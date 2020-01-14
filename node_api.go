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
	WaitForSelfDeals(context.Context, cid.Cid) ([]uint64, uint8, error)

	// SendPreCommitSector publishes the miner's pre-commitment of a sector to a
	// particular chain and returns the identity of the corresponding message.
	SendPreCommitSector(ctx context.Context, sectorID uint64, commR []byte, ticket SealTicket, pieces ...Piece) (cid.Cid, error)

	WaitForPreCommitSector(context.Context, cid.Cid) (uint64, uint8, error)

	// SendProveCommitSector publishes the miner's seal proof and returns the
	// the identity of the corresponding message.
	SendProveCommitSector(ctx context.Context, sectorID uint64, proof []byte, dealids ...uint64) (cid.Cid, error)

	WaitForProveCommitSector(context.Context, cid.Cid) (uint64, uint8, error)

	// GetSealTicket produces a ticket from the chain to which the miner commits
	// when they start encoding a sector.
	GetSealTicket(context.Context) (SealTicket, error)

	// GetSealSeed requests that a seal seed be provided through the return channel the given block interval after the preCommitMsg arrives on chain.
	// It expects to be notified through the invalidated channel if a re-org sets the chain back to before the height at the interval.
	GetSealSeed(ctx context.Context, preCommitMsg cid.Cid, interval uint64) (seed <-chan SealSeed, err <-chan error, invalidated <-chan struct{}, done <-chan struct{})
}

type PieceInfo struct {
	Size  uint64
	CommP [32]byte
}
