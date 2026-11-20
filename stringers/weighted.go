package stringers

import (
	"fmt"
	"math"
)

// Weighted holds a bunch of stringers and chooses among them on every call,
// honoring the relative positive weight of each item.
type Weighted struct {
	rndfn   func(int) int
	items   []fmt.Stringer
	weights []int
	total   int
}

// MakeWeighted creates a Weighted stringer from a list of stringers, their
// positive weights and a random func. Fallbacks to uniform selection are made
// for empty or inconsistent input, so malformed (but syntactically valid)
// state can never cause a panic.
func MakeWeighted(items []fmt.Stringer, weights []int, rndfn func(int) int) fmt.Stringer {
	if len(items) == 0 {
		return Empty{}
	}

	if len(weights) != len(items) {
		return MakeRandom(items, rndfn)
	}

	var total int64

	for _, w := range weights {
		if w <= 0 || total > math.MaxInt32-int64(w) {
			return MakeRandom(items, rndfn)
		}

		total += int64(w)
	}

	if total == 0 {
		return MakeRandom(items, rndfn)
	}

	return &Weighted{
		rndfn:   rndfn,
		items:   items,
		weights: weights,
		total:   int(total),
	}
}

// String implements stringer for Weighted.
func (w *Weighted) String() string {
	n := w.rndfn(w.total)

	for i, weight := range w.weights {
		if n < weight {
			return w.items[i].String()
		}

		n -= weight
	}

	// Defensive: a broken random source returned an out-of-range value.
	return w.items[len(w.items)-1].String()
}
