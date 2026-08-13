// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import "testing"

func TestMatchIPv4(t *testing.T) {
	tests := []struct {
		text string
		want string // the matched address, or "" for no match
	}{
		{"1.2.3.4", "1.2.3.4"},
		{"1.2.3.4/32", "1.2.3.4/32"},
		{"255.255.255.255/0", "255.255.255.255/0"},
		{"010.001.002.003", "010.001.002.003"},
		// A prefix length of 33 or more is not one, so the bare address
		// matches and the "/33" is left as text.
		{"1.2.3.4/33", "1.2.3.4"},
		// Junos writes a session port as "address/port", which must not be
		// mistaken for a prefix length.
		{"192.0.2.2/15851", "192.0.2.2"},
		{"1.2.3.4/", "1.2.3.4"},
		{"1.2.3.4/08", "1.2.3.4"},
		// An octet above 255 is not an octet, and a fifth one cannot be
		// tacked on to make a match.
		{"256.1.1.1", ""},
		{"1.2.3.256", ""},
		{"1.2.3.4567", ""},
		{"1.2.3", ""},
		{"1.2.3.4.5", "1.2.3.4"},
		{"9999", ""},
	}

	for _, tt := range tests {
		end := matchIPv4(tt.text, 0)
		got := ""
		if end >= 0 {
			got = tt.text[:end]
		}
		if got != tt.want {
			t.Errorf("matchIPv4(%q, 0) = %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestMatchIPv6(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"::", "::"},
		{"::/128", "::/128"},
		{"::1", "::1"},
		{"2001:db8::1", "2001:db8::1"},
		{"2001:db8::/32", "2001:db8::/32"},
		{"1:2:3:4:5:6:7:8", "1:2:3:4:5:6:7:8"},
		{"1:2:3:4:5:6:7:8/126", "1:2:3:4:5:6:7:8/126"},
		{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
		// Nine groups is not an address, and no shorter form of it is
		// either, because whatever follows a shorter form is a colon.
		{"1:2:3:4:5:6:7:8:9::", ""},
		// A group of five hex digits cannot be split into a valid one.
		{"12345::1", ""},
		// A prefix length above 128 is not one, and unlike IPv4 the bare
		// address does not match either: a "/" cannot follow an address.
		{"::1/129", ""},
		// The pattern the original was ported from stops at six groups on
		// either side of the "::", which leaves these unmatched.
		{"1:2:3:4:5:6:7::", ""},
		{"::1:2:3:4:5:6:7", ""},
		// An address has to be delimited: hex digits and colons run
		// together too easily to guess.
		{"abc::defg", ""},
		// A zone identifier is not part of the address, but it does not
		// stop the address from being one.
		{"2001:db8::1%eth0", "2001:db8::1"},
	}

	for _, tt := range tests {
		end := matchIPv6(tt.text, 0)
		got := ""
		if end >= 0 {
			got = tt.text[:end]
		}
		if got != tt.want {
			t.Errorf("matchIPv6(%q, 0) = %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestMatchIPv6RequiresDelimiter(t *testing.T) {
	// The character before an address may not look like part of one.
	for _, text := range []string{"a::1", "1::1", "-::1", ":::1", "z::1", "_::1"} {
		if end := matchIPv6(text, 1); end >= 0 {
			t.Errorf("matchIPv6(%q, 1) = %q, want no match", text, text[1:end])
		}
	}
	if end := matchIPv6("x ::1", 2); end != 5 {
		t.Errorf("matchIPv6(%q, 2) = %d, want 5", "x ::1", end)
	}
	// Interface descriptions full of "::" separators are not addresses, even
	// where a suffix of a word happens to be made of hex digits.
	if got := colorizeIPv6("IX::PROD::CAB"); got != "IX::PROD::CAB" {
		t.Errorf("colorizeIPv6(%q) = %q, want it unchanged", "IX::PROD::CAB", got)
	}
}

// TestHashStability pins the address-to-color mapping. These values are shared
// with the Perl implementation, and changing them would recolor every address
// a user has learned to recognize.
func TestHashStability(t *testing.T) {
	v4 := []struct {
		addr string
		want uint64
	}{
		{"1.2.3.4/32", 4328719392},
		{"1.2.3.4/30", 4328719390},
		{"1.2.3.4", 4328719392}, // no length is the same as /32
		{"0.0.0.0/0", 0},
	}
	for _, tt := range v4 {
		if got := hashIPv4(tt.addr); got != tt.want {
			t.Errorf("hashIPv4(%q) = %d, want %d", tt.addr, got, tt.want)
		}
	}

	v6 := []struct {
		addr string
		want uint64
	}{
		{"::", 128},
		{"::/128", 128},
		{"::/8", 8},
		{"::1/128", 160},
		{"1:2:3:4:5:6:7:8", 9773436800},
		{"1:2:3:4:5:6:7:8/126", 9773436798},
	}
	for _, tt := range v6 {
		if got := hashIPv6(tt.addr); got != tt.want {
			t.Errorf("hashIPv6(%q) = %d, want %d", tt.addr, got, tt.want)
		}
	}
}

func TestColorizeAddressesInContext(t *testing.T) {
	tests := []struct{ in, want string }{
		{
			"No ip addresses on this line.",
			"No ip addresses on this line.",
		},
		{
			"peer 1.2.3.4 up",
			"peer " + colorize("1.2.3.4", bgColors[hashIPv4("1.2.3.4")%15]) + " up",
		},
		{
			// Both addresses are colorized, and each keeps its own color.
			"1.2.3.4 -> 1.2.3.5",
			colorize("1.2.3.4", bgColors[hashIPv4("1.2.3.4")%15]) + " -> " +
				colorize("1.2.3.5", bgColors[hashIPv4("1.2.3.5")%15]),
		},
	}
	for _, tt := range tests {
		if got := colorizeIPv4(tt.in); got != tt.want {
			t.Errorf("colorizeIPv4(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// The same address must always get the same color, and a transposition must
// change it: that is the entire point of coloring addresses.
func TestAddressColorsDistinguishTranspositions(t *testing.T) {
	const a, b = "10.20.30.40", "10.20.30.04"
	if colorizeIPv4(a) == colorizeIPv4(b) {
		t.Errorf("%q and %q were colorized identically", a, b)
	}
	if colorizeIPv4("x "+a) != "x "+colorizeIPv4(a) {
		t.Errorf("%q was colorized differently depending on its context", a)
	}
}
