[![PkgGoDev](https://pkg.go.dev/badge/github.com/s0rg/fantasyname)](https://pkg.go.dev/github.com/s0rg/fantasyname)
[![License](https://img.shields.io/github/license/s0rg/fantasyname)](https://github.com/s0rg/fantasyname/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/s0rg/fantasyname)](go.mod)
[![Tag](https://img.shields.io/github/v/tag/s0rg/fantasyname?sort=semver)](https://github.com/s0rg/fantasyname/tags)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

[![CI](https://github.com/s0rg/fantasyname/workflows/ci/badge.svg)](https://github.com/s0rg/fantasyname/actions?query=workflow%3Aci)
[![Go Report Card](https://goreportcard.com/badge/github.com/s0rg/fantasyname)](https://goreportcard.com/report/github.com/s0rg/fantasyname)
[![Maintainability](https://qlty.sh/badges/09a35bec-e65e-4587-a845-4ba6982fd69f/maintainability.svg)](https://qlty.sh/gh/s0rg/projects/fantasyname)
[![Code Coverage](https://qlty.sh/badges/09a35bec-e65e-4587-a845-4ba6982fd69f/test_coverage.svg)](https://qlty.sh/gh/s0rg/projects/fantasyname)
![Issues](https://img.shields.io/github/issues/s0rg/fantasyname)

# fantasyname

This is a golang implementation of [name generator described at RinkWorks](http://rinkworks.com/namegen/),
its based on [https://github.com/skeeto/fantasyname](https://github.com/skeeto/fantasyname) code.

# example

How it looks like:
```go
import (
    "fmt"
    "log"
    "time"
    "math/rand"

    fn "github.com/s0rg/fantasyname"
)

func main() {
    rand.Seed(time.Now().UnixNano())

    gen, err := fn.Compile("sV'i", fn.Collapse(true), fn.RandFn(rand.Intn))
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(gen.String()) // will print something like: entheu'loaf
}
```

[Here](https://github.com/s0rg/fantasyname/blob/master/_example/main.go) is a full example.

You can run it with `go run _example/main.go` to see results.

# ready-to-use cli app

Created by [TLINDEN](https://github.com/TLINDEN/): [gfn](https://github.com/TLINDEN/gfn)

# pattern syntax

The letters `s`, `v`, `V`, `c`, `B`, `C`, `i`, `m`, `M`, `D`, and `d` represent different types of random replacements:

 - `s` - generic syllable
 - `v` - vowel
 - `V` - vowel or vowel combination
 - `c` - consonant
 - `B` - consonant or consonant combination suitable for beginning a word
 - `C` - consonant or consonant combination suitable anywhere in a word
 - `i` - insult
 - `m` - mushy name
 - `M` - mushy name ending
 - `D` - consonant suited for a stupid person's name
 - `d` - syllable suited for a stupid person's name (begins with a vowel)

Everything else is emitted literally.

All characters between parenthesis `()` are emitted literally. For example, the pattern `s(dim)`,
emits a random generic syllable followed by `dim`.

Characters between angle brackets `<>` emit patterns from the table above.
Imagine the entire pattern is wrapped in one of these.

In both types of groupings, a vertical bar `|` denotes a random choice. Empty groups are allowed.
For example, `(foo|bar)` emits either `foo` or `bar`. The pattern `<c|v|>` emits a constant, vowel,
or nothing at all.

An exclamation point `!` means to capitalize the component that follows it. For example,
`!(foo)` will emit `Foo` and `v!s` will emit a lowercase vowel followed by a capitalized syllable, like `eRod`.

# weights, conditions and fragments

The pattern language can be extended with three constructs. A pattern that only
uses the classic syntax above compiles exactly as before; the new characters are
recognized only where a construct actually starts.

## weighted branches

Curly braces `{` `}` group alternatives together with an optional positive
integer weight before each `weight:` branch. Branches without a weight count as
`1`. The generator picks one branch with probability proportional to its weight.
Weights are local to their group, groups may be nested, and the usual groups,
conditionals and dictionary symbols can be used inside a branch.

```
{3:(Al)|1:(El)}          # "Al" three times as often as "El"
{9:(Smith|Jones)|1:(Dragonslayer)}   # a rare heroic surname
{2:<s|cvc>|1:(x)}        # nested classic alternatives
```

A weight of zero, a negative weight, or a weight placed anywhere other than the
start of a weighted branch is reported as `ErrBadWeight`. Empty branches still
count as one alternative with weight `1`.

## conditional fragments

Square brackets `[` `]` enclose text that is expanded only when a label is
enabled. The construct is `[label:pattern]`, or `[!label:pattern]` for the
inverse (expanded only when the label is *not* enabled). Labels are passed to
[Compile] with the `Labels` option; a conditional whose label is not enabled
expands to nothing. Unlike `(a|b)`, the parts of a conditional are always
emitted in order - a conditional never chooses between alternatives, but it may
contain groups that do. Conditional blocks may be nested and combined with
weighted groups, dictionary symbols and fragments.

```go
gen, _ := fn.Compile("(Sir )[hero:!s](of )[hero:(the )]",
    fn.Labels("hero"), fn.RandFn(rand.Intn))

gen, _ = fn.Compile("[!hero:(the lowborn )](x)", fn.RandFn(rand.Intn))
```

By default an unknown label simply disables its block. Pass `StrictLabels(true)`
to make `[unknown:...]` a compile-time `ErrUnknownLabel` instead.

## named fragments

A `$name` reference inlines a pattern registered with the `Fragments` option.
Fragments may reference other fragments, which is useful for composing
surnames, titles and similar reusable pieces.

```go
gen, _ := fn.Compile(
    "[hero:$title]$surname",
    fn.Labels("hero"),
    fn.Fragments(map[string]string{
        "title":   "{4:(Ser )|1:(the Bold )}",
        "surname": "{9:(Smith|Jones)|1:(Wyrmbane)}",
    }),
)
```

Fragments that refer to each other in a cycle are rejected with
`ErrRecursiveCycle`. Empty or whitespace-only fragments expand to the empty
string.

## escaping

A backslash makes the next character literal, which is how to emit one of the
syntax characters themselves: `\{`, `\}`, `\[`, `\]`, `\$`, `\\`, `\|`, `\<`,
`\>`, `\(`, `\)`, `\!`, `\~` and `\:`. Any other escape is reported as
`ErrBadEscape`. A `$` without a name (for example `x$` or `x$1y`) is emitted
literally, so such patterns keep working; use `\$` to be explicit.

## limits and compatibility

Compilation is bounded: nesting deeper than 64 levels (`MaxNesting`), fragment
chains longer than 32 (`MaxFragmentDepth`) and more than 4096 total fragment
expansions (`MaxExpansions`) fail with `ErrTooDeep` instead of exhausting
memory. All syntax errors carry a byte offset, returned by `ErrorPosition(err)`,
and wrap a sentinel error (`ErrBadWeight`, `ErrMissingLabel`, `ErrBadLabel`,
`ErrUnknownLabel`, `ErrBadEscape`, `ErrUnknownFragment`, `ErrRecursiveCycle`,
`ErrTooDeep`), so callers can use `errors.Is`.

If you compile patterns that may contain `{`, `[`, `$` or `\` as plain text - for
example patterns stored before the extension existed - pass `Legacy(true)` to
disable the new constructs and reproduce the original behavior exactly.
