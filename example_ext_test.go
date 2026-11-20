package fantasyname_test

import (
	"fmt"
	"strings"

	"github.com/s0rg/fantasyname"
)

// xorshift keeps the example output deterministic and independent of math/rand.
func xorshift(seed uint32) func(int) int {
	state := seed

	return func(n int) int {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5

		return int(state>>1) % n
	}
}

func ExampleCompile_weighted() {
	gen, err := fantasyname.Compile("{3:(Al)|1:(El)}", fantasyname.RandFn(xorshift(99)))
	if err != nil {
		panic(err)
	}

	got := make([]string, 0, 6)
	for range 6 {
		got = append(got, gen.String())
	}

	fmt.Println(strings.Join(got, " "))
	// Output: Al Al El Al El El
}

func ExampleCompile_labels() {
	frags := map[string]string{
		"title":   "{4:(Ser )|1:(the Bold )}",
		"surname": "{9:(Smith|Jones)|1:(Wyrmbane)}",
	}

	common, err := fantasyname.Compile("[hero:$title]$surname",
		fantasyname.Fragments(frags),
		fantasyname.RandFn(xorshift(7)),
	)
	if err != nil {
		panic(err)
	}

	hero, err := fantasyname.Compile("[hero:$title]$surname",
		fantasyname.Labels("hero"),
		fantasyname.Fragments(frags),
		fantasyname.RandFn(xorshift(7)),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(strings.Contains(common.String(), " ")) // no title for a commoner
	fmt.Println(strings.Contains(hero.String(), " "))   // title for a hero
	// Output:
	// false
	// true
}
