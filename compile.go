package fantasyname

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/s0rg/fantasyname/stringers"
	"github.com/s0rg/fantasyname/wrappers"
)

// escapable holds runes that may follow a backslash in a pattern.
const escapable = `\{}[]()<>|!~:$`

// compiler turns a pattern into a stringer, one rune at a time.
type compiler struct {
	p    *parser
	conf *config
	// budget is the number of remaining fragment expansions for this Compile call.
	budget  *int
	runes   []rune
	offsets []int
	// frags is the chain of fragment names currently being expanded, it is
	// used to detect recursive references.
	frags []string
	depth int
}

// Compile creates fmt.Stringer from given pattern, you can call it as many times as
// you wish, it will produce new name on every call.
//
// In addition to the classic syntax the following extensions are supported:
//   - {weight:branch|weight:branch|branch} - pick a branch with relative weights;
//     weights are positive integers, an omitted weight means 1;
//   - [label:pattern] and [!label:pattern] - expand the nested pattern only when
//     the label was enabled with the [Labels] option (or was not, for "!");
//   - $name - inline a named fragment registered with the [Fragments] option;
//   - \x - emit x literally, where x is one of \{}[]()<>|!~:$.
func Compile(pattern string, opts ...Option) (rv fmt.Stringer, err error) {
	if pattern = strings.TrimSpace(pattern); pattern == "" {
		return nil, ErrEmptyStack
	}

	c := newConfig(opts...)
	budget := c.MaxExpansions

	return compilePattern(c, pattern, nil, &budget, 0, false)
}

// compilePattern compiles a single pattern (root or fragment) into a stringer.
func compilePattern(
	c *config, pattern string, frags []string, budget *int, depth int, literal bool,
) (rv fmt.Stringer, err error) {
	p := newParser(c, literal)

	cmp := &compiler{
		p:       p,
		conf:    c,
		runes:   []rune(pattern),
		offsets: make([]int, 0, len(pattern)+1),
		depth:   depth,
		budget:  budget,
		frags:   frags,
	}

	pos := 0

	for _, r := range pattern {
		cmp.offsets = append(cmp.offsets, pos)
		pos += len(string(r))
	}

	cmp.offsets = append(cmp.offsets, pos)

	if runErr := cmp.run(); runErr != nil {
		return nil, runErr
	}

	if rv, err = p.Build(); err != nil {
		if errors.Is(err, ErrUnbalancedGroup) {
			return nil, cmp.fail(len(cmp.runes), err)
		}

		return nil, err
	}

	return rv, nil
}

// posError is a compile error annotated with a byte offset into the pattern.
type posError struct {
	Err error
	Pos int
}

func (e *posError) Error() string {
	return fmt.Sprintf("pos: %d error: %v", e.Pos, e.Err)
}

func (e *posError) Unwrap() error {
	return e.Err
}

// ErrorPosition returns the byte offset into the pattern of a Compile error, if
// that error carries a position. Errors raised while expanding a fragment
// report the offset relative to that fragment's source, and their message names
// the fragment chain.
func ErrorPosition(err error) (pos int, ok bool) {
	var pe *posError
	if errors.As(err, &pe) {
		return pe.Pos, true
	}

	return 0, false
}

// fail annotates an error with a byte offset of rune index idx. Errors that
// already carry a position (e.g. from a nested fragment) are left untouched,
// so the innermost location and the fragment chain are preserved.
func (c *compiler) fail(idx int, err error) error {
	var pe *posError
	if errors.As(err, &pe) {
		return err
	}

	pos := 0
	if idx >= 0 && idx < len(c.offsets) {
		pos = c.offsets[idx]
	}

	return &posError{Pos: pos, Err: err}
}

func (c *compiler) run() (err error) {
	for i := 0; i < len(c.runes); i++ {
		if i, err = c.step(i); err != nil {
			return c.fail(i, err)
		}
	}

	return nil
}

