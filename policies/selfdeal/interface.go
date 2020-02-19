package selfdeal

import (
	"context"

	"github.com/filecoin-project/specs-actors/actors/abi"
)

// Schedule communicates the time bounds of a self-deal. The self-deal must
// appear in a sealed (proven) sector no later than StartEpoch, otherwise it
// is invalid.
type Schedule struct {
	StartEpoch abi.ChainEpoch
	EndEpoch   abi.ChainEpoch
}

type Policy interface {
	Schedule(ctx context.Context, pieces ...abi.PieceInfo) (Schedule, error)
}
