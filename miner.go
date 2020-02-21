package storage

import (
	"context"
	"errors"
	"io"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/go-storage-miner/apis/sectorbuilder"
	"github.com/filecoin-project/go-storage-miner/policies/precommit"
	"github.com/filecoin-project/go-storage-miner/policies/selfdeal"
	"github.com/filecoin-project/go-storage-miner/sealing"
)

var log = logging.Logger("storageminer")

type Miner struct {
	api     node.Interface
	maddr   address.Address
	sb      sectorbuilder.Interface
	ds      datastore.Batching
	sealing *sealing.Sealing
}

func NewMiner(api node.Interface, ds datastore.Batching, sb sectorbuilder.Interface, maddr address.Address, sdp selfdeal.Policy, pcp precommit.Policy) (*Miner, error) {
	return NewMinerWithOnSectorUpdated(api, ds, sb, maddr, sdp, pcp, nil)
}

func NewMinerWithOnSectorUpdated(api node.Interface, ds datastore.Batching, sb sectorbuilder.Interface, maddr address.Address, sdp selfdeal.Policy, pcp precommit.Policy, onSectorUpdated func(abi.SectorNumber, sealing.SectorState)) (*Miner, error) {
	m := &Miner{
		api:     api,
		maddr:   maddr,
		sb:      sb,
		ds:      ds,
		sealing: nil,
	}

	if onSectorUpdated != nil {
		m.sealing = sealing.NewSealingWithOnSectorUpdated(m.api, m.sb, m.ds, m.maddr, sdp, pcp, onSectorUpdated)
	} else {
		m.sealing = sealing.NewSealing(m.api, m.sb, m.ds, m.maddr, sdp, pcp)
	}

	return m, nil
}

// Run starts the Miner, which causes it (and its collaborating objects) to
// start listening for sector state-transitions. It is undefined behavior to
// call this method more than once. It is undefined behavior to call this method
// concurrently with any other Miner method.
func (m *Miner) Run(ctx context.Context) error {
	if err := m.runPreflightChecks(ctx); err != nil {
		return xerrors.Errorf("miner preflight checks failed: %w", err)
	}

	go m.sealing.Run(ctx) // nolint: errcheck

	return nil
}

// SealPiece writes the provided piece to a newly-created sector which it
// immediately seals.
func (m *Miner) SealPiece(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, sectorNum abi.SectorNumber, deal node.DealInfo) error {
	return m.sealing.SealPiece(ctx, size, r, sectorNum, deal)
}

// Stop causes the miner to stop listening for sector state transitions. It is
// undefined behavior to call this method before calling Start. It is undefined
// behavior to call this method more than once. It is undefined behavior to call
// this method concurrently with any other Miner method.
func (m *Miner) Stop(ctx context.Context) error {
	return m.sealing.Stop(ctx)
}

func (m *Miner) runPreflightChecks(ctx context.Context) error {
	tok, _, err := m.api.GetChainHead(ctx)
	if err != nil {
		return xerrors.Errorf("failed to get chain head: %w", err)
	}

	waddr, err := m.api.GetMinerWorkerAddress(ctx, tok)
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
