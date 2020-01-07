package storage

import (
	"context"
	"github.com/ipfs/go-cid"
)

type SelfDeal struct {
	size uint64
	CommP [32]byte
}

type NodeAPI interface {
	SendSelfDeals(ctx context.Context, sizes ...SelfDeal) (cid.Cid, error)
	StateWaitMsg(ctx context.Context, messageCid cid.Cid) error
}
