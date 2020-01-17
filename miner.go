package storage

import (
	"context"
	"io"

	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/go-statestore"
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
	api NodeAPI

	maddr  address.Address
	worker address.Address

	sb      SectorBuilderAPI
	sectors *statestore.StateStore

	// OnSectorUpdated is called each time a sector transitions from one state
	// to some other state, if defined. It is non-nil during test.
	OnSectorUpdated func(uint64, SectorState)

	sectorIncoming chan *SectorInfo
	sectorUpdated  chan sectorUpdate
	stop           chan struct{}
	stopped        chan struct{}
}

var _ Interface = new(Miner)

func NewMiner(api NodeAPI, ds datastore.Batching, sb SectorBuilderAPI) (*Miner, error) {
	return &Miner{
		api: api,

		sb: sb,

		sectors: statestore.New(namespace.Wrap(ds, datastore.NewKey(SectorStorePrefix))),

		sectorIncoming: make(chan *SectorInfo),
		sectorUpdated:  make(chan sectorUpdate),
		stop:           make(chan struct{}),
		stopped:        make(chan struct{}),
	}, nil
}

func (m *Miner) Start(ctx context.Context) error {
	if err := m.sectorStateLoop(ctx); err != nil {
		log.Errorf("%+v", err)
		return errors.Errorf("failed to start sector state loop: %s", err)
	}

	return nil
}

func (m *Miner) Stop(ctx context.Context) error {
	close(m.stop)
	select {
	case <-m.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
