package fantasyname

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

const (
	heroTag = "[hero:x]"
	dollar  = "$tag"
)

// seqRand returns a random func that yields the given values in order and then
// fails the test, so tests can assert exactly how many random choices were made.
func seqRand(t *testing.T, values ...int) func(int) int {
	t.Helper()

	var pos int

	return func(n int) int {
		if pos >= len(values) {
			t.Fatalf("unexpected random call #%d (n=%d)", pos, n)
		}

		v := values[pos]
		pos++

		if v < 0 || v >= n {
			t.Fatalf("value %d out of range [0,%d)", v, n)
		}

		return v
	}
}

func TestWeightedBranches(t *testing.T) {
	t.Parallel()

	// weights 3:1 for the two literal branches
	gen, err := Compile("{3:(aa)|1:(bb)}", RandFn(seqRand(t, 0, 2, 3, 2)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, want := range []string{"aa", "aa", "bb", "aa"} {
		if got := gen.String(); got != want {
			t.Errorf("call %d: got %q, want %q", i, got, want)
		}
	}
}

func TestWeightedDefaultsToOne(t *testing.T) {
	t.Parallel()

	// "aa" has default weight 1, "bb" has weight 2, "cc" default 1: total 4.
	gen, err := Compile("{(aa)|2:(bb)|(cc)}", RandFn(seqRand(t, 0, 1, 2, 3)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, want := range []string{"aa", "bb", "bb", "cc"} {
		if got := gen.String(); got != want {
			t.Errorf("call %d: got %q, want %q", i, got, want)
		}
	}
}

func TestWeightedNested(t *testing.T) {
	t.Parallel()

	// nested weighted group inside a weighted branch (outer total 4, inner total
	// 4); picking branch 0 consumes a second random call for the nested group
	gen, err := Compile("{2:{1:(xx)|3:(yy)}|2:(zz)}",
		RandFn(seqRand(t, 0, 1, 3, 0, 1)),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, want := range []string{"yy", "zz", "yy"} {
		if got := gen.String(); got != want {
			t.Errorf("call %d: got %q, want %q", i, got, want)
		}
	}
}

func TestWeightedSingleBranch(t *testing.T) {
	t.Parallel()

	// a group with one branch has no choice to make, the weight is irrelevant
	// but still validated
	for _, pat := range []string{"{3:(only)}", "{2:}x", "{1:(only)|}"} {
		if _, err := Compile(pat); err != nil {
			t.Errorf("%q: unexpected error: %v", pat, err)
		}
	}

	if _, err := Compile("{0:(only)}"); !errors.Is(err, ErrBadWeight) {
		t.Errorf("zero weight: unexpected error: %v", err)
	}
}

func TestWeightedClampedRandom(t *testing.T) {
	t.Parallel()

	// a broken random source must not panic: out-of-range values clamp
	gen, err := Compile("{1:(aa)|1:(bb)}", RandFn(func(int) int { return 1000 }))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gen.String(); got != "bb" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestConditionalLabels(t *testing.T) {
	t.Parallel()

	const patronymic = "(Sir )[hero:!s](of )[hero:(the )](House )[hero:$house]"

	gen, err := Compile(patronymic,
		Fragments(map[string]string{"house": "(Stark|Lannister)"}),
		RandFn(xorshift(1)),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for range 10 {
		if got := gen.String(); got != "Sir of House " {
			t.Errorf("disabled labels: unexpected result: %q", got)
		}
	}

	gen, err = Compile(patronymic, Labels("hero"),
		Fragments(map[string]string{"house": "(Stark|Lannister)"}),
		RandFn(xorshift(1)),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for range 10 {
		got := gen.String()
		if !strings.HasPrefix(got, "Sir ") || !strings.Contains(got, "of the House ") ||
			(!strings.HasSuffix(got, "Stark") && !strings.HasSuffix(got, "Lannister")) {
			t.Errorf("enabled labels: unexpected result: %q", got)
		}
	}
}

func TestConditionalNegation(t *testing.T) {
	t.Parallel()

	const pat = "x[!hero:(the )]y"

	gen, err := Compile(pat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gen.String(); got != "xthe y" {
		t.Errorf("unexpected result: %q", got)
	}

	gen, err = Compile(pat, Labels("hero"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gen.String(); got != "xy" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestUnbalancedGroupsReportEndPosition(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Pattern string
		Pos     int
	}{
		{"{a", 2},
		{"[hero:a", 7},
		{"{2:a", 4},
		{"x(a", 3},
		{"<a", 2},
	}

	for _, tc := range cases {
		_, err := Compile(tc.Pattern, Labels("hero"))
		if !errors.Is(err, ErrUnbalancedGroup) {
			t.Errorf("%q: unexpected error: %v", tc.Pattern, err)

			continue
		}

		pos, ok := ErrorPosition(err)
		if !ok || pos != tc.Pos {
			t.Errorf("%q: got pos %d (%v), want %d", tc.Pattern, pos, err, tc.Pos)
		}
	}
}

func TestConditionalNestedGroups(t *testing.T) {
	t.Parallel()

	// a conditional may contain an ordinary group
	gen, err := Compile("x[hero:<c|v>]y", Labels("hero"),
		RandFn(xorshift(3)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for range 10 {
		got := gen.String()
		if len(got) != 3 || !strings.HasPrefix(got, "x") || !strings.HasSuffix(got, "y") {
			t.Errorf("unexpected result: %q", got)
		}
	}
}

func TestConditionalNonSelecting(t *testing.T) {
	t.Parallel()

	// unlike (ab|cd), a conditional block emits all of its parts in order;
	// a single random call is made by the inner group, not by the conditional.
	gen, err := Compile("[hero:(ab|cd)]", Labels("hero"), RandFn(seqRand(t, 1)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gen.String(); got != "cd" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestConditionalInLiteralGroup(t *testing.T) {
	t.Parallel()

	// dictionary symbols inside () are literal; conditionals must inherit that
	gen, err := Compile("(x[hero:s]y)", Labels("hero"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gen.String(); got != "xsy" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestFragments(t *testing.T) {
	t.Parallel()

	frags := map[string]string{
		"first":   "!(<s|cvc>)",
		"surname": "{9:(Smith|Jones|Brown)|1:(Dragonslayer)}",
	}

	gen, err := Compile("$first $surname", Fragments(frags),
		RandFn(xorshift(7)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hero bool

	for range 200 {
		got := gen.String()

		if !strings.Contains(got, " ") {
			t.Fatalf("unexpected result: %q", got)
		}

		if strings.HasSuffix(got, "Dragonslayer") {
			hero = true
		}
	}

	if !hero {
		t.Error("weighted fragment never produced the rare surname")
	}
}

func TestFragmentsNested(t *testing.T) {
	t.Parallel()

	gen, err := Compile("$a!$b", Fragments(map[string]string{
		"a": "(big|small)",
		"b": "$c",
		"c": "(cat)",
	}), RandFn(seqRand(t, 0)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gen.String(); got != "bigCat" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestFragmentsEmpty(t *testing.T) {
	t.Parallel()

	gen, err := Compile("x$blank.y", Fragments(map[string]string{"blank": "  "}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gen.String(); got != "x.y" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestFragmentsInLiteralGroup(t *testing.T) {
	t.Parallel()

	const name = "frag"

	gen, err := Compile("(x$"+name+")", Fragments(map[string]string{name: "s"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gen.String(); got != "xs" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestEscapes(t *testing.T) {
	t.Parallel()

	cases := []struct{ pattern, want string }{
		{`a\|b`, "a|b"},
		{`\{x\}`, "{x}"},
		{`\(a\)`, "(a)"},
		{`\[hero:x\]`, heroTag},
		{`\$tag`, dollar},
		{`\!\~\:`, "!~:"},
		{`\<\>`, "<>"},
	}

	for _, tc := range cases {
		gen, err := Compile(tc.pattern)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.pattern, err)

			continue
		}

		if got := gen.String(); got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.pattern, got, tc.want)
		}
	}

	// escaping only the brackets leaves the symbol in between expandable
	gen, err := Compile(`\<s\>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := gen.String()
	if !strings.HasPrefix(got, "<") || !strings.HasSuffix(got, ">") || len(got) < 3 {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestLegacyMode(t *testing.T) {
	t.Parallel()

	// in legacy mode the extended runes are plain literals again
	for _, tc := range []struct{ pattern, want string }{
		{"{tag", "{tag"},
		{"[hero:x]", heroTag},
		{"$tag", dollar},
		{`a\b`, `a\b`},
	} {
		gen, err := Compile(tc.pattern, Legacy(true))
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tc.pattern, err)

			continue
		}

		if got := gen.String(); got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.pattern, got, tc.want)
		}
	}

	// "|" is still a group-only token: legacy patterns see the old error
	if _, err := Compile("a|b", Legacy(true)); !errors.Is(err, ErrInvalidSplit) {
		t.Errorf("legacy split: unexpected error: %v", err)
	}

	// and "{" does not open a group
	if _, err := Compile("{a|b}", Legacy(true)); !errors.Is(err, ErrInvalidSplit) {
		t.Errorf("legacy weight group: unexpected error: %v", err)
	}
}

func TestOldPatternsUnchanged(t *testing.T) {
	t.Parallel()

	patterns := []string{
		"sV'i",
		"<s|cvc>",
		"~(foo)c'<s|cvc>!(a)",
		"(a|b|c)",
		"<(a|b)|(c|d)>",
		"(a|)b",
		"!s~v",
		"(((((<(((((((((((((((a)))))))))))))))>)))))",

		"<i|s>v(mon|chu|zard|rtle)",
	}

	for _, pat := range patterns {
		gen, err := Compile(pat, RandFn(xorshift(42)))
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", pat, err)
		}

		for range 8 {
			if got := gen.String(); got == "" && pat != "<s|v|>" {
				t.Errorf("%q: unexpected empty result", pat)
			}
		}
	}

	// exact outputs for a deterministic pattern
	gen, err := Compile("!a!b!c", Dictionary(map[rune][]string{
		'a': {bigSyllableA}, 'b': {bigSyllableB}, 'c': {bigSyllableC},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := gen.String(); got != "AaaBbbCcc" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestErrorPositions(t *testing.T) {
	t.Parallel()

	type errCase struct {
		Err     error
		Pattern string
		Pos     int
	}

	cases := []errCase{
		{ErrBadWeight, "{0:a}", 1},
		{ErrBadWeight, "x{-2:a}", 2},
		{ErrBadWeight, "x{3:a|0:b}", 6},
		{ErrBadWeight, "x{99999999999999999999:a}", 2},
		{ErrMissingLabel, "[:a]", 0},
		{ErrBadLabel, "[hero]", 0},
		{ErrBadLabel, "[9:x]", 0},
		{ErrUnknownLabel, "[bogus:x]", 0},
		{ErrBadEscape, `\`, 0},
		{ErrBadEscape, `x\q`, 1},
		{ErrUnknownFragment, "$nope", 0},
		// byte offsets, not rune offsets
		{ErrBadWeight, "éé{0:a}", 5},
	}

	for _, tc := range cases {
		_, err := Compile(tc.Pattern, Labels("hero"), StrictLabels(true))
		if err == nil {
			t.Errorf("%q: no error", tc.Pattern)

			continue
		}

		if !errors.Is(err, tc.Err) {
			t.Errorf("%q: unexpected error: %v", tc.Pattern, err)
		}

		pos, ok := ErrorPosition(err)
		if !ok {
			t.Errorf("%q: no position in error: %v", tc.Pattern, err)

			continue
		}

		if pos != tc.Pos {
			t.Errorf("%q: got pos %d, want %d (%v)", tc.Pattern, pos, tc.Pos, err)
		}
	}
}

func TestErrorPositionAbsent(t *testing.T) {
	t.Parallel()

	if _, ok := ErrorPosition(ErrBadWeight); ok {
		t.Error("sentinel must not report a position")
	}

	if _, ok := ErrorPosition(nil); ok {
		t.Error("nil must not report a position")
	}
}

func TestNestingLimits(t *testing.T) {
	t.Parallel()

	deep := strings.Repeat("[hero:", 64) + "x" + strings.Repeat("]", 64)
	if _, clean := Compile(deep, Labels("hero")); clean != nil {
		t.Errorf("depth 64 must be allowed: %v", clean)
	}

	deep = strings.Repeat("[hero:", 65) + "x" + strings.Repeat("]", 65)
	if _, err := Compile(deep, Labels("hero")); !errors.Is(err, ErrTooDeep) {
		t.Errorf("depth 65: unexpected error: %v", err)
	}

	// an unbalanced pattern with many open groups must also hit the limit
	deep = strings.Repeat("[hero:", 100)
	if _, err := Compile(deep, Labels("hero")); !errors.Is(err, ErrTooDeep) {
		t.Errorf("unbalanced depth: unexpected error: %v", err)
	}

	nested := strings.Repeat("(", 65) + "x" + strings.Repeat(")", 65)
	if _, err := Compile(nested); !errors.Is(err, ErrTooDeep) {
		t.Errorf("group depth: unexpected error: %v", err)
	}

	if _, err := Compile(nested, MaxNesting(128)); err != nil {
		t.Errorf("custom limit: unexpected error: %v", err)
	}
}

func TestFragmentRecursion(t *testing.T) {
	t.Parallel()

	for name, frags := range map[string]map[string]string{
		"self":  {"a": "x$a"},
		"cycle": {"a": "x$b", "b": "y$a"},
	} {
		_, err := Compile("$a", Fragments(frags))
		if !errors.Is(err, ErrRecursiveCycle) {
			t.Errorf("%s: unexpected error: %v", name, err)
		}
	}

	// a fragment may be referenced any number of times, as long as not recursively
	if _, err := Compile("$a$a", Fragments(map[string]string{"a": "x"})); err != nil {
		t.Errorf("repeated reference: unexpected error: %v", err)
	}
}

func TestFragmentFanOutBudget(t *testing.T) {
	t.Parallel()

	// every fragment expands to two references of the next one: 2^N expansions
	frags := make(map[string]string, 33)
	for i := range 33 {
		frags[fmt.Sprintf("f%d", i)] = fmt.Sprintf("$f%d$f%d", i+1, i+1)
	}

	frags["f33"] = "x"

	_, err := Compile("$f0", Fragments(frags), MaxFragmentDepth(64), MaxExpansions(100))
	if !errors.Is(err, ErrTooDeep) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFragmentErrorsArePositional(t *testing.T) {
	t.Parallel()

	_, err := Compile("x$bad", Fragments(map[string]string{"bad": "y$nope"}))
	if !errors.Is(err, ErrUnknownFragment) {
		t.Fatalf("unexpected error: %v", err)
	}

	pos, ok := ErrorPosition(err)
	if !ok {
		t.Fatalf("no position in error: %v", err)
	}

	// the position is relative to the fragment body, and the fragment chain is
	// included in the message
	if pos != 1 {
		t.Errorf("want the position inside the fragment, got %d (%v)", pos, err)
	}

	if !strings.Contains(err.Error(), `fragment "bad"`) {
		t.Errorf("error must name the fragment: %v", err)
	}
}

// FuzzCompileExtended checks that extended syntax with all options never
// panics, on both accepted and rejected input.
func FuzzCompileExtended(f *testing.F) {
	f.Add("{3:(a)|1:(b)}")
	f.Add("[hero:(x)]")
	f.Add("[!hero:(x)]")
	f.Add("$frag")
	f.Add(`a\{b\}`)
	f.Add("{0:a}")
	f.Add("[$:]")

	frags := map[string]string{
		"frag": "{2:(aa)|1:(bb)}",
		"loop": "$loop",
	}

	f.Fuzz(func(_ *testing.T, arg string) {
		gen, err := Compile(arg,
			Labels("hero"),
			Fragments(frags),
			MaxNesting(8),
			MaxFragmentDepth(4),
			MaxExpansions(64),
		)
		if err != nil {
			return
		}

		_ = gen.String()
	})
}
