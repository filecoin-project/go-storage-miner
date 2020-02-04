package storage

import (
	"context"
	"io"

	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("storageminer")

const SectorStorePrefix = "/sectors"

// SectorBuilderAPI provides a method set used by the miner in order to interact
// with the go-sectorbuilder package. This method set exposes a subset of the
// methods defined on the sectorbuilder.Interface interface.
type SectorBuilderAPI interface {
	SectorSize() uint64
	SealPreCommit(ctx context.Context, sectorID uint64, ticket ffi.SealTicket, pieces []sectorbuilder.PublicPieceInfo) (sectorbuilder.RawSealPreCommitOutput, error)
	SealCommit(ctx context.Context, sectorID uint64, ticket ffi.SealTicket, seed ffi.SealSeed, pieces []sectorbuilder.PublicPieceInfo, rspco sectorbuilder.RawSealPreCommitOutput) (proof []byte, err error)
	RateLimit() func()
	AddPiece(ctx context.Context, pieceSize uint64, sectorID uint64, file io.Reader, existingPieceSizes []uint64) (sectorbuilder.PublicPieceInfo, error)
	AcquireSectorId() (uint64, error)
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
	onSectorUpdated func(uint64, SectorState)
}

func NewMiner(api NodeAPI, ds datastore.Batching, sb SectorBuilderAPI) (*Miner, error) {
	return NewMinerWithOnSectorUpdated(api, ds, sb, nil)
}

func NewMinerWithOnSectorUpdated(api NodeAPI, ds datastore.Batching, sb SectorBuilderAPI, onSectorUpdated func(uint64, SectorState)) (*Miner, error) {
	return &Miner{
		api:             api,
		ds:              ds,
		sb:              sb,
		onSectorUpdated: onSectorUpdated,
	}, nil
}

// AllocatePiece produces information about where a piece of a given size can
// be written.
//
// TODO: This signature doesn't make much sense. Returning a sector ID here
// means that we won't have the ability to move the piece around (i.e. do
// intelligent bin packing) after allocating. -- @laser
func (m *Miner) AllocatePiece(size uint64) (sectorID uint64, offset uint64, err error) {
	return m.sealing.AllocatePiece(size)
}

// AllocateSectorID allocates a new sector ID.
func (m *Miner) AllocateSectorID() (sectorID uint64, err error) {
	return m.sb.AcquireSectorId()
}

// ForceSectorState puts a sector with given ID into the given state.
func (m *Miner) ForceSectorState(ctx context.Context, id uint64, state SectorState) error {
	return m.sealing.ForceSectorState(ctx, id, state)
}

// GetSectorInfo produces information about a sector managed by this storage
// miner, or an error if the miner does not manage a sector with the
// provided identity.
func (m *Miner) GetSectorInfo(sectorID uint64) (SectorInfo, error) {
	return m.sealing.GetSectorInfo(sectorID)
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
	if m.onSectorUpdated != nil {
		m.sealing = NewWithOnSectorUpdated(m.api, m.sb, m.ds, m.worker, m.maddr, m.onSectorUpdated)
	} else {
		m.sealing = New(m.api, m.sb, m.ds, m.worker, m.maddr)
	}

	go m.sealing.Run(ctx) // nolint: errcheck

	return nil
}

// SealPiece writes the provided piece to a newly-created sector which it
// immediately seals.
func (m *Miner) SealPiece(ctx context.Context, size uint64, r io.Reader, sectorID uint64, dealID uint64) error {
	return m.sealing.SealPiece(ctx, size, r, sectorID, dealID)
}

// Stop causes the miner to stop listening for sector state transitions. It is
// undefined behavior to call this method before calling Start. It is undefined
// behavior to call this method more than once. It is undefined behavior to call
// this method concurrently with any other Miner method.
func (m *Miner) Stop(ctx context.Context) error {
	return m.sealing.Stop(ctx)
}
