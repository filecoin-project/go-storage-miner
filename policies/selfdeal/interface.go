package selfdeal

import (
	"context"

	"github.com/filecoin-project/go-storage-miner/apis/node"

	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("selfdeals")

type Policy interface {
	Schedule(ctx context.Context, pdis ...node.PieceWithOptionalDealInfo) (node.DealSchedule, error)
}
