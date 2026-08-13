// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The whole design rests on one property: colorizing the pieces of a text that
// was cut at line endings gives the same bytes as colorizing all of it.
func TestFormatTextParallelMatchesFormatText(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("testdata", "*.input"))
	if err != nil {
		t.Fatal(err)
	}

	for _, input := range inputs {
		data, err := os.ReadFile(input)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		want := FormatText(text)

		name := strings.TrimSuffix(filepath.Base(input), ".input")
		t.Run(name, func(t *testing.T) {
			// Every worker count, including more workers than there is
			// text to divide.
			for _, workers := range []int{0, 1, 2, 3, 7, 64, 4096} {
				if got := FormatTextParallel(text, workers); got != want {
					t.Errorf("FormatTextParallel(%s, %d) differs from FormatText", name, workers)
				}
			}
		})
	}
}

// Repeated text is divided into many small chunks, which puts the boundaries in
// different places than the captured files do.
func TestFormatTextParallelChunkBoundaries(t *testing.T) {
	unit := "Status: Ready\r\nplain\nEthernet1 is up, line protocol is up\n" +
		"1.2.3.4/32 2001:db8::1 1234567\nno-ending-here\r"
	for _, reps := range []int{1, 2, 17, 500} {
		text := strings.Repeat(unit, reps)
		want := FormatText(text)
		for _, workers := range []int{2, 5, 32} {
			if got := FormatTextParallel(text, workers); got != want {
				t.Errorf("%d repetitions, %d workers: output differs from FormatText",
					reps, workers)
			}
		}
	}
}

func TestSplitAtLines(t *testing.T) {
	tests := []struct {
		name string
		text string
		n    int
		want []string
	}{
		{"one worker", "a\nb\n", 1, []string{"a\nb\n"}},
		{"empty", "", 4, []string{""}},
		{"even split", "aaaa\nbbbb\n", 2, []string{"aaaa\n", "bbbb\n"}},
		// A cut lands past the target, at the next line ending.
		{"cut moves to the line ending", "aaaaaaaa\nbb\n", 2, []string{"aaaaaaaa\n", "bb\n"}},
		// Nothing left to cut at: the remainder is one piece.
		{"no line ending", "aaaaaaaaaa", 4, []string{"aaaaaaaaaa"}},
		{"trailing partial line", "aa\nbb\ncc", 3, []string{"aa\n", "bb\n", "cc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAtLines(tt.text, tt.n)
			if strings.Join(got, "") != tt.text {
				t.Errorf("splitAtLines(%q, %d) = %q, which does not rejoin to the input",
					tt.text, tt.n, got)
			}
			if len(got) != len(tt.want) || strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("splitAtLines(%q, %d) = %q, want %q", tt.text, tt.n, got, tt.want)
			}
			for i, chunk := range got[:len(got)-1] {
				if !strings.HasSuffix(chunk, "\n") {
					t.Errorf("chunk %d of %q does not end at a line ending: %q",
						i, tt.text, chunk)
				}
			}
		})
	}
}

// A chunk is never split so small that a goroutine costs more than the work.
func TestFormatTextParallelSkipsSmallInput(t *testing.T) {
	text := strings.Repeat("plain line\n", 8)
	if got := len(splitAtLines(text, 1)); got != 1 {
		t.Errorf("small input was divided into %d chunks, want 1", got)
	}
	if got, want := FormatTextParallel(text, 64), FormatText(text); got != want {
		t.Errorf("FormatTextParallel(%q, 64) = %q, want %q", text, got, want)
	}
}

func BenchmarkFormatTextParallel(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("testdata", "04-arista.input"))
	if err != nil {
		b.Fatal(err)
	}
	// One file is a single screenful of scrollback; a big capture is many.
	text := strings.Repeat(string(data), 8)

	for _, workers := range []int{1, 2, 4, 8, 0} {
		name := "workers=all"
		if workers > 0 {
			name = "workers=" + string(rune('0'+workers))
		}
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			for b.Loop() {
				FormatTextParallel(text, workers)
			}
		})
	}
}
