package test

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

type fakeNode struct {
	checkPieces              func(ctx context.Context, sectorNum abi.SectorNumber, pieces []node.Piece) *node.CheckPiecesError
	checkSealing             func(ctx context.Context, commD []byte, dealIDs []abi.DealID, ticket node.SealTicket) *node.CheckSealingError
	getMinerWorkerAddress    func(ctx context.Context, maddr address.Address) (address.Address, error)
	getReplicaCommitmentByID func(ctx context.Context, sectorNum abi.SectorNumber) (commR []byte, wasFound bool, err error)
	getSealSeed              func(ctx context.Context, msg cid.Cid, interval uint64) (<-chan node.SealSeed, <-chan node.SeedInvalidated, <-chan node.FinalityReached, <-chan node.GetSealSeedError)
	getSealTicket            func(context.Context) (node.SealTicket, error)
	sendPreCommitSector      func(ctx context.Context, sectorNum abi.SectorNumber, commR []byte, ticket node.SealTicket, pieces ...node.Piece) (cid.Cid, error)
	sendProveCommitSector    func(ctx context.Context, sectorNum abi.SectorNumber, proof []byte, dealIds ...abi.DealID) (cid.Cid, error)
	sendReportFaults         func(ctx context.Context, sectorNums ...abi.SectorNumber) (cid.Cid, error)
	sendSelfDeals            func(context.Context, ...abi.PieceInfo) (cid.Cid, error)
	waitForProveCommitSector func(context.Context, cid.Cid) (exitCode uint8, err error)
	waitForReportFaults      func(context.Context, cid.Cid) (uint8, error)
	waitForSelfDeals         func(context.Context, cid.Cid) ([]abi.DealID, uint8, error)
	walletHas                func(ctx context.Context, addr address.Address) (bool, error)
}

func newFakeNode() *fakeNode {
	return &fakeNode{
		checkPieces: func(ctx context.Context, sectorNum abi.SectorNumber, pieces []node.Piece) *node.CheckPiecesError {
			return nil
		},
		checkSealing: func(ctx context.Context, commD []byte, dealIDs []abi.DealID, ticket node.SealTicket) *node.CheckSealingError {
			return nil
		},
		getMinerWorkerAddress: func(ctx context.Context, maddr address.Address) (a address.Address, err error) {
			return address.NewIDAddress(42)
		},
		getReplicaCommitmentByID: func(ctx context.Context, sectorNum abi.SectorNumber) ([]byte, bool, error) {
			return nil, false, nil
		},
		getSealSeed: func(ctx context.Context, msg cid.Cid, interval uint64) (<-chan node.SealSeed, <-chan node.SeedInvalidated, <-chan node.FinalityReached, <-chan node.GetSealSeedError) {
			seedChan := make(chan node.SealSeed)
			go func() {
				seedChan <- node.SealSeed{
					BlockHeight: 42,
					TicketBytes: []byte{5, 6, 7},
				}
			}()

			return seedChan, make(chan node.SeedInvalidated), make(chan node.FinalityReached), make(chan node.GetSealSeedError)
		},
		getSealTicket: func(context.Context) (node.SealTicket, error) {
			return node.SealTicket{
				BlockHeight: 42,
				TicketBytes: []byte{1, 2, 3},
			}, nil
		},
		sendPreCommitSector: func(ctx context.Context, sectorNum abi.SectorNumber, commR []byte, ticket node.SealTicket, pieces ...node.Piece) (cid.Cid, error) {
			return createCidForTesting(42), nil
		},
		sendProveCommitSector: func(ctx context.Context, sectorNum abi.SectorNumber, proof []byte, dealIds ...abi.DealID) (cid.Cid, error) {
			return createCidForTesting(42), nil
		},
		sendReportFaults: func(ctx context.Context, sectorNumbers ...abi.SectorNumber) (i cid.Cid, e error) {
			return createCidForTesting(42), nil
		},
		sendSelfDeals: func(ctx context.Context, info ...abi.PieceInfo) (c cid.Cid, err error) {
			panic("by default, no self deals will be made")
		},
		waitForProveCommitSector: func(context.Context, cid.Cid) (exitCode uint8, err error) {
			return 0, nil
		},
		waitForReportFaults: func(i context.Context, i2 cid.Cid) (u uint8, e error) {
			return 0, nil
		},
		waitForSelfDeals: func(ctx context.Context, c cid.Cid) (ids []abi.DealID, u uint8, err error) {
			panic("by default, no self deals will be made")
		},
		walletHas: func(ctx context.Context, addr address.Address) (b bool, e error) {
			return true, nil
		},
	}
}

