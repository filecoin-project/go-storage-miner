package storage

import (
	"context"
	"errors"
	"io"

	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/go-storage-miner/apis/sectorbuilder"

	sealing2 "github.com/filecoin-project/go-storage-miner/sealing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"golang.org/x/xerrors"
)

var log = logging.Logger("storageminer")

type Miner struct {
	api     node.Interface
	maddr   address.Address
	sb      sectorbuilder.Interface
	ds      datastore.Batching
	sealing *sealing2.Sealing

	// onSectorUpdated is called each time a sector transitions from one state
	// to some other state, if defined. It is non-nil during test.
	onSectorUpdated func(abi.SectorNumber, sealing2.SectorState)
}

func NewMiner(api node.Interface, ds datastore.Batching, sb sectorbuilder.Interface, maddr address.Address) (*Miner, error) {
	return NewMinerWithOnSectorUpdated(api, ds, sb, maddr, nil)
}

func NewMinerWithOnSectorUpdated(api node.Interface, ds datastore.Batching, sb sectorbuilder.Interface, maddr address.Address, onSectorUpdated func(abi.SectorNumber, sealing2.SectorState)) (*Miner, error) {
	return &Miner{
		api:             api,
		maddr:           maddr,
		sb:              sb,
		ds:              ds,
		sealing:         nil,
		onSectorUpdated: onSectorUpdated,
	}, nil
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
		m.sealing = sealing2.NewSealingWithOnSectorUpdated(m.api, m.sb, m.ds, m.maddr, m.onSectorUpdated)
	} else {
		m.sealing = sealing2.NewSealing(m.api, m.sb, m.ds, m.maddr)
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
	waddr, err := m.api.GetMinerWorkerAddressFromChainHead(ctx, m.maddr)
	if err != nil {
		return xerrors.Errorf("error acquiring worker address: %w", err)
	}

	has, err := m.api.WalletHas(ctx, waddr)
	if err != nil {
		return xerrors.Errorf("failed to check wallet for worker key: %w", err)
	}

	if !has {
		return errors.New("key for worker not found in local wallet")
	}

	log.Infof("starting up miner %s, worker addr %s", m.maddr, waddr)

	return nil
}
