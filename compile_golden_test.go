package fantasyname

import (
	"testing"
)

// xorshift is a tiny deterministic pseudo-random source for tests, so pinned
// outputs do not depend on math/rand internals.
func xorshift(seed uint32) func(int) int {
	state := seed
	if state == 0 {
		state = 1
	}

	return func(n int) int {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5

		return int(state>>1) % n
	}
}

const (
	nameJones      = "jones"
	nameSmith      = "smith"
	knightOnly     = "Sir the of "
	knightOrCommon = "Sir of "
	lowborn0       = "lowborn xWyrmbane"
	lowborn1       = "lowborn xSmith"
	smith0         = "xWyrmbane"
	smith1         = "xSmith"
	rareWyrm       = "Wyrmbane"
)

func repeat6(v string) []string {
	return []string{v, v, v, v, v, v}
}

// TestGolden outputs are pinned to a fixed seed: they verify that weighted
// choice, conditional expansion and fragments are reproducible, and guard
// against accidental changes of the generator's random-call order.
func goldenCases() []struct {
	Pattern string
	Labels  []string
	Want    []string
} {
	return []struct {
		Pattern string
		Labels  []string
		Want    []string
	}{
		{
			Pattern: "{3:(Al)|1:(El)}",
			Labels:  nil,
			Want:    []string{"Al", "Al", "El", "Al", "El", "El"},
		},
		{
			Pattern: "{2:(smith)|3:(jones)}",
			Labels:  nil,
			Want:    []string{nameJones, nameJones, nameJones, nameJones, nameSmith, nameJones},
		},
		{
			Pattern: "(Sir )[hero:(the )](of )",
			Labels:  nil,
			Want:    repeat6(knightOrCommon),
		},
		{
			Pattern: "(Sir )[hero:(the )](of )",
			Labels:  []string{"hero"},
			Want:    repeat6(knightOnly),
		},
		{
			Pattern: "[!hero:(lowborn )](x)$sur",
			Labels:  nil,
			Want:    []string{lowborn0, lowborn0, lowborn1, lowborn1, lowborn1, lowborn1},
		},
		{
			Pattern: "[!hero:(lowborn )](x)$sur",
			Labels:  []string{"hero"},
			Want:    []string{smith0, smith0, smith1, smith1, smith1, smith1},
		},
		{
			Pattern: "{1:(a)|1:$frag}",
			Labels:  nil,
			Want:    []string{"a", "a", "zz", "a", "zz", "zz"},
		},
		{
			Pattern: "(<s|cvc>)'{1:(ia)|1:(or)}",
			Labels:  nil,
			Want:    []string{"eld'or", "as'or", "fif'or", "qaw'ia", "fog'ia", "ter'or"},
		},
	}
}

func TestGolden(t *testing.T) {
	t.Parallel()

	frags := map[string]string{
		"sur":  "{4:(Smith)|1:(" + rareWyrm + ")}",
		"frag": "(zz)",
	}

	for idx, tc := range goldenCases() {
		gen, err := Compile(tc.Pattern,
			Labels(tc.Labels...),
			Fragments(frags),
			RandFn(xorshift(99)),
		)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", idx, err)

			continue
		}

		for i, want := range tc.Want {
			if got := gen.String(); got != want {
				t.Errorf("case %d (%q) call %d: got %q, want %q", idx, tc.Pattern, i, got, want)
			}
		}
	}
}

// TestGoldenReproducible compiles the same pattern twice: both generators must
// produce the same sequence for the same seed.
func TestGoldenReproducible(t *testing.T) {
	t.Parallel()

	const pat = "{2:(A)|1:(B)}-[hero:(x)]-$f"

	opts := []Option{
		Labels("hero"),
		Fragments(map[string]string{"f": "{1:(p)|1:(q)}"}),
	}

	first, err := Compile(pat, append(opts, RandFn(xorshift(11)))...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := Compile(pat, append(opts, RandFn(xorshift(11)))...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 0; i < 20; i++ {
		a, b := first.String(), second.String()
		if a != b {
			t.Fatalf("call %d: results differ: %q vs %q", i, a, b)
		}
	}
}
