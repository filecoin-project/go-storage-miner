package test

import (
	"context"
	"testing"

	commcid "github.com/filecoin-project/go-fil-commcid"

	"github.com/filecoin-project/specs-actors/actors/abi"

	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/go-storage-miner/policies/selfdeal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChain struct {
	h abi.ChainEpoch
}

func (f *fakeChain) GetChainHead(ctx context.Context) (node.TipSetToken, abi.ChainEpoch, error) {
	return []byte{1, 2, 3}, f.h, nil
}

func TestBasicPolicyNoExistingSchedules(t *testing.T) {
	policy := selfdeal.NewBasicPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 10, 100)

	pieces := []node.PieceWithOptionalDealInfo{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: commcid.ReplicaCommitmentV1ToCID([]byte{1, 2, 3}),
			},
			DealInfo: nil,
		},
	}

	s1, err := policy.Schedule(context.Background())
	require.NoError(t, err)

	s2, err := policy.Schedule(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, int(s1.StartEpoch), int(s2.StartEpoch))
	assert.Equal(t, int(s1.EndEpoch), int(s2.EndEpoch))
}

func TestBasicPolicyMostConstrictiveSchedule(t *testing.T) {
	policy := selfdeal.NewBasicPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 10, 100)

	pieces := []node.PieceWithOptionalDealInfo{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: commcid.ReplicaCommitmentV1ToCID([]byte{1, 2, 3}),
			},
			DealInfo: &node.DealInfo{
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
			DealInfo: nil,
		},
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: commcid.ReplicaCommitmentV1ToCID([]byte{1, 2, 3}),
			},
			DealInfo: &node.DealInfo{
				DealID: abi.DealID(43),
				DealSchedule: node.DealSchedule{
					StartEpoch: abi.ChainEpoch(80),
					EndEpoch:   abi.ChainEpoch(100),
				},
			},
		},
	}

	s, err := policy.Schedule(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, 70, int(s.StartEpoch))
	assert.Equal(t, 100, int(s.EndEpoch))
}

func TestBasicDurationIgnoresExistingScheduleIfExpired(t *testing.T) {
	policy := selfdeal.NewBasicPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 10, 100)

	pieces := []node.PieceWithOptionalDealInfo{
		{
			Piece: abi.PieceInfo{
				Size:     abi.PaddedPieceSize(1024),
				PieceCID: commcid.ReplicaCommitmentV1ToCID([]byte{1, 2, 3}),
			},
			DealInfo: &node.DealInfo{
				DealID: abi.DealID(44),
				DealSchedule: node.DealSchedule{
					StartEpoch: abi.ChainEpoch(1),
					EndEpoch:   abi.ChainEpoch(100),
				},
			},
		},
	}

	s, err := policy.Schedule(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, 65, int(s.StartEpoch))
	assert.Equal(t, 165, int(s.EndEpoch))
}
