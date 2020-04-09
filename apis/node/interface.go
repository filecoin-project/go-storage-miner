package node

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
	fsm "github.com/filecoin-project/storage-fsm"
)

type Interface interface {
	// Sealing provides an interface to the node required by the finite state
	// machine.
	fsm.SealingAPI

	// Events calls various callbacks when the chain progresses to (or rolls
	// back to) an epoch.
	fsm.Events

	// GetMinerWorkerAddress produces the worker address associated with the
	// provided miner address.
	GetMinerWorkerAddress(ctx context.Context, maddr address.Address, tok fsm.TipSetToken) (address.Address, error)

	// GetSealTicket produces a ticket from the chain to which the miner commits
	// when they start encoding a sector.
	GetSealTicket(ctx context.Context, tok fsm.TipSetToken) (abi.SealRandomness, abi.ChainEpoch, error)

	// WalletHas checks the wallet for the key associated with the provided
	// address.
	WalletHas(ctx context.Context, addr address.Address) (bool, error)
}
