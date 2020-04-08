package node

import (
	"context"

	"github.com/filecoin-project/specs-actors/actors/abi"

	sealing2 "github.com/filecoin-project/storage-fsm"

	"github.com/filecoin-project/go-address"
)

type Interface interface {
	// Sealing provides an interface to the node required by the finite state
	// machine.
	sealing2.SealingAPI

	// Events calls various callbacks when the chain progresses to (or rolls
	// back to) an epoch.
	sealing2.Events

	// GetMinerWorkerAddress produces the worker address associated with the
	// provided miner address.
	GetMinerWorkerAddress(ctx context.Context, maddr address.Address, tok sealing2.TipSetToken) (address.Address, error)

	// GetSealTicket produces a ticket from the chain to which the miner commits
	// when they start encoding a sector.
	GetSealTicket(ctx context.Context, tok sealing2.TipSetToken) (abi.SealRandomness, abi.ChainEpoch, error)

	// WalletHas checks the wallet for the key associated with the provided
	// address.
	WalletHas(ctx context.Context, addr address.Address) (bool, error)
}