func (f *fakeNode) SendPreCommitSector(ctx context.Context, sectorNum abi.SectorNumber, commR []byte, ticket node.SealTicket, pieces ...node.Piece) (cid.Cid, error) {
	return f.sendPreCommitSector(ctx, sectorNum, commR, ticket, pieces...)
}

func (f *fakeNode) SendProveCommitSector(ctx context.Context, sectorNum abi.SectorNumber, proof []byte, dealIds ...abi.DealID) (cid.Cid, error) {
	return f.sendProveCommitSector(ctx, sectorNum, proof, dealIds...)
}

func (f *fakeNode) WaitForProveCommitSector(ctx context.Context, msg cid.Cid) (exitCode uint8, err error) {
	return f.waitForProveCommitSector(ctx, msg)
}

func (f *fakeNode) GetMinerWorkerAddress(ctx context.Context) (address.Address, error) {
	return f.getMinerWorkerAddress(ctx, maddr)
}

func (f *fakeNode) GetSealTicket(ctx context.Context) (node.SealTicket, error) {
	return f.getSealTicket(ctx)
}

func (f *fakeNode) GetSealSeed(ctx context.Context, preCommitMsgCid cid.Cid, interval uint64) (<-chan node.SealSeed, <-chan node.SeedInvalidated, <-chan node.FinalityReached, <-chan node.GetSealSeedError) {
	return f.getSealSeed(ctx, preCommitMsgCid, interval)
}

func (f *fakeNode) CheckPieces(ctx context.Context, sectorNum abi.SectorNumber, pieces []node.Piece) *node.CheckPiecesError {
	return f.checkPieces(ctx, sectorNum, pieces)
}

func (f *fakeNode) CheckSealing(ctx context.Context, commD []byte, dealIDs []abi.DealID, ticket node.SealTicket) *node.CheckSealingError {
	return f.checkSealing(ctx, commD, dealIDs, ticket)
}

func (f *fakeNode) GetReplicaCommitmentByID(ctx context.Context, sectorNum abi.SectorNumber) (commR []byte, wasFound bool, err error) {
	return f.getReplicaCommitmentByID(ctx, sectorNum)
}

func (f *fakeNode) SendReportFaults(ctx context.Context, sectorNumbers ...abi.SectorNumber) (cid.Cid, error) {
	return f.sendReportFaults(ctx, sectorNumbers...)
}

func (f *fakeNode) SendSelfDeals(ctx context.Context, pieces ...abi.PieceInfo) (cid.Cid, error) {
	return f.sendSelfDeals(ctx, pieces...)
}

func (f *fakeNode) WaitForReportFaults(ctx context.Context, msg cid.Cid) (uint8, error) {
	return f.waitForReportFaults(ctx, msg)
}

func (f *fakeNode) WaitForSelfDeals(ctx context.Context, msg cid.Cid) ([]abi.DealID, uint8, error) {
	return f.waitForSelfDeals(ctx, msg)
}

func (f *fakeNode) WalletHas(ctx context.Context, addr address.Address) (bool, error) {
	return f.walletHas(ctx, addr)
}

func createCidForTesting(n int) cid.Cid {
	useDefaultHashLengthFlag := -1
	obj, err := cbor.WrapObject([]int{n}, mh.IDENTITY, useDefaultHashLengthFlag)
	if err != nil {
		panic(err)
	}

	return obj.Cid()
}
