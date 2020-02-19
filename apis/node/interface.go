package node

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
)

type Interface interface {
	// SendSelfDeals publishes storage deals using the provided inputs and
	// returns the identity of the corresponding PublishStorageDeals message.
	SendSelfDeals(context.Context, ...abi.PieceInfo) (cid.Cid, error)

	// WaitForSelfDeals blocks until the PublishStorageDeals message is mined
	// into a block and then returns the referenced deal IDs.
	WaitForSelfDeals(context.Context, cid.Cid) ([]abi.DealID, uint8, error)

	// GetMinerWorkerAddress produces the worker address associated with the
	// miner.
	GetMinerWorkerAddress(context.Context) (address.Address, error)

	// SendPreCommitSector publishes the miner's pre-commitment of a sector to a
	// particular chain and returns the identity of the corresponding message.
	SendPreCommitSector(ctx context.Context, sectorNum abi.SectorNumber, commR []byte, ticket SealTicket, pieces ...Piece) (cid.Cid, error)

	// SendProveCommitSector publishes the miner's seal proof and returns the
	// the identity of the corresponding message.
	SendProveCommitSector(ctx context.Context, sectorNum abi.SectorNumber, proof []byte, dealids ...abi.DealID) (cid.Cid, error)

	// WaitForProveCommitSector blocks until the provided message is mined into
	// a block.
	WaitForProveCommitSector(context.Context, cid.Cid) (uint8, error)

	// SendReportFaults reports sectors as faulty.
	SendReportFaults(ctx context.Context, sectorIDs ...abi.SectorNumber) (cid.Cid, error)

	// WaitForReportFaults blocks until the provided message is mined into a
	// block.
	WaitForReportFaults(context.Context, cid.Cid) (uint8, error)

	// GetSealTicket produces a ticket from the chain to which the miner commits
	// when they start encoding a sector.
	GetSealTicket(context.Context) (SealTicket, error)

	// GetReplicaCommitmentByID produces the CommR associated with the given
	// sector as it appears in a pre-commit message. If the sector has not been
	// pre-committed, wasFound will be false.
	GetReplicaCommitmentByID(ctx context.Context, sectorNum abi.SectorNumber) (commR []byte, wasFound bool, err error)

	// GetSealSeed requests that a seal seed be provided through the return channel the given block interval after the preCommitMsg arrives on chain.
	// It expects to be notified through the invalidated channel if a re-org sets the chain back to before the height at the interval.
	GetSealSeed(ctx context.Context, preCommitMsg cid.Cid, interval uint64) (<-chan SealSeed, <-chan SeedInvalidated, <-chan FinalityReached, <-chan GetSealSeedError)

	// CheckPieces ensures that the provides pieces' metadata exist in
	// not yet-expired on-chain storage deals.
	CheckPieces(ctx context.Context, sectorNum abi.SectorNumber, pieces []Piece) *CheckPiecesError

	// CheckSealing ensures that the given data commitment matches the
	// commitment of the given pieces associated with the given deals. The
	// ordering of the deals must match the ordering of the related pieces in
	// the sector.
	CheckSealing(ctx context.Context, commD []byte, dealIDs []abi.DealID, ticket SealTicket) *CheckSealingError

	// WalletHas checks the wallet for the key associated with the provided
	// address.
	WalletHas(ctx context.Context, addr address.Address) (bool, error)
}
