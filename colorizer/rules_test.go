// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import "testing"

// colorizedWith reports whether FormatText painted the whole line in c.
func colorizedWith(line string, c color) bool {
	return FormatText(line) == colorize(line, c)
}

// The severity column of "alarm show" is padded differently depending on the
// device and the command, and either kind of padding has to be recognized.
func TestCienaAlarmSeverityPadding(t *testing.T) {
	const (
		unpadded = `|    4|   |   |   51|critical| Thu Sep 15 17:12:30 2022 |    PTP-1/5| Rx Loss of Signal |`
		padded   = `|    4|   |   |   51| critical | Thu Sep 15 17:12:30 2022 |    PTP-1/5| Rx Loss of Signal |`
		info     = `|    4|   |   |   51| info | Thu Sep 15 17:12:30 2022 |    PTP-1/5| Rx Loss of Signal |`
	)
	for _, line := range []string{unpadded, padded, info} {
		if out := FormatText(line); out == line {
			t.Errorf("alarm line was not colorized: %q", line)
		}
	}
}

// Junos interface names are matched by family. A name is a name; a colon is
// not part of one.
func TestJunosInterfaceNames(t *testing.T) {
	up := []string{
		"ge-0/0/1 up up",
		"xe-0/0/1 up up",
		"et-0/0/1 up up",
		"et-0/0/1:0 up up",   // a channel of a channelized port
		"et-0/0/1:0.0 up up", // and a logical unit on that channel
		"xe-0/0/1:3 up up",
		"ae0 up up",
		"ae0.100 up up",
		"lo0.0 up up",
		"irb up up",
		"vlan.100 up up",
		"fti0 up up",
		"gr-1/2/3 up up",
		"pfe-0/0/0 up up",
		"vcp-0/0 up up",
	}
	for _, line := range up {
		if !colorizedWith(line, green) {
			t.Errorf("FormatText(%q) = %q, want it green", line, FormatText(line))
		}
	}

	down := []string{"ge-0/0/1 up down", "ae0 up down"}
	for _, line := range down {
		if !colorizedWith(line, red) {
			t.Errorf("FormatText(%q) = %q, want it red", line, FormatText(line))
		}
	}

	admin := []string{"ge-0/0/1 down down", "ae0 down up"}
	for _, line := range admin {
		if !colorizedWith(line, orange) {
			t.Errorf("FormatText(%q) = %q, want it orange", line, FormatText(line))
		}
	}

	notNames := []string{":fti0 up up", "gr:-1/2/3 up up", "Ethernet1 up up"}
	for _, line := range notNames {
		if out := FormatText(line); out != line {
			t.Errorf("FormatText(%q) = %q, want it left alone", line, out)
		}
	}
}

// Cisco appends "(connected)" to a link that is up. Nothing else may come
// between the state and it.
func TestInterfaceUpLine(t *testing.T) {
	for _, line := range []string{
		"Ethernet1 is up, line protocol is up",
		"Ethernet1 is up, line protocol is up (connected)",
		"GigabitEthernet0/1 is up, line protocol is up (connected)",
	} {
		if !colorizedWith(line, green) {
			t.Errorf("FormatText(%q) = %q, want it green", line, FormatText(line))
		}
	}

	// A colon before "(connected)" is not something a device emits, and is
	// not accepted.
	line := "Ethernet1 is up, line protocol is up: (connected)"
	if colorizedWith(line, green) {
		t.Errorf("FormatText(%q) was colorized green, want it left alone", line)
	}
}

// Multi-lane Arista optics add a channel column to "show interfaces
// transceiver" and print the full interface name.
func TestAristaTransceiverChannels(t *testing.T) {
	tests := []struct {
		line string
		want color
	}{
		{"Ethernet9/30       1 39.47     3.32    90.01   2.88     -8.20    0:00:02 ago", info},
		{"Ethernet9/30       3 39.47     3.32    120.00  2.72     -6.03    0:00:02 ago", info},
		{"Et9/30             1 39.47     3.32    90.01   2.88     -8.20    0:00:02 ago", info},
		{"Ethernet9/30       4 39.47     3.32    120.00  2.60     -30.00   0:00:02 ago", red},
		{"Ethernet9/30       4 39.47     3.32    120.00  2.60     - Inf    0:00:02 ago", red},
		// Only the receive level counts: a dim transmitter is not red.
		{"Ethernet9/30       4 39.47     3.32    0.00    -30.00   -8.20    0:00:02 ago", info},
		{"Ethernet9/30       1 N/A       N/A     N/A     N/A      N/A      N/A", orange},
	}
	for _, tt := range tests {
		if !colorizedWith(tt.line, tt.want) {
			t.Errorf("FormatText(%q) = %q, want it %v", tt.line, FormatText(tt.line), tt.want)
		}
	}

	// The form without a channel column is unaffected by the one with it.
	for _, tt := range []struct {
		line string
		want color
	}{
		{"Et49/1     45.55      3.29      5.62     0.81      -4.46     0:00:03 ago", info},
		{"Et49/8     45.55      3.29      6.15     1.02      -30.00    0:00:03 ago", red},
	} {
		if !colorizedWith(tt.line, tt.want) {
			t.Errorf("FormatText(%q) = %q, want it %v", tt.line, FormatText(tt.line), tt.want)
		}
	}
}

// Every rule table has to compile, and each pattern has to have exactly the
// submatches its paints refer to. The rule constructors check this as the
// tables are built, so merely reaching this test proves it, but name the
// requirement so a failure explains itself.
func TestRuleTablesAreWellFormed(t *testing.T) {
	tables := map[string][]rule{
		"arista (before counters)": aristaRulesBefore,
		"arista (after counters)":  aristaRulesAfter,
		"junos":                    junosRules,
		"vyos":                     vyosRules,
		"ciena":                    cienaRules,
	}
	for name, rules := range tables {
		if len(rules) == 0 {
			t.Errorf("%s rule table is empty", name)
		}
		for i, r := range rules {
			if len(r.paints) == 0 {
				t.Errorf("%s rule %d paints nothing", name, i)
			}
			for _, p := range r.paints {
				if p.group < 1 || p.group > r.re.NumSubexp() {
					t.Errorf("%s rule %d paints submatch %d of %q, which has %d",
						name, i, p.group, r.re, r.re.NumSubexp())
				}
			}
		}
	}
}
