package fantasyname

import (
	"testing"
)

// detRand is deterministic and independent of math/rand, so the golden values
// below are stable across Go releases.
func detRand() func(int) int {
	c := 0

	return func(n int) int {
		c++

		return (c*7 + c/3) % n
	}
}

const (
	bigSyllableA = "aaa"
	bigSyllableB = "bbb"
	bigSyllableC = "ccc"
)

type legacyCase struct {
	Pattern string
	Want    []string
}

// legacyCases are patterns valid before the syntax extension. The values
// were captured from the pre-extension implementation.
//
//nolint:funlen // a single data table is easier to audit than several.
func legacyCases() []legacyCase {
	return []legacyCase{
		{
			"sV'i",
			[]string{"angey'goof", "eldey'wad", "ingey'finger", "osey'thimble", "taney'dork",
				"adey'skull", "deley'dip", "ightey'numb", "oldey'clot", "serey'muck"},
		},
		{
			"<s|cvc>",
			[]string{"vum", "hon", "yap", "sam", "jis", "bel", "xuv", "lor", "cay", "unt"},
		},
		{
			"~(foo)c'<s|cvc>!(a)",
			[]string{"oofl'danA", "oofm'honA", "oofn'nyA", "oofp'samA", "oofr'vesA",
				"oofs'belA", "ooft'etA", "oofv'lorA", "oofw'rilA", "oofy'untA"},
		},
		{
			"(a|b|c)",
			[]string{"b", "c", "b", "c", "a", "c", "a", "b", "a", "b"},
		},
		{
			"<(a|b)|(c|d)>",
			[]string{"c", "b", "a", "c", "b", "a", "c", "b", "a", "c"},
		},
		{
			"(a|)b",
			[]string{"b", "ab", "ab", "b", "ab", "ab", "b", "ab", "ab", "b"},
		},
		{
			"!s~v",
			[]string{"Angi", "Dany", "Eri", "Ingu", "Nye", "Raku", "Tana", "Veso", "Arda", "Deli"},
		},
		{
			"(((((<(((((((((((((((a)))))))))))))))>)))))",
			[]string{"a", "a", "a", "a", "a", "a", "a", "a", "a", "a"},
		},
		{
			"<s|v|>",
			[]string{"i", "y", "hon", "kim", "os", "", "y", "ves", "ard", "del"},
		},
		{
			"<i|s>v(mon|chu|zard|rtle)",
			[]string{"aughuchu", "wadozard", "knockerimon", "turichu", "skullezard",
				"footamon", "perachu", "muckyzard", "doltumon", "imuchu"},
		},
		{
			"sv(nia|lia|cia|sia)",
			[]string{"angicia", "eldania", "ingucia", "osinia", "tanacia", "adunia",
				"delicia", "ightania", "olducia", "serinia"},
		},
		{
			"c'<s|cvc>",
			[]string{"l'dan", "m'hon", "n'ny", "p'sam", "r'ves", "s'bel", "t'et", "v'lor", "w'ril", "y'unt"},
		},
		{
			"<>",
			[]string{"", "", "", "", "", "", "", "", "", ""},
		},
		{
			"()",
			[]string{"", "", "", "", "", "", "", "", "", ""},
		},
		{
			"(a||b)",
			[]string{"", "b", "", "b", "a", "b", "a", "", "a", ""},
		},
		{
			"<a|>",
			[]string{"", "a", "a", "", "a", "a", "", "a", "a", ""},
		},
		{
			"!<s|c>",
			[]string{"V", "Eld", "Hon", "Y", "Os", "Sam", "J", "Ad", "Bel", "X"},
		},
		{
			"~<s|c>",
			[]string{"v", "dle", "noh", "y", "so", "mas", "j", "da", "leb", "x"},
		},
		{
			"B<v|V>c(ios)",
			[]string{"lumios", "snoyios", "steygios", "nisios", "swexios", "bleymios",
				"rayios", "tryfios", "chreysios", "tuxios"},
		},
		{
			"<VC|vC|B|BVs|Vs><v|V|v|<v(l|n|r)|vc>>(th)",
			[]string{"vyth", "iwath", "stratanith", "sweth", "uzith", "jaseruth",
				"choth", "achuth", "zharodath", "byth"},
		},
	}
}

// TestLegacyOutputsUnchanged pins the output of patterns valid before the
// syntax extension, using a deterministic random source; they must never change.
func TestLegacyOutputsUnchanged(t *testing.T) {
	t.Parallel()

	for idx, tc := range legacyCases() {
		gen, err := Compile(tc.Pattern, RandFn(detRand()))
		if err != nil {
			t.Errorf("case %d (%q): unexpected error: %v", idx, tc.Pattern, err)

			continue
		}

		for i, want := range tc.Want {
			if got := gen.String(); got != want {
				t.Errorf("case %d (%q) call %d: got %q, want %q", idx, tc.Pattern, i, got, want)
			}
		}
	}

	// unmatched close bracket keeps its historical error
	if _, err := Compile("><", RandFn(detRand())); err == nil {
		t.Error("unmatched bracket: no error")
	}

	// custom dictionary and collapse options stay as they were
	gen, err := Compile("!a!b!c", Dictionary(map[rune][]string{
		'a': {bigSyllableA, "bb"}, 'b': {bigSyllableB}, 'c': {bigSyllableC, "d"},
	}), RandFn(detRand()))
	if err != nil {
		t.Fatalf("dictionary: unexpected error: %v", err)
	}

	for i := 0; i < 10; i++ {
		if got := gen.String(); got != "BbBbbCcc" {
			t.Errorf("dictionary call %d: unexpected result: %q", i, got)
		}
	}

	gen, err = Compile("(xxooo)~s!v", Collapse(true), RandFn(detRand()))
	if err != nil {
		t.Fatalf("collapse: unexpected error: %v", err)
	}

	for i, want := range []string{
		"xoognaI", "xoonadY", "xooreI", "xoogniU", "xooynE",
		"xookarU", "xoonatA", "xoosevO", "xoodraA", "xooledI",
	} {
		if got := gen.String(); got != want {
			t.Errorf("collapse call %d: got %q, want %q", i, got, want)
		}
	}
}
