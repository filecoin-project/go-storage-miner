package sealing

import (
	"io"
	"math/bits"
	"math/rand"
	"sync"

	"github.com/ipfs/go-cid"

	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/xerrors"
)

func handle(format string, x ...interface{}) error {
	err := xerrors.Errorf(format, x)
	log.Error(err)

	return err
}

func fillersFromRem(toFill abi.UnpaddedPieceSize) ([]abi.UnpaddedPieceSize, error) {
	// Convert to in-sector bytes for easier math:
	//
	// Sector size to user bytes ratio is constant, e.g. for 1024B we have 1016B
	// of user-usable data.
	//
	// (1024/1016 = 128/127)
	//
	// Given that we can get sector size by simply adding 1/127 of the user
	// bytes
	//
	// (we convert to sector bytes as they are nice round binary numbers)

	n := uint64(toFill)

	n += n / 127

	// We need to fill the sector with pieces that are powers of 2. Conveniently
	// computers store numbers in binary, which means we can look at 1s to get
	// all the piece sizes we need to fill the sector. It also means that number
	// of pieces is the number of 1s in the number of remaining bytes to fill
	out := make([]abi.UnpaddedPieceSize, bits.OnesCount64(n))
	for i := range out {
		// Extract the next lowest non-zero bit
		next := bits.TrailingZeros64(n)
		psize := uint64(1) << next
		// e.g: if the number is 0b010100, psize will be 0b000100

		// set that bit to 0 by XORing it, so the next iteration looks at the
		// next bit
		n ^= psize

		// Add the piece size to the list of pieces we need to create
		out[i] = abi.PaddedPieceSize(psize).Unpadded()
	}

	return out, nil
}

// FastPledgeCommitment generates a file containing parts-quantity pieces, each
// containing size-quantity junk bytes, and returns that file's unsealed CID.
//
// NOTE: This function was copied from lotus, and lotus has since deleted it. We
// will delete it too when we re-extract the lotus storage mining code.
func (m *Sealing) FastPledgeCommitment(size abi.UnpaddedPieceSize, parts uint64) (unsealedCID cid.Cid, err error) {
	n := uint64(size)

	parts = 1 << bits.Len64(parts) // round down to nearest power of 2
	if n/parts < 127 {
		parts = n / 127
	}

	unpaddedPieceSize := abi.PaddedPieceSize(abi.SectorSize((n + n/127) / parts)).Unpadded()
	out := make([]abi.PieceInfo, parts)
	var lk sync.Mutex

	var wg sync.WaitGroup
	wg.Add(int(parts))
	for i := uint64(0); i < parts; i++ {
		go func(i uint64) {
			defer wg.Done()
			pieceCID, perr := sectorbuilder.GeneratePieceCIDFromFile(m.sb.SealProofType(), io.LimitReader(rand.New(rand.NewSource(42+int64(i))), int64(unpaddedPieceSize)), unpaddedPieceSize)

			lk.Lock()
			if perr != nil {
				err = multierror.Append(err, perr)
			}
			out[i] = abi.PieceInfo{
				Size:     unpaddedPieceSize.Padded(),
				PieceCID: pieceCID,
			}
			lk.Unlock()
		}(i)
	}
	wg.Wait()

	if err != nil {
		return cid.Undef, err
	}

	return sectorbuilder.GenerateUnsealedCID(m.sb.SealProofType(), out)
}

func (m *Sealing) ListSectors() ([]SectorInfo, error) {
	var sectors []SectorInfo
	if err := m.sectors.List(&sectors); err != nil {
		return nil, err
	}
	return sectors, nil
}

func (m *Sealing) GetSectorInfo(num abi.SectorNumber) (SectorInfo, error) {
	var out SectorInfo
	err := m.sectors.Get(num).Get(nil)
	return out, err
}
