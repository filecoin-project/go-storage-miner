package test

import (
	"context"
	"testing"

	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/specs-actors/actors/abi"

	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/go-storage-miner/policies/precommit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChain struct {
	h abi.ChainEpoch
}

func (f *fakeChain) GetChainHead(ctx context.Context) (node.TipSetToken, abi.ChainEpoch, error) {
	return []byte{1, 2, 3}, f.h, nil
}

func TestBasicPolicyEmptySector(t *testing.T) {
	policy := precommit.NewBasicPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 10)

	exp, err := policy.Expiration(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 65, int(exp))
}

func TestBasicPolicyMostConstrictiveSchedule(t *testing.T) {
	policy := precommit.NewBasicPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 100)

	pieces := []node.PieceWithDealInfo{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: commcid.ReplicaCommitmentV1ToCID([]byte{1, 2, 3}),
			},
			DealInfo: node.DealInfo{
				DealID: abi.DealID(42),
				DealSchedule: node.DealSchedule{
					StartEpoch: abi.ChainEpoch(70),
					EndEpoch:   abi.ChainEpoch(75),
				},
			},
		},
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: commcid.ReplicaCommitmentV1ToCID([]byte{1, 2, 3}),
			},
			DealInfo: node.DealInfo{
				DealID: abi.DealID(43),
				DealSchedule: node.DealSchedule{
					StartEpoch: abi.ChainEpoch(80),
					EndEpoch:   abi.ChainEpoch(100),
				},
			},
		},
	}

	exp, err := policy.Expiration(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, 100, int(exp))
}

func TestBasicPolicyIgnoresExistingScheduleIfExpired(t *testing.T) {
	policy := precommit.NewBasicPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 100)

	pieces := []node.PieceWithDealInfo{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: commcid.ReplicaCommitmentV1ToCID([]byte{1, 2, 3}),
			},
			DealInfo: node.DealInfo{
				DealID: abi.DealID(44),
				DealSchedule: node.DealSchedule{
					StartEpoch: abi.ChainEpoch(1),
					EndEpoch:   abi.ChainEpoch(10),
				},
			},
		},
	}

	exp, err := policy.Expiration(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, 155, int(exp))
}