// step consumes one pattern token starting at index i and returns the index of
// the last consumed rune.
func (c *compiler) step(i int) (last int, err error) {
	r := c.runes[i]

	if c.conf.Legacy {
		return c.legacy(i, r)
	}

	switch r {
	case '<', '(', '{', '[':
		return c.open(i, r)
	case '>', ')', '}', ']':
		return c.close(i, r)
	default:
		return c.special(i, r)
	}
}

// legacy reproduces the pre-extension behavior: only <>()|!~ tokens are
// special, everything else (including {, [, ], $ and \) is emitted literally.
func (c *compiler) legacy(i int, r rune) (last int, err error) {
	switch r {
	case '<', '(':
		return c.open(i, r)
	case '>', ')':
		return c.close(i, r)
	case '|':
		return c.split(i)
	case '!':
		return i, c.p.Wrap(wrappers.Capitalized)
	case '~':
		return i, c.p.Wrap(wrappers.Reversed)
	default:
		return i, c.p.OnSymbol(r)
	}
}

// open handles '<', '(', '{' and '['.
func (c *compiler) open(i int, r rune) (last int, err error) {
	switch r {
	case '<':
		if err := c.push(); err != nil {
			return i, err
		}

		c.p.OnGroupStart(false)
	case '(':
		if err := c.push(); err != nil {
			return i, err
		}

		c.p.OnGroupStart(true)
	case '[':
		return c.conditional(i)
	default: // '{'
		if err := c.push(); err != nil {
			return i, err
		}

		c.p.OnWeightedGroupStart(c.p.literal())

		return c.weight(i + 1)
	}

	return i, nil
}

// close handles '>', ')', '}' and ']'.
func (c *compiler) close(i int, r rune) (last int, err error) {
	switch r {
	case '>':
		c.pop()

		return i, c.p.OnGroupEnd(false)
	case ')':
		c.pop()

		return i, c.p.OnGroupEnd(true)
	case ']':
		c.pop()

		return i, c.p.OnConditionalEnd()
	default: // '}'
		c.pop()

		return i, c.p.OnWeightedGroupEnd()
	}
}

// special handles tokens that do not open or close a group.
func (c *compiler) special(i int, r rune) (last int, err error) {
	switch r {
	case '\\':
		return c.escape(i)
	case '|':
		return c.split(i)
	case '!':
		return i, c.p.Wrap(wrappers.Capitalized)
	case '~':
		return i, c.p.Wrap(wrappers.Reversed)
	case '$':
		return c.fragment(i)
	default:
		return i, c.p.OnSymbol(r)
	}
}

func (c *compiler) push() error {
	c.depth++

	if c.depth > c.conf.MaxNesting {
		return ErrTooDeep
	}

	return nil
}

func (c *compiler) pop() {
	if c.depth > 0 {
		c.depth--
	}
}

// split handles '|'.
func (c *compiler) split(i int) (last int, err error) {
	if err := c.p.OnGroupSplit(); err != nil {
		return i, err
	}

	if top, ok := c.p.stack.Top(); ok && top.IsWeighted && !c.conf.Legacy {
		return c.weight(i + 1)
	}

	return i, nil
}

// escape handles a backslash at index i.
func (c *compiler) escape(i int) (last int, err error) {
	if i+1 >= len(c.runes) || !strings.ContainsRune(escapable, c.runes[i+1]) {
		return i, ErrBadEscape
	}

	return i + 1, c.p.OnSymbolLiteral(c.runes[i+1])
}

// weight parses an optional "N:" / "+N:" / "-N:" prefix starting at index i.
// It is a no-op when the runes do not form a weight token.
func (c *compiler) weight(i int) (last int, err error) {
	raw, colon := c.scanWeight(i)

	if raw == "" {
		return i - 1, nil
	}

	n, convErr := strconv.Atoi(raw)
	if convErr != nil || n <= 0 {
		return i, ErrBadWeight
	}

	if err := c.p.SetWeight(n); err != nil {
		return i, err
	}

	return colon, nil
}

