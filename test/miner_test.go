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
	"github.com/multiformats/go-multihash"

	"github.com/filecoin-project/go-storage-miner"
	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/go-storage-miner/policies/precommit"
	"github.com/filecoin-project/go-storage-miner/policies/selfdeal"
	"github.com/filecoin-project/storage-fsm"
	sealing "github.com/filecoin-project/storage-fsm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const DefaultSectorID = 42
const UserBytesOneKiBSector = 1016 // also known as "unpadded" bytes

var DefaultDealInfo = node.DealInfo{
	DealID: 42,
	DealSchedule: node.DealSchedule{
		StartEpoch: 100,
		EndEpoch:   200,
	},
}

func TestSuccessfulPieceSealingFlow(t *testing.T) {
	ctx := context.Background()

	maddr, err := address.NewIDAddress(55)
	require.NoError(t, err)

	api := newFakeNode()
	sdp := selfdeal.NewBasicPolicy(api, 10, 100)
	pcp := precommit.NewBasicPolicy(api, 300)

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

	miner, err := storage.NewMinerWithOnSectorUpdated(api, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, &sdp, &pcp, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// start the internal runloop
	require.NoError(t, miner.Run(ctx))

	// kick off the state machine
	require.NoError(t, miner.SealPiece(ctx, UserBytesOneKiBSector, io.LimitReader(rand.New(rand.NewSource(42)), int64(UserBytesOneKiBSector)), DefaultSectorID, DefaultDealInfo))

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

	// we'll assert the contents of these slices at the end of the test
	var selfDealPieceSizes []abi.UnpaddedPieceSize
	var selfDealSchedule node.DealSchedule
	var preCommitSectorPieceSizes []abi.UnpaddedPieceSize

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.getChainHead = func(ctx context.Context) (token node.TipSetToken, epoch abi.ChainEpoch, err error) {
			return []byte{1, 2, 3}, abi.ChainEpoch(66), nil
		}

		n.sendSelfDeals = func(i context.Context, start, end abi.ChainEpoch, info ...abi.PieceInfo) (cid cid.Cid, e error) {
			selfDealPieceSizes = append(selfDealPieceSizes, info[0].Size.Unpadded())
			selfDealPieceSizes = append(selfDealPieceSizes, info[1].Size.Unpadded())

			selfDealSchedule = node.DealSchedule{
				StartEpoch: start,
				EndEpoch:   end,
			}

			return createCidForTesting(42), nil
		}

		n.waitForSelfDeals = func(i context.Context, i2 cid.Cid) (dealIds []abi.DealID, exitCode uint8, err error) {
			return []abi.DealID{abi.DealID(42), abi.DealID(43)}, 0, nil
		}

		n.sendPreCommitSector = func(ctx context.Context, proofType abi.RegisteredProof, sectorNum abi.SectorNumber, sealedCID cid.Cid, sealEpoch, expiration abi.ChainEpoch, pieces ...node.PieceWithDealInfo) (c cid.Cid, err error) {
			for idx := range pieces {
				preCommitSectorPieceSizes = append(preCommitSectorPieceSizes, pieces[idx].Piece.Size.Unpadded())
			}

			encode, err := multihash.Encode([]byte{42, 42, 42}, multihash.IDENTITY)
			require.NoError(t, err)

			return cid.NewCidV1(cid.Raw, encode), nil
		}

		return n
	}()

	sdp := selfdeal.NewBasicPolicy(fakeNode, 10, 100)
	pcp := precommit.NewBasicPolicy(fakeNode, 300)

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

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), sb, maddr, &sdp, &pcp, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// create a piece which fills up a quarter of the sector
	pieceSize := abi.UnpaddedPieceSize(uint64(1016 / 4))
	pieceReader := io.LimitReader(rand.New(rand.NewSource(42)), int64(pieceSize))

	// kick off state transitions
	require.NoError(t, miner.Run(ctx))
	require.NoError(t, miner.SealPiece(ctx, pieceSize, pieceReader, sectorID, DefaultDealInfo))

	select {
	case <-doneCh:
		require.Equal(t, 2, len(selfDealPieceSizes), "expected two self-deals")
		require.Equal(t, 3, len(preCommitSectorPieceSizes), "expected three pieces in the sector")
		assert.Equal(t, int(abi.UnpaddedPieceSize(254)), int(selfDealPieceSizes[0]))
		assert.Equal(t, int(abi.UnpaddedPieceSize(508)), int(selfDealPieceSizes[1]))
		assert.Greater(t, int(selfDealSchedule.StartEpoch), 0)
		assert.Greater(t, int(selfDealSchedule.EndEpoch), 0)
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

		n.sendPreCommitSector = func(ctx context.Context, proofType abi.RegisteredProof, sectorNum abi.SectorNumber, sealedCID cid.Cid, sealEpoch, expiration abi.ChainEpoch, pieces ...node.PieceWithDealInfo) (cid.Cid, error) {
			return cid.Undef, errors.New("expected error")
		}

		return n
	}()

	sdp := selfdeal.NewBasicPolicy(fakeNode, 10, 100)
	pcp := precommit.NewBasicPolicy(fakeNode, 300)

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, sealing.Packing).
		then(sealing.Unsealed).
		then(sealing.PreCommitting).
		then(sealing.PreCommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, &sdp, &pcp, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// kick off state transitions
	require.NoError(t, miner.Run(ctx))
	require.NoError(t, miner.SealPiece(ctx, UserBytesOneKiBSector, io.LimitReader(rand.New(rand.NewSource(42)), int64(UserBytesOneKiBSector)), DefaultSectorID, DefaultDealInfo))

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

		n.sendProveCommitSector = func(ctx context.Context, proofType abi.RegisteredProof, sectorNum abi.SectorNumber, proof []byte, dealIds ...abi.DealID) (i cid.Cid, e error) {
			return cid.Undef, errors.New("expected error")
		}

		return n
	}()

	sdp := selfdeal.NewBasicPolicy(fakeNode, 10, 100)
	pcp := precommit.NewBasicPolicy(fakeNode, 300)

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, sealing.Packing).
		then(sealing.Unsealed).
		then(sealing.PreCommitting).
		then(sealing.WaitSeed).
		then(sealing.Committing).
		then(sealing.CommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, &sdp, &pcp, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// kick off state transitions
	require.NoError(t, miner.Run(ctx))
	require.NoError(t, miner.SealPiece(ctx, UserBytesOneKiBSector, io.LimitReader(rand.New(rand.NewSource(42)), int64(UserBytesOneKiBSector)), DefaultSectorID, DefaultDealInfo))

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

	sdp := selfdeal.NewBasicPolicy(fakeNode, 10, 100)
	pcp := precommit.NewBasicPolicy(fakeNode, 300)

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, sealing.Packing).
		then(sealing.Unsealed).
		then(sealing.PreCommitting).
		then(sealing.WaitSeed).
		then(sealing.Committing).
		then(sealing.CommitWait).
		then(sealing.CommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, &sdp, &pcp, onSectorUpdatedFunc)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, miner.Stop(ctx))
	}()

	// kick off state transitions
	require.NoError(t, miner.Run(ctx))
	require.NoError(t, miner.SealPiece(ctx, UserBytesOneKiBSector, io.LimitReader(rand.New(rand.NewSource(42)), int64(UserBytesOneKiBSector)), DefaultSectorID, DefaultDealInfo))

	select {
	case <-doneCh:
		// done
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for sequence to complete: %s", getSequenceStatusFunc())
	}
}
