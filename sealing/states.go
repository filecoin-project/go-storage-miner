package sealing

import (
	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/go-sectorbuilder/fs"
	"github.com/filecoin-project/go-statemachine"
	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"golang.org/x/xerrors"
)

const InteractivePoRepDelay = 8

func (m *Sealing) handlePacking(ctx statemachine.Context, sector SectorInfo) error {
	log.Infow("performing filling up rest of the sector...", "sector", sector.SectorNum)

	var unpaddedAllocated abi.UnpaddedPieceSize
	for _, paddedQty := range sector.existingPieces() {
		unpaddedAllocated += sectorbuilder.UserBytesForSectorSize(abi.SectorSize(paddedQty))
	}

	unpaddedTotal := sectorbuilder.UserBytesForSectorSize(m.sb.SectorSize())

	if unpaddedAllocated > unpaddedTotal {
		return xerrors.Errorf("too much data in sector: %d > %d", unpaddedAllocated, unpaddedTotal)
	}

	fillerSizes, err := fillersFromRem(unpaddedTotal - unpaddedAllocated)
	if err != nil {
		return err
	}

	if len(fillerSizes) > 0 {
		log.Warnf("Creating %d filler pieces for sector %d", len(fillerSizes), sector.SectorNum)
	}

	fillerPieceSizes := make([]abi.UnpaddedPieceSize, len(fillerSizes))
	for idx := range fillerSizes {
		fillerPieceSizes[idx] = fillerSizes[idx]
	}

	existingPadded := sector.existingPieces()
	existingUnpadded := make([]abi.UnpaddedPieceSize, len(existingPadded))
	for idx := range existingPadded {
		existingUnpadded[idx] = existingPadded[idx].Unpadded()
	}

	pieces, err := m.pledgeSector(ctx.Context(), sector.SectorNum, existingUnpadded, fillerPieceSizes...)
	if err != nil {
		return xerrors.Errorf("filling up the sector (%v): %w", fillerSizes, err)
	}

	return ctx.Send(SectorPacked{pieces: pieces})
}

func (m *Sealing) handleUnsealed(ctx statemachine.Context, sector SectorInfo) error {
	if err := m.api.CheckPieces(ctx.Context(), sector.SectorNum, sector.Pieces); err != nil { // Sanity check state
		switch err.EType {
		case node.CheckPiecesAPI:
			log.Errorf("handleUnsealed: api error, not proceeding: %+v", err)
			return nil
		case node.CheckPiecesInvalidDeals:
			return ctx.Send(SectorPackingFailed{xerrors.Errorf("invalid deals in sector: %w", err)})
		case node.CheckPiecesExpiredDeals: // Probably not much we can do here, maybe re-pack the sector?
			return ctx.Send(SectorPackingFailed{xerrors.Errorf("expired deals in sector: %w", err)})
		default:
			return xerrors.Errorf("checkPieces sanity check error: %w", err)
		}
	}

	tok, err := m.api.GetChainHead(ctx.Context())
	if err != nil {
		return xerrors.Errorf("failed to get chain head: %w", err)
	}

	log.Infow("performing sector replication...", "sector", sector.SectorNum)
	ticket, err := m.api.GetSealTicket(ctx.Context(), tok)
	if err != nil {
		return ctx.Send(SectorSealFailed{xerrors.Errorf("getting ticket failed: %w", err)})
	}

	pis, err := sector.pieceInfos()
	if err != nil {
		return ctx.Send(SectorSealFailed{xerrors.Errorf("getting piece infos failed: %w", err)})
	}

	rspco, err := m.sb.SealPreCommit(ctx.Context(), sector.SectorNum, ticket.SB(), pis)
	if err != nil {
		return ctx.Send(SectorSealFailed{xerrors.Errorf("seal pre commit failed: %w", err)})
	}

	return ctx.Send(SectorSealed{
		commD: rspco.CommD[:],
		commR: rspco.CommR[:],
		ticket: node.SealTicket{
			BlockHeight: ticket.BlockHeight,
			TicketBytes: ticket.TicketBytes[:],
		},
	})
}

func (m *Sealing) handlePreCommitting(ctx statemachine.Context, sector SectorInfo) error {
	if err := m.api.CheckSealing(ctx.Context(), sector.CommD, sector.deals(), sector.Ticket); err != nil {
		switch err.EType {
		case node.CheckSealingAPI:
			log.Errorf("handlePreCommitting: api error, not proceeding: %+v", err)
			return nil
		case node.CheckSealingBadCommD: // TODO: Should this just back to packing? (not really needed since handleUnsealed will do that too)
			return ctx.Send(SectorSealFailed{xerrors.Errorf("bad CommD error: %w", err)})
		case node.CheckSealingExpiredTicket:
			return ctx.Send(SectorSealFailed{xerrors.Errorf("ticket expired error: %w", err)})
		default:
			return xerrors.Errorf("checkSeal sanity check error: %w", err)
		}
	}

	smsg, err := m.api.SendPreCommitSector(ctx.Context(), sector.SectorNum, sector.CommR, sector.Ticket, sector.Pieces...)
	if err != nil {
		return ctx.Send(SectorPreCommitFailed{xerrors.Errorf("failed to send pre-commit message: %w", err)})
	}

	return ctx.Send(SectorPreCommitted{message: smsg})
}

