package sealing

import (
	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
)

type Log struct {
	Timestamp uint64
	Trace     string // for errors

	Message string

	// additional data (Event info)
	Kind string
}

type SectorInfo struct {
	State     SectorState
	SectorNum abi.SectorNumber
	Nonce     uint64 // TODO: remove

	// Packing

	Pieces []node.PieceWithDealInfo

	// PreCommit
	CommD  []byte
	CommR  []byte
	Proof  []byte
	Ticket node.SealTicket

	PreCommitMessage *cid.Cid

	// WaitSeed
	Seed node.SealSeed

	// Committing
	CommitMessage *cid.Cid

	// Faults
	FaultReportMsg *cid.Cid

	// Debug
	LastErr string

	Log []Log
}

func (t *SectorInfo) deals() []abi.DealID {
	out := make([]abi.DealID, len(t.Pieces))
	for i, piece := range t.Pieces {
		out[i] = piece.DealInfo.DealID
	}

	return out
}

func (t *SectorInfo) existingPieces() []abi.PaddedPieceSize {
	out := make([]abi.PaddedPieceSize, len(t.Pieces))
	for i, piece := range t.Pieces {
		out[i] = piece.Piece.Size
	}

	return out
}
