package test

import (
	"context"
	"io"

	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-sectorbuilder"
)

type fakeSectorBuilder struct{}

func (fakeSectorBuilder) SectorSize() uint64 {
	return 1024
}

func (fakeSectorBuilder) SealPreCommit(ctx context.Context, sectorID uint64, ticket ffi.SealTicket, pieces []sectorbuilder.PublicPieceInfo) (sectorbuilder.RawSealPreCommitOutput, error) {
	return sectorbuilder.RawSealPreCommitOutput{}, nil
}

func (fakeSectorBuilder) SealCommit(ctx context.Context, sectorID uint64, ticket ffi.SealTicket, seed ffi.SealSeed, pieces []sectorbuilder.PublicPieceInfo, rspco sectorbuilder.RawSealPreCommitOutput) (proof []byte, err error) {
	return []byte{42}, nil
}

func (fakeSectorBuilder) RateLimit() func() {
	return func() {}
}

func (fakeSectorBuilder) AddPiece(pieceSize uint64, sectorID uint64, file io.Reader, existingPieceSizes []uint64) (sectorbuilder.PublicPieceInfo, error) {
	return sectorbuilder.PublicPieceInfo{Size: pieceSize}, nil
}

//nolint:golint
func (fakeSectorBuilder) AcquireSectorId() (uint64, error) {
	return 42, nil
}
