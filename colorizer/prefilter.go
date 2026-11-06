// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"regexp/syntax"
	"strings"
	"unicode/utf8"
)

// Every line of output is offered to every rule, which is upwards of a hundred
// and fifty patterns, and all but a handful of them fail on the first
// character. Running a regular expression to learn that costs far more than
// the comparison it comes down to, because the engine sets up its state before
// it looks at anything.
//
// A prefilter is that comparison, extracted from the pattern itself so that it
// cannot drift away from what the pattern requires. It is a necessary
// condition for a match, never a sufficient one: a line it accepts is still
// handed to the regular expression, and a line it rejects could not have
// matched. Where a pattern is too open-ended to say anything about, the
// prefilter is empty and accepts everything, which is merely the original
// behavior.

// A byteSet is a set of byte values.
type byteSet [4]uint64

func (s *byteSet) add(b byte) { s[b>>6] |= 1 << (b & 63) }

func (s *byteSet) addRange(lo, hi byte) {
	for b := int(lo); b <= int(hi); b++ {
		s.add(byte(b))
	}
}

func (s *byteSet) has(b byte) bool { return s[b>>6]&(1<<(b&63)) != 0 }

// A prefilter requires a line to begin with lit, to continue with one byte
// from each of classes, and to contain must somewhere. A zero prefilter
// requires nothing.
//
// The first two are the same thing said two ways: a position that admits a
// single byte joins lit, which compares as a string, and one that admits
// several becomes a class. Splitting them keeps the common case -- a pattern
// that opens with a literal phrase -- down to one string comparison.
//
// must catches the rest. A table-shaped pattern opens with something as
// unselective as "|", and what actually distinguishes it -- " Enabled ",
// " peer-group member" -- sits in the middle of the line, where a condition on
// the start of the line can say nothing about it.
type prefilter struct {
	lit     string
	classes []byteSet
	must    string

	// least is the shortest line that can satisfy the three, checked first
	// so that the length tests the others would each make are made once.
	least int
}

// match reports whether line satisfies f, and so whether the pattern f came
// from might match it.
func (f *prefilter) match(line string) bool {
	if len(line) < f.least {
		return false
	}
	if f.lit != "" {
		// Comparing the first byte here rejects most lines outright, and
		// does it without the call that comparing the strings costs.
		if line[0] != f.lit[0] || !strings.HasPrefix(line, f.lit) {
			return false
		}
	}
	rest := line[len(f.lit):]
	for i := range f.classes {
		if !f.classes[i].has(rest[i]) {
			return false
		}
	}
	return strings.Contains(line, f.must)
}

// newPrefilter derives a prefilter from pattern, which must be written in the
// syntax regexp.Compile accepts. A pattern that is not anchored to the start
// of the text, or whose opening is not a fixed number of bytes, yields the
// zero prefilter.
func newPrefilter(pattern string) prefilter {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return prefilter{}
	}

	re = re.Simplify()

	var e extractor
	e.walk(re)
	f := prefilter{must: requiredLiteral(re)}
	if e.anchored {
		f.lit, f.classes = e.lit.String(), e.classes
	}
	// A literal the start of the line already accounts for is not worth
	// searching the whole line for a second time.
	if strings.Contains(f.lit, f.must) {
		f.must = ""
	}
	f.least = max(len(f.lit)+len(f.classes), len(f.must))
	return f
}

// requiredLiteral returns the longest run of literal text that every match of
// re has to contain.
//
// Only the subexpressions a match must pass through are considered: the
// branches of an alternation and the body of a repetition are skipped, because
// a match need not enter either one.
func requiredLiteral(re *syntax.Regexp) string {
	best := ""
	switch re.Op {
	case syntax.OpLiteral:
		if re.Flags&syntax.FoldCase == 0 {
			best = string(re.Rune)
		}
	case syntax.OpCapture:
		best = requiredLiteral(re.Sub[0])
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if s := requiredLiteral(sub); len(s) > len(best) {
				best = s
			}
		}
	}
	return best
}

// An extractor accumulates the leading bytes of a pattern as it walks the
// pattern's syntax tree in match order.
type extractor struct {
	anchored bool
	lit      strings.Builder
	classes  []byteSet
	stopped  bool
}

// literal records that the pattern requires the byte b at the next position.
func (e *extractor) literal(b byte) {
	if len(e.classes) == 0 {
		e.lit.WriteByte(b)
		return
	}
	var s byteSet
	s.add(b)
	e.classes = append(e.classes, s)
}

