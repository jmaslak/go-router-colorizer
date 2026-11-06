// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// benchCorpus is every captured device output in testdata, concatenated, so
// that a measurement covers all the vendors rather than only Arista.
func benchCorpus(tb testing.TB) string {
	tb.Helper()
	inputs, err := filepath.Glob(filepath.Join("testdata", "*.input"))
	if err != nil || len(inputs) == 0 {
		tb.Fatalf("no testdata: %v", err)
	}
	var b strings.Builder
	for _, input := range inputs {
		data, err := os.ReadFile(input)
		if err != nil {
			tb.Fatal(err)
		}
		b.Write(data)
		b.WriteString("\n")
	}
	return b.String()
}

func BenchmarkFormatTextCorpus(b *testing.B) {
	corpus := benchCorpus(b)
	b.SetBytes(int64(len(corpus)))
	b.ReportAllocs()
	for b.Loop() {
		FormatText(corpus)
	}
}

// BenchmarkFormatLine breaks the cost down by the kind of line, since the
// rules bail out at different points for each.
func BenchmarkFormatLine(b *testing.B) {
	cases := []struct{ name, line string }{
		{"plain", "This line matches nothing at all in any rule table.\n"},
		{"blank", "\n"},
		{"interface", "Ethernet1 is up, line protocol is up (connected)\n"},
		{"counters", "     0 input errors, 0 CRC, 0 alignment, 0 symbol, 0 input discards\n"},
		{"ipv4", "  Internet address is 192.0.2.1/24, broadcast 192.0.2.255\n"},
		{"ipv6", "  IPv6 address is 2001:db8:1234:5678::1/64\n"},
		{"numbers", "     1234567890 packets input, 9876543210 bytes\n"},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.SetBytes(int64(len(c.line)))
			b.ReportAllocs()
			for b.Loop() {
				FormatText(c.line)
			}
		})
	}
}
