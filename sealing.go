package storage

import (
	"context"

	sealing2 "github.com/filecoin-project/go-storage-miner/sealing"
	"github.com/filecoin-project/specs-actors/actors/abi"
)

// AllocatePiece produces information about where a piece of a given size can
// be written.
//
// TODO: This signature doesn't make much sense. Returning a sector ID here
// means that we won't have the ability to move the piece around (i.e. do
// intelligent bin packing) after allocating. -- @laser
func (m *Miner) AllocatePiece(size abi.UnpaddedPieceSize) (sectorNum abi.SectorNumber, offset uint64, err error) {
	return m.sealing.AllocatePiece(size)
}

// ListSectors lists all the sectors managed by this storage miner (sealed
// or otherwise).
func (m *Miner) ListSectors() ([]sealing2.SectorInfo, error) {
	return m.sealing.ListSectors()
}

// GetSectorInfo produces information about a sector managed by this storage
// miner, or an error if the miner does not manage a sector with the
// provided identity.
func (m *Miner) GetSectorInfo(sectorNum abi.SectorNumber) (sealing2.SectorInfo, error) {
	return m.sealing.GetSectorInfo(sectorNum)
}

// PledgeSector allocates a new sector, fills it with self-deal junk, and
// seals that sector.
func (m *Miner) PledgeSector() error {
	return m.sealing.PledgeSector()
}

// ForceSectorState puts a sector with given ID into the given state.
func (m *Miner) ForceSectorState(ctx context.Context, num abi.SectorNumber, state sealing2.SectorState) error {
	return m.sealing.ForceSectorState(ctx, num, state)
}

// AllocateSectorID allocates a new sector ID.
func (m *Miner) AllocateSectorID() (sectorNum abi.SectorNumber, err error) {
	return m.sb.AcquireSectorNumber()
}
