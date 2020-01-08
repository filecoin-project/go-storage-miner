package test

import (
	"fmt"
	"testing"

	"github.com/filecoin-project/lotus/api"
	"gotest.tools/assert"
)

type sectorStateTracker struct {
	actualSequence   []api.SectorState
	expectedSequence []api.SectorState
	sectorID         uint64
	t                *testing.T
}

func begin(t *testing.T, sectorID uint64, initialState api.SectorState) *sectorStateTracker {
	return &sectorStateTracker{
		actualSequence:   []api.SectorState{},
		expectedSequence: []api.SectorState{initialState},
		sectorID:         sectorID,
		t:                t,
	}
}

func (f *sectorStateTracker) then(s api.SectorState) *sectorStateTracker {
	f.expectedSequence = append(f.expectedSequence, s)
	return f
}

func (f *sectorStateTracker) end() (func(uint64, api.SectorState), func() string, <-chan struct{}) {
	var indx int
	var last api.SectorState
	done := make(chan struct{})

	next := func(sectorID uint64, curr api.SectorState) {
		if sectorID != f.sectorID {
			return
		}

		if indx < len(f.expectedSequence) {
			assert.Equal(f.t, f.expectedSequence[indx], curr, "unexpected transition from %s to %s (expected transition to %s)", api.SectorStates[last], api.SectorStates[curr], api.SectorStates[f.expectedSequence[indx]])
		}

		last = curr
		indx++
		f.actualSequence = append(f.actualSequence, curr)

		// if this is the last value we care about, signal completion
		if f.expectedSequence[len(f.expectedSequence)-1] == curr {
			go func() {
				done <- struct{}{}
			}()
		}
	}

	status := func() string {
		expected := make([]string, len(f.expectedSequence))
		for i, s := range f.expectedSequence {
			expected[i] = api.SectorStates[s]
		}

		actual := make([]string, len(f.actualSequence))
		for i, s := range f.actualSequence {
			actual[i] = api.SectorStates[s]
		}

		return fmt.Sprintf("expected transitions: %+v, actual transitions: %+v", expected, actual)
	}

	return next, status, done
}
