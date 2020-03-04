package sectorbuilder

import (
	"context"
	"io"

	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
)

// Interface provides a method set used by the miner in order to interact
// with the go-sectorbuilder package. This method set exposes a subset of the
// methods defined on the sectorbuilder.Interface interface.
type Interface interface {
	AcquireSectorNumber() (abi.SectorNumber, error)
	AddPiece(context.Context, abi.UnpaddedPieceSize, abi.SectorNumber, io.Reader, []abi.UnpaddedPieceSize) (abi.PieceInfo, error)
	DropStaged(context.Context, abi.SectorNumber) error
	FinalizeSector(context.Context, abi.SectorNumber) error
	RateLimit() func()
	SealCommit(ctx context.Context, sectorNum abi.SectorNumber, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, sealedCID cid.Cid, unsealedCID cid.Cid) (proof []byte, err error)
	SealPreCommit(ctx context.Context, sectorNum abi.SectorNumber, ticket abi.SealRandomness, pieces []abi.PieceInfo) (sealedCID cid.Cid, unsealedCID cid.Cid, err error)
	SectorSize() abi.SectorSize
	SealProofType() abi.RegisteredProof
}

var _ Interface = new(sectorbuilder.SectorBuilder)
