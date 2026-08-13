// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import "strings"

// ANSI escape sequences used for foreground colorization. They assume a
// terminal with a dark background.
const (
	green  = "\x1b[32m"   // "good" values
	red    = "\x1b[1;31m" // error conditions (bold red)
	orange = "\x1b[33m"   // neither good nor bad
	info   = "\x1b[36m"   // potentially important (cyan)
	reset  = "\x1b[0m"    // return to the terminal default

	underlineOn  = "\x1b[4m"
	underlineOff = "\x1b[24m"
)

// bgColors holds the color pairs used to highlight IP addresses. An address is
// hashed onto one of these entries, so a given address always renders in the
// same color and a transposed digit shows up as a color change.
//
// The number of entries matters: changing it changes every address's color.
var bgColors = []string{
	"\x1b[30m\x1b[47m",  // black on white
	"\x1b[35m\x1b[47m",  // magenta on white
	"\x1b[90m\x1b[47m",  // gray on white
	"\x1b[30m\x1b[41m",  // black on red
	"\x1b[90m\x1b[41m",  // gray on red
	"\x1b[37m\x1b[41m",  // white on red
	"\x1b[30m\x1b[42m",  // black on green
	"\x1b[30m\x1b[43m",  // black on yellow (orange)
	"\x1b[31m\x1b[43m",  // red on yellow (orange)
	"\x1b[37m\x1b[44m",  // white on blue
	"\x1b[30m\x1b[45m",  // black on magenta
	"\x1b[37m\x1b[45m",  // white on magenta
	"\x1b[30m\x1b[46m",  // black on cyan
	"\x1b[30m\x1b[100m", // black on gray
	"\x1b[97m\x1b[100m", // white on gray
}

// colorize renders text in color.
//
// Any color already present inside text ends with a reset, which would return
// the remainder of text to the terminal default; the first such reset is
// rewritten to color so that the surrounding colorization survives a nested
// one. Only the first is rewritten, matching the Perl original.
func colorize(text, color string) string {
	return color + strings.Replace(text, reset, color, 1) + reset
}
