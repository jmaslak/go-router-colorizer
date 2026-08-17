// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"fmt"
	"sort"
	"strings"
)

// A color names the semantic role a piece of text is colorized with. The rule
// tables are built once, at package initialization, so they store roles rather
// than escape sequences; a role is looked up in the active palette each time
// text is painted, which is what lets SetPalette work after the rules exist.
type color string

// The roles are named after the colors the default palette gives them, which
// is what the rule tables have always called them.
const (
	green  color = "green"  // "good" values
	red    color = "red"    // error conditions
	orange color = "orange" // neither good nor bad
	info   color = "info"   // potentially important
)

const (
	reset = "\x1b[0m" // return to the terminal default

	underlineOn  = "\x1b[4m"
	underlineOff = "\x1b[24m"
)

type palette struct {
	fg map[color]string

	// ip holds the color pairs used to highlight IP addresses.
	//
	// The number of entries matters: an odd number of IP address colors is
	// highly recommended, as otherwise not all colors will apepar for bare
	// IP addresses.
	ip []string
}

var defaultPalette = palette{
	fg: map[color]string{
		green:  "\x1b[32m",
		red:    "\x1b[1;31m", // bold red
		orange: "\x1b[33m",   // rendered yellow on most terminals
		info:   "\x1b[36m",   // cyan
	},
	ip: []string{
		"\x1b[30m\x1b[47m",  // black on white
		"\x1b[35m\x1b[47m",  // magenta on white
		"\x1b[90m\x1b[47m",  // gray on white
		"\x1b[30m\x1b[41m",  // black on red
		"\x1b[93m\x1b[41m",  // gray on red
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
	},
}

// deuteranopiaPalette avoids leaning on the red/green axis. "Good" moves from
// green to bright blue, so the good/error contrast becomes blue versus red
// rather than green versus red; error keeps its bold for a cue that is not a
// color at all. The IP backgrounds drop green and red in favor of blue, cyan,
// yellow, magenta, white, and gray, which stay mutually distinguishable with
// deuteranopia (and largely with protanopia as well).
var deuteranopiaPalette = palette{
	fg: map[color]string{
		green:  "\x1b[94m",   // bright blue
		red:    "\x1b[1;31m", // bold red
		orange: "\x1b[33m",   // yellow
		info:   "\x1b[36m",   // cyan
	},
	ip: []string{
		"\x1b[30m\x1b[47m",  // black on white
		"\x1b[34m\x1b[47m",  // blue on white
		"\x1b[35m\x1b[47m",  // magenta on white
		"\x1b[37m\x1b[44m",  // white on blue
		"\x1b[93m\x1b[44m",  // bright yellow on blue
		"\x1b[96m\x1b[44m",  // bright cyan on blue
		"\x1b[30m\x1b[46m",  // black on cyan
		"\x1b[34m\x1b[46m",  // blue on cyan
		"\x1b[30m\x1b[43m",  // black on yellow
		"\x1b[34m\x1b[43m",  // blue on yellow
		"\x1b[30m\x1b[45m",  // black on magenta
		"\x1b[37m\x1b[45m",  // white on magenta
		"\x1b[93m\x1b[45m",  // bright yellow on magenta
		"\x1b[30m\x1b[100m", // black on gray
		"\x1b[97m\x1b[100m", // white on gray
	},
}

// palettes holds the palettes SetPalette can select.
var palettes = map[string]palette{
	"default":      defaultPalette,
	"deuteranopia": deuteranopiaPalette,
}

// active is the palette all colorization reads from.
var active = defaultPalette

// SetPalette selects the named palette for all subsequent colorization and
// reports an error naming the valid choices if there is no such palette.
//
// It replaces what every later paint reads, so it must be called before
// formatting starts: it is not safe to call concurrently with FormatText,
// FormatTextParallel, or a running Filter.
func SetPalette(name string) error {
	p, ok := palettes[name]
	if !ok {
		names := make([]string, 0, len(palettes))
		for n := range palettes {
			names = append(names, n)
		}
		sort.Strings(names)
		return fmt.Errorf("unknown palette %q (valid palettes: %s)", name, strings.Join(names, ", "))
	}
	active = p
	Green, Cyan, Yellow = p.fg[green], p.fg[info], p.fg[orange]
	return nil
}

// Exported names for the active palette's escape sequences, used by
// cmd/router-colorizer for its own messages.
var (
	Green  = defaultPalette.fg[green]
	Cyan   = defaultPalette.fg[info]
	Yellow = defaultPalette.fg[orange]
)

const Reset = reset

// colorize renders text in the active palette's escape for c.
func colorize(text string, c color) string {
	return colorizeEscape(text, active.fg[c])
}

// colorizeEscape renders text in color, given as a raw escape sequence.
func colorizeEscape(text, color string) string {
	return color + strings.Replace(text, reset, color, 1) + reset
}
