// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"strings"
	"testing"
)

// groupDigits collects writeGroupedDigits into a string, which is how the
// grouping is easiest to state expectations about.
func groupDigits(s string) string {
	var b strings.Builder
	writeGroupedDigits(&b, s)
	return b.String()
}

func TestGroupDigits(t *testing.T) {
	u := func(s string) string { return underlineOn + s + underlineOff }

	tests := []struct{ in, want string }{
		{"", ""},
		{"1", "1"},
		{"123", "123"},
		{"1234", "1234"},
		{"12345", u("12") + "345"},
		{"123456", u("123") + "456"},
		{"1234567", "1" + u("234") + "567"},
		{"100000000", "100" + u("000") + "000"},
		{"1234567890", u("1") + "234" + u("567") + "890"},
		{"12345678901234", "12" + u("345") + "678" + u("901") + "234"},
	}
	for _, tt := range tests {
		if got := groupDigits(tt.in); got != tt.want {
			t.Errorf("groupDigits(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestColorizeNumbers(t *testing.T) {
	u := func(s string) string { return underlineOn + s + underlineOff }

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"short numbers are left alone", "Number 333 in a string", "Number 333 in a string"},
		{"a four-digit number is left alone", "Number 3333", "Number 3333"},
		{"only the integer part", "100000001.992933", "100" + u("000") + "001.992933"},
		{"already grouped by commas", "1,000,000", "1,000,000"},
		{"a time is not a number", "12:345678:9", "12:345678:9"},
		{
			"an escape sequence parameter is not a number",
			"\x1b[1034h and \x1b[?1034h",
			"\x1b[1034h and \x1b[?1034h",
		},
		{
			"a number after an escape sequence still is one",
			"\x1b[32m1234567",
			"\x1b[32m1" + u("234") + "567",
		},
		{"several numbers", "1234 and 5678", "1234 and 5678"},
		{"several numbers", "12345 and 67890", u("12") + "345 and " + u("67") + "890"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colorizeNumbers(tt.in); got != tt.want {
				t.Errorf("colorizeNumbers(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