func (m *Sealing) handleWaitSeed(ctx statemachine.Context, sector SectorInfo) error {
	seedChan, invalidated, done, errChan := m.api.GetSealSeed(ctx.Context(), *sector.PreCommitMessage, InteractivePoRepDelay)

	for {
		select {
		case seed := <-seedChan:
			return ctx.Send(SectorSeedReady{seed: seed})
		case err := <-errChan:
			log.Error("error waiting for precommit", err)

			switch err.EType {
			case node.GetSealSeedFailedError:
				return ctx.Send(SectorPreCommitFailed{err.Unwrap()})
			case node.GetSealSeedFatalError:
				return ctx.Send(SectorFatalError{err.Unwrap()})
			default:
				log.Error("unhandled error from GetSealSeed: %+v", err)
				return ctx.Send(SectorFatalError{err.Unwrap()})
			}
		case <-invalidated:
			log.Warn("revert in interactive commit sector step")

			return nil
		case <-done:
			return nil
		case <-ctx.Context().Done():
			return nil
		}
	}
}

func (m *Sealing) handleCommitting(ctx statemachine.Context, sector SectorInfo) error {
	log.Info("scheduling seal proof computation...")

	pis, err := sector.pieceInfos()
	if err != nil {
		return ctx.Send(SectorComputeProofFailed{xerrors.Errorf("failed to get piece infos for computing seal proof: %w", err)})
	}

	proof, err := m.sb.SealCommit(ctx.Context(), sector.SectorNum, sector.Ticket.SB(), sector.Seed.SB(), pis, sector.rspco())
	if err != nil {
		return ctx.Send(SectorComputeProofFailed{xerrors.Errorf("computing seal proof failed: %w", err)})
	}

	smsg, err := m.api.SendProveCommitSector(ctx.Context(), sector.SectorNum, proof, sector.deals()...)
	if err != nil {
		return ctx.Send(SectorCommitFailed{xerrors.Errorf("error sending prove commit sector: %w", err)})
	}

	return ctx.Send(SectorCommitted{
		proof:   proof,
		message: smsg,
	})
}

func (m *Sealing) handleCommitWait(ctx statemachine.Context, sector SectorInfo) error {
	if sector.CommitMessage == nil {
		log.Errorf("sector %d entered commit wait state without a message cid", sector.SectorNum)
		return ctx.Send(SectorCommitFailed{xerrors.Errorf("entered commit wait with no commit cid")})
	}

	exitCode, err := m.api.WaitForProveCommitSector(ctx.Context(), *sector.CommitMessage)
	if err != nil {
		return ctx.Send(SectorCommitFailed{xerrors.Errorf("failed to wait for porep inclusion: %w", err)})
	}

	if exitCode != 0 {
		return ctx.Send(SectorCommitFailed{xerrors.Errorf("submitting sector proof failed (exit=%d, msg=%s) (t:%x; s:%x(%d); p:%x)", exitCode, sector.CommitMessage, sector.Ticket.TicketBytes, sector.Seed.TicketBytes, sector.Seed.BlockHeight, sector.Proof)})
	}

	return ctx.Send(SectorProving{})
}

func (m *Sealing) handleFinalizeSector(ctx statemachine.Context, sector SectorInfo) error {
	// TODO: Maybe wait for some finality

	if err := m.sb.FinalizeSector(ctx.Context(), sector.SectorNum); err != nil {
		if !xerrors.Is(err, fs.ErrNoSuitablePath) {
			return ctx.Send(SectorFinalizeFailed{xerrors.Errorf("finalize sector: %w", err)})
		}
		log.Warnf("finalize sector: %v", err)
	}

	if err := m.sb.DropStaged(ctx.Context(), sector.SectorNum); err != nil {
		return ctx.Send(SectorFinalizeFailed{xerrors.Errorf("drop staged: %w", err)})
	}

	return ctx.Send(SectorFinalized{})
}

func (m *Sealing) handleFaulty(ctx statemachine.Context, sector SectorInfo) error {
	//// TODO: check if the fault has already been reported, and that this sector is even valid
	//
	//// TODO: coalesce faulty sector reporting
	smsg, err := m.api.SendReportFaults(ctx.Context(), sector.SectorNum)
	if err != nil {
		return xerrors.Errorf("failed to push declare faults message to network: %w", err)
	}

	return ctx.Send(SectorFaultReported{reportMsg: smsg})
}

func (m *Sealing) handleFaultReported(ctx statemachine.Context, sector SectorInfo) error {
	if sector.FaultReportMsg == nil {
		return xerrors.Errorf("entered fault reported state without a FaultReportMsg cid")
	}

	exitCode, err := m.api.WaitForReportFaults(ctx.Context(), *sector.FaultReportMsg)
	if err != nil {
		return xerrors.Errorf("failed to wait for fault declaration: %w", err)
	}

	if exitCode != 0 {
		log.Errorf("UNHANDLED: declaring sector fault failed (exit=%d, msg=%s) (num: %d)", exitCode, *sector.FaultReportMsg, sector.SectorNum)
		return xerrors.Errorf("UNHANDLED: submitting fault declaration failed (exit %d)", exitCode)
	}

	return ctx.Send(SectorFaultedFinal{})
}
