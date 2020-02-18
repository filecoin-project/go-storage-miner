package test

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"

	"github.com/filecoin-project/go-storage-miner"
	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/go-storage-miner/sealing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const DefaultDealID = 42
const DefaultSectorID = 42
const UserBytesOneKiBSector = 1016 // also known as "unpadded" bytes

func TestSuccessfulPieceSealingFlow(t *testing.T) {
	ctx := context.Background()

	maddr, err := address.NewIDAddress(55)
	require.NoError(t, err)

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, sealing.Packing).
		then(sealing.Unsealed).
		then(sealing.PreCommitting).
		then(sealing.WaitSeed).
		then(sealing.Committing).
		then(sealing.CommitWait).
		then(sealing.FinalizeSector).
		then(sealing.Proving).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(newFakeNode(), datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, onSectorUpdatedFunc)
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

	maddr, err := address.NewIDAddress(55)
	require.NoError(t, err)

	// we'll assert the contents of this slice at the end of the test
	var selfDealPieceSizes []abi.UnpaddedPieceSize

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.sendSelfDeals = func(i context.Context, info ...abi.PieceInfo) (cid cid.Cid, e error) {
			selfDealPieceSizes = append(selfDealPieceSizes, info[0].Size.Unpadded())
			selfDealPieceSizes = append(selfDealPieceSizes, info[1].Size.Unpadded())

			return createCidForTesting(42), nil
		}

		n.waitForSelfDeals = func(i context.Context, i2 cid.Cid) (dealIds []abi.DealID, exitCode uint8, err error) {
			return []abi.DealID{abi.DealID(42), abi.DealID(43)}, 0, nil
		}

		return n
	}()

	sb := &fakeSectorBuilder{}

	// a sequence of sector state transitions we expect to observe
	sectorID, err := sb.AcquireSectorNumber()
	require.NoError(t, err)

	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, sectorID, sealing.Packing).
		then(sealing.Unsealed).
		then(sealing.PreCommitting).
		then(sealing.WaitSeed).
		then(sealing.Committing).
		then(sealing.CommitWait).
		then(sealing.FinalizeSector).
		then(sealing.Proving).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), sb, maddr, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// create a piece which fills up a quarter of the sector
	pieceSize := abi.UnpaddedPieceSize(uint64(1016 / 4))
	pieceReader := io.LimitReader(rand.New(rand.NewSource(42)), int64(pieceSize))

	// kick off state transitions
	require.NoError(t, miner.Run(ctx))
	require.NoError(t, miner.SealPiece(ctx, pieceSize, pieceReader, sectorID, DefaultDealID))

	select {
	case <-doneCh:
		require.Equal(t, 2, len(selfDealPieceSizes), "expected two self-deals")
		assert.Equal(t, int(abi.UnpaddedPieceSize(254)), int(selfDealPieceSizes[0]))
		assert.Equal(t, int(abi.UnpaddedPieceSize(508)), int(selfDealPieceSizes[1]))
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for sequence to complete: %s", getSequenceStatusFunc())
	}
}

func TestHandlesPreCommitSectorSendFailed(t *testing.T) {
	ctx := context.Background()

	maddr, err := address.NewIDAddress(55)
	require.NoError(t, err)

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.sendPreCommitSector = func(ctx context.Context, sectorNum abi.SectorNumber, commR []byte, ticket node.SealTicket, pieces ...node.Piece) (i cid.Cid, e error) {
			return cid.Undef, errors.New("expected error")
		}

		return n
	}()

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, sealing.Packing).
		then(sealing.Unsealed).
		then(sealing.PreCommitting).
		then(sealing.PreCommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, onSectorUpdatedFunc)
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

	maddr, err := address.NewIDAddress(55)
	require.NoError(t, err)

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.sendProveCommitSector = func(ctx context.Context, sectorNum abi.SectorNumber, proof []byte, dealIds ...abi.DealID) (i cid.Cid, e error) {
			return cid.Undef, errors.New("expected error")
		}

		return n
	}()

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, sealing.Packing).
		then(sealing.Unsealed).
		then(sealing.PreCommitting).
		then(sealing.WaitSeed).
		then(sealing.Committing).
		then(sealing.CommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, onSectorUpdatedFunc)
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

	maddr, err := address.NewIDAddress(55)
	require.NoError(t, err)

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.waitForProveCommitSector = func(i context.Context, i2 cid.Cid) (uint8, error) {
			return 0, errors.New("expected behavior")
		}

		return n
	}()

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, sealing.Packing).
		then(sealing.Unsealed).
		then(sealing.PreCommitting).
		then(sealing.WaitSeed).
		then(sealing.Committing).
		then(sealing.CommitWait).
		then(sealing.CommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, onSectorUpdatedFunc)
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