// class records that the pattern requires a byte from s at the next position.
func (e *extractor) class(s byteSet) {
	e.classes = append(e.classes, s)
}

// walk adds what re requires, stopping at the first subexpression whose length
// or content is not fixed. Everything recorded before that point is still
// required, which is what makes a partial walk useful.
func (e *extractor) walk(re *syntax.Regexp) {
	if e.stopped {
		return
	}
	switch re.Op {
	case syntax.OpBeginText:
		// Only an anchor the pattern opens with says where a match starts.
		if e.lit.Len() == 0 && len(e.classes) == 0 {
			e.anchored = true
		}

	case syntax.OpEndText, syntax.OpEndLine, syntax.OpWordBoundary,
		syntax.OpNoWordBoundary, syntax.OpEmptyMatch:
		// Consumes nothing, so it moves no position.

	case syntax.OpCapture:
		e.walk(re.Sub[0])

	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if e.walk(sub); e.stopped {
				return
			}
		}

	case syntax.OpLiteral:
		if re.Flags&syntax.FoldCase != 0 {
			e.stopped = true
			return
		}
		for _, r := range re.Rune {
			if r > 0x7f {
				// A multi-byte rune, so the positions after it are
				// not where counting bytes would put them.
				e.stopped = true
				return
			}
			e.literal(byte(r))
		}

	case syntax.OpCharClass:
		s, ascii := classBytes(re.Rune)
		e.class(s)
		if !ascii {
			// The class admits a multi-byte rune, so this is the last
			// position that can be counted to.
			e.stopped = true
		}

	case syntax.OpAlternate:
		// Only the first byte is common to every branch.
		var s byteSet
		for _, sub := range re.Sub {
			if !firstBytes(sub, &s) {
				e.stopped = true
				return
			}
		}
		e.class(s)
		e.stopped = true

	case syntax.OpPlus:
		// One repetition is required, so its first byte is required, but
		// how many follow it is not fixed.
		var s byteSet
		if !firstBytes(re.Sub[0], &s) {
			e.stopped = true
			return
		}
		e.class(s)
		e.stopped = true

	default:
		// A repetition, "any character", or anything else that may match
		// a number of bytes that is not fixed.
		e.stopped = true
	}
}

// firstBytes adds to s every byte a match of re can begin with, and reports
// whether it was able to: a subexpression that can match nothing at all, or
// whose first byte is not determined, has no such set.
func firstBytes(re *syntax.Regexp, s *byteSet) bool {
	switch re.Op {
	case syntax.OpCapture:
		return firstBytes(re.Sub[0], s)

	case syntax.OpConcat:
		if len(re.Sub) == 0 {
			return false
		}
		return firstBytes(re.Sub[0], s)

	case syntax.OpLiteral:
		if len(re.Rune) == 0 || re.Flags&syntax.FoldCase != 0 {
			return false
		}
		addRune(s, re.Rune[0])
		return true

	case syntax.OpCharClass:
		class, _ := classBytes(re.Rune)
		for i := range s {
			s[i] |= class[i]
		}
		return true

	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			if !firstBytes(sub, s) {
				return false
			}
		}
		return true

	case syntax.OpPlus:
		return firstBytes(re.Sub[0], s)

	default:
		// Includes everything that can match the empty string, which has
		// no first byte, and so says nothing about the line.
		return false
	}
}

// classBytes returns the bytes a rune in the class can begin with, given the
// class as the pairs of range endpoints a parsed character class holds, and
// reports whether every rune in it is a single byte.
func classBytes(runes []rune) (s byteSet, ascii bool) {
	ascii = true
	for i := 0; i+1 < len(runes); i += 2 {
		lo, hi := runes[i], runes[i+1]
		if lo <= 0x7f {
			s.addRange(byte(lo), byte(min(hi, 0x7f)))
		}
		if hi > 0x7f {
			// Any byte that can lead a multi-byte UTF-8 encoding. This
			// is a superset, which is all a prefilter needs.
			s.addRange(0xc2, 0xf4)
			ascii = false
		}
	}
	return s, ascii
}

// addRune adds the byte r's encoding begins with.
func addRune(s *byteSet, r rune) {
	if r <= 0x7f {
		s.add(byte(r))
		return
	}
	var buf [utf8.UTFMax]byte
	utf8.EncodeRune(buf[:], r)
	s.add(buf[0])
}
