package storage

import (
	"bytes"
	"context"
	"io"
	"math/bits"
	"math/rand"
	"runtime"

	commcid "github.com/filecoin-project/go-fil-commcid"
	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/specs-actors/actors/abi"
	builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	market "github.com/filecoin-project/specs-actors/actors/builtin/market"
	"golang.org/x/xerrors"
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

func (m *Sealing) pledgeSector(ctx context.Context, sectorNum abi.SectorNumber, existingPieceSizes []abi.UnpaddedPieceSize, sizes ...abi.UnpaddedPieceSize) ([]Piece, error) {
	if len(sizes) == 0 {
		return nil, nil
	}

	log.Infof("Pledge %d, contains %+v", sectorNum, existingPieceSizes)

	arg := &market.PublishStorageDealsParams{
		Deals: make([]market.ClientDealProposal, len(sizes)),
	}

	for i, size := range sizes {
		commP, err := m.FastPledgeCommitment(size, uint64(1))
		if err != nil {
			return nil, handle("failed to generate pledge commitment: ", err)
		}

		sdp := market.ClientDealProposal{
			Proposal: market.DealProposal{
				Client:               m.worker,
				ClientCollateral:     abi.NewTokenAmount(0),
				EndEpoch:             abi.ChainEpoch(0),
				PieceCID:             commcid.PieceCommitmentV1ToCID(commP[:]),
				PieceSize:            size.Padded(),
				Provider:             m.maddr,
				ProviderCollateral:   abi.NewTokenAmount(0),
				StartEpoch:           abi.ChainEpoch(0),
				StoragePricePerEpoch: abi.NewTokenAmount(0),
			},
		}

		arg.Deals[i] = sdp
	}

	log.Infof("Publishing deals for %d", sectorNum)

	argBuf := new(bytes.Buffer)
	if err := arg.MarshalCBOR(argBuf); err != nil {
		return nil, handle("marshaling PublishStorageDealsParams failed: ", err)
	}

	mcid, err := m.api.SendMessage(m.worker, builtin.StorageMarketActorAddr, builtin.MethodsMarket.PublishStorageDeals, abi.NewTokenAmount(0), argBuf.Bytes())
	if err != nil {
		return nil, handle("failed to send message to storage market actor: ", err)
	}

	wmsg, err := m.api.WaitMessage(ctx, mcid)
	if err != nil {
		return nil, handle("failed to wait for message: ", err)
	}

	if wmsg.Receipt.ExitCode != 0 {
		return nil, handle("publishing deal failed: exit %d", wmsg.Receipt.ExitCode)
	}

	ret := new(market.PublishStorageDealsReturn)
	if err = ret.UnmarshalCBOR(bytes.NewReader(wmsg.Receipt.Return)); err != nil {
		return nil, handle("unmarshaling PublishStorageDealsReturn failed: ", err)
	}

	if len(ret.IDs) != len(sizes) {
		return nil, handle("got unexpected number of DealIDs from PublishStorageDeals")
	}

	out := make([]Piece, len(sizes))

	for i, size := range sizes {
		ppi, err := m.sb.AddPiece(ctx, size, sectorNum, m.pledgeReader(size, uint64(runtime.NumCPU())), existingPieceSizes)
		if err != nil {
			return nil, xerrors.Errorf("add piece: %w", err)
		}

		existingPieceSizes = append(existingPieceSizes, size)

		out[i] = Piece{
			DealID:   ret.IDs[i],
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

		ppi, err := pieces[0].ppi()
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

func handle(format string, x ...interface{}) error {
	err := xerrors.Errorf(format, x)
	log.Error(err)

	return err
}
