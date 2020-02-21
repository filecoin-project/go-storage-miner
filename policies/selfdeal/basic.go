package selfdeal

import (
	"context"

	"github.com/filecoin-project/specs-actors/actors/abi"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

type Chain interface {
	GetChainHead(ctx context.Context) (node.TipSetToken, abi.ChainEpoch, error)
}

// BasicPolicy satisfies selfdeal.Policy. It has two modes:
//
// Mode 1: The sector contains a non-zero quantity of client-provided pieces
// Mode 2: The sector contains only self-deal pieces
//
// The BasicPolicy#Schedule method is given a slice of the pieces which the
// miner intends to seal into the sector, and from that slice picks either the
// first or second mode.
//
// If we're in Mode 1: The self-deal start and end-dates of the returned
// DealSchedule will be set to the minimum and maximum start and end epoch
// for pieces in the sector as long as at least one piece's deal contains both
// start and end-dates which is in the future.
//
// If we're in Mode 2: The self-deal start date will be set to the current
// epoch + provingDelay. The self-deal end date will be set to the current epoch
// + provingDelay + duration.
type BasicPolicy struct {
	api Chain

	// An estimate for the number of blocks between the current chain head and
	// when the sector should have been proven. Used to compute the self-deal
	// start epoch.
	provingDelay abi.ChainEpoch

	// The number of epochs for which the self-dealing miner will be required to
	// honor the self-deal.
	duration abi.ChainEpoch
}

// NewBasicPolicy produces a new BasicPolicy.
func NewBasicPolicy(api Chain, delay abi.ChainEpoch, duration abi.ChainEpoch) BasicPolicy {
	return BasicPolicy{api: api, provingDelay: delay, duration: duration}
}

// Schedule produces the deal schedule for the self-deals to reside in a sector
// containing the provided pieces.
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
