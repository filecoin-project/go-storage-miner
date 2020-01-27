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
	AddPiece(pieceSize uint64, sectorID uint64, file io.Reader, existingPieceSizes []uint64) (sectorbuilder.PublicPieceInfo, error)
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
	return NewMinerForTesting(api, ds, sb, nil)
}

func NewMinerForTesting(api NodeAPI, ds datastore.Batching, sb SectorBuilderAPI, onSectorUpdated func(uint64, SectorState)) (*Miner, error) {
	return &Miner{
		api:             api,
		ds:              ds,
		sb:              sb,
		onSectorUpdated: onSectorUpdated,
	}, nil
}

func (m *Miner) Start(ctx context.Context) error {
	if m.onSectorUpdated != nil {
		m.sealing = NewForTesting(m.api, m.sb, m.ds, m.worker, m.maddr, m.onSectorUpdated)
	} else {
		m.sealing = New(m.api, m.sb, m.ds, m.worker, m.maddr)
	}

	go m.sealing.Run(ctx) // nolint: errcheck

	return nil
}

func (m *Miner) Stop(ctx context.Context) error {
	defer m.sealing.Stop(ctx) // nolint: errcheck

	return nil
}

// AllocateSectorID allocates a new sector ID.
func (m *Miner) AllocateSectorID() (sectorID uint64, err error) {
	return m.sb.AcquireSectorId()
}

// PledgeSector allocates a new sector, fills it with self-deal junk, and
// seals that sector.
func (m *Miner) PledgeSector() error {
	return m.sealing.PledgeSector()
}

// SealPiece writes the provided piece to a newly-created sector which it
// immediately seals.
func (m *Miner) SealPiece(ctx context.Context, size uint64, r io.Reader, sectorID uint64, dealID uint64) error {
	return m.sealing.SealPiece(ctx, size, r, sectorID, dealID)
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
