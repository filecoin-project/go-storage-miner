package storage

import (
	"context"
	"errors"
	"io"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	sectorstorage "github.com/filecoin-project/sector-storage"
	"github.com/filecoin-project/sector-storage/ffiwrapper"
	"github.com/filecoin-project/specs-actors/actors/abi"
	fsm "github.com/filecoin-project/storage-fsm"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

var log = logging.Logger("storageminer")

type MinerAddressGetter = func(context.Context) (address.Address, error)

type Miner struct {
	api node.Interface              // this Miner's interface to the outside world
	dst datastore.Batching          // responsible for persisting/loading state machine to/from disk
	fsm *fsm.Sealing                // finite-state machine for sectors
	mag MinerAddressGetter          // used to acquire the miner's address, which can change over time
	pcp fsm.PreCommitPolicy         // used to set pre-commit expiry
	sid fsm.SectorIDCounter         // used to ensure that sectors are numbered uniquely for a given miner
	mgr sectorstorage.SectorManager // a concrete implementation of sector-storage
	ver ffiwrapper.Verifier         // an interface to proof verification
}

func NewMiner(api node.Interface, minerAddressGetter MinerAddressGetter, dst datastore.Batching, mgr sectorstorage.SectorManager, sid fsm.SectorIDCounter, ver ffiwrapper.Verifier, pcp fsm.PreCommitPolicy) *Miner {
	return &Miner{
		api: api,
		mgr: mgr,
		mag: minerAddressGetter,
		dst: dst,
		sid: sid,
		ver: ver,
		pcp: pcp,
	}
}

// Run starts the Miner, which causes it (and its collaborating objects) to
// start listening for sector state-transitions. It is undefined behavior to
// call this method more than once. It is undefined behavior to call this method
// concurrently with any other Miner method. This method blocks until all
// sectors have been restarted (in the finite-state machine), and returns any
// preflight check-errors or errors encountered when restarting sectors.
func (m *Miner) Run(ctx context.Context) error {
	tok, _, err := m.api.ChainHead(ctx)
	if err != nil {
		return xerrors.Errorf("failed to get chain head during startup: %w", err)
	}

	maddr, err := m.mag(ctx)
	if err != nil {
		return xerrors.Errorf("failed to get miner address on startup: %w", err)
	}

	waddr, err := m.api.GetMinerWorkerAddress(ctx, maddr, tok)
	if err != nil {
		return xerrors.Errorf("failed to get miner worker address on startup: %w", err)
	}

	if err := m.runPreflightChecks(ctx, waddr); err != nil {
		return xerrors.Errorf("miner preflight checks failed: %w", err)
	}

	m.fsm = fsm.New(m.api, m.api, maddr, waddr, m.dst, m.mgr, m.sid, m.ver, m.api.GetSealTicket, m.pcp)

	log.Infof("starting up miner %s, worker addr %s", maddr, waddr)

	return m.fsm.Run(ctx)
}

// SealPiece writes the provided piece to a newly-created sector which it
// immediately seals.
func (m *Miner) SealPiece(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, sectorNum abi.SectorNumber, deal fsm.DealInfo) error {
	return m.fsm.SealPiece(ctx, size, r, sectorNum, deal)
}

// Stop causes the miner to stop listening for sector state transitions. It is
// undefined behavior to call this method before calling Start. It is undefined
// behavior to call this method more than once. It is undefined behavior to call
// this method concurrently with any other Miner method.
func (m *Miner) Stop(ctx context.Context) error {
	return m.fsm.Stop(ctx)
}

// AllocatePiece produces information about where a piece of a given size can
// be written.
//
// TODO: This signature doesn't make much sense. Returning a sector ID here
// means that we won't have the ability to move the piece around (i.e. do
// intelligent bin packing) after allocating. -- @laser
func (m *Miner) AllocatePiece(size abi.UnpaddedPieceSize) (sectorNum abi.SectorNumber, offset uint64, err error) {
	return m.fsm.AllocatePiece(size)
}

// ListSectors lists all the sectors managed by this storage miner (sealed
// or otherwise).
func (m *Miner) ListSectors() ([]fsm.SectorInfo, error) {
	return m.fsm.ListSectors()
}

// GetSectorInfo produces information about a sector managed by this storage
// miner, or an error if the miner does not manage a sector with the
// provided identity.
func (m *Miner) GetSectorInfo(sectorNum abi.SectorNumber) (fsm.SectorInfo, error) {
	return m.fsm.GetSectorInfo(sectorNum)
}

// PledgeSector allocates a new sector, fills it with self-deal junk, and
// seals that sector.
func (m *Miner) PledgeSector(ctx context.Context) error {
	// TODO: plumb context into the FSM
	return m.fsm.PledgeSector()
}

// ForceSectorState puts a sector with given ID into the given state.
func (m *Miner) ForceSectorState(ctx context.Context, num abi.SectorNumber, state fsm.SectorState) error {
	return m.fsm.ForceSectorState(ctx, num, state)
}

func (m *Miner) runPreflightChecks(ctx context.Context, waddr address.Address) error {
	has, err := m.api.WalletHas(ctx, waddr)
	if err != nil {
		return xerrors.Errorf("failed to check wallet for worker key: %w", err)
	}

	if !has {
		return errors.New("key for worker not found in local wallet")
	}

	return nil
}
