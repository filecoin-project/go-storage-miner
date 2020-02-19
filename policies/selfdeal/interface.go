package selfdeal

import (
	"context"

	"github.com/filecoin-project/specs-actors/actors/abi"
)

type Schedule struct {
	StartEpoch  abi.ChainEpoch
	ExpiryEpoch abi.ChainEpoch
}

type Policy interface {
	Schedule(ctx context.Context, pieces ...abi.PieceInfo) (Schedule, error)
}
