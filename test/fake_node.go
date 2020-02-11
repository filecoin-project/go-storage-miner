package test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/filecoin-project/go-storage-miner/apis/node"

	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/specs-actors/actors/builtin/market"

	"github.com/filecoin-project/specs-actors/actors/runtime"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"
)

type fakeNode struct {
	checkPieces              func(ctx context.Context, sectorNum abi.SectorNumber, pieces []node.Piece) *node.CheckPiecesError
	checkSealing             func(ctx context.Context, commD []byte, dealIDs []abi.DealID, ticket node.SealTicket) *node.CheckSealingError
	getMinerWorkerAddress    func(ctx context.Context, maddr address.Address) (address.Address, error)
	getReplicaCommitmentByID func(ctx context.Context, sectorNum abi.SectorNumber) (commR []byte, wasFound bool, err error)
	getSealSeed              func(ctx context.Context, msg cid.Cid, interval uint64) (<-chan node.SealSeed, <-chan node.SeedInvalidated, <-chan node.FinalityReached, <-chan *node.GetSealSeedError)
	getSealTicket            func(context.Context) (node.SealTicket, error)
	sendMessage              func(from, to address.Address, method abi.MethodNum, value abi.TokenAmount, params []byte) (cid.Cid, error)
	sendPreCommitSector      func(ctx context.Context, sectorNum abi.SectorNumber, commR []byte, ticket node.SealTicket, pieces ...node.Piece) (cid.Cid, error)
	sendProveCommitSector    func(ctx context.Context, sectorNum abi.SectorNumber, proof []byte, dealIds ...abi.DealID) (cid.Cid, error)
	sendReportFaults         func(ctx context.Context, sectorNums ...abi.SectorNumber) (cid.Cid, error)
	waitForProveCommitSector func(context.Context, cid.Cid) (exitCode uint8, err error)
	waitForReportFaults      func(context.Context, cid.Cid) (uint8, error)
	waitMessage              func(ctx context.Context, msg cid.Cid) (node.MsgWait, error)
	walletHas                func(ctx context.Context, addr address.Address) (bool, error)
}

func newFakeNode() *fakeNode {
	m := make(map[cid.Cid]runtime.CBORMarshaler)

	m[createCidForTesting(int(builtin.MethodsMarket.PublishStorageDeals))] = &market.PublishStorageDealsReturn{
		IDs: []abi.DealID{42, 99},
	}

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
		getSealSeed: func(ctx context.Context, msg cid.Cid, interval uint64) (<-chan node.SealSeed, <-chan node.SeedInvalidated, <-chan node.FinalityReached, <-chan *node.GetSealSeedError) {
			seedChan := make(chan node.SealSeed)
			go func() {
				seedChan <- node.SealSeed{
					BlockHeight: 42,
					TicketBytes: []byte{5, 6, 7},
				}
			}()

			return seedChan, make(chan node.SeedInvalidated), make(chan node.FinalityReached), make(chan *node.GetSealSeedError)
		},
		getSealTicket: func(context.Context) (node.SealTicket, error) {
			return node.SealTicket{
				BlockHeight: 42,
				TicketBytes: []byte{1, 2, 3},
			}, nil
		},
		sendMessage: func(from, to address.Address, method abi.MethodNum, value abi.TokenAmount, params []byte) (c cid.Cid, err error) {
			return createCidForTesting(int(method)), nil
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
		waitForProveCommitSector: func(context.Context, cid.Cid) (exitCode uint8, err error) {
			return 0, nil
		},
		waitForReportFaults: func(i context.Context, i2 cid.Cid) (u uint8, e error) {
			return 0, nil
		},
		waitMessage: func(ctx context.Context, msg cid.Cid) (wait node.MsgWait, err error) {
			v, ok := m[msg]

			if !ok {
				panic(fmt.Sprintf("test setup is missing message cid: %s, map: %+v", msg, m))
			}

			buf := new(bytes.Buffer)
			if err := v.MarshalCBOR(buf); err != nil {
				panic(fmt.Sprintf("test failed to marshal CBOR bytes: %s", err))
			}

			return node.MsgWait{
				Receipt: node.MessageReceipt{
					ExitCode: 0,
					Return:   buf.Bytes(),
					GasUsed:  abi.NewTokenAmount(0),
				},
				Height: 0,
			}, nil
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

func (f *fakeNode) GetMinerWorkerAddressFromChainHead(ctx context.Context, maddr address.Address) (address.Address, error) {
	return f.getMinerWorkerAddress(ctx, maddr)
}

func (f *fakeNode) GetSealTicket(ctx context.Context) (node.SealTicket, error) {
	return f.getSealTicket(ctx)
}

func (f *fakeNode) GetSealSeed(ctx context.Context, preCommitMsgCid cid.Cid, interval uint64) (<-chan node.SealSeed, <-chan node.SeedInvalidated, <-chan node.FinalityReached, <-chan *node.GetSealSeedError) {
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

func (f *fakeNode) SendMessage(from, to address.Address, method abi.MethodNum, value abi.TokenAmount, params []byte) (cid.Cid, error) {
	return f.sendMessage(from, to, method, value, params)
}

func (f *fakeNode) SendReportFaults(ctx context.Context, sectorNumbers ...abi.SectorNumber) (cid.Cid, error) {
	return f.sendReportFaults(ctx, sectorNumbers...)
}

func (f *fakeNode) WaitForReportFaults(ctx context.Context, msg cid.Cid) (uint8, error) {
	return f.waitForReportFaults(ctx, msg)
}

func (f *fakeNode) WaitMessage(ctx context.Context, msg cid.Cid) (node.MsgWait, error) {
	return f.waitMessage(ctx, msg)
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
