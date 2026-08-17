// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package main

import "testing"

func TestResolvePalette(t *testing.T) {
	tests := []struct {
		name       string
		paletteSet bool
		cb         bool
		want       string
		wantErr    bool
	}{
		{"default", false, false, "default", false},
		{"deuteranopia", true, false, "deuteranopia", false},
		{"default", false, true, "deuteranopia", false},
		// -cb agrees with an explicit -palette deuteranopia.
		{"deuteranopia", true, true, "deuteranopia", false},
		// -cb against a different explicit palette is a contradiction.
		{"default", true, true, "", true},
	}
	for _, tt := range tests {
		got, err := resolvePalette(tt.name, tt.paletteSet, tt.cb)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("resolvePalette(%q, %v, %v) = (%q, %v), want (%q, wantErr=%v)",
				tt.name, tt.paletteSet, tt.cb, got, err, tt.want, tt.wantErr)
		}
	}
}
