package precommit

import (
	"context"

	"github.com/filecoin-project/specs-actors/actors/abi"
	logging "github.com/ipfs/go-log"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

var log = logging.Logger("precommit")

type Policy interface {
	Expiration(ctx context.Context, pdis ...node.PieceWithDealInfo) (abi.ChainEpoch, error)
}
