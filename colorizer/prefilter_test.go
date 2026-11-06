// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"strings"
	"testing"
)

// allRules is every rule in every table, which the prefilter tests check as a
// group: a prefilter that rejects too much is a rule that silently stops
// firing, and nothing else in the suite would say which rule it was.
func allRules() map[string][]rule {
	return map[string][]rule{
		"arista (before counters)": aristaRulesBefore,
		"arista (after counters)":  aristaRulesAfter,
		"junos":                    junosRules,
		"vyos":                     vyosRules,
		"ciena":                    cienaRules,
	}
}

func TestPrefilterExtraction(t *testing.T) {
	tests := []struct {
		pattern string
		lit     string
		classes int
		must    string
		accepts []string
		rejects []string
	}{
		// A pattern that opens with a phrase reduces to that phrase.
		{
			pattern: `^(Status: Ready)$`,
			lit:     "Status: Ready",
			accepts: []string{"Status: Ready"},
			rejects: []string{"Status: Down", "status: Ready", ""},
		},
		// Character classes become one class per position.
		{
			pattern: `^([A-Z][a-z][0-9]x)$`,
			lit:     "",
			classes: 4,
			must:    "x",
			accepts: []string{"Ab0x"},
			rejects: []string{"ab0x", "AB0x", "Abcx", "Ab0"},
		},
		// A literal after a class still filters, though only one position
		// at a time: the run of literal bytes has to be a class each once
		// the positions are being counted off one by one.
		{
			pattern: `^(Line\s[0-9], Loc:)$`,
			lit:     "Line",
			classes: 1 + 1 + len(", Loc:"), // the space, the digit, then the rest
			must:    ", Loc:",
			accepts: []string{"Line 1, Loc:"},
			rejects: []string{"Line11, Loc:", "Lin  1"},
		},
		// A "+" requires one repetition, so its first byte is required.
		{
			pattern: `^([A-Za-z0-9_\-\.]+ uptime is .*)$`,
			lit:     "",
			classes: 1,
			must:    " uptime is ",
			accepts: []string{"router1 uptime is 3 days"},
			rejects: []string{" router1 uptime is 3 days", ""},
		},
		// An alternation contributes only the bytes its branches share a
		// position with, and nothing after it.
		{
			pattern: `^((?:Local|Remote) TCP address is .*)$`,
			lit:     "",
			classes: 1,
			must:    " TCP address is ",
			accepts: []string{"Local TCP address is x", "Remote TCP address is x"},
			rejects: []string{"Other TCP address is x", ""},
		},
		// An unanchored pattern says nothing about the start of a line,
		// but its text still has to be somewhere in one.
		{
			pattern: `(abc)`,
			lit:     "",
			classes: 0,
			must:    "abc",
			accepts: []string{"xyzabc", "abc"},
			rejects: []string{"", "nothing like it"},
		},
		// Neither does a pattern that opens with something optional, which
		// is where searching the line for required text earns its keep.
		{
			pattern: `^((?: |   )? Description: .*)$`,
			lit:     "",
			classes: 0,
			must:    " Description: ",
			accepts: []string{"   Description: uplink", " Description: uplink"},
			rejects: []string{"anything at all", "Description: uplink"},
		},
		// Text the start of the line already accounts for is not searched
		// for a second time.
		{
			pattern: `^(Modem state: Idle)$`,
			lit:     "Modem state: Idle",
			must:    "",
			accepts: []string{"Modem state: Idle"},
			rejects: []string{"Modem state: Busy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			f := newPrefilter(tt.pattern)
			if f.lit != tt.lit {
				t.Errorf("newPrefilter(%q).lit = %q, want %q", tt.pattern, f.lit, tt.lit)
			}
			if len(f.classes) != tt.classes {
				t.Errorf("newPrefilter(%q) has %d classes, want %d",
					tt.pattern, len(f.classes), tt.classes)
			}
			if f.must != tt.must {
				t.Errorf("newPrefilter(%q).must = %q, want %q", tt.pattern, f.must, tt.must)
			}
			for _, line := range tt.accepts {
				if !f.match(line) {
					t.Errorf("newPrefilter(%q).match(%q) = false, want true", tt.pattern, line)
				}
			}
			for _, line := range tt.rejects {
				if f.match(line) {
					t.Errorf("newPrefilter(%q).match(%q) = true, want false", tt.pattern, line)
				}
			}
		})
	}
}

// A prefilter may only ever reject a line its pattern would not have matched.
// This is the property the whole optimization rests on, so it is checked
// against every rule, over the captured device output and over the awkward
// input the corpus does not contain.
func TestPrefilterRejectsOnlyNonMatches(t *testing.T) {
	lines := corpusLines(t)
	lines = append(lines,
		"", " ", "\x1b[0m", "\x1b[32mEthernet1 is up, line protocol is up",
		"Status: Ready", "Status:", "\r", "\x00", "é is up, line protocol is up",
		strings.Repeat("A", 300),
	)

	for name, rules := range allRules() {
		for i := range rules {
			for _, line := range lines {
				checkPrefilter(t, name, i, &rules[i], line)
			}
		}
	}
}

func FuzzPrefilter(f *testing.F) {
	for _, seed := range []string{
		"", "Status: Ready", "Ethernet1 is up, line protocol is up",
		"| Admin State | Enabled |", "  BGP state is Established",
		"ge-0/0/0.0 up up", "\x1b[Ké \x00",
	} {
		f.Add(seed)
	}

	rules := allRules()
	f.Fuzz(func(t *testing.T, line string) {
		for name, table := range rules {
			for i := range table {
				checkPrefilter(t, name, i, &table[i], line)
			}
		}
	})
}

// checkPrefilter fails the test if r's prefilter rejects a line that r's
// pattern matches.
func checkPrefilter(t *testing.T, table string, i int, r *rule, line string) {
	t.Helper()
	if r.pre.match(line) {
		return
	}
	if r.re.MatchString(line) {
		t.Errorf("%s rule %d: prefilter rejected %q, which %v matches",
			table, i, line, r.re)
	}
}

// The prefilters have to actually filter. A refactor that left them all empty
// would keep every other test passing while quietly making the colorizer
// several times slower, so hold the tables to the coverage they have now.
func TestPrefilterCoversMostRules(t *testing.T) {
	total, filtered := 0, 0
	for name, rules := range allRules() {
		for i := range rules {
			total++
			f := &rules[i].pre
			if f.lit != "" || len(f.classes) > 0 || f.must != "" {
				filtered++
				continue
			}
			t.Logf("%s rule %d has no prefilter: %v", name, i, rules[i].re)
		}
	}
	if min := total * 9 / 10; filtered < min {
		t.Errorf("only %d of %d rules have a prefilter, want at least %d",
			filtered, total, min)
	}
}

// The pattern colorizeInterfaceCounters uses is filtered the same way, and is
// not part of any rule table.
func TestInterfaceCountersPrefilter(t *testing.T) {
	if interfaceCountersPre.lit == "" {
		t.Fatal("the interface counter pattern has no prefilter")
	}
	for _, line := range corpusLines(t) {
		if !interfaceCountersPre.match(line) && interfaceCounters.MatchString(line) {
			t.Errorf("prefilter rejected %q, which the counter pattern matches", line)
		}
	}
}

// corpusLines is every line of captured device output in testdata.
func corpusLines(t *testing.T) []string {
	t.Helper()
	return strings.Split(benchCorpus(t), "\n")
}
