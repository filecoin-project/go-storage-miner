package test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/filecoin-project/specs-actors/actors/builtin/market"

	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
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

	maddr, err := address.NewIDAddress(55)
	require.NoError(t, err)

	waddr, err := address.NewIDAddress(66)
	require.NoError(t, err)

	// a sequence of sector state transitions we expect to observe
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.WaitSeed).
		then(storage.Committing).
		then(storage.CommitWait).
		then(storage.FinalizeSector).
		then(storage.Proving).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(newFakeNode(), datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, waddr, onSectorUpdatedFunc)
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

	waddr, err := address.NewIDAddress(66)
	require.NoError(t, err)

	// we'll assert the contents of this slice at the end of the test
	var selfDealPieceSizes []abi.UnpaddedPieceSize

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.sendMessage = func(from, to address.Address, method abi.MethodNum, value abi.TokenAmount, params []byte) (c cid.Cid, err error) {
			switch method {
			case builtin.MethodsMarket.PublishStorageDeals:
				arg := new(market.PublishStorageDealsParams)
				if err = arg.UnmarshalCBOR(bytes.NewReader(params)); err != nil {
					panic(fmt.Sprintf("unmarshaling PublishStorageDealsParams failed: %w", err))
				}

				selfDealPieceSizes = append(selfDealPieceSizes, arg.Deals[0].Proposal.PieceSize.Unpadded())
				selfDealPieceSizes = append(selfDealPieceSizes, arg.Deals[1].Proposal.PieceSize.Unpadded())
			default:
				panic(fmt.Sprintf("unhandled method call in test: %d", method))
			}

			return createCidForTesting(42), nil
		}

		n.waitMessage = func(ctx context.Context, msg cid.Cid) (wait storage.MsgWait, err error) {
			ret := &market.PublishStorageDealsReturn{
				IDs: []abi.DealID{42, 99},
			}

			buf := new(bytes.Buffer)
			if err := ret.MarshalCBOR(buf); err != nil {
				panic(fmt.Sprintf("failed to marshal PublishStorageDealsReturn CBOR bytes: %w", err))
			}

			return storage.MsgWait{
				Receipt: storage.MessageReceipt{
					ExitCode: 0,
					Return:   buf.Bytes(),
					GasUsed:  abi.NewTokenAmount(0),
				},
				Height: 0,
			}, nil
		}

		return n
	}()

	sb := &fakeSectorBuilder{}

	// a sequence of sector state transitions we expect to observe
	sectorID, err := sb.AcquireSectorNumber()
	require.NoError(t, err)

	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, sectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.WaitSeed).
		then(storage.Committing).
		then(storage.CommitWait).
		then(storage.FinalizeSector).
		then(storage.Proving).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), sb, maddr, waddr, onSectorUpdatedFunc)
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

	waddr, err := address.NewIDAddress(66)
	require.NoError(t, err)

	// configure behavior of the fake node
	fakeNode := func() *fakeNode {
		n := newFakeNode()

		n.sendPreCommitSector = func(ctx context.Context, sectorNum abi.SectorNumber, commR []byte, ticket storage.SealTicket, pieces ...storage.Piece) (i cid.Cid, e error) {
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

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, waddr, onSectorUpdatedFunc)
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

	waddr, err := address.NewIDAddress(66)
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
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.WaitSeed).
		then(storage.Committing).
		then(storage.CommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, waddr, onSectorUpdatedFunc)
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

	waddr, err := address.NewIDAddress(66)
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
	onSectorUpdatedFunc, getSequenceStatusFunc, doneCh := begin(t, DefaultSectorID, storage.Packing).
		then(storage.Unsealed).
		then(storage.PreCommitting).
		then(storage.WaitSeed).
		then(storage.Committing).
		then(storage.CommitWait).
		then(storage.CommitFailed).
		end()

	miner, err := storage.NewMinerWithOnSectorUpdated(fakeNode, datastore.NewMapDatastore(), &fakeSectorBuilder{}, maddr, waddr, onSectorUpdatedFunc)
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
