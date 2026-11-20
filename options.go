package fantasyname

import (
	"fmt"
	"maps"
	"math/rand"
)

const (
	defaultMaxNesting       = 64
	defaultMaxFragmentDepth = 32
	defaultMaxExpansions    = 4096
)

type config struct {
	RandIntN         func(int) int
	Dictionary       map[rune][]fmt.Stringer
	Fragments        map[string]string
	Labels           map[string]struct{}
	Collapse         bool
	StrictLabels     bool
	Legacy           bool
	MaxNesting       int
	MaxFragmentDepth int
	MaxExpansions    int
}

func newConfig(opts ...Option) (c *config) {
	c = &config{}

	for _, o := range opts {
		o(c)
	}

	c.validate()

	return c
}

func (c *config) validate() {
	if c.RandIntN == nil {
		c.RandIntN = rand.Intn
	}

	if c.Dictionary == nil {
		c.Dictionary = symbolMap
	}

	if c.Labels == nil {
		c.Labels = make(map[string]struct{})
	}

	if c.MaxNesting <= 0 {
		c.MaxNesting = defaultMaxNesting
	}

	if c.MaxFragmentDepth <= 0 {
		c.MaxFragmentDepth = defaultMaxFragmentDepth
	}

	if c.MaxExpansions <= 0 {
		c.MaxExpansions = defaultMaxExpansions
	}
}

// Option is a configuration func.
type Option func(*config)

// Collapse creates option for collapsing doubles in result.
func Collapse(v bool) Option {
	return func(c *config) {
		c.Collapse = v
	}
}

// RandFn creates option with custom random func.
func RandFn(v func(int) int) Option {
	return func(c *config) {
		c.RandIntN = v
	}
}

// Dictionary sets custom dictionary (that maps runes to possible strings) for generation.
func Dictionary(d map[rune][]string) Option {
	return func(c *config) {
		t := make(map[rune][]fmt.Stringer)

		for k, v := range d {
			t[k] = convert(v)
		}

		c.Dictionary = t
	}
}

// Labels enables conditional fragments (see [label:pattern] syntax) that
// reference any of the given label names. Labels not listed here make their
// fragments expand to nothing, unless [StrictLabels] is enabled.
func Labels(names ...string) Option {
	return func(c *config) {
		if c.Labels == nil {
			c.Labels = make(map[string]struct{}, len(names))
		}

		for _, n := range names {
			c.Labels[n] = struct{}{}
		}
	}
}

// Legacy disables the extended syntax, so that {, }, [, ], $ and \ are emitted
// literally again, exactly as before the extension existed. Use it to compile
// stored patterns whose literal text may contain those characters.
func Legacy(v bool) Option {
	return func(c *config) {
		c.Legacy = v
	}
}

// StrictLabels makes a reference to a label that was not declared with
// [Labels] a compile-time error (ErrUnknownLabel) instead of silently
// disabling the fragment.
func StrictLabels(v bool) Option {
	return func(c *config) {
		c.StrictLabels = v
	}
}

// Fragments registers named patterns, referenced from a pattern as $name.
// Fragments may reference other fragments, but recursive cycles are rejected
// with ErrRecursiveCycle.
func Fragments(m map[string]string) Option {
	return func(c *config) {
		c.Fragments = make(map[string]string, len(m))
		maps.Copy(c.Fragments, m)
	}
}

// MaxNesting sets the maximum depth of nested groups, conditionals and
// fragment references, deeper patterns fail with ErrTooDeep. Non-positive
// values keep the default of 64.
func MaxNesting(n int) Option {
	return func(c *config) {
		c.MaxNesting = n
	}
}

// MaxFragmentDepth sets the maximum chain of nested $fragment references,
// longer chains fail with ErrTooDeep. Non-positive values keep the default
// of 32.
func MaxFragmentDepth(n int) Option {
	return func(c *config) {
		c.MaxFragmentDepth = n
	}
}

// MaxExpansions sets the maximum total number of fragment expansions in a
// single Compile call, exceeding it fails with ErrTooDeep. This bounds
// exponential fan-out of fragments that reference the same fragment several
// times. Non-positive values keep the default of 4096.
func MaxExpansions(n int) Option {
	return func(c *config) {
		c.MaxExpansions = n
	}
}
