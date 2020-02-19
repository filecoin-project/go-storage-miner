package node

import (
	commcid "github.com/filecoin-project/go-fil-commcid"
	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
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

type Piece struct {
	DealID   abi.DealID
	Size     abi.PaddedPieceSize
	PieceCID cid.Cid
}

func (p *Piece) SB() (out sectorbuilder.PublicPieceInfo, err error) {
	out.Size = p.Size.Unpadded()

	commP, err := commcid.CIDToPieceCommitmentV1(p.PieceCID)
	if err != nil {
		return sectorbuilder.PublicPieceInfo{}, xerrors.Errorf("failed to map CID to CommP: ", err)
	}

	copy(out.CommP[:], commP)

	return out, nil
}