// scanWeight returns the raw weight token starting at i (empty when there is
// none) and the index of the ':' that follows it.
func (c *compiler) scanWeight(i int) (raw string, colon int) {
	j := i

	if j < len(c.runes) && (c.runes[j] == '+' || c.runes[j] == '-') {
		j++
	}

	digits := j

	for j < len(c.runes) && c.runes[j] >= '0' && c.runes[j] <= '9' {
		j++
	}

	if j == digits || j >= len(c.runes) || c.runes[j] != ':' {
		return "", 0
	}

	return string(c.runes[i:j]), j
}

// conditional parses a [label:pattern] or [!label:pattern] construct at index i.
func (c *compiler) conditional(i int) (last int, err error) {
	name, negate, colon, err := c.scanLabel(i + 1)
	if err != nil {
		return i, err
	}

	_, enabled := c.conf.Labels[name]

	if !enabled && c.conf.StrictLabels {
		return i, fmt.Errorf("%w: %q", ErrUnknownLabel, name)
	}

	if err := c.push(); err != nil {
		return i, err
	}

	c.p.OnConditionalStart(enabled != negate, c.p.literal())

	return colon, nil
}

// scanLabel reads an optional '!' and a label name starting at i, returning the
// name, the negation flag and the index of the ':' separator.
func (c *compiler) scanLabel(i int) (name string, negate bool, colon int, err error) {
	j := i

	if j < len(c.runes) && c.runes[j] == '!' {
		negate = true
		j++
	}

	begin := j

	for j < len(c.runes) && isLabelRune(c.runes[j], j == begin) {
		j++
	}

	if j == begin {
		if j+1 < len(c.runes) && c.runes[j] == ':' {
			return "", false, 0, ErrMissingLabel
		}

		return "", false, 0, ErrBadLabel
	}

	if j >= len(c.runes) || c.runes[j] != ':' {
		return "", false, 0, ErrBadLabel
	}

	return string(c.runes[begin:j]), negate, j, nil
}

// fragment parses a $name reference at index i and inlines the fragment. A
// "$" that cannot start a name is emitted literally, so patterns that used it
// as plain text keep working.
func (c *compiler) fragment(i int) (last int, err error) {
	j := i + 1
	begin := j

	for j < len(c.runes) && isLabelRune(c.runes[j], j == begin) {
		j++
	}

	if j == begin {
		return i, c.p.OnSymbol('$')
	}

	v, err := c.expand(string(c.runes[begin:j]))
	if err != nil {
		return i, err
	}

	return j - 1, c.p.Add(v)
}

// expand resolves a fragment reference into a stringer.
func (c *compiler) expand(fname string) (rv fmt.Stringer, err error) {
	src, ok := c.conf.Fragments[fname]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownFragment, fname)
	}

	if slices.Contains(c.frags, fname) {
		return nil, fmt.Errorf("%w: %q", ErrRecursiveCycle, fname)
	}

	if len(c.frags)+1 > c.conf.MaxFragmentDepth {
		return nil, fmt.Errorf("%w: fragment depth exceeds %d", ErrTooDeep, c.conf.MaxFragmentDepth)
	}

	if *c.budget <= 0 {
		return nil, fmt.Errorf("%w: fragment expansion budget exceeded", ErrTooDeep)
	}

	*c.budget--

	if strings.TrimSpace(src) == "" {
		return stringers.Empty{}, nil
	}

	chain := make([]string, len(c.frags), len(c.frags)+1)
	copy(chain, c.frags)
	chain = append(chain, fname)

	if rv, err = compilePattern(c.conf, src, chain, c.budget, c.depth+1, c.p.literal()); err != nil {
		return nil, fmt.Errorf("fragment %q: %w", fname, err)
	}

	return rv, nil
}

func isLabelRune(r rune, first bool) bool {
	letter := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')

	if first {
		return letter
	}

	return letter || r == '-' || (r >= '0' && r <= '9')
}
