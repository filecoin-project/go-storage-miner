package test

import (
	"context"

	"github.com/filecoin-project/go-storage-mining/storage"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"
)

type fakeNode struct{}

func (f *fakeNode) SendSelfDeals(context.Context, ...storage.PieceInfo) (cid.Cid, error) {
	return createCidForTesting(42), nil
}

func (f *fakeNode) WaitForSelfDeals(context.Context, cid.Cid) (dealIds []uint64, err error) {
	return []uint64{42, 42}, nil
}

func (f *fakeNode) SendPreCommitSector(ctx context.Context, sectorID uint64, ticket storage.SealTicket, pieces ...storage.Piece) (cid.Cid, error) {
	return createCidForTesting(42), nil
}

func (f *fakeNode) WaitForPreCommitSector(context.Context, cid.Cid) (processedAtBlockHeight uint64, err error) {
	return 42, nil
}

func (f *fakeNode) SendProveCommitSector(ctx context.Context, sectorID uint64, proof []byte, dealIds ...uint64) (cid.Cid, error) {
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

func createCidForTesting(n int) cid.Cid {
	useDefaultHashLengthFlag := -1
	obj, err := cbor.WrapObject([]int{n}, mh.IDENTITY, useDefaultHashLengthFlag)
	if err != nil {
		panic(err)
	}

	return obj.Cid()
}
