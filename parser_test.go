package fantasyname

import (
	"errors"
	"testing"

	"github.com/s0rg/fantasyname/stringers"
	"github.com/s0rg/fantasyname/wrappers"
)

func TestParserErrors(t *testing.T) {
	t.Parallel()

	var e error

	p := &parser{}

	if e = p.OnGroupEnd(true); e == nil {
		t.Error("group end: no error")
	}

	if !errors.Is(e, ErrEmptyStack) {
		t.Error("group end: unexpected error")
	}

	if e = p.OnGroupSplit(); e == nil {
		t.Error("group split: no error")
	}

	if !errors.Is(e, ErrEmptyStack) {
		t.Error("group split: unexpected error")
	}

	if e = p.OnSymbol(' '); e == nil {
		t.Error("symbol: no error")
	}

	if !errors.Is(e, ErrEmptyStack) {
		t.Error("symbol: unexpected error")
	}

	if e = p.Wrap(wrappers.Collapsed); e == nil {
		t.Error("wrap: no error")
	}

	if !errors.Is(e, ErrEmptyStack) {
		t.Error("wrap: unexpected error")
	}
}

func TestParserErrorsExtended(t *testing.T) {
	t.Parallel()

	p := &parser{}

	if e := p.SetWeight(1); !errors.Is(e, ErrEmptyStack) {
		t.Error("weight: unexpected error")
	}

	if e := p.Add(stringers.Literal("x")); !errors.Is(e, ErrEmptyStack) {
		t.Error("add: unexpected error")
	}

	if e := p.OnSymbolLiteral('x'); !errors.Is(e, ErrEmptyStack) {
		t.Error("literal: unexpected error")
	}
}

func TestParserMismatchedEnds(t *testing.T) {
	t.Parallel()

	newParser := func() *parser {
		return &parser{}
	}

	p := newParser()
	p.stack.Push(&state{IsGroup: true, IsLiteral: true})
	p.stack.Push(&state{IsGroup: true, IsConditional: true, IsLiteral: true})

	if err := p.OnGroupEnd(true); !errors.Is(err, ErrUnbalancedGroup) {
		t.Errorf("conditional closed as group: unexpected error: %v", err)
	}

	p = newParser()
	p.stack.Push(&state{IsGroup: true, IsLiteral: true})
	p.stack.Push(&state{IsGroup: true, IsWeighted: true, IsLiteral: true})

	if err := p.OnGroupEnd(true); !errors.Is(err, ErrUnbalancedGroup) {
		t.Errorf("weighted closed as group: unexpected error: %v", err)
	}

	p = newParser()
	p.stack.Push(&state{IsGroup: true, IsLiteral: true})
	p.stack.Push(&state{IsGroup: true, IsLiteral: false})

	if err := p.OnGroupEnd(true); !errors.Is(err, ErrUnbalancedGroup) {
		t.Errorf("literal mismatch: unexpected error: %v", err)
	}

	p = newParser()
	p.stack.Push(&state{IsGroup: true, IsLiteral: true})
	p.stack.Push(&state{IsGroup: true, IsLiteral: true})

	if err := p.OnWeightedGroupEnd(); !errors.Is(err, ErrUnbalancedGroup) {
		t.Errorf("plain closed as weighted: unexpected error: %v", err)
	}

	p = newParser()
	p.stack.Push(&state{IsGroup: true, IsLiteral: true})
	p.stack.Push(&state{IsGroup: true, IsConditional: true, IsLiteral: true})

	if err := p.OnConditionalEnd(); err != nil {
		t.Errorf("conditional end: unexpected error: %v", err)
	}
}

func TestParserConditionalSplit(t *testing.T) {
	t.Parallel()

	p := &parser{}
	p.stack.Push(&state{IsGroup: true, IsConditional: true, IsLiteral: true})

	if err := p.OnGroupSplit(); !errors.Is(err, ErrInvalidSplit) {
		t.Errorf("conditional split: unexpected error: %v", err)
	}
}

func TestParserWeightChecks(t *testing.T) {
	t.Parallel()

	p := &parser{}
	p.stack.Push(&state{IsGroup: true, IsLiteral: true})

	if err := p.SetWeight(0); !errors.Is(err, ErrBadWeight) {
		t.Errorf("weight outside weighted group: unexpected error: %v", err)
	}

	if err := p.SetWeight(-1); !errors.Is(err, ErrBadWeight) {
		t.Errorf("negative weight: unexpected error: %v", err)
	}

	p = &parser{}
	p.stack.Push(&state{IsGroup: true, IsWeighted: true, IsLiteral: true, branchWeight: 1})

	if err := p.SetWeight(0); !errors.Is(err, ErrBadWeight) {
		t.Errorf("zero weight: unexpected error: %v", err)
	}

	if err := p.SetWeight(maxWeight + 1); !errors.Is(err, ErrBadWeight) {
		t.Errorf("huge weight: unexpected error: %v", err)
	}

	if err := p.SetWeight(2); err != nil {
		t.Errorf("valid weight: unexpected error: %v", err)
	}
}
