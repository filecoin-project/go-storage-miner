package selfdeal

import (
	"context"

	"github.com/filecoin-project/specs-actors/actors/abi"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

type Chain interface {
	GetChainHead(ctx context.Context) (node.TipSetToken, abi.ChainEpoch, error)
}

type BasicPolicy struct {
	api Chain

	// An estimate for the number of blocks between the current chain head and
	// when a sector should have been proven. Used to compute the self-deal
	// StartEpoch.
	provingDelay abi.ChainEpoch

	// The number of epochs for which the self-dealing miner will receive power.
	duration abi.ChainEpoch
}

// NewBasicPolicy produces a new fixed duration self-deal policy.
func NewBasicPolicy(api Chain, delay abi.ChainEpoch, duration abi.ChainEpoch) BasicPolicy {
	return BasicPolicy{api: api, provingDelay: delay, duration: duration}
}

// Schedule produces the deal terms for this fixed duration self-deal policy.
func (p *BasicPolicy) Schedule(ctx context.Context, pieces ...node.PieceWithOptionalDealInfo) (node.DealSchedule, error) {
	_, epoch, err := p.api.GetChainHead(ctx)
	if err != nil {
		return node.DealSchedule{}, err
	}

	var start *abi.ChainEpoch
	var end *abi.ChainEpoch

	for _, p := range pieces {
		if p.DealInfo != nil {
			if p.DealInfo.DealSchedule.StartEpoch < epoch {
				log.Warnf("piece schedule %+v starts before current epoch %d", p, epoch)
				continue
			}

			if p.DealInfo.DealSchedule.EndEpoch < epoch {
				log.Warnf("piece schedule %+v ended before current epoch %d", p, epoch)
				continue
			}

			if start == nil || *start > p.DealInfo.DealSchedule.StartEpoch {
				start = &p.DealInfo.DealSchedule.StartEpoch
			}

			if end == nil || *end < p.DealInfo.DealSchedule.EndEpoch {
				end = &p.DealInfo.DealSchedule.EndEpoch
			}
		}
	}

	if start == nil {
		tmp := epoch + p.provingDelay
		start = &tmp
	}

	if end == nil {
		tmp := epoch + p.provingDelay + p.duration
		end = &tmp
	}

	return node.DealSchedule{
		StartEpoch: *start,
		EndEpoch:   *end,
	}, nil
}
