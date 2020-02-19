package selfdeal

import (
	"context"

	"github.com/filecoin-project/specs-actors/actors/abi"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

type Chain interface {
	GetChainHead(ctx context.Context) (node.TipSetToken, abi.ChainEpoch, error)
}

type FixedDurationPolicy struct {
	api Chain

	// An estimate for the number of blocks between the current chain head and
	// when a sector should have been proven. Used to compute the self-deal
	// StartEpoch.
	provingDelay abi.ChainEpoch

	// The number of epochs for which the self-dealing miner will receive power.
	duration abi.ChainEpoch
}

// NewFixedDurationPolicy produces a new fixed duration self-deal policy.
func NewFixedDurationPolicy(api Chain, delay abi.ChainEpoch, duration abi.ChainEpoch) FixedDurationPolicy {
	return FixedDurationPolicy{api: api, provingDelay: delay, duration: duration}
}

// Schedule produces the deal terms for this fixed duration self-deal policy.
func (p *FixedDurationPolicy) Schedule(ctx context.Context, pieces ...abi.PieceInfo) (Schedule, error) {
	_, epoch, err := p.api.GetChainHead(ctx)
	if err != nil {
		return Schedule{}, err
	}

	return Schedule{
		StartEpoch: epoch + p.provingDelay,
		EndEpoch:   epoch + p.provingDelay + p.duration,
	}, nil
}
