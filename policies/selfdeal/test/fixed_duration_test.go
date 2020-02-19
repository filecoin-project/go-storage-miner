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

func TestFixedDurationScheduleIgnoresSelfDealPieces(t *testing.T) {
	policy := selfdeal.NewFixedDurationPolicy(&fakeChain{
		h: abi.ChainEpoch(55),
	}, 10, 100)

	pieces := []abi.PieceInfo{{
		Size:     abi.PaddedPieceSize(1024),
		PieceCID: commcid.ReplicaCommitmentV1ToCID([]byte{1, 2, 3}),
	}, {
		Size:     abi.PaddedPieceSize(1024),
		PieceCID: commcid.ReplicaCommitmentV1ToCID([]byte{1, 2, 3}),
	}}

	s1, err := policy.Schedule(context.Background())
	require.NoError(t, err)

	s2, err := policy.Schedule(context.Background(), pieces...)
	require.NoError(t, err)

	assert.Equal(t, int(s1.StartEpoch), int(s2.StartEpoch))
	assert.Equal(t, int(s1.EndEpoch), int(s2.EndEpoch))
}

func TestFixedDurationScheduleBounds(t *testing.T) {
	delay := abi.ChainEpoch(10)
	duration := abi.ChainEpoch(100)
	headEpoch := abi.ChainEpoch(77)

	policy := selfdeal.NewFixedDurationPolicy(&fakeChain{
		h: headEpoch,
	}, delay, duration)

	s, err := policy.Schedule(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int(s.StartEpoch), int(headEpoch+delay))
	assert.Equal(t, int(duration), int(s.EndEpoch)-int(s.StartEpoch))
}
