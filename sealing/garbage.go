package sealing

import (
	"context"
	"io"
	"math/bits"
	"math/rand"
	"runtime"

	"github.com/filecoin-project/specs-actors/actors/abi"

	"github.com/filecoin-project/go-storage-miner/apis/node"
)

func (m *Sealing) pledgeReader(size abi.UnpaddedPieceSize, parts uint64) io.Reader {
	n := uint64(size)

	parts = 1 << bits.Len64(parts) // round down to nearest power of 2
	if n/parts < 127 {
		parts = n / 127
	}

	piece := abi.PaddedPieceSize(abi.SectorSize((n/127 + n) / parts)).Unpadded()

	readers := make([]io.Reader, parts)
	for i := range readers {
		readers[i] = io.LimitReader(rand.New(rand.NewSource(42+int64(i))), int64(piece))
	}

	return io.MultiReader(readers...)
}

// pledgeSector writes junk (filler) pieces to the provided sector and creates
// self-deals for those junk pieces. Junk pieces are written to the target
// sector respecting alignment of the sector's existing pieces. After the sector
// has been completely filled (with junk and/or client data), a slice of all
// piece and deal metadata associated with that sector is returned.
func (m *Sealing) pledgeSector(ctx context.Context, sectorNum abi.SectorNumber, existing []node.PieceWithDealInfo, fillerPieceSizes ...abi.UnpaddedPieceSize) ([]node.PieceWithDealInfo, error) {
	if len(fillerPieceSizes) == 0 {
		return existing, nil
	}

	log.Infof("Pledge %d, contains %+v", sectorNum, existing)

	fillers := make([]node.PieceWithOptionalDealInfo, len(fillerPieceSizes))
	for idx := range fillerPieceSizes {
		// NOTE: This software was copied from lotus, and lotus assumed that
		// the returned data commitment could be used in place of a piece
		// commitment. Today, the underlying CID codec is identical, so that
		// assumption holds.
		unsealedCID, err := m.FastPledgeCommitment(fillerPieceSizes[idx], uint64(1))
		if err != nil {
			return nil, handle("failed to generate pledge commitment: ", err)
		}

		fillers[idx] = node.PieceWithOptionalDealInfo{
			Piece: abi.PieceInfo{
				Size:     fillerPieceSizes[idx].Padded(),
				PieceCID: unsealedCID,
			},
			DealInfo: nil,
		}
	}

	// the self-deal scheduling algorithm needs to know about all of the pieces
	// to be written into the sector - both self-deal junk pieces and pieces
	// received from consummating storage deals
	existingPrime := make([]node.PieceWithOptionalDealInfo, len(existing))
	for idx := range existingPrime {
		existingPrime[idx] = node.PieceWithOptionalDealInfo{
			Piece:    existing[idx].Piece,
			DealInfo: &existing[idx].DealInfo,
		}
	}

	schedule, err := m.selfDealPolicy.Schedule(ctx, append(fillers, existingPrime...)...)
	if err != nil {
		return nil, handle("failed to create self-deal schedule", err)
	}

	// we send self-deals only for the filler pieces, as the other pieces are
	// already associated with on-chain deals
	fillersPrime := make([]abi.PieceInfo, len(fillers))
	for idx := range fillersPrime {
		fillersPrime[idx] = abi.PieceInfo{
			Size:     fillers[idx].Piece.Size,
			PieceCID: fillers[idx].Piece.PieceCID,
		}
	}

	mcid, err := m.api.SendSelfDeals(ctx, schedule.StartEpoch, schedule.EndEpoch, fillersPrime...)
	if err != nil {
		return nil, handle("failed to send self-deals to node", err)
	}

	dealIDs, exitCode, err := m.api.WaitForSelfDeals(ctx, mcid)
	if err != nil {
		return nil, handle("node produced an error waiting for self deals", err)
	}

	if exitCode != 0 {
		return nil, handle("publishing deal failed: exit %d", exitCode)
	}

	if len(dealIDs) != len(fillerPieceSizes) {
		return nil, handle("got unexpected number of deal IDs from PublishStorageDeals (len(dealIDs)=%d != len(fillerPieceSizes)=%d)", len(dealIDs), len(fillerPieceSizes))
	}

	// the sizes of the pieces already written to the sector into which the
	// junk pieces will be written
	existingSizes := make([]abi.UnpaddedPieceSize, len(existing))
	for idx := range existingSizes {
		existingSizes[idx] = existing[idx].Piece.Size.Unpadded()
	}

	out := make([]node.PieceWithDealInfo, len(existing))
	for idx := range existing {
		out[idx] = existing[idx]
	}

	for idx := range fillerPieceSizes {
		ppi, err := m.sb.AddPiece(ctx, fillerPieceSizes[idx], sectorNum, m.pledgeReader(fillerPieceSizes[idx], uint64(runtime.NumCPU())), existingSizes)
		if err != nil {
			return nil, handle("add piece: %w", err)
		}

		existingSizes = append(existingSizes, fillerPieceSizes[idx])

		// extend the slice with the newly-self-dealt piece
		out = append(out, node.PieceWithDealInfo{
			Piece: ppi,
			DealInfo: node.DealInfo{
				DealID:       dealIDs[idx],
				DealSchedule: schedule,
			},
		})
	}

	return out, nil
}

func (m *Sealing) PledgeSector(ctx context.Context) error {
	size := abi.PaddedPieceSize(m.sb.SectorSize()).Unpadded()

	num, err := m.sb.AcquireSectorNumber()
	if err != nil {
		return handle("failed to acquire sector number: %w", err)
	}

	pieces, err := m.pledgeSector(ctx, num, []node.PieceWithDealInfo{}, size)
	if err != nil {
		return handle("pledge sector failed: %w", err)
	}

	if err := m.newSector(ctx, num, pieces...); err != nil {
		return handle("failed to create new sector: %w", err)
	}

	return nil
}
