// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

// Package colorizer colorizes the output of network device command line
// interfaces.
//
// Text is recognized by the shapes that Arista, Cisco, Juniper, and VyOS
// routers and switches and Ciena WDM devices produce, so it colorizes best
// when fed output from one of those. Anything it does not recognize is passed
// through unchanged, which makes it safe to pipe arbitrary text through.
//
// Three things are colorized:
//
//   - Lines whose meaning is worth calling out. Green is good, orange is
//     neither good nor bad, red is an error, and cyan is merely notable. Most
//     lines are not colorized at all.
//   - IP addresses, each on a colored background chosen by hashing the
//     address, so that a given address always looks the same and a transposed
//     digit shows up as a change in color.
//   - Long runs of digits, which get alternating groups of three underlined so
//     that 1000000 can be told from 10000000 at a glance.
//
// The colors assume a terminal with a dark background.
//
// FormatText colorizes a string, Filter colorizes a stream, and
// FormatTextParallel colorizes a large string across several goroutines. Lines
// are colorized independently of each other, so all three agree byte for byte
// on the same input.
package colorizer

import (
	"regexp"
	"strings"
)

// morePrompt is the pager prompt Arista EOS emits, which is redrawn and erased
// in the middle of the output it is paging.
const morePrompt = "\x1b[3m --More-- \x1b[23m\x1b[K\r\x1b[K"

// Unnecessary backspace removal for some Arista output
var re_unnecessary_bs *regexp.Regexp = regexp.MustCompile(` \x08([^\x08\n\r])`)

// FormatText returns text with ANSI color escapes added.
//
// Input may be any fragment of a stream: text is colorized a line at a time,
// and a trailing partial line is colorized as far as it goes. Line endings,
// including bare carriage returns, are preserved.
func FormatText(text string) string {
	text = strings.ReplaceAll(text, morePrompt, "")

	var b strings.Builder
	b.Grow(len(text) + len(text)/4)
	for len(text) > 0 {
		chunk, rest := nextLine(text)
		b.WriteString(formatLine(chunk))
		text = rest
	}
	return b.String()
}

// nextLine splits off the first line of text: a run of characters that are
// neither carriage return nor line feed, followed by at most one of each. A
// bare carriage return therefore ends a line, because devices use it to redraw
// one.
func nextLine(text string) (line, rest string) {
	end := strings.IndexAny(text, "\r\n")
	if end < 0 {
		return text, ""
	}
	if text[end] == '\r' {
		end++
	}
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return text[:end], text[end:]
}

// formatLine colorizes a single line, which may carry its line ending.
func formatLine(text string) string {
	line, eol := text, ""
	if strings.HasSuffix(line, "\n") {
		line, eol = line[:len(line)-1], "\n"
	}

	// Terminal control sequences at the start and end of the line are set
	// aside rather than colorized: they position the cursor and erase, and
	// wrapping them in a color changes what the terminal draws.
	preamble, body := splitPreamble(line)
	line = trimMorePrompt(body)

	// A space that is immediately backspaced over draws nothing.
	line = re_unnecessary_bs.ReplaceAllString(line, "$1")

	trailer := ""
	if strings.HasSuffix(line, "\r") {
		line, trailer = line[:len(line)-1], "\r"
	}

	// Every vendor's rules see every line: the CLIs overlap, and a line only
	// one vendor emits is only recognized by one vendor's patterns. The order
	// of the families is therefore part of the behavior, not a preference,
	// because the first rule to colorize a line usually shuts out the rest.
	line = applyAll(line, aristaRulesBefore)
	line = colorizeInterfaceCounters(line)
	line = applyAll(line, aristaRulesAfter)
	line = applyAll(line, vyosRules)
	line = applyAll(line, junosRules)
	line = applyAll(line, cienaRules)

	line = colorizeIPv4(line)
	line = colorizeIPv6(line)
	line = colorizeNumbers(line)

	if preamble == "" && trailer == "" && eol == "" {
		return line
	}
	return preamble + line + trailer + eol
}

// splitPreamble splits off the leading terminal control sequences of a line,
// which run through the last "erase to end of line" or "reverse index" escape
// the line contains. An escape that ends italics is not treated as a control
// sequence: it belongs to the pager prompt that trimMorePrompt removes.
func splitPreamble(line string) (preamble, rest string) {
	const italicsOff = "\x1b[23m"

	for i := len(line) - 1; i >= 0; i-- {
		if line[i] != '\x1b' {
			continue
		}
		n := 0
		switch {
		case strings.HasPrefix(line[i:], "\x1b[K"):
			n = len("\x1b[K")
		case strings.HasPrefix(line[i:], "\x1bM"):
			n = len("\x1bM")
		default:
			continue
		}
		if strings.HasSuffix(line[:i], italicsOff) {
			continue
		}
		return line[:i+n], line[i+n:]
	}
	return "", line
}

// trimMorePrompt removes a pager prompt at the start of the line, through the
// last point at which the device redrew the line over it.
func trimMorePrompt(line string) string {
	const prefix = "\x1b[3m --More-- "
	const erase = "\r\x1b[K"

	if !strings.HasPrefix(line, prefix) {
		return line
	}
	end := strings.LastIndex(line, erase)
	if end < len(prefix) {
		return line
	}
	return line[end+len(erase):]
}
