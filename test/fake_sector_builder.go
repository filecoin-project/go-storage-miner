package test

import (
	"context"
	"io"

	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-cid"
)

type fakeSectorBuilder struct{}

func (f fakeSectorBuilder) AcquireSectorNumber() (abi.SectorNumber, error) {
	return abi.SectorNumber(42), nil
}

func (f fakeSectorBuilder) AddPiece(ctx context.Context, size abi.UnpaddedPieceSize, s abi.SectorNumber, r io.Reader, existing []abi.UnpaddedPieceSize) (abi.PieceInfo, error) {
	commP := [32]byte{9, 10, 11}

	return abi.PieceInfo{
		Size:     size.Padded(),
		PieceCID: commcid.PieceCommitmentV1ToCID(commP[:]),
	}, nil
}

func (f fakeSectorBuilder) DropStaged(context.Context, abi.SectorNumber) error {
	return nil
}

func (f fakeSectorBuilder) FinalizeSector(context.Context, abi.SectorNumber) error {
	return nil
}

func (f fakeSectorBuilder) RateLimit() func() {
	return func() {}
}

func (f fakeSectorBuilder) SealCommit(ctx context.Context, sectorNum abi.SectorNumber, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, sealedCID cid.Cid, unsealedCID cid.Cid) (proof []byte, err error) {
	return []byte{42}, nil
}

func (f fakeSectorBuilder) SealPreCommit(ctx context.Context, sectorNum abi.SectorNumber, ticket abi.SealRandomness, pieces []abi.PieceInfo) (sealedCID cid.Cid, unsealedCID cid.Cid, err error) {
	commR := [32]byte{1, 2, 3}
	commD := [32]byte{4, 5, 6}

	return commcid.ReplicaCommitmentV1ToCID(commR[:]), commcid.DataCommitmentV1ToCID(commD[:]), nil
}

func (f fakeSectorBuilder) SectorSize() abi.SectorSize {
	return abi.SectorSize(1024)
}

func (f fakeSectorBuilder) SealProofType() abi.RegisteredProof {
	return abi.RegisteredProof_StackedDRG2KiBSeal
}
