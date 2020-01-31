package test

import (
	"context"

	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"

	"github.com/filecoin-project/go-storage-miner"
)

type fakeNode struct {
	checkPieces              func(ctx context.Context, sectorID uint64, pieces []storage.Piece) *storage.CheckPiecesError
	checkSealing             func(ctx context.Context, commD []byte, dealIDs []uint64, ticket storage.SealTicket) *storage.CheckSealingError
	getReplicaCommitmentByID func(ctx context.Context, sectorID uint64) (commR []byte, wasFound bool, err error)
	getSealSeed              func(ctx context.Context, msg cid.Cid, interval uint64) (<-chan storage.SealSeed, <-chan storage.SeedInvalidated, <-chan storage.FinalityReached, <-chan *storage.GetSealSeedError)
	getSealTicket            func(context.Context) (storage.SealTicket, error)
	sendPreCommitSector      func(ctx context.Context, sectorID uint64, commR []byte, ticket storage.SealTicket, pieces ...storage.Piece) (cid.Cid, error)
	sendProveCommitSector    func(ctx context.Context, sectorID uint64, proof []byte, dealIds ...uint64) (cid.Cid, error)
	sendReportFaults         func(ctx context.Context, sectorIDs ...uint64) (cid.Cid, error)
	sendSelfDeals            func(context.Context, ...storage.PieceInfo) (cid.Cid, error)
	waitForProveCommitSector func(context.Context, cid.Cid) (exitCode uint8, err error)
	waitForReportFaults      func(context.Context, cid.Cid) (uint8, error)
	waitForSelfDeals         func(context.Context, cid.Cid) (dealIds []uint64, exitCode uint8, err error)
}

func newFakeNode() *fakeNode {
	return &fakeNode{
		sendSelfDeals: func(context.Context, ...storage.PieceInfo) (cid.Cid, error) {
			panic("by default, no self deals will be made")
		},
		waitForSelfDeals: func(context.Context, cid.Cid) (dealIds []uint64, exitCode uint8, err error) {
			panic("by default, no self deals will be made")
		},
		sendPreCommitSector: func(ctx context.Context, sectorID uint64, commR []byte, ticket storage.SealTicket, pieces ...storage.Piece) (cid.Cid, error) {
			return createCidForTesting(42), nil
		},
		sendProveCommitSector: func(ctx context.Context, sectorID uint64, proof []byte, dealIds ...uint64) (cid.Cid, error) {
			return createCidForTesting(42), nil
		},
		waitForProveCommitSector: func(context.Context, cid.Cid) (exitCode uint8, err error) {
			return 0, nil
		},
		getSealTicket: func(context.Context) (storage.SealTicket, error) {
			return storage.SealTicket{
				BlockHeight: 42,
				TicketBytes: []byte{1, 2, 3},
			}, nil
		},
		getSealSeed: func(ctx context.Context, msg cid.Cid, interval uint64) (<-chan storage.SealSeed, <-chan storage.SeedInvalidated, <-chan storage.FinalityReached, <-chan *storage.GetSealSeedError) {
			seedChan := make(chan storage.SealSeed)
			go func() {
				seedChan <- storage.SealSeed{
					BlockHeight: 42,
					TicketBytes: []byte{5, 6, 7},
				}
			}()

			return seedChan, make(chan storage.SeedInvalidated), make(chan storage.FinalityReached), make(chan *storage.GetSealSeedError)
		},
		checkPieces: func(ctx context.Context, sectorID uint64, pieces []storage.Piece) *storage.CheckPiecesError {
			return nil
		},
		checkSealing: func(ctx context.Context, commD []byte, dealIDs []uint64, ticket storage.SealTicket) *storage.CheckSealingError {
			return nil
		},
		sendReportFaults: func(ctx context.Context, sectorIDs ...uint64) (i cid.Cid, e error) {
			return createCidForTesting(42), nil
		},
		waitForReportFaults: func(i context.Context, i2 cid.Cid) (u uint8, e error) {
			return 0, nil
		},
		getReplicaCommitmentByID: func(ctx context.Context, sectorID uint64) ([]byte, bool, error) {
			return nil, false, nil
		},
	}
}

func (f *fakeNode) SendSelfDeals(ctx context.Context, pieces ...storage.PieceInfo) (cid.Cid, error) {
	return f.sendSelfDeals(ctx, pieces...)
}

func (f *fakeNode) WaitForSelfDeals(ctx context.Context, msg cid.Cid) (dealIds []uint64, exitCode uint8, err error) {
	return f.waitForSelfDeals(ctx, msg)
}

func (f *fakeNode) SendPreCommitSector(ctx context.Context, sectorID uint64, commR []byte, ticket storage.SealTicket, pieces ...storage.Piece) (cid.Cid, error) {
	return f.sendPreCommitSector(ctx, sectorID, commR, ticket, pieces...)
}

func (f *fakeNode) SendProveCommitSector(ctx context.Context, sectorID uint64, proof []byte, dealIds ...uint64) (cid.Cid, error) {
	return f.sendProveCommitSector(ctx, sectorID, proof, dealIds...)
}

func (f *fakeNode) WaitForProveCommitSector(ctx context.Context, msg cid.Cid) (exitCode uint8, err error) {
	return f.waitForProveCommitSector(ctx, msg)
}

func (f *fakeNode) GetSealTicket(ctx context.Context) (storage.SealTicket, error) {
	return f.getSealTicket(ctx)
}

func (f *fakeNode) GetSealSeed(ctx context.Context, preCommitMsgCid cid.Cid, interval uint64) (<-chan storage.SealSeed, <-chan storage.SeedInvalidated, <-chan storage.FinalityReached, <-chan *storage.GetSealSeedError) {
	return f.getSealSeed(ctx, preCommitMsgCid, interval)
}

func (f *fakeNode) CheckPieces(ctx context.Context, sectorID uint64, pieces []storage.Piece) *storage.CheckPiecesError {
	return f.checkPieces(ctx, sectorID, pieces)
}

func (f *fakeNode) CheckSealing(ctx context.Context, commD []byte, dealIDs []uint64, ticket storage.SealTicket) *storage.CheckSealingError {
	return f.checkSealing(ctx, commD, dealIDs, ticket)
}

func (f *fakeNode) GetReplicaCommitmentByID(ctx context.Context, sectorID uint64) (commR []byte, wasFound bool, err error) {
	return f.getReplicaCommitmentByID(ctx, sectorID)
}

func (f *fakeNode) SendReportFaults(ctx context.Context, sectorIDs ...uint64) (cid.Cid, error) {
	return f.sendReportFaults(ctx, sectorIDs...)
}

func (f *fakeNode) WaitForReportFaults(ctx context.Context, msg cid.Cid) (uint8, error) {
	return f.waitForReportFaults(ctx, msg)
}

func createCidForTesting(n int) cid.Cid {
	useDefaultHashLengthFlag := -1
	obj, err := cbor.WrapObject([]int{n}, mh.IDENTITY, useDefaultHashLengthFlag)
	if err != nil {
		panic(err)
	}

	return obj.Cid()
}
