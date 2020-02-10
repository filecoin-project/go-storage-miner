package storage

import (
	"testing"

	"github.com/filecoin-project/specs-actors/actors/abi"

	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/stretchr/testify/assert"
)

func testFill(t *testing.T, n abi.UnpaddedPieceSize, exp []abi.UnpaddedPieceSize) {
	f, err := fillersFromRem(n)
	assert.NoError(t, err)
	assert.Equal(t, exp, f)

	var sum abi.UnpaddedPieceSize
	for _, u := range f {
		sum += u
	}
	assert.Equal(t, n, sum)
}

func TestFillersFromRem(t *testing.T) {
	for i := 8; i < 32; i++ {
		// single
		ub := sectorbuilder.UserBytesForSectorSize(abi.SectorSize(1) << i)
		testFill(t, ub, []abi.UnpaddedPieceSize{ub})

		// 2
		ub = sectorbuilder.UserBytesForSectorSize(abi.SectorSize(5) << i)
		ub1 := sectorbuilder.UserBytesForSectorSize(abi.SectorSize(1) << i)
		ub3 := sectorbuilder.UserBytesForSectorSize(abi.SectorSize(4) << i)
		testFill(t, ub, []abi.UnpaddedPieceSize{ub1, ub3})

		// 4
		ub = sectorbuilder.UserBytesForSectorSize(abi.SectorSize(15) << i)
		ub2 := sectorbuilder.UserBytesForSectorSize(abi.SectorSize(2) << i)
		ub4 := sectorbuilder.UserBytesForSectorSize(abi.SectorSize(8) << i)
		testFill(t, ub, []abi.UnpaddedPieceSize{ub1, ub2, ub3, ub4})

		// different 2
		ub = sectorbuilder.UserBytesForSectorSize(abi.SectorSize(9) << i)
		testFill(t, ub, []abi.UnpaddedPieceSize{ub1, ub4})
	}
}
