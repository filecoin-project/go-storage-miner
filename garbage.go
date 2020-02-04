package storage

import (
	"context"
	"io"
	"math/bits"
	"math/rand"
	"runtime"

	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"golang.org/x/xerrors"
)

func (m *Sealing) pledgeReader(size uint64, parts uint64) io.Reader {
	parts = 1 << bits.Len64(parts) // round down to nearest power of 2
	if size/parts < 127 {
		parts = size / 127
	}

	piece := sectorbuilder.UserBytesForSectorSize((size/127 + size) / parts)

	readers := make([]io.Reader, parts)
	for i := range readers {
		readers[i] = io.LimitReader(rand.New(rand.NewSource(42+int64(i))), int64(piece))
	}

	return io.MultiReader(readers...)
}

func (m *Sealing) pledgeSector(ctx context.Context, sectorID uint64, existingPieceSizes []uint64, sizes ...uint64) ([]Piece, error) {
	if len(sizes) == 0 {
		return nil, nil
	}

	log.Infof("Pledge %d, contains %+v", sectorID, existingPieceSizes)

	pieces := make([]PieceInfo, len(sizes))
	for i, size := range sizes {
		commP, err := m.FastPledgeCommitment(size, uint64(runtime.NumCPU()))
		if err != nil {
			return nil, err
		}

		pieces[i] = PieceInfo{
			Size:  size,
			CommP: commP,
		}
	}

	log.Infof("Publishing deals for %d", sectorID)

	mcid, err := m.api.SendSelfDeals(ctx, pieces...)
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

	out := make([]Piece, len(sizes))

	for i, size := range sizes {
		ppi, err := m.sb.AddPiece(ctx, size, sectorID, m.pledgeReader(size, uint64(runtime.NumCPU())), existingPieceSizes)
		if err != nil {
			return nil, xerrors.Errorf("add piece: %w", err)
		}

		existingPieceSizes = append(existingPieceSizes, size)

		out[i] = Piece{
			DealID: dealIDs[i],
			Size:   ppi.Size,
			CommP:  ppi.CommP[:],
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

		sid, err := m.sb.AcquireSectorId()
		if err != nil {
			log.Errorf("%+v", err)
			return
		}

		pieces, err := m.pledgeSector(ctx, sid, []uint64{}, size)
		if err != nil {
			log.Errorf("%+v", err)
			return
		}

		if err := m.newSector(context.TODO(), sid, pieces[0].DealID, pieces[0].ppi()); err != nil {
			log.Errorf("%+v", err)
			return
		}
	}()
	return nil
}
