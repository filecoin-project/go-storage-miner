package sealing

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/filecoin-project/go-statemachine"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"golang.org/x/xerrors"
)

func (m *Sealing) Plan(events []statemachine.Event, user interface{}) (interface{}, uint64, error) {
	next, err := m.plan(events, user.(*SectorInfo))
	if err != nil || next == nil {
		return nil, 0, err
	}

	return func(ctx statemachine.Context, si SectorInfo) error {
		err := next(ctx, si)
		if err != nil {
			log.Errorf("unhandled sector error (%d): %+v", si.SectorNum, err)
			return nil
		}

		return nil
	}, uint64(len(events)), nil
}

var fsmPlanners = []func(events []statemachine.Event, state *SectorInfo) error{
	UndefinedSectorState: planOne(on(SectorStart{}, Packing)),
	Packing:              planOne(on(SectorPacked{}, Unsealed)),
	Unsealed: planOne(
		on(SectorSealed{}, PreCommitting),
		on(SectorSealFailed{}, SealFailed),
		on(SectorPackingFailed{}, PackingFailed),
	),
	PreCommitting: planOne(
		on(SectorSealFailed{}, SealFailed),
		on(SectorPreCommitted{}, WaitSeed),
		on(SectorPreCommitFailed{}, PreCommitFailed),
	),
	WaitSeed: planOne(
		on(SectorSeedReady{}, Committing),
		on(SectorPreCommitFailed{}, PreCommitFailed),
	),
	Committing: planCommitting,
	CommitWait: planOne(
		on(SectorProving{}, FinalizeSector),
		on(SectorCommitFailed{}, CommitFailed),
	),

	FinalizeSector: planOne(
		on(SectorFinalized{}, Proving),
	),

	Proving: planOne(
		on(SectorFaultReported{}, FaultReported),
		on(SectorFaulty{}, Faulty),
	),

	SealFailed: planOne(
		on(SectorRetrySeal{}, Unsealed),
	),
	PreCommitFailed: planOne(
		on(SectorRetryPreCommit{}, PreCommitting),
		on(SectorRetryWaitSeed{}, WaitSeed),
		on(SectorSealFailed{}, SealFailed),
	),

	Faulty: planOne(
		on(SectorFaultReported{}, FaultReported),
	),
	FaultedFinal: final,
}

func (m *Sealing) plan(events []statemachine.Event, state *SectorInfo) (func(statemachine.Context, SectorInfo) error, error) {
	/////
	// First process all events

	for _, event := range events {
		l := Log{
			Timestamp: uint64(time.Now().Unix()),
			Message:   fmt.Sprintf("%+v", event),
			Kind:      fmt.Sprintf("event;%T", event.User),
		}

		if err, iserr := event.User.(xerrors.Formatter); iserr {
			l.Trace = fmt.Sprintf("%+v", err)
		}

		state.Log = append(state.Log, l)
	}

	p := fsmPlanners[state.State]
	if p == nil {
		return nil, xerrors.Errorf("planner for state %s not found", SectorStates[state.State])
	}

	if err := p(events, state); err != nil {
		return nil, xerrors.Errorf("running planner for state %s failed: %w", SectorStates[state.State], err)
	}

	if m.onSectorUpdated != nil {
		m.onSectorUpdated(state.SectorNum, state.State)
	}

	/////
	// Now decide what to do next

	/*

		*   Empty
		|   |
		|   v
		*<- Packing <- incoming
		|   |
		|   v
		*<- Unsealed <--> SealFailed
		|   |
		|   v
		*   PreCommitting <--> PreCommitFailed
		|   |                  ^
		|   v                  |
		*<- WaitSeed ----------/
		|   |||
		|   vvv      v--> SealCommitFailed
		*<- Committing
		|   |        ^--> CommitFailed
		|   v             ^
		*<- CommitWait ---/
		|   |
		|   v
		*<- Proving
		|
		v
		FailedUnrecoverable

		UndefinedSectorState <- ¯\_(ツ)_/¯
		    |                     ^
		    *---------------------/

	*/

	switch state.State {
	// Happy path
	case Packing:
		return m.handlePacking, nil
	case Unsealed:
		return m.handleUnsealed, nil
	case PreCommitting:
		return m.handlePreCommitting, nil
	case WaitSeed:
		return m.handleWaitSeed, nil
	case Committing:
		return m.handleCommitting, nil
	case CommitWait:
		return m.handleCommitWait, nil
	case FinalizeSector:
		return m.handleFinalizeSector, nil
	case Proving:
		// TODO: track sector health / expiration
		log.Infof("Proving sector %d", state.SectorNum)

	// Handled failure modes
	case SealFailed:
		return m.handleSealFailed, nil
	case PreCommitFailed:
		return m.handlePreCommitFailed, nil
	case SealCommitFailed:
		log.Warnf("sector %d entered unimplemented state 'SealCommitFailed'", state.SectorNum)
	case CommitFailed:
		log.Warnf("sector %d entered unimplemented state 'CommitFailed'", state.SectorNum)

		// Faults
	case Faulty:
		return m.handleFaulty, nil
	case FaultReported:
		return m.handleFaultReported, nil

	// Fatal errors
	case UndefinedSectorState:
		log.Error("sector update with undefined state!")
	case FailedUnrecoverable:
		log.Errorf("sector %d failed unrecoverably", state.SectorNum)
	default:
		log.Errorf("unexpected sector update state: %d", state.State)
	}

	return nil, nil
}

