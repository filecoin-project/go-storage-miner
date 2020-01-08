package storage_test

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/lotus/api"
	"github.com/stretchr/testify/assert"

	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	cbor "github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"

	"github.com/filecoin-project/go-storage-mining/storage"
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

func createCidForTesting(n int) cid.Cid {
	useDefaultHashLengthFlag := -1
	obj, err := cbor.WrapObject([]int{n}, mh.IDENTITY, useDefaultHashLengthFlag)
	if err != nil {
		panic(err)
	}

	return obj.Cid()
}

////

type fakeNode struct{}

func (f *fakeNode) SendSelfDeals(context.Context, ...storage.PieceInfo) (cid.Cid, error) {
	return createCidForTesting(42), nil
}

func (f *fakeNode) WaitForSelfDeals(context.Context, cid.Cid) (dealIds []uint64, err error) {
	return []uint64{42, 42}, nil
}

func (f *fakeNode) SendPreCommitSector(context.Context, storage.SectorId, storage.SealTicket, ...storage.Piece) (cid.Cid, error) {
	return createCidForTesting(42), nil
}

func (f *fakeNode) WaitForPreCommitSector(context.Context, cid.Cid) (processedAtBlockHeight uint64, err error) {
	return 42, nil
}

func (f *fakeNode) SendProveCommitSector(context.Context, storage.SectorId, storage.SealProof, ...storage.DealId) (cid.Cid, error) {
	return createCidForTesting(42), nil
}

func (f *fakeNode) WaitForProveCommitSector(context.Context, cid.Cid) (processedAtBlockHeight uint64, err error) {
	return 42, nil
}

func (f *fakeNode) GetSealTicket(context.Context) (storage.SealTicket, error) {
	return storage.SealTicket{
		BlockHeight: 42,
		TicketBytes: []byte{1, 2, 3},
	}, nil
}

func (f *fakeNode) SetSealSeedHandler(ctx context.Context, msg cid.Cid, available func(storage.SealSeed), invalidated func()) {
	go func() {
		available(storage.SealSeed{
			BlockHeight: 42,
			TicketBytes: []byte{5, 6, 7},
		})
	}()
}

////

type fakeSectorBuilder struct{}

func (fakeSectorBuilder) SectorSize() uint64 {
	return 1024
}

func (fakeSectorBuilder) SealPreCommit(sectorID uint64, ticket ffi.SealTicket, pieces []sectorbuilder.PublicPieceInfo) (sectorbuilder.RawSealPreCommitOutput, error) {
	return sectorbuilder.RawSealPreCommitOutput{}, nil
}

func (fakeSectorBuilder) SealCommit(sectorID uint64, ticket ffi.SealTicket, seed ffi.SealSeed, pieces []sectorbuilder.PublicPieceInfo, rspco sectorbuilder.RawSealPreCommitOutput) (proof []byte, err error) {
	return []byte{42}, nil
}

func (fakeSectorBuilder) RateLimit() func() {
	return func() {}
}

func (fakeSectorBuilder) AddPiece(pieceSize uint64, sectorId uint64, file io.Reader, existingPieceSizes []uint64) (sectorbuilder.PublicPieceInfo, error) {
	return sectorbuilder.PublicPieceInfo{Size: pieceSize}, nil
}

func (fakeSectorBuilder) AcquireSectorId() (uint64, error) {
	return 42, nil
}

////

type sectorStateSequence struct {
	actualSequence   []api.SectorState
	expectedSequence []api.SectorState
	sectorId         uint64
	t                *testing.T
}

func begin(t *testing.T, sectorId uint64, initialState api.SectorState) *sectorStateSequence {
	return &sectorStateSequence{
		actualSequence:   []api.SectorState{},
		expectedSequence: []api.SectorState{initialState},
		sectorId:         sectorId,
		t:                t,
	}
}

func (f *sectorStateSequence) then(s api.SectorState) *sectorStateSequence {
	f.expectedSequence = append(f.expectedSequence, s)
	return f
}

func (f *sectorStateSequence) end() (func(uint64, api.SectorState), func() string, <-chan struct{}) {
	var indx int
	var last api.SectorState
	done := make(chan struct{})

	next := func(sectorId uint64, curr api.SectorState) {
		if sectorId != f.sectorId {
			return
		}

		if indx < len(f.expectedSequence) {
			assert.Equal(f.t, f.expectedSequence[indx], curr, "unexpected transition from %s to %s (expected transition to %s)", api.SectorStates[last], api.SectorStates[curr], api.SectorStates[f.expectedSequence[indx]])
		}

		last = curr
		indx++
		f.actualSequence = append(f.actualSequence, curr)

		// if this is the last value we care about, signal completion
		if f.expectedSequence[len(f.expectedSequence)-1] == curr {
			go func() {
				done <- struct{}{}
			}()
		}
	}

	status := func() string {
		expected := make([]string, len(f.expectedSequence))
		for i, s := range f.expectedSequence {
			expected[i] = api.SectorStates[s]
		}

		actual := make([]string, len(f.actualSequence))
		for i, s := range f.actualSequence {
			actual[i] = api.SectorStates[s]
		}

		return fmt.Sprintf("expected transitions: %+v, actual transitions: %+v", expected, actual)
	}

	return next, status, done
}
