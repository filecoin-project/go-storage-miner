package test

import (
	"context"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ipfs/go-datastore"

	"github.com/filecoin-project/go-storage-mining/storage"
)

func TestSuccessfulPieceSealingFlow(t *testing.T) {
	ctx := context.Background()

	miner, err := storage.NewMiner(&fakeNode{}, datastore.NewMapDatastore(), &fakeSectorBuilder{})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	pieceSize := uint64(1016 / 4)
	pieceReader := io.LimitReader(rand.New(rand.NewSource(42)), int64(pieceSize))

	sectorID, _, err := miner.AllocatePiece(pieceSize)
	if err != nil {
		t.Fatalf("failed to allocate: %s", err)
	}

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, sectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.PreCommitted).
		then(storage.Committing).
		then(storage.CommitWait).
		then(storage.Proving).
		end()

	// register our event handler
	miner.OnSectorUpdated = onSectorUpdatedFunc

	// start the internal runloop
	require.NoError(t, miner.Run(ctx))

	// kick off the state machine
	require.NoError(t, miner.SealPiece(ctx, pieceSize, pieceReader, sectorID, 1))

	select {
	case <-doneCh:
		// success; we're done
	case <-time.After(2 * time.Second):
		// if we've stubbed everything properly, 2 seconds should be sufficient
		t.Fatalf("timed out waiting for sequence to complete: %s", getSequenceStatusFunc())
	}
}

func TestSealPieceCreatesSelfDealsToFillSector(t *testing.T) {
	// creates two self-deals to fill the remaining space in the sector
	t.Skip("NodeAPI#SendSelfDeals should be called with two pieces")
}

func TestHandlesSectorBuilderPreCommitError(t *testing.T) {
	// SectorBuilderAPI#SealPreCommit produced an error
	t.Skip("the sector should end up in a SealFailed state")
}

func TestHandlesPreCommitSectorSendFailed(t *testing.T) {
	// NodeAPI#SendPreCommitSector produced an error
	t.Skip("the sector should end up in a PreCommitFailed state (or perhaps PreCommitting if error was transient)")
}

func TestHandlesSectorBuilderCommitError(t *testing.T) {
	// SectorBuilderAPI#SealCommit produced an error
	t.Skip("the sector should end up in a SealCommitFailed state")
}

func TestHandlesCommitSectorMessageSendFailed(t *testing.T) {
	// NodeAPI#SendProveCommitSector produced an error
	t.Skip("the sector should end up in a CommitFailed state (or perhaps Committing if error was transient)")
}

func TestHandlesCommitSectorMessageWaitFailure(t *testing.T) {
	// // NodeAPI#WaitForProveCommitSector produced an error
	t.Skip("the sector should end up in a CommitFailed state (or perhaps Committing if error was transient)")
}

func TestHandlesCommitSectorMessageNeverIncludedInBlock(t *testing.T) {
	// what happens if the CommitSector message, which we're waiting on, doesn't
	// show up in the chain in a timely manner?
	t.Skip("the sector should end up in a CommitFailed state (or perhaps Committing if error was transient)")
}

func TestSealSeedInvalidated(t *testing.T) {
	// SetSealSeedHandler called our "seed available" handler, and then some
	// time later called our "seed invalidated" handler
	t.Skip("the sector should go back to a PreCommitted state")
}
