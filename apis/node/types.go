package node

import (
	"github.com/filecoin-project/specs-actors/actors/abi"
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
	TicketBytes abi.SealRandomness
}

type SealSeed struct {
	BlockHeight uint64
	TicketBytes abi.InteractiveSealRandomness
}

func (t *SealSeed) Equals(o *SealSeed) bool {
	return string(t.TicketBytes) == string(o.TicketBytes) && t.BlockHeight == o.BlockHeight
}

type PieceWithDealInfo struct {
	Piece    abi.PieceInfo
	DealInfo DealInfo
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
