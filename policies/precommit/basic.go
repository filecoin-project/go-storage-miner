package precommit

import (
	"context"

	"github.com/filecoin-project/specs-actors/actors/abi"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

type Chain interface {
	GetChainHead(ctx context.Context) (node.TipSetToken, abi.ChainEpoch, error)
}

// BasicPolicy satisfies precommit.Policy. It has two modes:
//
// Mode 1: The sector contains a non-zero quantity of pieces
// Mode 2: The sector contains no pieces
//
// The BasicPolicy#Expiration method is given a slice of the pieces which the
// miner has encoded into the sector, and from that slice picks either the
// first or second mode.
//
// If we're in Mode 1: The pre-commit expiration epoch will be the maximum
// deal end epoch of a piece in the sector.
//
// If we're in Mode 2: The pre-commit expiration epoch will be set to the
// current epoch + the provided default duration.
type BasicPolicy struct {
	api Chain

	duration abi.ChainEpoch
}

// NewBasicPolicy produces a BasicPolicy
func NewBasicPolicy(api Chain, duration abi.ChainEpoch) BasicPolicy {
	return BasicPolicy{
		api:      api,
		duration: duration,
	}
}

// Expiration produces the pre-commit sector expiration epoch for an encoded
// replica containing the provided enumeration of pieces and deals.
func (p *BasicPolicy) Expiration(ctx context.Context, pdis ...node.PieceWithDealInfo) (abi.ChainEpoch, error) {
	_, epoch, err := p.api.GetChainHead(ctx)
	if err != nil {
		return 0, nil
	}

	var end *abi.ChainEpoch

	for _, p := range pdis {
		if p.DealInfo.DealSchedule.EndEpoch < epoch {
			log.Warnf("piece schedule %+v ended before current epoch %d", p, epoch)
			continue
		}

		if end == nil || *end < p.DealInfo.DealSchedule.EndEpoch {
			end = &p.DealInfo.DealSchedule.EndEpoch
		}
	}

	if end == nil {
		tmp := epoch + p.duration
		end = &tmp
	}

	return *end, nil
}
