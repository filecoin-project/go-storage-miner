package storage_test

import (
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/go-sectorbuilder/paramfetch"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	cbor "github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"

	"github.com/filecoin-project/go-storage-mining/storage"
)

func TestSectorLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cleanup, builder := createSectorBuilderForTest(t)
	defer cleanup()

	miner, err := storage.NewMiner(&fakeNodeApi{}, datastore.NewMapDatastore(), builder)
	if err != nil {
		t.Fatalf("failed to create miner: %s", err)
	}

	defer func() {
		if err := miner.Stop(ctx); err != nil {
			t.Errorf("failed to stop miner: %s", err)
		}
	}()

	if err = miner.Run(ctx); err != nil {
		t.Fatalf("failed to start miner: %s", err)
	}

	pieceSize := sectorbuilder.UserBytesForSectorSize(builder.SectorSize()) / 4
	pieceReader := io.LimitReader(rand.New(rand.NewSource(42)), int64(pieceSize))

	sectorID, _, err := miner.AllocatePiece(pieceSize)
	if err != nil {
		t.Fatalf("failed to allocate: %s", err)
	}

	if err = miner.SealPiece(ctx, pieceSize, pieceReader, sectorID, 1); err != nil {
		t.Fatalf("failed to start piece-sealing flow: %s", err)
	}

	time.Sleep(time.Minute * 30)
}

func createCidForTesting(n int) cid.Cid {
	useDefaultHashLengthFlag := -1
	obj, err := cbor.WrapObject([]int{n}, mh.IDENTITY, useDefaultHashLengthFlag)
	if err != nil {
		panic(err)
	}

	return obj.Cid()
}

func createSectorBuilderForTest(t *testing.T) (func(), *sectorbuilder.SectorBuilder) {
	sectorSize := uint64(1024)

	if err := paramfetch.GetParams(sectorSize); err != nil {
		t.Fatalf("failed to acquire Groth parameters and verifying keys: %s", err)
	}

	dir, err := ioutil.TempDir("", "storage-mining-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err)
	}

	builder, err := sectorbuilder.TempSectorbuilderDir(dir, sectorSize, datastore.NewMapDatastore())
	if err != nil {
		t.Fatalf("%+v", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("failed to remove dir: %s", err)
		}
	}

	return cleanup, builder
}

type fakeNodeApi struct{}

func (f *fakeNodeApi) SendSelfDeals(context.Context, ...storage.PieceInfo) (cid.Cid, error) {
	return createCidForTesting(42), nil
}

func (f *fakeNodeApi) WaitForSelfDeals(context.Context, cid.Cid) (dealIds []uint64, err error) {
	return []uint64{42, 42}, nil
}

func (f *fakeNodeApi) SendPreCommitSector(context.Context, storage.SectorId, storage.SealTicket, ...storage.Piece) (cid.Cid, error) {
	return createCidForTesting(42), nil
}

func (f *fakeNodeApi) WaitForPreCommitSector(context.Context, cid.Cid) (processedAtBlockHeight uint64, err error) {
	return 42, nil
}

func (f *fakeNodeApi) SendProveCommitSector(context.Context, storage.SectorId, storage.SealProof, ...storage.DealId) (cid.Cid, error) {
	return createCidForTesting(42), nil
}

func (f *fakeNodeApi) WaitForProveCommitSector(context.Context, cid.Cid) (processedAtBlockHeight uint64, err error) {
	return 42, nil
}

func (f *fakeNodeApi) GetSealTicket(context.Context) (storage.SealTicket, error) {
	return storage.SealTicket{
		BlockHeight: 42,
		TicketBytes: []byte{1, 2, 3},
	}, nil
}

func (f *fakeNodeApi) SetSealSeedHandler(ctx context.Context, msg cid.Cid, available func(storage.SealSeed), invalidated func()) {
	go func() {
		available(storage.SealSeed{
			BlockHeight: 42,
			TicketBytes: []byte{5, 6, 7},
		})
	}()
}
