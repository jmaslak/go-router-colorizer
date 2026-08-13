// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"strings"
	"testing"
)

func TestNextLine(t *testing.T) {
	tests := []struct {
		in    string
		lines []string
	}{
		{"", nil},
		{"abc", []string{"abc"}},
		{"a\nb\n", []string{"a\n", "b\n"}},
		{"a\r\nb", []string{"a\r\n", "b"}},
		{"a\rb", []string{"a\r", "b"}},
		// A carriage return ends a line even without a line feed, because a
		// device uses one to draw over the line it just wrote.
		{"a\r\r\n", []string{"a\r", "\r\n"}},
		{"\n\r", []string{"\n", "\r"}},
	}
	for _, tt := range tests {
		var got []string
		for rest := tt.in; rest != ""; {
			var line string
			line, rest = nextLine(rest)
			got = append(got, line)
		}
		if strings.Join(got, "|") != strings.Join(tt.lines, "|") {
			t.Errorf("splitting %q gave %q, want %q", tt.in, got, tt.lines)
		}
	}
}

func TestSplitPreamble(t *testing.T) {
	tests := []struct {
		in       string
		preamble string
		rest     string
	}{
		{"plain text", "", "plain text"},
		{"\x1b[Ktext", "\x1b[K", "text"},
		{"\x1bMtext", "\x1bM", "text"},
		// The last control sequence in the line ends the preamble.
		{"\x1b[Kone\x1b[Ktwo", "\x1b[Kone\x1b[K", "two"},
		// The erase that belongs to the pager prompt is not a preamble: it
		// is removed with the prompt instead.
		{"\x1b[23m\x1b[Ktext", "", "\x1b[23m\x1b[Ktext"},
		{"\x1b[Ka\x1b[23m\x1b[Kb", "\x1b[K", "a\x1b[23m\x1b[Kb"},
	}
	for _, tt := range tests {
		preamble, rest := splitPreamble(tt.in)
		if preamble != tt.preamble || rest != tt.rest {
			t.Errorf("splitPreamble(%q) = (%q, %q), want (%q, %q)",
				tt.in, preamble, rest, tt.preamble, tt.rest)
		}
	}
}

func TestFormatTextPreservesStructure(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty input", "", ""},
		{"unrecognized text is untouched", "hello, world\n", "hello, world\n"},
		{
			"line endings survive colorization",
			"Status: Ready\r\n",
			green + "Status: Ready" + reset + "\r\n",
		},
		{
			"a line without an ending is still colorized",
			"Status: Ready",
			green + "Status: Ready" + reset,
		},
		{
			"cursor control is left outside the color",
			"\x1b[KStatus: Ready\n",
			"\x1b[K" + green + "Status: Ready" + reset + "\n",
		},
		{
			"the pager prompt is removed",
			"\x1b[3m --More-- \x1b[23m\x1b[K\r\x1b[KStatus: Ready\n",
			green + "Status: Ready" + reset + "\n",
		},
		{
			"a space backspaced over draws nothing",
			"Status: Ready \x08\n",
			green + "Status: Ready" + reset + "\n",
		},
		{
			"each line is colorized on its own",
			"Status: Ready\nStatus: Busy\n",
			green + "Status: Ready" + reset + "\n" + info + "Status: Busy" + reset + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatText(tt.in); got != tt.want {
				t.Errorf("FormatText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// The first rule to recognize a line wins, which is what lets a specific rule
// be written ahead of a general one.
func TestFirstRuleWins(t *testing.T) {
	tests := []struct {
		in    string
		color string
	}{
		{"Status: Ready", green},
		{"Status: anything else", info},
		{"  BGP state is Established", green},
		{"  BGP state is Idle", red},
		{"  BGP state = Established, up for 1w2d", green},
		{"  BGP state = Idle", red},
	}
	for _, tt := range tests {
		want := colorize(tt.in, tt.color)
		if got := FormatText(tt.in); got != want {
			t.Errorf("FormatText(%q) = %q, want %q", tt.in, got, want)
		}
	}
}

// Interface counter lines are judged as a whole: a nonzero counter makes the
// whole line an error, and some counters do not count at all.
func TestInterfaceCounters(t *testing.T) {
	tests := []struct {
		in    string
		color string
	}{
		{"     0 input errors, 0 CRC, 0 alignment, 0 symbol", green},
		{"     0 input errors, 7 CRC, 0 alignment, 0 symbol", red},
		{"     3 PAUSE input, 0 CRC", info},
		{"     0 packets input, 1024 bytes", ""}, // a counter, not a fault
	}
	for _, tt := range tests {
		want := tt.in
		if tt.color != "" {
			want = colorize(tt.in, tt.color)
		}
		got := colorizeInterfaceCounters(tt.in)
		if got != want {
			t.Errorf("colorizeInterfaceCounters(%q) = %q, want %q", tt.in, got, want)
		}
	}

	// A line of "show users" output looks like counters but is not.
	tty := "   16 VTY 0 input errors, 3 CRC"
	if got := colorizeInterfaceCounters(tty); got != tty {
		t.Errorf("colorizeInterfaceCounters(%q) = %q, want it unchanged", tty, got)
	}
}

func TestColorizeNested(t *testing.T) {
	// Colorizing text that is already colorized has to restore the outer
	// color where the inner one ends, or the rest of the line goes plain.
	inner := colorize("value", green)
	got := colorize("field "+inner+" trailing", red)
	want := red + "field " + green + "value" + red + " trailing" + reset
	if got != want {
		t.Errorf("colorize(%q, red) = %q, want %q", "field "+inner+" trailing", got, want)
	}
}
