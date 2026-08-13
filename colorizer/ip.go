// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"regexp"
	"strconv"
	"strings"
)

// Address components are recognized by matching a whole run of digits or hex
// digits against these patterns, rather than by letting a regular expression
// pick a shorter prefix of the run. An address is only an address if it is
// followed by something that is not part of one, which is the effect the Perl
// original gets from a trailing negative lookahead.
var (
	ipv4Octet     = regexp.MustCompile(`^(?:25[0-5]|2[0-4][0-9]|[0-1]?[0-9]{1,2})$`)
	ipv4PrefixLen = regexp.MustCompile(`^(?:[12][0-9]|3[0-2]|[0-9])$`)
	ipv6PrefixLen = regexp.MustCompile(`^(?:1[01][0-9]|12[0-8]|[1-9][0-9]|[0-9])$`)
)

// colorizeIPv4 gives every IPv4 address or prefix in line its own background
// color.
func colorizeIPv4(line string) string {
	return colorizeAddresses(line, matchIPv4, hashIPv4)
}

// colorizeIPv6 gives every IPv6 address or prefix in line its own background
// color. Unlike IPv4, a candidate is only considered when it is delimited:
// hex digits look enough like ordinary words that matching them mid-token
// would colorize things that are not addresses.
func colorizeIPv6(line string) string {
	return colorizeAddresses(line, matchIPv6, hashIPv6)
}

// colorizeAddresses walks line left to right, colorizing every address that
// match reports, and returns the result. Text between addresses is untouched.
func colorizeAddresses(line string, match func(string, int) int, hash func(string) uint64) string {
	var b strings.Builder
	last := 0
	for pos := 0; pos < len(line); {
		end := match(line, pos)
		if end <= pos { // no match here, or an empty one
			pos++
			continue
		}
		addr := line[pos:end]
		// Sized on the first address found, so that a line without one --
		// most lines -- costs nothing.
		if b.Cap() == 0 {
			b.Grow(len(line) + 16)
		}
		b.WriteString(line[last:pos])
		b.WriteString(colorize(addr, bgColors[hash(addr)%uint64(len(bgColors))]))
		last, pos = end, end
	}
	if last == 0 {
		return line
	}
	b.WriteString(line[last:])
	return b.String()
}

// matchIPv4 reports the end of the IPv4 address or prefix starting at pos in
// s, or -1 if there is none.
func matchIPv4(s string, pos int) int {
	end := pos
	for octet := range 4 {
		if octet > 0 {
			if end >= len(s) || s[end] != '.' {
				return -1
			}
			end++
		}
		run := digitRun(s, end)
		if run == 0 || !ipv4Octet.MatchString(s[end:end+run]) {
			return -1
		}
		end += run
	}
	end = skipPrefixLen(s, end, ipv4PrefixLen)
	if end < len(s) && isDigit(s[end]) {
		return -1
	}
	return end
}

// matchIPv6 reports the end of the IPv6 address or prefix starting at pos in
// s, or -1 if there is none.
//
// The forms are tried in the order the Perl original's pattern lists them:
// eight groups first, then the "::" forms, by ascending number of groups
// before the "::" and descending number after it.
func matchIPv6(s string, pos int) int {
	// A candidate has to start at the beginning of a token. The Perl original
	// only rejected a preceding hex digit, ":" or "-", which let a candidate
	// start in the middle of a word made of letters: in "IX::PROD::CAB", the
	// "D::CAB" that follows "PRO" looks like an address. Any preceding word
	// character rules the candidate out.
	if pos > 0 {
		if c := s[pos-1]; isWord(c) || c == ':' || c == '-' {
			return -1
		}
	}

	if end, ok := matchIPv6Form(s, pos, 8, -1); ok {
		return end
	}
	for lead := 0; lead <= 6; lead++ {
		for trail := 6 - lead; trail >= 0; trail-- {
			if end, ok := matchIPv6Form(s, pos, lead, trail); ok {
				return end
			}
		}
	}
	return -1
}

