package sealing

import (
	"context"
	"io"
	"math/bits"
	"math/rand"
	"runtime"

	commcid "github.com/filecoin-project/go-fil-commcid"
	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

func (m *Sealing) pledgeReader(size abi.UnpaddedPieceSize, parts uint64) io.Reader {
	n := uint64(size)

	parts = 1 << bits.Len64(parts) // round down to nearest power of 2
	if n/parts < 127 {
		parts = n / 127
	}

	piece := sectorbuilder.UserBytesForSectorSize(abi.SectorSize((n/127 + n) / parts))

	readers := make([]io.Reader, parts)
	for i := range readers {
		readers[i] = io.LimitReader(rand.New(rand.NewSource(42+int64(i))), int64(piece))
	}

	return io.MultiReader(readers...)
}

func (m *Sealing) pledgeSector(ctx context.Context, sectorNum abi.SectorNumber, existingPieceSizes []abi.UnpaddedPieceSize, sizes ...abi.UnpaddedPieceSize) ([]node.Piece, error) {
	if len(sizes) == 0 {
		return nil, nil
	}

	log.Infof("Pledge %d, contains %+v", sectorNum, existingPieceSizes)

	pieces := make([]abi.PieceInfo, len(sizes))
	for i, size := range sizes {
		commP, err := m.FastPledgeCommitment(size, uint64(1))
		if err != nil {
			return nil, handle("failed to generate pledge commitment: ", err)
		}

		pieces[i] = abi.PieceInfo{
			Size:     size.Padded(),
			PieceCID: commcid.PieceCommitmentV1ToCID(commP[:]),
		}
	}

	_, epoch, err := m.api.GetChainHead(ctx)
	if err != nil {
		return nil, err
	}

	panic("TODO: implement a real policy")

	mcid, err := m.api.SendSelfDeals(ctx, epoch+100, epoch+1000, pieces...)
	if err != nil {
		return nil, err
	}

	dealIDs, exitCode, err := m.api.WaitForSelfDeals(ctx, mcid)
	if err != nil {
		return nil, err
	}

	if exitCode != 0 {
		err := xerrors.Errorf("publishing deal failed: exit %d", exitCode)
		log.Error(err)
		return nil, err
	}

	if len(dealIDs) != len(sizes) {
		err := xerrors.New("got unexpected number of DealIDs from PublishStorageDeals")
		log.Error(err)
		return nil, err
	}

	out := make([]node.Piece, len(sizes))
	for i, size := range sizes {
		ppi, err := m.sb.AddPiece(ctx, size, sectorNum, m.pledgeReader(size, uint64(runtime.NumCPU())), existingPieceSizes)
		if err != nil {
			return nil, xerrors.Errorf("add piece: %w", err)
		}

		existingPieceSizes = append(existingPieceSizes, size)

		out[i] = node.Piece{
			DealID:   dealIDs[i],
			Size:     ppi.Size.Padded(),
			PieceCID: commcid.PieceCommitmentV1ToCID(ppi.CommP[:]),
		}
	}

	return out, nil
}

func (m *Sealing) PledgeSector() error {
	go func() {
		ctx := context.TODO() // we can't use the context from command which invokes
		// this, as we run everything here async, and it's cancelled when the
		// command exits

		size := sectorbuilder.UserBytesForSectorSize(m.sb.SectorSize())

		num, err := m.sb.AcquireSectorNumber()
		if err != nil {
			log.Errorf("%+v", err)
			return
		}

		pieces, err := m.pledgeSector(ctx, num, []abi.UnpaddedPieceSize{}, size)
		if err != nil {
			log.Errorf("%+v", err)
			return
		}

		ppi, err := pieces[0].SB()
		if err != nil {
			log.Errorf("%+v", err)
			return
		}

		if err := m.newSector(context.TODO(), num, pieces[0].DealID, ppi); err != nil {
			log.Errorf("%+v", err)
			return
		}
	}()
	return nil
}
