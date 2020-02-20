package sealing

import (
	"context"
	"io"
	"sync"

	commcid "github.com/filecoin-project/go-fil-commcid"

	"github.com/filecoin-project/go-storage-miner/policies/selfdeal"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-padreader"
	"github.com/filecoin-project/go-statemachine"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/go-storage-miner/apis/sectorbuilder"
)

const SectorStorePrefix = "/sectors"

var log = logging.Logger("sectors")

type Sealing struct {
	api node.Interface

	maddr address.Address

	sb      sectorbuilder.Interface
	sectors *statemachine.StateGroup

	// used to compute self-deal schedule (e.g. start and end epochs), this
	// value is inherited from the Miner which creates this Sealing struct
	selfDealPolicy selfdeal.Policy

	// onSectorUpdated is called each time a sector transitions from one state
	// to some other state, if defined. It is non-nil during test.
	onSectorUpdated func(abi.SectorNumber, SectorState)

	// runCompleteWg is incremented when Mining is created, and will prevent
	// new sectors from being added to the StateGroup before existing sectors
	// have been restarted. When Mining#Run exits, runCompleteWg is decremented.
	runCompleteWg sync.WaitGroup
}

func NewSealing(api node.Interface, sb sectorbuilder.Interface, ds datastore.Batching, maddr address.Address, sdp selfdeal.Policy) *Sealing {
	return NewSealingWithOnSectorUpdated(api, sb, ds, maddr, sdp, nil)
}

func NewSealingWithOnSectorUpdated(api node.Interface, sb sectorbuilder.Interface, ds datastore.Batching, maddr address.Address, sdp selfdeal.Policy, onSectorUpdated func(abi.SectorNumber, SectorState)) *Sealing {
	s := &Sealing{
		api:             api,
		maddr:           maddr,
		sb:              sb,
		onSectorUpdated: onSectorUpdated,
		selfDealPolicy:  sdp,
	}

	s.runCompleteWg.Add(1)

	s.sectors = statemachine.New(namespace.Wrap(ds, datastore.NewKey(SectorStorePrefix)), s, SectorInfo{})

	return s
}

func (m *Sealing) Run(ctx context.Context) error {
	defer m.runCompleteWg.Done()

	if err := m.restartSectors(ctx); err != nil {
		log.Errorf("%+v", err)
		return xerrors.Errorf("failed load sector states: %w", err)
	}

	return nil
}

func (m *Sealing) Stop(ctx context.Context) error {
	m.runCompleteWg.Add(1)

	return m.sectors.Stop(ctx)
}

func (m *Sealing) AllocatePiece(size abi.UnpaddedPieceSize) (sectorNum abi.SectorNumber, offset uint64, err error) {
	if padreader.PaddedSize(uint64(size)) != size {
		return 0, 0, xerrors.Errorf("cannot allocate unpadded piece")
	}

	sid, err := m.sb.AcquireSectorNumber() // TODO: Put more than one thing in a sector
	if err != nil {
		return 0, 0, xerrors.Errorf("acquiring sector ID: %w", err)
	}

	// offset hard-coded to 0 since we only put one thing in a sector for now
	return sid, 0, nil
}

func (m *Sealing) SealPiece(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, sectorNum abi.SectorNumber, dealInfo node.DealInfo) error {
	log.Infof("Seal piece for deal %d", dealInfo.DealID)

	ppi, err := m.sb.AddPiece(ctx, size, sectorNum, r, []abi.UnpaddedPieceSize{})
	if err != nil {
		return xerrors.Errorf("adding piece to sector: %w", err)
	}

	return m.newSector(ctx, sectorNum, node.PieceWithDealInfo{
		Piece: abi.PieceInfo{
			Size:     size.Padded(),
			PieceCID: commcid.PieceCommitmentV1ToCID(ppi.CommP[:]),
		},
		DealInfo: dealInfo,
	})
}

func (m *Sealing) newSector(ctx context.Context, sectorNum abi.SectorNumber, piecesInSector ...node.PieceWithDealInfo) error {
	m.runCompleteWg.Wait()

	log.Infof("Start sealing %d", sectorNum)

	return m.sectors.Send(uint64(sectorNum), SectorStart{
		num:    sectorNum,
		pieces: piecesInSector,
	})
}
