// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"strings"
	"testing"
)

// setPalette switches the palette for one test and restores the default (and
// the exported escapes) when the test ends.
func setPalette(t *testing.T, name string) {
	t.Helper()
	if err := SetPalette(name); err != nil {
		t.Fatalf("SetPalette(%q): %v", name, err)
	}
	t.Cleanup(func() {
		if err := SetPalette("default"); err != nil {
			t.Fatalf("SetPalette(%q): %v", "default", err)
		}
	})
}

func TestSetPaletteUnknown(t *testing.T) {
	err := SetPalette("no-such-palette")
	if err == nil {
		t.Fatal("SetPalette of an unknown name did not fail")
	}
	// The error is how the -palette flag reports its valid values, so it has
	// to name them.
	for _, name := range []string{"default", "deuteranopia"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("SetPalette error %q does not mention palette %q", err, name)
		}
	}
	if FormatText("Status: Ready") != colorize("Status: Ready", green) {
		t.Error("a failed SetPalette changed the active palette")
	}
}

func TestSetPaletteChangesRuleColors(t *testing.T) {
	const line = "Status: Ready"
	before := FormatText(line)

	setPalette(t, "deuteranopia")
	after := FormatText(line)

	if after != colorize(line, green) {
		t.Errorf("FormatText(%q) = %q, want the whole line in the good color", line, after)
	}
	if after == before {
		t.Errorf("FormatText(%q) = %q under both palettes, want the good color to change", line, after)
	}
}

func TestSetPaletteChangesIPColors(t *testing.T) {
	const addr = "1.2.3.4"
	before := FormatText(addr)

	setPalette(t, "deuteranopia")
	after := FormatText(addr)

	if want := colorizeEscape(addr, active.ip[hashIPv4(addr)%uint64(len(active.ip))]); after != want {
		t.Errorf("FormatText(%q) = %q, want %q", addr, after, want)
	}
	if after == before {
		t.Errorf("FormatText(%q) = %q under both palettes, want %q's slot to change color", addr, after, addr)
	}
}

func TestSetPaletteUpdatesExportedEscapes(t *testing.T) {
	setPalette(t, "deuteranopia")
	if Green != active.fg[green] || Cyan != active.fg[info] || Yellow != active.fg[orange] {
		t.Error("SetPalette left Green, Cyan, or Yellow on the previous palette")
	}
}

// A palette may define fewer IP colors than the default; addresses spread
// across however many it has.
func TestSmallIPPalette(t *testing.T) {
	small := defaultPalette
	small.ip = []string{
		"\x1b[37m\x1b[44m", // white on blue
		"\x1b[30m\x1b[47m", // black on white
		"\x1b[30m\x1b[46m", // black on cyan
	}
	palettes["test-small"] = small
	t.Cleanup(func() { delete(palettes, "test-small") })
	setPalette(t, "test-small")

	// Consecutive addresses to land on both entries.
	for _, addr := range []string{"1.2.3.4", "1.2.3.5"} {
		if got, want := FormatText(addr), ipColor(addr); got != want {
			t.Errorf("FormatText(%q) = %q, want %q", addr, got, want)
		}
	}
	if ipColor("1.2.3.4") == ipColor("1.2.3.5") {
		t.Error("adjacent addresses landed on the same entry, so the test exercised only one")
	}
}

func TestPalettesAreComplete(t *testing.T) {
	for name, p := range palettes {
		for _, c := range []color{green, red, orange, info} {
			if p.fg[c] == "" {
				t.Errorf("palette %q has no escape for %q", name, c)
			}
		}
		if len(p.ip) == 0 {
			t.Errorf("palette %q has no IP colors", name)
		}
	}
	// We need an odd number of IP colors due to how we build the hash of
	// IP numbers. If an even number is used, it's possible that not all colors
	// will be used for bare IPs.
	if len(defaultPalette.ip)%2 != 1 {
		t.Errorf("the default palette has %d IP colors, want an odd number",
			len(defaultPalette.ip))
	}
}
