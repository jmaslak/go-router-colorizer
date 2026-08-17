// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"fmt"
	"regexp"
	"strings"
)

// A rule recognizes one shape of router output and colorizes part of it.
//
// Rules are applied to a line in order, and each one rewrites at most the
// first match of its pattern. Order is significant in two ways:
//
//   - A rule that has colorized a line leaves an escape sequence at the start
//     of the colorized text, which stops later, more general rules whose
//     patterns are anchored at the start of the line from matching. This is
//     how "first rule wins" is expressed: the specific rule ("Status: Ready"
//     is good) precedes the general one ("Status: anything" is informational).
//   - Rules that colorize different columns of the same line do stack, because
//     each one only rewrites the columns it captured.
type rule struct {
	re     *regexp.Regexp
	paints []paint
}

// A paint renders one submatch of a rule's pattern. Matched text that no paint
// refers to is dropped from the output, which a handful of rules rely on to
// trim trailing whitespace.
type paint struct {
	group int   // 1-based submatch index
	color color // color role, or "" to emit the submatch unchanged
}

// whole builds a rule that colorizes submatch 1 of pattern.
func whole(pattern string, c color) rule {
	return rule{re: mustCompile(pattern, 1), paints: []paint{{1, c}}}
}

// group builds a rule that colorizes each submatch of pattern with the
// corresponding color, where an empty color emits that submatch unchanged.
// One color per submatch must be given.
func group(pattern string, colors ...color) rule {
	re := mustCompile(pattern, len(colors))
	paints := make([]paint, len(colors))
	for i, c := range colors {
		paints[i] = paint{i + 1, c}
	}
	return rule{re: re, paints: paints}
}

// alternate builds a rule for the table-shaped output of the Ciena CLI, whose
// patterns capture alternating column separators and column bodies: the
// odd-numbered submatches (the "|" separators) are emitted unchanged and the
// even-numbered ones (the column bodies) are colorized.
func alternate(pattern string, body color) rule {
	re := regexp.MustCompile(pattern)
	paints := make([]paint, re.NumSubexp())
	for i := range paints {
		c := color("")
		if i%2 == 1 { // 0-based index 1 is submatch 2
			c = body
		}
		paints[i] = paint{i + 1, c}
	}
	return rule{re: re, paints: paints}
}

// mustCompile compiles pattern and panics unless it has exactly want
// submatches, which guards the rule tables against a stray capture group
// silently shifting every later submatch index.
func mustCompile(pattern string, want int) *regexp.Regexp {
	re := regexp.MustCompile(pattern)
	if got := re.NumSubexp(); got != want {
		panic(fmt.Sprintf("colorizer: pattern %q has %d submatches, want %d", pattern, got, want))
	}
	return re
}

// apply rewrites the first match of r in line. Lines that do not match are
// returned unchanged.
func (r rule) apply(line string) string {
	m := r.re.FindStringSubmatchIndex(line)
	if m == nil {
		return line
	}

	var b strings.Builder
	b.Grow(len(line) + 16)
	b.WriteString(line[:m[0]])
	for _, p := range r.paints {
		start, end := m[2*p.group], m[2*p.group+1]
		if start < 0 { // submatch did not participate in the match
			continue
		}
		if p.color == "" {
			b.WriteString(line[start:end])
		} else {
			b.WriteString(colorize(line[start:end], p.color))
		}
	}
	b.WriteString(line[m[1]:])
	return b.String()
}

// applyAll runs every rule against line, in order.
func applyAll(line string, rules []rule) string {
	for _, r := range rules {
		line = r.apply(line)
	}
	return line
}
