package fantasyname

import (
	"fmt"

	"github.com/s0rg/fantasyname/stringers"
)

// maxWeight bounds the sum of weights in a single weighted group.
const maxWeight = 1 << 20

type wrapper func(fmt.Stringer) fmt.Stringer

type state struct {
	rand          func(int) int
	items         []fmt.Stringer
	weights       []int
	wrappers      []wrapper
	splitpos      int
	branchWeight  int
	IsLiteral     bool
	IsGroup       bool
	IsWeighted    bool
	IsConditional bool
	Active        bool
}

func (s *state) Add(v fmt.Stringer) {
	if !s.IsGroup {
		v = s.applyWrappers(v)
	}

	s.items = append(s.items, v)
}

func (s *state) Wrap(w wrapper) {
	s.wrappers = append(s.wrappers, w)
}

// SetWeight assigns a weight to the branch that is currently being built.
// It is only valid for weighted groups.
func (s *state) SetWeight(w int) error {
	if !s.IsWeighted || w <= 0 || w > maxWeight {
		return ErrBadWeight
	}

	total := int64(w)

	for _, have := range s.weights {
		total += int64(have)
	}

	if total > maxWeight {
		return ErrBadWeight
	}

	s.branchWeight = w

	return nil
}

func (s *state) Split() (rv *state) {
	cutoff := len(s.items) - s.splitpos

	if len(s.items) == 0 || cutoff == 0 {
		s.items = append(s.items, stringers.Empty{})
		s.commitWeight()

		s.splitpos++

		return s
	}

	switch cutoff {
	case 1:
		// optimization - do not create sequence for single stringer.
	default:
		tmp := make([]fmt.Stringer, cutoff)
		copy(tmp, s.items[s.splitpos:])

		s.items[s.splitpos] = stringers.Sequence(tmp)
	}

	s.commitWeight()

	s.splitpos++
	s.items = s.items[:s.splitpos]

	return s
}

func (s *state) Stringer() (rv fmt.Stringer) {
	switch {
	case len(s.items) == 0:
		rv = stringers.Empty{}
	case len(s.items) == 1:
		rv = s.items[0]
	case s.IsWeighted:
		rv = stringers.MakeWeighted(s.items, s.weights, s.rand)
	case s.IsGroup:
		rv = stringers.MakeRandom(s.items, s.rand)
	default:
		rv = stringers.Sequence(s.items)
	}

	return s.applyWrappers(rv)
}

// Conditional returns the stringer for a conditional block. Unlike a group its
// parts are always emitted in order - a conditional never picks a branch.
func (s *state) Conditional() (rv fmt.Stringer) {
	switch len(s.items) {
	case 0:
		rv = stringers.Empty{}
	case 1:
		rv = s.items[0]
	default:
		rv = stringers.Sequence(s.items)
	}

	return s.applyWrappers(rv)
}

func (s *state) applyWrappers(v fmt.Stringer) (rv fmt.Stringer) {
	rv = v

	if len(s.wrappers) > 0 {
		for i, w := range s.wrappers {
			rv = w(rv)
			s.wrappers[i] = nil
		}

		s.wrappers = s.wrappers[:0]
	}

	return rv
}

func (s *state) commitWeight() {
	if !s.IsWeighted {
		return
	}

	w := s.branchWeight
	if w <= 0 {
		w = 1
	}

	s.weights = append(s.weights, w)
	s.branchWeight = 1
}
