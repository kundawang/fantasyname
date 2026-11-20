package stringers

import (
	"fmt"
	"testing"
)

func TestWeightedBoundaries(t *testing.T) {
	t.Parallel()

	values := []int{0, 1, 2, 3, 4, 5}
	calls := 0

	rnd := func(n int) int {
		if n != 6 {
			t.Errorf("unexpected total: %d", n)
		}

		v := values[calls%len(values)]
		calls++

		return v
	}

	val := MakeWeighted(
		[]fmt.Stringer{Literal("a"), Literal("b"), Literal("c")},
		[]int{3, 1, 2},
		rnd,
	)

	want := []string{"a", "a", "a", "b", "c", "c"}

	for i, exp := range want {
		if got := val.String(); got != exp {
			t.Errorf("call %d: unexpected result: %q, want %q", i, got, exp)
		}
	}

	if calls != len(want) {
		t.Errorf("unexpected number of random calls: %d", calls)
	}
}

func TestWeightedFallbackToUniform(t *testing.T) {
	t.Parallel()

	items := []fmt.Stringer{Literal("a"), Literal("b")}

	for _, weights := range [][]int{nil, {1}, {1, 0}, {-1, 2}} {
		if _, ok := MakeWeighted(items, weights, func(int) int { return 0 }).(*Random); !ok {
			t.Errorf("expected uniform fallback for weights: %v", weights)
		}
	}
}

func TestWeightedEmpty(t *testing.T) {
	t.Parallel()

	if s := MakeWeighted(nil, nil, func(int) int { return 0 }); s.String() != "" {
		t.Errorf("unexpected result: %q", s.String())
	}
}
