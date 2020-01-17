package storage

import (
	"context"
	"io"
)

type Interface interface {
	// AllocateSectorID allocates a new sector ID.
	AllocateSectorID() (sectorID uint64, err error)

	// PledgeSector allocates a new sector, fills it with self-deal junk, and
	// seals that sector.
	PledgeSector() error

	// SealPiece writes the provided piece to a newly-created sector which it
	// immediately seals.
	SealPiece(ctx context.Context, size uint64, r io.Reader, sectorID uint64, dealID uint64) error

	// GetSectorInfo produces information about a sector managed by this storage
	// miner, or an error if the miner does not manage a sector with the
	// provided identity.
	GetSectorInfo(sectorID uint64) (SectorInfo, error)

	// ListSectors lists all the sectors managed by this storage miner (sealed
	// or otherwise).
	ListSectors() ([]SectorInfo, error)
}
