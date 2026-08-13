// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzFormatText checks the properties that hold for any input at all.
// Colorization is inserted into text nobody has validated -- it comes off a
// device, possibly mid-escape-sequence -- so it has to survive anything.
func FuzzFormatText(f *testing.F) {
	f.Add("")
	f.Add("Status: Ready\n")
	f.Add("| Admin State | Enabled | \n")
	f.Add("Ethernet1 is up, line protocol is up\r\n")
	f.Add("1.2.3.4/32 2001:db8::1/128 1234567\n")
	f.Add("\x1b[3m --More-- \x1b[23m\x1b[K\r\x1b[Kdata\n")
	f.Add("\x1b[Kfoo\r\x1b[0m\x1b[32m\x08 \n\n\r")

	f.Fuzz(func(t *testing.T, in string) {
		out := FormatText(in)

		// Line structure is what a terminal lays the text out by, and
		// colorization must not change it.
		if got, want := strings.Count(out, "\n"), strings.Count(in, "\n"); got != want {
			t.Errorf("FormatText(%q) = %q: %d line feeds, want %d", in, out, got, want)
		}

		// Colorization only ever adds escape sequences, so the output
		// cannot be shorter than the input, except for the three things
		// that are deliberately dropped.
		dropped := len(morePrompt)*strings.Count(in, morePrompt) +
			2*strings.Count(in, " \x08")
		if len(out)+dropped < len(in) {
			t.Errorf("FormatText(%q) = %q: lost %d bytes",
				in, out, len(in)-len(out)-dropped)
		}

		if utf8.ValidString(in) && !utf8.ValidString(out) {
			t.Errorf("FormatText(%q) = %q: valid input became invalid UTF-8", in, out)
		}

		// Dividing the work must not change the result. Force a division
		// the input is too small to earn, so that the chunk boundaries land
		// in every position the fuzzer can reach.
		if divided := joinFormatted(splitAtLines(in, 3)); divided != out {
			t.Errorf("FormatText(%q) = %q, but colorizing it in three pieces gave %q",
				in, out, divided)
		}
		if got := FormatTextParallel(in, 4); got != out {
			t.Errorf("FormatTextParallel(%q, 4) = %q, want %q", in, got, out)
		}
	})
}

// joinFormatted colorizes each piece on its own and joins the results, which
// is what FormatTextParallel does without the goroutines.
func joinFormatted(chunks []string) string {
	var b strings.Builder
	for _, chunk := range chunks {
		b.WriteString(FormatText(chunk))
	}
	return b.String()
}

// BenchmarkFormatText measures a full screen of device output, the unit of
// work that matters when a command dumps thousands of lines at once.
func BenchmarkFormatText(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "04-arista.input"))
	if err != nil {
		b.Fatal(err)
	}
	text := string(data)

	b.SetBytes(int64(len(text)))
	for b.Loop() {
		FormatText(text)
	}
}
