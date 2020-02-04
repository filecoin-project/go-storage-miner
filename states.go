package storage

import (
	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/filecoin-project/go-sectorbuilder/fs"
	"github.com/filecoin-project/go-statemachine"
	"golang.org/x/xerrors"
)

const InteractivePoRepDelay = 8

func (m *Sealing) handlePacking(ctx statemachine.Context, sector SectorInfo) error {
	log.Infow("performing filling up rest of the sector...", "sector", sector.SectorID)

	var allocated uint64
	for _, piece := range sector.Pieces {
		allocated += piece.Size
	}

	ubytes := sectorbuilder.UserBytesForSectorSize(m.sb.SectorSize())

	if allocated > ubytes {
		return xerrors.Errorf("too much data in sector: %d > %d", allocated, ubytes)
	}

	fillerSizes, err := fillersFromRem(ubytes - allocated)
	if err != nil {
		return err
	}

	if len(fillerSizes) > 0 {
		log.Warnf("Creating %d filler pieces for sector %d", len(fillerSizes), sector.SectorID)
	}

	pieces, err := m.pledgeSector(ctx.Context(), sector.SectorID, sector.existingPieces(), fillerSizes...)
	if err != nil {
		return xerrors.Errorf("filling up the sector (%v): %w", fillerSizes, err)
	}

	return ctx.Send(SectorPacked{pieces: pieces})
}

func (m *Sealing) handleUnsealed(ctx statemachine.Context, sector SectorInfo) error {
	if err := m.api.CheckPieces(ctx.Context(), sector.SectorID, sector.Pieces); err != nil { // Sanity check state
		switch err.EType {
		case CheckPiecesAPI:
			log.Errorf("handleUnsealed: api error, not proceeding: %+v", err)
			return nil
		case CheckPiecesInvalidDeals:
			return ctx.Send(SectorPackingFailed{xerrors.Errorf("invalid deals in sector: %w", err)})
		case CheckPiecesExpiredDeals: // Probably not much we can do here, maybe re-pack the sector?
			return ctx.Send(SectorPackingFailed{xerrors.Errorf("expired deals in sector: %w", err)})
		default:
			return xerrors.Errorf("checkPieces sanity check error: %w", err)
		}
	}

	log.Infow("performing sector replication...", "sector", sector.SectorID)
	ticket, err := m.api.GetSealTicket(ctx.Context())
	if err != nil {
		return ctx.Send(SectorSealFailed{xerrors.Errorf("getting ticket failed: %w", err)})
	}

	rspco, err := m.sb.SealPreCommit(ctx.Context(), sector.SectorID, ticket.SB(), sector.pieceInfos())
	if err != nil {
		return ctx.Send(SectorSealFailed{xerrors.Errorf("seal pre commit failed: %w", err)})
	}

	return ctx.Send(SectorSealed{
		commD: rspco.CommD[:],
		commR: rspco.CommR[:],
		ticket: SealTicket{
			BlockHeight: ticket.BlockHeight,
			TicketBytes: ticket.TicketBytes[:],
		},
	})
}

func (m *Sealing) handlePreCommitting(ctx statemachine.Context, sector SectorInfo) error {
	if err := m.api.CheckSealing(ctx.Context(), sector.CommD, sector.deals(), sector.Ticket); err != nil {
		switch err.EType {
		case CheckSealingAPI:
			log.Errorf("handlePreCommitting: api error, not proceeding: %+v", err)
			return nil
		case CheckSealingBadCommD: // TODO: Should this just back to packing? (not really needed since handleUnsealed will do that too)
			return ctx.Send(SectorSealFailed{xerrors.Errorf("bad CommD error: %w", err)})
		case CheckSealingExpiredTicket:
			return ctx.Send(SectorSealFailed{xerrors.Errorf("ticket expired error: %w", err)})
		default:
			return xerrors.Errorf("checkSeal sanity check error: %w", err)
		}
	}

	smsg, err := m.api.SendPreCommitSector(ctx.Context(), sector.SectorID, sector.CommR, sector.Ticket, sector.Pieces...)
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
			case GetSealSeedFailedError:
				return ctx.Send(SectorPreCommitFailed{err.inner})
			case GetSealSeedFatalError:
				return ctx.Send(SectorFatalError{err.inner})
			default:
				log.Error("unhandled error from GetSealSeed: %+v", err)
				return ctx.Send(SectorFatalError{err.inner})
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

	proof, err := m.sb.SealCommit(ctx.Context(), sector.SectorID, sector.Ticket.SB(), sector.Seed.SB(), sector.pieceInfos(), sector.rspco())
	if err != nil {
		return ctx.Send(SectorComputeProofFailed{xerrors.Errorf("computing seal proof failed: %w", err)})
	}

	smsg, err := m.api.SendProveCommitSector(ctx.Context(), sector.SectorID, proof, sector.deals()...)
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
		log.Errorf("sector %d entered commit wait state without a message cid", sector.SectorID)
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

	if err := m.sb.FinalizeSector(ctx.Context(), sector.SectorID); err != nil {
		if !xerrors.Is(err, fs.ErrNoSuitablePath) {
			return ctx.Send(SectorFinalizeFailed{xerrors.Errorf("finalize sector: %w", err)})
		}
		log.Warnf("finalize sector: %v", err)
	}

	if err := m.sb.DropStaged(ctx.Context(), sector.SectorID); err != nil {
		return ctx.Send(SectorFinalizeFailed{xerrors.Errorf("drop staged: %w", err)})
	}

	return ctx.Send(SectorFinalized{})
}

func (m *Sealing) handleFaulty(ctx statemachine.Context, sector SectorInfo) error {
	//// TODO: check if the fault has already been reported, and that this sector is even valid
	//
	//// TODO: coalesce faulty sector reporting
	smsg, err := m.api.SendReportFaults(ctx.Context(), sector.SectorID)
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
		log.Errorf("UNHANDLED: declaring sector fault failed (exit=%d, msg=%s) (id: %d)", exitCode, *sector.FaultReportMsg, sector.SectorID)
		return xerrors.Errorf("UNHANDLED: submitting fault declaration failed (exit %d)", exitCode)
	}

	return ctx.Send(SectorFaultedFinal{})
}
