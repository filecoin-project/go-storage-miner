package sectorbuilder

import (
	"context"
	"io"

	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/specs-actors/actors/abi"
)

// Interface provides a method set used by the miner in order to interact
// with the go-sectorbuilder package. This method set exposes a subset of the
// methods defined on the sectorbuilder.Interface interface.
type Interface interface {
	AcquireSectorNumber() (abi.SectorNumber, error)
	AddPiece(ctx context.Context, pieceSize abi.UnpaddedPieceSize, sectorNumber abi.SectorNumber, file io.Reader, existingPieceSizes []abi.UnpaddedPieceSize) (sectorbuilder.PublicPieceInfo, error)
	DropStaged(context.Context, abi.SectorNumber) error
	FinalizeSector(context.Context, abi.SectorNumber) error
	RateLimit() func()
	SealCommit(ctx context.Context, sectorNumber abi.SectorNumber, ticket ffi.SealTicket, seed ffi.SealSeed, pieces []sectorbuilder.PublicPieceInfo, rspco sectorbuilder.RawSealPreCommitOutput) (proof []byte, err error)
	SealPreCommit(ctx context.Context, sectorNumber abi.SectorNumber, ticket ffi.SealTicket, pieces []sectorbuilder.PublicPieceInfo) (sectorbuilder.RawSealPreCommitOutput, error)
	SectorSize() abi.SectorSize
}

var _ Interface = new(sectorbuilder.SectorBuilder)
