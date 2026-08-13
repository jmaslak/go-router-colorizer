// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import "strings"

// colorizeNumbers underlines alternating groups of three digits in every run
// of digits in line, so that a long number can be read without counting
// digits.
//
// Runs that are part of something other than a number are left alone: the
// fractional part of a decimal, the fields of an address or a time, and the
// numeric parameters of an ANSI escape sequence.
func colorizeNumbers(line string) string {
	var b strings.Builder
	last := 0
	for pos := 0; pos < len(line); {
		run := digitRun(line, pos)
		if run == 0 {
			pos++
			continue
		}
		end := pos + run
		// Three digits or fewer are never grouped, so a line of short
		// numbers -- most lines -- is returned without being rebuilt.
		if run > 3 && groupable(line, pos, end) {
			if b.Cap() == 0 {
				b.Grow(len(line) + 16)
			}
			b.WriteString(line[last:pos])
			writeGroupedDigits(&b, line[pos:end])
			last = end
		}
		pos = end
	}
	if last == 0 {
		return line
	}
	b.WriteString(line[last:])
	return b.String()
}

// groupable reports whether the run of digits at [start,end) is a number in
// its own right, rather than a continuation of one or a piece of syntax.
func groupable(s string, start, end int) bool {
	if start > 0 {
		switch prev := s[start-1]; prev {
		case ':', '.':
			// A dotted or colon-separated field: an address, a time,
			// or the fractional part of a decimal.
			return false
		}
		// A parameter of an ANSI escape sequence.
		if strings.HasSuffix(s[:start], "\x1b[") || strings.HasSuffix(s[:start], "\x1b[?") {
			return false
		}
	}
	if end < len(s) && s[end] == ':' {
		// The first field of a time or a similar colon-separated value.
		return false
	}
	return true
}

// writeGroupedDigits writes s to b, underlining every other group of three
// digits, counting the groups from the right and leaving the rightmost one
// alone. A number of three digits or fewer is written unchanged.
//
// The leftmost group is short whenever the length is not a multiple of three,
// which is why the groups are indexed from the right.
func writeGroupedDigits(b *strings.Builder, s string) {
	for group := (len(s)+2)/3 - 1; group >= 0; group-- {
		start := max(len(s)-3*(group+1), 0)
		end := len(s) - 3*group
		if group%2 == 0 {
			b.WriteString(s[start:end])
			continue
		}
		b.WriteString(underlineOn)
		b.WriteString(s[start:end])
		b.WriteString(underlineOff)
	}
}
