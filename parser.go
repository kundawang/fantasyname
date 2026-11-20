package fantasyname

import (
	"errors"
	"fmt"

	"github.com/s0rg/fantasyname/stringers"
	"github.com/s0rg/fantasyname/wrappers"
)

var (
	// ErrEmptyStack is returned, if parser counts stack as non-empty, but it was.
	ErrEmptyStack = errors.New("empty stack")
	// ErrInvalidSplit is returned when split - "|" is found in pattern out of any group.
	ErrInvalidSplit = errors.New("invalid split")
	// ErrUnbalancedGroup is returned when stuck upon unexpected group close/open brackets.
	ErrUnbalancedGroup = errors.New("unbalanced group")
	// ErrBadWeight is returned for zero, negative, too large or misplaced weights.
	ErrBadWeight = errors.New("invalid weight")
	// ErrMissingLabel is returned when a conditional has no label before the ":".
	ErrMissingLabel = errors.New("missing label")
	// ErrBadLabel is returned when a conditional label is malformed or is not
	// followed by the ":" separator.
	ErrBadLabel = errors.New("invalid label")
	// ErrUnknownLabel is returned for labels not declared with the Labels option.
	ErrUnknownLabel = errors.New("unknown label")
	// ErrBadEscape is returned for invalid or dangling escape sequences.
	ErrBadEscape = errors.New("invalid escape")
	// ErrUnknownFragment is returned when a referenced fragment was not registered.
	ErrUnknownFragment = errors.New("unknown fragment")
	// ErrRecursiveCycle is returned when fragments reference each other in a cycle.
	ErrRecursiveCycle = errors.New("recursive fragment cycle")
	// ErrTooDeep is returned when nesting exceeds the configured limits.
	ErrTooDeep = errors.New("nesting too deep")
)

type parser struct {
	conf  *config
	stack statestack
}

func newParser(c *config, literal bool) (p *parser) {
	p = &parser{conf: c}

	// "root" state
	p.stack.Push(&state{
		IsLiteral:    literal,
		rand:         p.conf.RandIntN,
		branchWeight: 1,
	})

	return p
}

func (p *parser) OnGroupStart(isLiteral bool) {
	p.stack.Push(&state{
		IsGroup:      true,
		IsLiteral:    isLiteral,
		rand:         p.conf.RandIntN,
		branchWeight: 1,
	})
}

func (p *parser) OnGroupEnd(isLiteral bool) (err error) {
	cur, ok := p.stack.Pop()
	if !ok {
		return ErrEmptyStack
	}

	top, ok := p.stack.Top()
	if !ok {
		return ErrEmptyStack
	}

	if cur.IsConditional || cur.IsWeighted || !cur.IsGroup || cur.IsLiteral != isLiteral {
		return ErrUnbalancedGroup
	}

	top.Add(cur.Split().Stringer())

	return nil
}

func (p *parser) OnWeightedGroupStart(isLiteral bool) {
	p.stack.Push(&state{
		IsGroup:      true,
		IsLiteral:    isLiteral,
		IsWeighted:   true,
		rand:         p.conf.RandIntN,
		branchWeight: 1,
	})
}

func (p *parser) OnWeightedGroupEnd() (err error) {
	cur, ok := p.stack.Pop()
	if !ok {
		return ErrEmptyStack
	}

	top, ok := p.stack.Top()
	if !ok {
		return ErrEmptyStack
	}

	if !cur.IsGroup || !cur.IsWeighted || cur.IsConditional {
		return ErrUnbalancedGroup
	}

	top.Add(cur.Split().Stringer())

	return nil
}

func (p *parser) OnConditionalStart(active, isLiteral bool) {
	p.stack.Push(&state{
		IsGroup:       true,
		IsConditional: true,
		IsLiteral:     isLiteral,
		Active:        active,
		rand:          p.conf.RandIntN,
		branchWeight:  1,
	})
}

func (p *parser) OnConditionalEnd() (err error) {
	cur, ok := p.stack.Pop()
	if !ok {
		return ErrEmptyStack
	}

	top, ok := p.stack.Top()
	if !ok {
		return ErrEmptyStack
	}

	if !cur.IsConditional || cur.IsWeighted {
		return ErrUnbalancedGroup
	}

	v := cur.Conditional()
	if !cur.Active {
		v = stringers.Empty{}
	}

	top.Add(v)

	return nil
}

func (p *parser) OnGroupSplit() (err error) {
	top, ok := p.stack.Top()
	if !ok {
		return ErrEmptyStack
	}

	if !top.IsGroup || top.IsConditional {
		return ErrInvalidSplit
	}

	top.Split()

	return nil
}

func (p *parser) SetWeight(w int) (err error) {
	top, ok := p.stack.Top()
	if !ok {
		return ErrEmptyStack
	}

	return top.SetWeight(w)
}

func (p *parser) Add(v fmt.Stringer) (err error) {
	top, ok := p.stack.Top()
	if !ok {
		return ErrEmptyStack
	}

	top.Add(v)

	return nil
}

func (p *parser) Wrap(w wrapper) (err error) {
	top, ok := p.stack.Top()
	if !ok {
		return ErrEmptyStack
	}

	top.Wrap(w)

	return nil
}

func (p *parser) OnSymbol(r rune) (err error) {
	top, ok := p.stack.Top()
	if !ok {
		return ErrEmptyStack
	}

	var v fmt.Stringer

	if !top.IsLiteral {
		if sym, ok := p.conf.Dictionary[r]; ok {
			v = stringers.MakeRandom(sym, p.conf.RandIntN)
		}
	}

	if v == nil {
		v = stringers.Literal(string(r))
	}

	top.Add(v)

	return nil
}

func (p *parser) OnSymbolLiteral(r rune) (err error) {
	top, ok := p.stack.Top()
	if !ok {
		return ErrEmptyStack
	}

	top.Add(stringers.Literal(string(r)))

	return nil
}

func (p *parser) Build() (s fmt.Stringer, err error) {
	if len(p.stack) != 1 {
		return nil, ErrUnbalancedGroup
	}

	// no need to check `ok` - its already checked above.
	top, _ := p.stack.Top()

	s = top.Stringer()

	if p.conf.Collapse {
		s = wrappers.Collapsed(s)
	}

	return s, nil
}

// literal reports whether the innermost construction context is literal, so
// nested conditionals and weighted groups can inherit it.
func (p *parser) literal() bool {
	if top, ok := p.stack.Top(); ok {
		return top.IsLiteral
	}

	return false
}