func planCommitting(events []statemachine.Event, state *SectorInfo) error {
	for _, event := range events {
		switch e := event.User.(type) {
		case globalMutator:
			if e.applyGlobal(state) {
				return nil
			}
		case SectorCommitted: // the normal case
			e.apply(state)
			state.State = CommitWait
		case SectorSeedReady: // seed changed :/
			if e.seed.Equals(&state.Seed) {
				log.Warnf("planCommitting: got SectorSeedReady, but the seed didn't change")
				continue // or it didn't!
			}
			log.Warnf("planCommitting: commit Seed changed")
			e.apply(state)
			state.State = Committing
			return nil
		case SectorComputeProofFailed:
			state.State = SealCommitFailed
		case SectorSealFailed:
			state.State = CommitFailed
		case SectorCommitFailed:
			state.State = CommitFailed
		default:
			return xerrors.Errorf("planCommitting got event of unknown type %T, events: %+v", event.User, events)
		}
	}
	return nil
}

func (m *Sealing) restartSectors(ctx context.Context) error {
	trackedSectors, err := m.ListSectors()
	if err != nil {
		log.Errorf("loading sector list: %+v", err)
	}

	for _, sector := range trackedSectors {
		if err := m.sectors.Send(uint64(sector.SectorNum), SectorRestart{}); err != nil {
			log.Errorf("restarting sector %d: %+v", sector.SectorNum, err)
		}
	}

	// TODO: Grab on-chain sector set and diff with trackedSectors

	return nil
}

// ForceSectorState puts a sector with given ID into the given state.
func (m *Sealing) ForceSectorState(ctx context.Context, num abi.SectorNumber, state SectorState) error {
	return m.sectors.Send(uint64(num), SectorForceState{state})
}

func final(events []statemachine.Event, state *SectorInfo) error {
	return xerrors.Errorf("didn't expect any events in state %s, got %+v", SectorStates[state.State], events)
}

func on(mut mutator, next SectorState) func() (mutator, SectorState) {
	return func() (mutator, SectorState) {
		return mut, next
	}
}

func planOne(ts ...func() (mut mutator, next SectorState)) func(events []statemachine.Event, state *SectorInfo) error {
	return func(events []statemachine.Event, state *SectorInfo) error {
		if len(events) != 1 {
			for _, event := range events {
				if gm, ok := event.User.(globalMutator); ok {
					gm.applyGlobal(state)
					return nil
				}
			}
			return xerrors.Errorf("planner for state %s only has a plan for a single event only, got %+v", SectorStates[state.State], events)
		}

		if gm, ok := events[0].User.(globalMutator); ok {
			gm.applyGlobal(state)
			return nil
		}

		for _, t := range ts {
			mut, next := t()

			if reflect.TypeOf(events[0].User) != reflect.TypeOf(mut) {
				continue
			}

			if err, iserr := events[0].User.(error); iserr {
				log.Warnf("sector %d got error event %T: %+v", state.SectorNum, events[0].User, err)
			}

			events[0].User.(mutator).apply(state)
			state.State = next
			return nil
		}

		return xerrors.Errorf("planner for state %s received unexpected event %T (%+v)", SectorStates[state.State], events[0].User, events[0])
	}
}
