package storage

import (
	"context"
	"github.com/filecoin-project/filecoin-ffi"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/pkg/errors"
	"io"

	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/lotus/chain/address"
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/lib/statestore"
)

var log = logging.Logger("storageminer")

const SectorStorePrefix = "/sectors"

type SectorBuilderAPI interface {
	SectorSize() uint64
	SealPreCommit(sectorID uint64, ticket ffi.SealTicket, pieces []sectorbuilder.PublicPieceInfo) (sectorbuilder.RawSealPreCommitOutput, error)
	SealCommit(sectorID uint64, ticket ffi.SealTicket, seed ffi.SealSeed, pieces []sectorbuilder.PublicPieceInfo, rspco sectorbuilder.RawSealPreCommitOutput) (proof []byte, err error)
	RateLimit()
	AddPiece(pieceSize uint64, sectorId uint64, file io.Reader, existingPieceSizes []uint64) (sectorbuilder.PublicPieceInfo, error)
	AcquireSectorId() (uint64, error)
}


type Miner struct {
	api    NodeAPI
	events *events.Events
	h      host.Host

	maddr  address.Address
	worker address.Address

	// Sealing
	sb      SectorBuilderAPI
	sectors *statestore.StateStore

	sectorIncoming chan *SectorInfo
	sectorUpdated  chan sectorUpdate
	stop           chan struct{}
	stopped        chan struct{}
}

func NewMiner(api NodeAPI, ds datastore.Batching, sb *sectorbuilder.SectorBuilder) (*Miner, error) {
	return &Miner{
		api: api,

		sb:    sb,

		sectors: statestore.New(namespace.Wrap(ds, datastore.NewKey(SectorStorePrefix))),

		sectorIncoming: make(chan *SectorInfo),
		sectorUpdated:  make(chan sectorUpdate),
		stop:           make(chan struct{}),
		stopped:        make(chan struct{}),
	}, nil
}

func (m *Miner) Run(ctx context.Context) error {
	if err := m.sectorStateLoop(ctx); err != nil {
		log.Errorf("%+v", err)
		return errors.Errorf("failed to startup sector state loop: %w", err)
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
