package sealing

import (
	"bytes"
	"time"

	"github.com/filecoin-project/go-storage-miner/apis/node"

	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/go-statemachine"
	"golang.org/x/xerrors"
)

const minRetryTime = 1 * time.Minute

func failedCooldown(ctx statemachine.Context, sector SectorInfo) error {
	retryStart := time.Unix(int64(sector.Log[len(sector.Log)-1].Timestamp), 0).Add(minRetryTime)
	if len(sector.Log) > 0 && !time.Now().After(retryStart) {
		log.Infof("%s(%d), waiting %s before retrying", SectorStates[sector.State], sector.SectorNum, time.Until(retryStart))
		select {
		case <-time.After(time.Until(retryStart)):
		case <-ctx.Context().Done():
			return ctx.Context().Err()
		}
	}

	return nil
}

func (m *Sealing) checkPreCommitted(ctx statemachine.Context, sector SectorInfo) ([]byte, bool, error) {
	tok, _, err := m.api.GetChainHead(ctx.Context())
	if err != nil {
		return nil, false, xerrors.Errorf("failed to get chain head: %w", err)
	}

	sealedCID, found, err := m.api.GetSealedCID(ctx.Context(), tok, sector.SectorNum)
	if err != nil {
		log.Errorf("checkPreCommitted(%d): temp error: %+v", sector.SectorNum, err)
		return nil, false, err
	}

	commR, err := commcid.CIDToReplicaCommitmentV1(sealedCID)
	if err != nil {
		log.Errorf("failed to map from sealed CID to CommR: %+v", sector.SectorNum, err)
		return nil, false, err
	}

	if found {
		// TODO: If not expired yet, we can just try reusing sealticket
		log.Warnf("sector %d found in miner preseal array", sector.SectorNum)
		return commR, true, nil
	}

	return nil, false, nil
}

func (m *Sealing) handleSealFailed(ctx statemachine.Context, sector SectorInfo) error {
	if _, is, _ := m.checkPreCommitted(ctx, sector); is {
		// TODO: Remove this after we can re-precommit
		return nil // noop, for now
	}

	if err := failedCooldown(ctx, sector); err != nil {
		return err
	}

	return ctx.Send(SectorRetrySeal{})
}

func (m *Sealing) handlePreCommitFailed(ctx statemachine.Context, sector SectorInfo) error {
	if err := m.api.CheckSealing(ctx.Context(), sector.CommD, sector.deals(), sector.Ticket); err != nil {
		switch err.EType {
		case node.CheckSealingAPI:
			log.Errorf("handlePreCommitFailed: api error, not proceeding: %+v", err)
			return nil
		case node.CheckSealingBadCommD: // TODO: Should this just back to packing? (not really needed since handleUnsealed will do that too)
			return ctx.Send(SectorSealFailed{xerrors.Errorf("bad CommD error: %w", err)})
		case node.CheckSealingExpiredTicket:
			return ctx.Send(SectorSealFailed{xerrors.Errorf("ticket expired error: %w", err)})
		default:
			return xerrors.Errorf("checkSeal sanity check error: %w", err)
		}
	}

	var null [32]byte
	if commR, is, _ := m.checkPreCommitted(ctx, sector); is && !bytes.Equal(commR[:], null[:]) {
		if sector.PreCommitMessage != nil {
			log.Warn("sector %d is precommitted on chain, but we don't have precommit message", sector.SectorNum)
			return nil // TODO: SeedWait needs this currently
		}

		if string(commR[:]) != string(sector.CommR) {
			log.Warn("sector %d is precommitted on chain, with different CommR: %x != %x", sector.SectorNum, commR, sector.CommR)
			return nil // TODO: remove when the actor allows re-precommit
		}

		// TODO: we could compare more things, but I don't think we really need to
		//  CommR tells us that CommD (and CommPs), and the ticket are all matching

		if err := failedCooldown(ctx, sector); err != nil {
			return err
		}

		return ctx.Send(SectorRetryWaitSeed{})
	}

	if sector.PreCommitMessage != nil {
		log.Warn("retrying precommit even though the message failed to apply")
	}

	if err := failedCooldown(ctx, sector); err != nil {
		return err
	}

	return ctx.Send(SectorRetryPreCommit{})
}
