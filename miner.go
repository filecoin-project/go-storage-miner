package storage

import (
	"context"
	"errors"
	"io"

	"github.com/filecoin-project/specs-actors/actors/abi"

	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"golang.org/x/xerrors"
)

var log = logging.Logger("storageminer")

// SectorBuilderAPI provides a method set used by the miner in order to interact
// with the go-sectorbuilder package. This method set exposes a subset of the
// methods defined on the sectorbuilder.Interface interface.
type SectorBuilderAPI interface {
	AcquireSectorNumber() (abi.SectorNumber, error)
	AddPiece(ctx context.Context, pieceSize abi.UnpaddedPieceSize, sectorNumber abi.SectorNumber, file io.Reader, existingPieceSizes []abi.UnpaddedPieceSize) (sectorbuilder.PublicPieceInfo, error)
	DropStaged(context.Context, abi.SectorNumber) error
	FinalizeSector(context.Context, abi.SectorNumber) error
	RateLimit() func()
	SealCommit(ctx context.Context, sectorNumber abi.SectorNumber, ticket ffi.SealTicket, seed ffi.SealSeed, pieces []sectorbuilder.PublicPieceInfo, rspco sectorbuilder.RawSealPreCommitOutput) (proof []byte, err error)
	SealPreCommit(ctx context.Context, sectorNumber abi.SectorNumber, ticket ffi.SealTicket, pieces []sectorbuilder.PublicPieceInfo) (sectorbuilder.RawSealPreCommitOutput, error)
	SectorSize() abi.SectorSize
}

var _ SectorBuilderAPI = new(sectorbuilder.SectorBuilder)

type Miner struct {
	api     NodeAPI
	maddr   address.Address
	worker  address.Address
	sb      SectorBuilderAPI
	ds      datastore.Batching
	sealing *Sealing

	// onSectorUpdated is called each time a sector transitions from one state
	// to some other state, if defined. It is non-nil during test.
	onSectorUpdated func(abi.SectorNumber, SectorState)
}

func NewMiner(api NodeAPI, ds datastore.Batching, sb SectorBuilderAPI, maddr, waddr address.Address) (*Miner, error) {
	return NewMinerWithOnSectorUpdated(api, ds, sb, maddr, waddr, nil)
}

func NewMinerWithOnSectorUpdated(api NodeAPI, ds datastore.Batching, sb SectorBuilderAPI, maddr, waddr address.Address, onSectorUpdated func(abi.SectorNumber, SectorState)) (*Miner, error) {
	return &Miner{
		api:             api,
		maddr:           maddr,
		worker:          waddr,
		sb:              sb,
		ds:              ds,
		sealing:         nil,
		onSectorUpdated: onSectorUpdated,
	}, nil
}

// AllocatePiece produces information about where a piece of a given size can
// be written.
//
// TODO: This signature doesn't make much sense. Returning a sector ID here
// means that we won't have the ability to move the piece around (i.e. do
// intelligent bin packing) after allocating. -- @laser
func (m *Miner) AllocatePiece(size abi.UnpaddedPieceSize) (sectorNum abi.SectorNumber, offset uint64, err error) {
	return m.sealing.AllocatePiece(size)
}

// AllocateSectorID allocates a new sector ID.
func (m *Miner) AllocateSectorID() (sectorNum abi.SectorNumber, err error) {
	return m.sb.AcquireSectorNumber()
}

// ForceSectorState puts a sector with given ID into the given state.
func (m *Miner) ForceSectorState(ctx context.Context, num abi.SectorNumber, state SectorState) error {
	return m.sealing.ForceSectorState(ctx, num, state)
}

// GetSectorInfo produces information about a sector managed by this storage
// miner, or an error if the miner does not manage a sector with the
// provided identity.
func (m *Miner) GetSectorInfo(sectorNum abi.SectorNumber) (SectorInfo, error) {
	return m.sealing.GetSectorInfo(sectorNum)
}

// ListSectors lists all the sectors managed by this storage miner (sealed
// or otherwise).
func (m *Miner) ListSectors() ([]SectorInfo, error) {
	return m.sealing.ListSectors()
}

// PledgeSector allocates a new sector, fills it with self-deal junk, and
// seals that sector.
func (m *Miner) PledgeSector() error {
	return m.sealing.PledgeSector()
}

// Run starts the Miner, which causes it (and its collaborating objects) to
// start listening for sector state-transitions. It is undefined behavior to
// call this method more than once. It is undefined behavior to call this method
// concurrently with any other Miner method.
func (m *Miner) Run(ctx context.Context) error {
	if err := m.runPreflightChecks(ctx); err != nil {
		return xerrors.Errorf("miner preflight checks failed: %w", err)
	}

	if m.onSectorUpdated != nil {
		m.sealing = NewSealingWithOnSectorUpdated(m.api, m.sb, m.ds, m.worker, m.maddr, m.onSectorUpdated)
	} else {
		m.sealing = NewSealing(m.api, m.sb, m.ds, m.worker, m.maddr)
	}

	go m.sealing.Run(ctx) // nolint: errcheck

	return nil
}

// SealPiece writes the provided piece to a newly-created sector which it
// immediately seals.
func (m *Miner) SealPiece(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, sectorNum abi.SectorNumber, dealID abi.DealID) error {
	return m.sealing.SealPiece(ctx, size, r, sectorNum, dealID)
}

// Stop causes the miner to stop listening for sector state transitions. It is
// undefined behavior to call this method before calling Start. It is undefined
// behavior to call this method more than once. It is undefined behavior to call
// this method concurrently with any other Miner method.
func (m *Miner) Stop(ctx context.Context) error {
	return m.sealing.Stop(ctx)
}

func (m *Miner) runPreflightChecks(ctx context.Context) error {
	if m.worker == address.Undef {
		return xerrors.New("miner worker address has not been set")
	}

	has, err := m.api.WalletHas(ctx, m.worker)
	if err != nil {
		return xerrors.Errorf("failed to check wallet for worker key: %w", err)
	}

	if !has {
		return errors.New("key for worker not found in local wallet")
	}

	log.Infof("starting up miner %s, worker addr %s", m.maddr, m.worker)
	return nil
}