// matchIPv6Form matches one address form at pos: lead groups, then a "::" and
// trail more groups, unless trail is negative, in which case there is no "::"
// and lead is the whole address.
func matchIPv6Form(s string, pos, lead, trail int) (int, bool) {
	end := pos
	for i := range lead {
		if i > 0 {
			if end >= len(s) || s[end] != ':' {
				return 0, false
			}
			end++
		}
		run := hexRun(s, end)
		if run == 0 || run > 4 {
			return 0, false
		}
		end += run
	}
	if trail >= 0 {
		if !strings.HasPrefix(s[end:], "::") {
			return 0, false
		}
		end += 2
		for i := range trail {
			if i > 0 {
				if end >= len(s) || s[end] != ':' {
					return 0, false
				}
				end++
			}
			run := hexRun(s, end)
			if run == 0 || run > 4 {
				return 0, false
			}
			end += run
		}
	}

	end = skipPrefixLen(s, end, ipv6PrefixLen)
	// The address has to be followed by something that cannot continue it.
	if end < len(s) {
		if c := s[end]; isWord(c) || c == ':' || c == '.' || c == '/' {
			return 0, false
		}
	}
	return end, true
}

// skipPrefixLen advances past a "/length" suffix at pos, if there is a
// complete and valid one.
func skipPrefixLen(s string, pos int, valid *regexp.Regexp) int {
	if pos >= len(s) || s[pos] != '/' {
		return pos
	}
	run := digitRun(s, pos+1)
	if run == 0 || !valid.MatchString(s[pos+1:pos+1+run]) {
		return pos
	}
	return pos + 1 + run
}

// hashIPv4 hashes an IPv4 address or prefix onto a color. The length is part
// of the hash, so 10.0.0.0/8 and 10.0.0.0/24 get different colors.
func hashIPv4(addr string) uint64 {
	subnet, length := splitPrefix(addr, 32)

	var val uint64
	for octet := range strings.SplitSeq(subnet, ".") {
		n, _ := strconv.ParseUint(octet, 10, 16)
		val = val*256 + n
	}
	return val*256 + uint64(length)
}

// hashIPv6 hashes an IPv6 address or prefix onto a color.
//
// The hash mixes each group with the group in the same position counted from
// the other end of the address, and shifts by only four bits per group, so it
// does not use anything like the full 128 bits of the address. It is kept bit
// for bit compatible with the Perl original so that the two agree on colors;
// the result stays well inside 64 bits because a valid address never has a
// group at both ends of the same position.
func hashIPv6(addr string) uint64 {
	subnet, length := splitPrefix(addr, 128)

	// Groups written before the "::", left aligned, and after it, right
	// aligned. An address with no "::" is entirely leading groups.
	var leading, trailing [8]string
	first, second, contracted := strings.Cut(subnet, "::")
	copy(leading[:], strings.Split(first, ":"))
	if contracted && second != "" {
		groups := strings.Split(second, ":")
		if len(groups) > len(trailing) {
			groups = groups[len(groups)-len(trailing):]
		}
		copy(trailing[len(trailing)-len(groups):], groups)
	}

	var val uint64
	for i := range leading {
		n, _ := strconv.ParseUint(leading[i]+trailing[i], 16, 64)
		val = val*16 + n
	}
	return val*32 + uint64(length)
}

// splitPrefix splits an address into its subnet and prefix length, using
// deflt when no length is written.
func splitPrefix(addr string, deflt int) (subnet string, length int) {
	subnet, lengthStr, found := strings.Cut(addr, "/")
	if !found {
		return subnet, deflt
	}
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return subnet, deflt
	}
	return subnet, length
}

// digitRun returns the length of the run of decimal digits at pos.
func digitRun(s string, pos int) int {
	n := 0
	for pos+n < len(s) && isDigit(s[pos+n]) {
		n++
	}
	return n
}

// hexRun returns the length of the run of hexadecimal digits at pos.
func hexRun(s string, pos int) int {
	n := 0
	for pos+n < len(s) && isHexDigit(s[pos+n]) {
		n++
	}
	return n
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isWord(c byte) bool {
	return isDigit(c) || c == '_' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
