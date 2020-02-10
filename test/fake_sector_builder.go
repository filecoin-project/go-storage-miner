package test

import (
	"context"
	"io"

	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/specs-actors/actors/abi"
)

type fakeSectorBuilder struct{}

func (fakeSectorBuilder) SectorSize() abi.SectorSize {
	return 1024
}

func (fakeSectorBuilder) SealPreCommit(ctx context.Context, sectorNum abi.SectorNumber, ticket ffi.SealTicket, pieces []sectorbuilder.PublicPieceInfo) (sectorbuilder.RawSealPreCommitOutput, error) {
	return sectorbuilder.RawSealPreCommitOutput{}, nil
}

func (fakeSectorBuilder) SealCommit(ctx context.Context, sectorNum abi.SectorNumber, ticket ffi.SealTicket, seed ffi.SealSeed, pieces []sectorbuilder.PublicPieceInfo, rspco sectorbuilder.RawSealPreCommitOutput) (proof []byte, err error) {
	return []byte{42}, nil
}

func (fakeSectorBuilder) RateLimit() func() {
	return func() {}
}

func (fakeSectorBuilder) AddPiece(ctx context.Context, pieceSize abi.UnpaddedPieceSize, sectorNum abi.SectorNumber, file io.Reader, existingPieceSizes []abi.UnpaddedPieceSize) (sectorbuilder.PublicPieceInfo, error) {
	return sectorbuilder.PublicPieceInfo{Size: pieceSize}, nil
}

//nolint:golint
func (fakeSectorBuilder) AcquireSectorNumber() (abi.SectorNumber, error) {
	return 42, nil
}

func (fakeSectorBuilder) DropStaged(context.Context, abi.SectorNumber) error {
	return nil
}

func (fakeSectorBuilder) FinalizeSector(context.Context, abi.SectorNumber) error {
	return nil
}
