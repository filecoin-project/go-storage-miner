package storage

import (
	commcid "github.com/filecoin-project/go-fil-commcid"
	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

type SealTicket struct {
	BlockHeight uint64
	TicketBytes []byte
}

func (t *SealTicket) SB() sectorbuilder.SealTicket {
	out := sectorbuilder.SealTicket{BlockHeight: t.BlockHeight}
	copy(out.TicketBytes[:], t.TicketBytes)
	return out
}

type SealSeed struct {
	BlockHeight uint64
	TicketBytes []byte
}

func (t *SealSeed) SB() sectorbuilder.SealSeed {
	out := sectorbuilder.SealSeed{BlockHeight: t.BlockHeight}
	copy(out.TicketBytes[:], t.TicketBytes)
	return out
}

func (t *SealSeed) Equals(o *SealSeed) bool {
	return string(t.TicketBytes) == string(o.TicketBytes) && t.BlockHeight == o.BlockHeight
}

type Piece struct {
	DealID   abi.DealID
	Size     abi.PaddedPieceSize
	PieceCID cid.Cid
}

func (p *Piece) ppi() (out sectorbuilder.PublicPieceInfo, err error) {
	out.Size = p.Size.Unpadded()

	commP, err := commcid.CIDToPieceCommitmentV1(p.PieceCID)
	if err != nil {
		return sectorbuilder.PublicPieceInfo{}, xerrors.Errorf("failed to map CID to CommP: ", err)
	}

	copy(out.CommP[:], commP)

	return out, nil
}

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

	Pieces []Piece

	// PreCommit
	CommD  []byte
	CommR  []byte
	Proof  []byte
	Ticket SealTicket

	PreCommitMessage *cid.Cid

	// WaitSeed
	Seed SealSeed

	// Committing
	CommitMessage *cid.Cid

	// Faults
	FaultReportMsg *cid.Cid

	// Debug
	LastErr string

	Log []Log
}

func (t *SectorInfo) pieceInfos() ([]sectorbuilder.PublicPieceInfo, error) {
	out := make([]sectorbuilder.PublicPieceInfo, len(t.Pieces))
	for i, piece := range t.Pieces {
		ppi, err := piece.ppi()
		if err != nil {
			return nil, xerrors.Errorf("failed to map to PublicPieceInfo: ", err)
		}

		out[i] = ppi
	}

	return out, nil
}

func (t *SectorInfo) deals() []abi.DealID {
	out := make([]abi.DealID, len(t.Pieces))
	for i, piece := range t.Pieces {
		out[i] = piece.DealID
	}

	return out
}

func (t *SectorInfo) existingPieces() []abi.PaddedPieceSize {
	out := make([]abi.PaddedPieceSize, len(t.Pieces))
	for i, piece := range t.Pieces {
		out[i] = piece.Size
	}

	return out
}

func (t *SectorInfo) rspco() sectorbuilder.RawSealPreCommitOutput {
	var out sectorbuilder.RawSealPreCommitOutput

	copy(out.CommD[:], t.CommD)
	copy(out.CommR[:], t.CommR)

	return out
}
