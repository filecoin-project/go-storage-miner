package storage

import (
	"context"
	"io"
	"math/rand"

	"github.com/filecoin-project/go-sectorbuilder"
	"golang.org/x/xerrors"
)

func (m *Sealing) pledgeSector(ctx context.Context, sectorID uint64, existingPieceSizes []uint64, sizes ...uint64) ([]Piece, error) {
	if len(sizes) == 0 {
		return nil, nil
	}

	pieces := make([]PieceInfo, len(sizes))
	for i, size := range sizes {
		release := m.sb.RateLimit()
		commP, err := sectorbuilder.GeneratePieceCommitment(io.LimitReader(rand.New(rand.NewSource(42)), int64(size)), size)
		release()

		if err != nil {
			return nil, err
		}

		pieces[i] = PieceInfo{
			Size:  size,
			CommP: commP,
		}
	}

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
		ppi, err := m.sb.AddPiece(size, sectorID, io.LimitReader(rand.New(rand.NewSource(42)), int64(size)), existingPieceSizes)
		if err != nil {
			return nil, err
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
