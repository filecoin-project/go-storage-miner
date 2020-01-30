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

	// SendProveCommitSector publishes the miner's seal proof and returns the
	// the identity of the corresponding message.
	SendProveCommitSector(ctx context.Context, sectorID uint64, proof []byte, dealids ...uint64) (cid.Cid, error)

	// WaitForProveCommitSector blocks until the provided message is mined into
	// a block.
	WaitForProveCommitSector(context.Context, cid.Cid) (uint64, uint8, error)

	// SendReportFaults reports sectors as faulty.
	SendReportFaults(ctx context.Context, sectorIDs ...uint64) (cid.Cid, error)

	// WaitForReportFaults blocks until the provided message is mined into a
	// block.
	WaitForReportFaults(context.Context, cid.Cid) (uint8, error)

	// GetSealTicket produces a ticket from the chain to which the miner commits
	// when they start encoding a sector.
	GetSealTicket(context.Context) (SealTicket, error)

	// GetReplicaCommitmentByID produces the CommR associated with the given
	// sector as it appears in a pre-commit message. If the sector has not been
	// pre-committed, wasFound will be false.
	GetReplicaCommitmentByID(ctx context.Context, sectorID uint64) (commR []byte, wasFound bool, err error)

	// GetSealSeed requests that a seal seed be provided through the return channel the given block interval after the preCommitMsg arrives on chain.
	// It expects to be notified through the invalidated channel if a re-org sets the chain back to before the height at the interval.
	GetSealSeed(ctx context.Context, preCommitMsg cid.Cid, interval uint64) (seed <-chan SealSeed, err <-chan error, invalidated <-chan struct{}, done <-chan struct{})

	// CheckPieces ensures that the provides pieces' metadata exist in
	// not yet-expired on-chain storage deals.
	CheckPieces(ctx context.Context, sectorID uint64, pieces []Piece) *CheckPiecesError

	// CheckSealing ensures that the given data commitment matches the
	// commitment of the given pieces associated with the given deals. The
	// ordering of the deals must match the ordering of the related pieces in
	// the sector.
	CheckSealing(ctx context.Context, commD []byte, dealIDs []uint64) *CheckSealingError
}

type PieceInfo struct {
	Size  uint64
	CommP [32]byte
}

type CheckPiecesErrorType = uint64

const (
	UndefinedCheckPiecesErrorType CheckPiecesErrorType = iota
	CheckPiecesAPI
	CheckPiecesInvalidDeals
	CheckPiecesExpiredDeals
)

type CheckPiecesError struct {
	inner error
	EType CheckPiecesErrorType
}

func (c CheckPiecesError) Error() string {
	return c.inner.Error()
}

type CheckSealingErrorType = uint64

const (
	UndefinedCheckSealingErrorType CheckSealingErrorType = iota
	CheckSealingAPI
	CheckSealingBadCommD
	CheckSealingExpiredTicket
)

type CheckSealingError struct {
	inner error
	EType CheckSealingErrorType
}

func (c CheckSealingError) Error() string {
	return c.inner.Error()
}
