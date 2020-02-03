package test

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-storage-miner"
)

const DefaultDealID = 42
const DefaultSectorID = 42
const UserBytesOneKiBSector = 1016 // also known as "unpadded" bytes
const PaddedBytesOneKiBSector = 1024

func TestSuccessfulPieceSealingFlow(t *testing.T) {
	ctx := context.Background()

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.WaitSeed).
		then(storage.Committing).
		then(storage.CommitWait).
		then(storage.Proving).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(newFakeNode(), datastore.NewMapDatastore(), &fakeSectorBuilder{}, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// start the internal runloop
	require.NoError(t, miner.Run(ctx))

	// kick off the state machine
	require.NoError(t, miner.SealPiece(ctx, UserBytesOneKiBSector, io.LimitReader(rand.New(rand.NewSource(42)), int64(UserBytesOneKiBSector)), DefaultSectorID, DefaultDealID))

	select {
	case <-doneCh:
		// success; we're done
	case <-time.After(2 * time.Second):
		// if we've stubbed everything properly, 2 seconds should be sufficient
		t.Fatalf("timed out waiting for sequence to complete: %s", getSequenceStatusFunc())
	}
}

func TestSealPieceCreatesSelfDealsToFillSector(t *testing.T) {
	ctx := context.Background()

	// we'll assert the contents of this slice at the end of the test
	var selfDealPieceSizes []uint64

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.sendSelfDeals = func(i context.Context, info ...storage.PieceInfo) (cid cid.Cid, e error) {
			selfDealPieceSizes = append(selfDealPieceSizes, info[0].Size)
			selfDealPieceSizes = append(selfDealPieceSizes, info[1].Size)

			return createCidForTesting(42), nil
		}

		n.waitForSelfDeals = func(i context.Context, i2 cid.Cid) (dealIds []uint64, exitCode uint8, err error) {
			return []uint64{42, 43}, 0, nil
		}

		return n
	}()

	sb := &fakeSectorBuilder{}

	// a sequence of sector state transitions we expect to observe
	sectorID, err := sb.AcquireSectorId()
	require.NoError(t, err)

	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, sectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.WaitSeed).
		then(storage.Committing).
		then(storage.CommitWait).
		then(storage.Proving).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), sb, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// create a piece which fills up a quarter of the sector
	pieceSize := uint64(1016 / 4)
	pieceReader := io.LimitReader(rand.New(rand.NewSource(42)), int64(pieceSize))

	// kick off state transitions
	require.NoError(t, miner.Run(ctx))
	require.NoError(t, miner.SealPiece(ctx, pieceSize, pieceReader, sectorID, DefaultDealID))

	select {
	case <-doneCh:
		require.Equal(t, 2, len(selfDealPieceSizes), "expected two self-deals")
		assert.Equal(t, uint64(254), selfDealPieceSizes[0])
		assert.Equal(t, uint64(508), selfDealPieceSizes[1])
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for sequence to complete: %s", getSequenceStatusFunc())
	}
}

func TestHandlesPreCommitSectorSendFailed(t *testing.T) {
	ctx := context.Background()

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.sendPreCommitSector = func(ctx context.Context, sectorID uint64, commR []byte, ticket storage.SealTicket, pieces ...storage.Piece) (i cid.Cid, e error) {
			return cid.Undef, errors.New("expected error")
		}

		return n
	}()

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.PreCommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// kick off state transitions
	require.NoError(t, miner.Run(ctx))
	require.NoError(t, miner.SealPiece(ctx, UserBytesOneKiBSector, io.LimitReader(rand.New(rand.NewSource(42)), int64(UserBytesOneKiBSector)), DefaultSectorID, DefaultDealID))

	select {
	case <-doneCh:
		// done
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for sequence to complete: %s", getSequenceStatusFunc())
	}
}

func TestHandlesProveCommitSectorMessageSendFailed(t *testing.T) {
	ctx := context.Background()

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.sendProveCommitSector = func(ctx context.Context, sectorID uint64, proof []byte, dealIds ...uint64) (i cid.Cid, e error) {
			return cid.Undef, errors.New("expected error")
		}

		return n
	}()

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.WaitSeed).
		then(storage.Committing).
		then(storage.CommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// kick off state transitions
	require.NoError(t, miner.Run(ctx))
	require.NoError(t, miner.SealPiece(ctx, UserBytesOneKiBSector, io.LimitReader(rand.New(rand.NewSource(42)), int64(UserBytesOneKiBSector)), DefaultSectorID, DefaultDealID))

	select {
	case <-doneCh:
		// done
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for sequence to complete: %s", getSequenceStatusFunc())
	}
}

func TestHandlesCommitSectorMessageWaitFailure(t *testing.T) {
	ctx := context.Background()

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.waitForProveCommitSector = func(i context.Context, i2 cid.Cid) (uint8, error) {
			return 0, errors.New("expected behavior")
		}

		return n
	}()

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.WaitSeed).
		then(storage.Committing).
		then(storage.CommitWait).
		then(storage.CommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// kick off state transitions
	require.NoError(t, miner.Run(ctx))
	require.NoError(t, miner.SealPiece(ctx, UserBytesOneKiBSector, io.LimitReader(rand.New(rand.NewSource(42)), int64(UserBytesOneKiBSector)), DefaultSectorID, DefaultDealID))

	select {
	case <-doneCh:
		// done
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for sequence to complete: %s", getSequenceStatusFunc())
	}
}

func TestHandlesSectorBuilderPreCommitError(t *testing.T) {
	// SectorBuilderAPI#SealPreCommit produced an error
	t.Skip("the sector should end up in a SealFailed state")
}

func TestHandlesSectorBuilderCommitError(t *testing.T) {
	// SectorBuilderAPI#SealCommit produced an error
	t.Skip("the sector should end up in a SealCommitFailed state")
}

func TestHandlesCommitSectorMessageNeverIncludedInBlock(t *testing.T) {
	// what happens if the CommitSector message, which we're waiting on, doesn't
	// show up in the chain in a timely manner?
	t.Skip("the sector should end up in a CommitFailed state (or perhaps Committing if error was transient)")
}

func TestSealSeedInvalidated(t *testing.T) {
	// GetSealSeed called our "seed available" handler, and then some
	// time later called our "seed invalidated" handler
	t.Skip("the sector should go back to a PreCommitted state")
}
