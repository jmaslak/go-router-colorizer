// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGolden checks each line of captured device output against the
// colorization recorded for it. The cases are shared with the Perl
// implementation this package was ported from, one line per test, so that a
// difference points at the line that changed rather than at the file.
func TestGolden(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("testdata", "*.input"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("no test cases found in testdata")
	}

	for _, input := range inputs {
		name := strings.TrimSuffix(filepath.Base(input), ".input")
		t.Run(name, func(t *testing.T) {
			in := readLines(t, input)
			want := readLines(t, strings.TrimSuffix(input, ".input")+".output")
			if len(in) != len(want) {
				t.Fatalf("%s has %d lines, but its expected output has %d",
					name, len(in), len(want))
			}
			for i := range in {
				if got := FormatText(in[i]); got != want[i] {
					t.Errorf("line %d: FormatText(%q)\n got %q\nwant %q",
						i+1, in[i], got, want[i])
				}
			}
		})
	}
}

// readLines reads a file as a list of lines with their endings removed, which
// is how the shared test cases are structured.
func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSuffix(string(data), "\n")
	return strings.Split(text, "\n")
}
