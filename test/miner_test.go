package test

import (
	"context"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/go-storage-mining/storage"
	"github.com/filecoin-project/lotus/api"
	"github.com/ipfs/go-datastore"
)

func TestSectorLifecycle(t *testing.T) {
	ctx := context.Background()

	builder := &fakeSectorBuilder{}

	miner, err := storage.NewMiner(&fakeNode{}, datastore.NewMapDatastore(), builder)
	if err != nil {
		t.Fatalf("failed to create miner: %s", err)
	}

	defer func() {
		if err := miner.Stop(ctx); err != nil {
			t.Errorf("failed to stop miner: %s", err)
		}
	}()

	pieceSize := sectorbuilder.UserBytesForSectorSize(builder.SectorSize()) / 4
	pieceReader := io.LimitReader(rand.New(rand.NewSource(42)), int64(pieceSize))

	sectorID, _, err := miner.AllocatePiece(pieceSize)
	if err != nil {
		t.Fatalf("failed to allocate: %s", err)
	}

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, sectorID, api.Packing).
		then(api.Unsealed).
		then(api.PreCommitting).
		then(api.PreCommitted).
		then(api.Committing).
		then(api.CommitWait).
		then(api.Proving).
		end()

	miner.OnSectorUpdated = onSectorUpdatedFunc

	if err = miner.Run(ctx); err != nil {
		t.Fatalf("failed to start miner: %s", err)
	}

	if err = miner.SealPiece(ctx, pieceSize, pieceReader, sectorID, 1); err != nil {
		t.Fatalf("failed to start piece-sealing flow: %s", err)
	}

	select {
	case <-doneCh:
		// success
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for sequence to complete: %s", getSequenceStatusFunc())
	}
}
