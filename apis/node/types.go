package node

import (
	commcid "github.com/filecoin-project/go-fil-commcid"
	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"golang.org/x/xerrors"
)

// TipSetToken is the implementation-nonspecific identity for a tipset.
type TipSetToken []byte

type FinalityReached struct{}
type SeedInvalidated struct{}

type MsgWait struct {
	Receipt MessageReceipt
	Height  abi.ChainEpoch
}

type MessageReceipt struct {
	ExitCode uint8
	Return   []byte
	GasUsed  abi.TokenAmount
}

type PieceInfo struct {
	Size  uint64
	CommP [32]byte
}

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

type PieceWithDealInfo struct {
	Piece    abi.PieceInfo
	DealInfo DealInfo
}

func (p *PieceWithDealInfo) SB() (out sectorbuilder.PublicPieceInfo, err error) {
	out.Size = p.Piece.Size.Unpadded()

	commP, err := commcid.CIDToPieceCommitmentV1(p.Piece.PieceCID)
	if err != nil {
		return sectorbuilder.PublicPieceInfo{}, xerrors.Errorf("failed to map CID to CommP: ", err)
	}

	copy(out.CommP[:], commP)

	return out, nil
}

// PieceWithOptionalDealInfo is a tuple of piece info and optional deal
type PieceWithOptionalDealInfo struct {
	Piece    abi.PieceInfo
	DealInfo *DealInfo // nil for pieces which do not yet appear in self-deals
}

// DealInfo is a tuple of deal identity and its schedule
type DealInfo struct {
	DealID       abi.DealID
	DealSchedule DealSchedule
}

// DealSchedule communicates the time interval of a storage deal. The deal must
// appear in a sealed (proven) sector no later than StartEpoch, otherwise it
// is invalid.
type DealSchedule struct {
	StartEpoch abi.ChainEpoch
	EndEpoch   abi.ChainEpoch
}
