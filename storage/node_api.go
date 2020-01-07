package storage

import (
	"context"

	"github.com/ipfs/go-cid"
)

type SealProof [32]byte
type UnpaddedPieceBytes uint64
type PaddedPieceBytes uint64
type SectorId uint64
type DealId uint64

type NodeAPI interface {
	// WaitMsg blocks until a message with the provided identity is mined into a
	// block and returns message processing metadata.
	WaitMsg(context.Context, cid.Cid) (MsgProcessedInfo, error)

	// SendSelfDeals publishes storage deals using the provided inputs and
	// returns the identity of the corresponding PublishStorageDeals message.
	SendSelfDeals(context.Context, ...PieceInfo) (cid.Cid, error)

	// SendPreCommitSector publishes the miner's pre-commitment of a sector to a
	// particular chain and returns the identity of the corresponding message.
	SendPreCommitSector(context.Context, SectorId, SealTicket, ...PieceInfo) (cid.Cid, error)

	// SendProveCommitSector publishes the miner's seal proof and returns the
	// the identity of the corresponding message.
	SendProveCommitSector(context.Context, SectorId, SealProof, ...DealId) (cid.Cid, error)

	// SendDeclareFaults publishes a notification that the miner has detected
	// one or more storage faults and returns the identity of the corresponding
	// message.
	SendDeclareFaults(context.Context, ...SectorId) (cid.Cid, error)

	// SealTicket produces a ticket from the chain to which the miner commits
	// when they start encoding a sector.
	SealTicket(context.Context) (SealTicket, error)

	// SetSealSeedHandler sets the seal seed handler associated with the
	// provided pre-commit message. Any handler previously associated with the
	// provided pre-commit message is replaced.
	SetSealSeedHandler(ctx context.Context, preCommitMsg cid.Cid, h SealSeedHandler)
}

type PieceInfo struct {
	Size  UnpaddedPieceBytes
	CommP [32]byte
}

type MsgProcessedInfo struct {
	Height   uint64
	ExitCode uint8 // TODO: should we replace this with an opaque string or `error`?
}

type SealSeedHandler interface {
	// SealSeedAvailable is called when a seal seed becomes available. This
	// method may be called multiple times if re-orgs occur.
	SealSeedAvailable(SealSeed)

	// SealSeedInvalidated is called when a seal seed becomes invalid. This
	// method may be called multiple times if a re-org occurs.
	SealSeedInvalidated()
}
