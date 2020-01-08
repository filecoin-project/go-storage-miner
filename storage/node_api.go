package storage

import (
	"context"

	"github.com/ipfs/go-cid"
)

type NodeAPI interface {
	// SendSelfDeals publishes storage deals using the provided inputs and
	// returns the identity of the corresponding PublishStorageDeals message.
	SendSelfDeals(context.Context, ...PieceInfo) (cid.Cid, error)

	// WaitForSelfDeals blocks until a
	WaitForSelfDeals(context.Context, cid.Cid) ([]uint64, error)

	// SendPreCommitSector publishes the miner's pre-commitment of a sector to a
	// particular chain and returns the identity of the corresponding message.
	SendPreCommitSector(context.Context, uint64, SealTicket, ...Piece) (cid.Cid, error)

	WaitForPreCommitSector(context.Context, cid.Cid) (uint64, error)

	// SendProveCommitSector publishes the miner's seal proof and returns the
	// the identity of the corresponding message.
	SendProveCommitSector(context.Context, uint64, []byte, ...uint64) (cid.Cid, error)

	WaitForProveCommitSector(context.Context, cid.Cid) (uint64, error)

	// GetSealTicket produces a ticket from the chain to which the miner commits
	// when they start encoding a sector.
	GetSealTicket(context.Context) (SealTicket, error)

	// SetSealSeedHandler sets the seal seed handler associated with the
	// provided pre-commit message. Any handler previously associated with the
	// provided pre-commit message is replaced.
	SetSealSeedHandler(context.Context, cid.Cid, func(SealSeed), func())
}

type PieceInfo struct {
	Size  uint64
	CommP [32]byte
}
