package storage

import (
	"context"
	"io"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-padreader"
	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/go-statemachine"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"golang.org/x/xerrors"
)

const SectorStorePrefix = "/sectors"

type Sealing struct {
	api NodeAPI

	maddr  address.Address
	worker address.Address

	sb      SectorBuilderAPI
	sectors *statemachine.StateGroup

	// onSectorUpdated is called each time a sector transitions from one state
	// to some other state, if defined. It is non-nil during test.
	onSectorUpdated func(uint64, SectorState)

	// runCompleteWg is incremented when Mining is created, and will prevent
	// new sectors from being added to the StateGroup before existing sectors
	// have been restarted. When Mining#Run exits, runCompleteWg is decremented.
	runCompleteWg sync.WaitGroup
}

func New(api NodeAPI, sb SectorBuilderAPI, ds datastore.Batching, worker address.Address, maddr address.Address) *Sealing {
	return NewWithOnSectorUpdated(api, sb, ds, worker, maddr, nil)
}

func NewWithOnSectorUpdated(api NodeAPI, sb SectorBuilderAPI, ds datastore.Batching, worker address.Address, maddr address.Address, onSectorUpdated func(uint64, SectorState)) *Sealing {
	s := &Sealing{
		api:             api,
		maddr:           maddr,
		sb:              sb,
		worker:          worker,
		onSectorUpdated: onSectorUpdated,
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

func (m *Sealing) AllocatePiece(size uint64) (sectorID uint64, offset uint64, err error) {
	if padreader.PaddedSize(size) != size {
		return 0, 0, xerrors.Errorf("cannot allocate unpadded piece")
	}

	sid, err := m.sb.AcquireSectorId() // TODO: Put more than one thing in a sector
	if err != nil {
		return 0, 0, xerrors.Errorf("acquiring sector ID: %w", err)
	}

	// offset hard-coded to 0 since we only put one thing in a sector for now
	return sid, 0, nil
}

func (m *Sealing) SealPiece(ctx context.Context, size uint64, r io.Reader, sectorID uint64, dealID uint64) error {
	log.Infof("Seal piece for deal %d", dealID)

	ppi, err := m.sb.AddPiece(ctx, size, sectorID, r, []uint64{})
	if err != nil {
		return xerrors.Errorf("adding piece to sector: %w", err)
	}

	return m.newSector(ctx, sectorID, dealID, ppi)
}

func (m *Sealing) newSector(ctx context.Context, sid uint64, dealID uint64, ppi sectorbuilder.PublicPieceInfo) error {
	m.runCompleteWg.Wait()

	log.Infof("Start sealing %d", sid)

	return m.sectors.Send(sid, SectorStart{
		id: sid,
		pieces: []Piece{
			{
				DealID: dealID,

				Size:  ppi.Size,
				CommP: ppi.CommP[:],
			},
		},
	})
}
