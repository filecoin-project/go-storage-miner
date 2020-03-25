package test

import (
	"context"

	"github.com/filecoin-project/go-address"
	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

type fakeNode struct {
	checkPieces              func(ctx context.Context, sectorNum abi.SectorNumber, pieces []node.PieceWithDealInfo) *node.CheckPiecesError
	checkSealing             func(ctx context.Context, commD []byte, dealIDs []abi.DealID, ticket node.SealTicket) *node.CheckSealingError
	getChainHead             func(ctx context.Context) (node.TipSetToken, abi.ChainEpoch, error)
	getMinerWorkerAddress    func(context.Context, node.TipSetToken) (address.Address, error)
	getSealedCID             func(ctx context.Context, tok node.TipSetToken, sectorNum abi.SectorNumber) (cid.Cid, bool, error)
	getSealSeed              func(ctx context.Context, msg cid.Cid, interval uint64) (<-chan node.SealSeed, <-chan node.SeedInvalidated, <-chan node.FinalityReached, <-chan node.GetSealSeedError)
	getSealTicket            func(context.Context, node.TipSetToken) (node.SealTicket, error)
	sendPreCommitSector      func(ctx context.Context, proofType abi.RegisteredProof, sectorNum abi.SectorNumber, sealedCID cid.Cid, sealEpoch, expiration abi.ChainEpoch, pieces ...node.PieceWithDealInfo) (cid.Cid, error)
	sendProveCommitSector    func(ctx context.Context, proofType abi.RegisteredProof, sectorNum abi.SectorNumber, proof []byte, dealIds ...abi.DealID) (cid.Cid, error)
	sendReportFaults         func(ctx context.Context, sectorNums ...abi.SectorNumber) (cid.Cid, error)
	sendSelfDeals            func(context.Context, abi.ChainEpoch, abi.ChainEpoch, ...abi.PieceInfo) (cid.Cid, error)
	waitForProveCommitSector func(context.Context, cid.Cid) (exitCode uint8, err error)
	waitForReportFaults      func(context.Context, cid.Cid) (uint8, error)
	waitForSelfDeals         func(context.Context, cid.Cid) ([]abi.DealID, uint8, error)
	walletHas                func(ctx context.Context, addr address.Address) (bool, error)
}

func newFakeNode() *fakeNode {
	return &fakeNode{
		checkPieces: func(ctx context.Context, sectorNum abi.SectorNumber, pieces []node.PieceWithDealInfo) *node.CheckPiecesError {
			return nil
		},
		checkSealing: func(ctx context.Context, commD []byte, dealIDs []abi.DealID, ticket node.SealTicket) *node.CheckSealingError {
			return nil
		},
		getChainHead: func(ctx context.Context) (token node.TipSetToken, epoch abi.ChainEpoch, err error) {
			return node.TipSetToken{1, 2, 3}, 42, nil
		},
		getMinerWorkerAddress: func(ctx context.Context, tok node.TipSetToken) (a address.Address, err error) {
			return address.NewIDAddress(42)
		},
		getSealedCID: func(ctx context.Context, tok node.TipSetToken, sectorNum abi.SectorNumber) (cid.Cid, bool, error) {
			commR := [32]byte{1, 2, 3}
			return commcid.ReplicaCommitmentV1ToCID(commR[:]), false, nil
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
		getSealTicket: func(context.Context, node.TipSetToken) (node.SealTicket, error) {
			return node.SealTicket{
				BlockHeight: 42,
				TicketBytes: []byte{1, 2, 3},
			}, nil
		},
		sendPreCommitSector: func(ctx context.Context, proofType abi.RegisteredProof, sectorNum abi.SectorNumber, sealedCID cid.Cid, sealEpoch, expiration abi.ChainEpoch, pieces ...node.PieceWithDealInfo) (cid.Cid, error) {
			return createCidForTesting(42), nil
		},
		sendReportFaults: func(ctx context.Context, sectorNumbers ...abi.SectorNumber) (i cid.Cid, e error) {
			return createCidForTesting(42), nil
		},
		sendSelfDeals: func(ctx context.Context, start, end abi.ChainEpoch, info ...abi.PieceInfo) (c cid.Cid, err error) {
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
		sendProveCommitSector: func(ctx context.Context, proofType abi.RegisteredProof, sectorNum abi.SectorNumber, proof []byte, dealIds ...abi.DealID) (cid.Cid, error) {
			return createCidForTesting(42), nil
		},
	}
}

func (f *fakeNode) CheckPieces(ctx context.Context, sectorNum abi.SectorNumber, pieces []node.PieceWithDealInfo) *node.CheckPiecesError {
	return f.checkPieces(ctx, sectorNum, pieces)
}

func (f *fakeNode) CheckSealing(ctx context.Context, commD []byte, dealIDs []abi.DealID, ticket node.SealTicket) *node.CheckSealingError {
	return f.checkSealing(ctx, commD, dealIDs, ticket)
}

func (f *fakeNode) GetChainHead(ctx context.Context) (node.TipSetToken, abi.ChainEpoch, error) {
	return f.getChainHead(ctx)
}

func (f *fakeNode) GetMinerWorkerAddress(ctx context.Context, tok node.TipSetToken) (address.Address, error) {
	return f.getMinerWorkerAddress(ctx, tok)
}

func (f *fakeNode) GetSealedCID(ctx context.Context, tok node.TipSetToken, sectorNum abi.SectorNumber) (cid.Cid, bool, error) {
	return f.getSealedCID(ctx, tok, sectorNum)
}

func (f *fakeNode) GetSealSeed(ctx context.Context, preCommitMsg cid.Cid, interval uint64) (<-chan node.SealSeed, <-chan node.SeedInvalidated, <-chan node.FinalityReached, <-chan node.GetSealSeedError) {
	return f.getSealSeed(ctx, preCommitMsg, interval)
}

func (f *fakeNode) GetSealTicket(ctx context.Context, tok node.TipSetToken) (node.SealTicket, error) {
	return f.getSealTicket(ctx, tok)
}

func (f *fakeNode) SendPreCommitSector(ctx context.Context, proofType abi.RegisteredProof, sectorNum abi.SectorNumber, sealedCID cid.Cid, sealEpoch, expiration abi.ChainEpoch, pieces ...node.PieceWithDealInfo) (cid.Cid, error) {
	return f.sendPreCommitSector(ctx, proofType, sectorNum, sealedCID, sealEpoch, expiration, pieces...)
}

func (f *fakeNode) SendProveCommitSector(ctx context.Context, proofType abi.RegisteredProof, sectorNum abi.SectorNumber, proof []byte, dealIds ...abi.DealID) (cid.Cid, error) {
	return f.sendProveCommitSector(ctx, proofType, sectorNum, proof, dealIds...)
}

func (f *fakeNode) SendReportFaults(ctx context.Context, sectorNumbers ...abi.SectorNumber) (cid.Cid, error) {
	return f.sendReportFaults(ctx, sectorNumbers...)
}

func (f *fakeNode) SendSelfDeals(ctx context.Context, start, end abi.ChainEpoch, pieces ...abi.PieceInfo) (cid.Cid, error) {
	return f.sendSelfDeals(ctx, start, end, pieces...)
}

func (f *fakeNode) WaitForProveCommitSector(ctx context.Context, msg cid.Cid) (exitCode uint8, err error) {
	return f.waitForProveCommitSector(ctx, msg)
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
