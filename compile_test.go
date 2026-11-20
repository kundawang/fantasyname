package fantasyname

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
)

func TestCompileErrors(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Err     error
		Pattern string
	}

	cases := []testCase{
		{Pattern: "", Err: ErrEmptyStack},
		{Pattern: "a>", Err: ErrEmptyStack},
		{Pattern: "a)", Err: ErrEmptyStack},
		{Pattern: "<a", Err: ErrUnbalancedGroup},
		{Pattern: "(a", Err: ErrUnbalancedGroup},
		{Pattern: "<a)", Err: ErrUnbalancedGroup},
		{Pattern: "(a>", Err: ErrUnbalancedGroup},
		{Pattern: "a|b", Err: ErrInvalidSplit},
		// extended syntax
		{Pattern: "{a", Err: ErrUnbalancedGroup},
		{Pattern: "a}", Err: ErrEmptyStack},
		{Pattern: "[hero:a", Err: ErrUnbalancedGroup},
		{Pattern: "a]", Err: ErrEmptyStack},
		{Pattern: "<[hero:a>", Err: ErrUnbalancedGroup},
		{Pattern: "[hero:<a]", Err: ErrUnbalancedGroup},
		{Pattern: "{0:a}", Err: ErrBadWeight},
		{Pattern: "{-2:a}", Err: ErrBadWeight},
		{Pattern: "{a|0:b}", Err: ErrBadWeight},
		{Pattern: "{99999999999999999999:a}", Err: ErrBadWeight},
		{Pattern: "[:a]", Err: ErrMissingLabel},
		{Pattern: "[hero]", Err: ErrBadLabel},
		{Pattern: "[9:x]", Err: ErrBadLabel},
		{Pattern: "[a b:x]", Err: ErrBadLabel},
		{Pattern: "[bogus:x]", Err: ErrUnknownLabel},
		{Pattern: `\`, Err: ErrBadEscape},
		{Pattern: `\q`, Err: ErrBadEscape},
		{Pattern: "$nope", Err: ErrUnknownFragment},
	}

	for idx, tc := range cases {
		_, err := Compile(tc.Pattern,
			Labels("hero"),
			StrictLabels(true),
			Fragments(map[string]string{"a": "v"}),
		)
		if err == nil {
			t.Errorf("no error for case: %d", idx)
		}

		if !errors.Is(err, tc.Err) {
			t.Errorf("unexpected error for case: %d: %v", idx, err)
		}
	}
}

func TestCompileTricky(t *testing.T) {
	t.Parallel()

	const (
		pat = "(((((<(((((((((((((((a)))))))))))))))>)))))"
		val = "a"
	)

	gen, err := Compile(pat)
	if err != nil {
		t.Errorf("uexpected error: %v", err)
	}

	if rv := gen.String(); rv != val {
		t.Errorf("unexpected result: '%s'", rv)
	}
}

func TestCompileMain(t *testing.T) {
	t.Parallel()

	const (
		pat  = "~(foo)c'<s|cvc>!(a)"
		valp = "A"
		vals = "oof"
	)

	gen, err := Compile(pat, RandFn(rand.Intn))
	if err != nil {
		t.Errorf("uexpected error: %v", err)
	}

	if rv := gen.String(); !strings.HasPrefix(rv, vals) || !strings.HasSuffix(rv, valp) {
		t.Errorf("unexpected result: '%s'", rv)
	}
}

func TestCompileCollapse(t *testing.T) {
	t.Parallel()

	const (
		pat = "(xxooo)"
		val = "xoo"
	)

	gen, err := Compile(pat, Collapse(true))
	if err != nil {
		t.Errorf("uexpected error: %v", err)
	}

	if rv := gen.String(); rv != val {
		t.Errorf("unexpected result: '%s'", rv)
	}
}

func TestCompileDictionary(t *testing.T) {
	t.Parallel()

	const (
		pat = "!a!b!c"
		val = "AaaBbbCcc"
	)

	custom := map[rune][]string{
		'a': {bigSyllableA},
		'b': {bigSyllableB},
		'c': {bigSyllableC},
	}

	gen, err := Compile(pat, Dictionary(custom))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if rv := gen.String(); rv != val {
		t.Errorf("unexpected result: '%s'", rv)
	}
}

func FuzzCompile(f *testing.F) {
	f.Add("<i|s>v(mon|chu|zard|rtle)")
	f.Fuzz(func(t *testing.T, arg string) {
		_, err := Compile(arg)
		if err != nil {
			t.Skip("only correct templates are intresting")
		}
	})
}
