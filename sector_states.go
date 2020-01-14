package storage

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-sectorbuilder"
)

// SectorState enumerates the states which a sector can be in as it progresses
// from newly-provisioned, through packing, to sealed-and-proven.
type SectorState = uint64

const (
	UndefinedSectorState SectorState = iota

	Empty   // TODO: Is this useful
	Packing // sector not in sealStore, and not on chain

	Unsealed      // sealing / queued
	PreCommitting // on chain pre-commit
	PreCommitted  // waiting for seed
	Committing
	CommitWait // waiting for message to land on chain
	Proving

	SealFailed
	PreCommitFailed
	SealCommitFailed
	CommitFailed

	FailedUnrecoverable
)

var SectorStates = []string{
	UndefinedSectorState: "UndefinedSectorState",
	Empty:                "Empty",
	Packing:              "Packing",
	Unsealed:             "Unsealed",
	PreCommitting:        "PreCommitting",
	PreCommitted:         "PreCommitted",
	Committing:           "Committing",
	CommitWait:           "CommitWait",
	Proving:              "Proving",

	SealFailed:       "SealFailed",
	PreCommitFailed:  "PreCommitFailed",
	SealCommitFailed: "SealCommitFailed",
	CommitFailed:     "CommitFailed",

	FailedUnrecoverable: "FailedUnrecoverable",
}

type providerHandlerFunc func(ctx context.Context, deal SectorInfo) *sectorUpdate

func (m *Miner) handleSectorUpdate(ctx context.Context, sector SectorInfo, cb providerHandlerFunc) {
	go func() {
		update := cb(ctx, sector)

		if update == nil {
			return // async
		}

		select {
		case m.sectorUpdated <- *update:
		case <-m.stop:
		}
	}()
}

func (m *Miner) handlePacking(ctx context.Context, sector SectorInfo) *sectorUpdate {
	log.Infow("performing filling up rest of the sector...", "sector", sector.SectorID)

	var allocated uint64
	for _, piece := range sector.Pieces {
		allocated += piece.Size
	}

	ubytes := sectorbuilder.UserBytesForSectorSize(m.sb.SectorSize())

	if allocated > ubytes {
		return sector.upd().fatal(xerrors.Errorf("too much data in sector: %d > %d", allocated, ubytes))
	}

	fillerSizes, err := fillersFromRem(ubytes - allocated)
	if err != nil {
		return sector.upd().fatal(err)
	}

	if len(fillerSizes) > 0 {
		log.Warnf("Creating %d filler pieces for sector %d", len(fillerSizes), sector.SectorID)
	}

	pieces, err := m.pledgeSector(ctx, sector.SectorID, sector.existingPieces(), fillerSizes...)
	if err != nil {
		return sector.upd().fatal(xerrors.Errorf("filling up the sector (%v): %w", fillerSizes, err))
	}

	return sector.upd().to(Unsealed).state(func(info *SectorInfo) {
		info.Pieces = append(info.Pieces, pieces...)
	})
}

func (m *Miner) handleUnsealed(ctx context.Context, sector SectorInfo) *sectorUpdate {
	log.Infow("performing sector replication...", "sector", sector.SectorID)
	ticket, err := m.api.GetSealTicket(ctx)
	if err != nil {
		return sector.upd().fatal(err)
	}

	rspco, err := m.sb.SealPreCommit(ctx, sector.SectorID, ticket.SB(), sector.pieceInfos())
	if err != nil {
		return sector.upd().to(SealFailed).error(xerrors.Errorf("seal pre commit failed: %w", err))
	}

	return sector.upd().to(PreCommitting).state(func(info *SectorInfo) {
		info.CommD = rspco.CommD[:]
		info.CommR = rspco.CommR[:]
		info.Ticket = SealTicket{
			BlockHeight: ticket.BlockHeight,
			TicketBytes: ticket.TicketBytes[:],
		}
	})
}

func (m *Miner) handlePreCommitting(ctx context.Context, sector SectorInfo) *sectorUpdate {
	mcid, err := m.api.SendPreCommitSector(ctx, sector.SectorID, sector.CommR, sector.Ticket, sector.Pieces...)
	if err != nil {
		return sector.upd().to(PreCommitFailed).error(err)
	}

	return sector.upd().to(PreCommitted).state(func(info *SectorInfo) {
		info.PreCommitMessage = &mcid
	})
}

func (m *Miner) handlePreCommitted(ctx context.Context, sector SectorInfo) *sectorUpdate {
	updateNonce := sector.Nonce

	m.api.SetSealSeedHandler(ctx, *sector.PreCommitMessage, func(seed SealSeed) {
		m.sectorUpdated <- *sector.upd().to(Committing).setNonce(updateNonce).state(func(info *SectorInfo) {
			info.Seed = seed
		})

		updateNonce++
	}, func() {
		log.Warn("revert in interactive commit sector step")
	})

	return nil
}

func (m *Miner) handleCommitting(ctx context.Context, sector SectorInfo) *sectorUpdate {
	log.Info("scheduling seal proof computation...")

	proof, err := m.sb.SealCommit(ctx, sector.SectorID, sector.Ticket.SB(), sector.Seed.SB(), sector.pieceInfos(), sector.rspco())
	if err != nil {
		return sector.upd().to(SealCommitFailed).error(xerrors.Errorf("computing seal proof failed: %w", err))
	}

	mcid, err := m.api.SendProveCommitSector(ctx, sector.SectorID, proof, sector.deals()...)
	if err != nil {
		return sector.upd().to(CommitFailed).error(xerrors.Errorf("sending commit message: %w", err))
	}

	return sector.upd().to(CommitWait).state(func(info *SectorInfo) {
		info.CommitMessage = &mcid
		info.Proof = proof
	})
}

func (m *Miner) handleCommitWait(ctx context.Context, sector SectorInfo) *sectorUpdate {
	if sector.CommitMessage == nil {
		log.Errorf("sector %d entered commit wait state without a message cid", sector.SectorID)
		return sector.upd().to(CommitFailed).error(xerrors.Errorf("entered commit wait with no commit cid"))
	}

	_, exitCode, err := m.api.WaitForProveCommitSector(ctx, *sector.CommitMessage)
	if err != nil {
		return sector.upd().to(CommitFailed).error(err)
	}

	if exitCode != 0 {
		log.Errorf("UNHANDLED: submitting sector proof failed (exit=%d, msg=%s) (t:%x; s:%x(%d); p:%x)", exitCode, sector.CommitMessage, sector.Ticket.TicketBytes, sector.Seed.TicketBytes, sector.Seed.BlockHeight, sector.Proof)
		return sector.upd().fatal(xerrors.Errorf("UNHANDLED: submitting sector proof failed (exit: %d)", exitCode))
	}

	return sector.upd().to(Proving).state(func(info *SectorInfo) {
	})
}
