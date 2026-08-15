// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"regexp"
	"slices"
	"strings"
)

// bgpErrors are the per-neighbor error counters in "show ip bgp neighbors"
// output. A nonzero counter is worth shouting about; a zero one is not, which
// is why the count is required to have no leading zero.
const bgpErrors = `(?:` +
	`AS path loop detection` +
	`|Enforced First AS` +
	`|Malformed MPBGP routes` +
	`|Originator ID matches local router ID` +
	`|Nexthop matches local IP address` +
	`|Resulting in removal of all paths in update \(treat as withdraw\)` +
	`|Resulting in AFI/SAFI disable` +
	`|Resulting in attribute ignore` +
	`|Disabled AFI/SAFIs` +
	`|IPv4 labeled-unicast NLRIs dropped due to excessive labels` +
	`|IPv6 labeled-unicast NLRIs dropped due to excessive labels` +
	`|IPv4 local address not available` +
	`|IPv6 local address not available` +
	`|Unexpected IPv6 nexthop for IPv4 routes` +
	`)`

// aristaRulesBefore and aristaRulesAfter are applied around
// colorizeInterfaceCounters, which is not expressible as a rule.
//
// These cover Arista EOS and, because their CLIs are close relatives, most of
// Cisco IOS as well.
var (
	aristaRulesBefore = []rule{
		// BGP neighbor state.
		whole(`^(  BGP state is Established[^\n]*)$`, green),
		whole(`^(  BGP state is [^\n]*)$`, red),
		whole(`^(  Last (?:sent||rcvd) (?:socket-error||notification):[^\n]+)$`, info),
		whole(`^(  (?:Inbound||Outbound) route map is [^\n]*)$`, info),
		whole(`^(  Inherits configuration from and member of peer-group [^\n]+)$`, info),
		whole(`^(    (?:IPv4|IPv6) Unicast:     [^\n]*)$`, info),
		whole(`^(  Configured maximum total number of routes is [0-9]+(?:, warning limit is [0-9]+)?)$`, info),
		whole(`^(    `+bgpErrors+`: `+pPosInt+`)$`, red),
		whole(`^(BGP neighbor is [^\n]+)$`, info),
		whole(`^((?:Local|Remote) TCP address is [^\n]+)$`, info),

		// Terminals and serial lines.
		whole(`^(Baud rate \(TX/RX\) is .*)$`, info),
		whole(`^(Status: Ready)$`, green),
		whole(`^(Status: .*)$`, info),
		whole(`^(Modem state: Idle)$`, green),
		whole(`^(Modem state: Ready)$`, green),
		whole(`^(Modem state: .*)$`, info),
		whole(`^(Line\s`+pInt+`, Location: .*)$`, info),
	}

	aristaRulesAfter = []rule{
		// "show interfaces": link state.
		whole(`^(`+pInterface+` is up, line protocol is up(?: \(connected\))?\s?)$`, green),
		whole(`^(`+pInterface+` is administratively down,[^\n]+)$`, orange),
		whole(`^(`+pInterface+` is [^\n]+, line protocol is [^\n]+)$`, red),
		whole(`^(  Up [^\n]+)$`, green),
		whole(`^(  Down [^\n]+)$`, red),

		// "show interfaces": descriptions and rates.
		whole(`^((?: |   )? Description: [^\n]+)$`, info),
		whole(`^(  `+pNum+`\s\w+\s(?:input|output)\srate\s`+pNum+`\s[^\n]+)$`, info),

		// "show interfaces status".
		whole(`^(`+pIntShort+`[^\n]+ connected [^\n]+)$`, green),
		whole(`^(`+pIntShort+`[^\n]+ disabled [^\n]+)$`, orange),
		whole(`^(`+pIntShort+`[^\n]+ errdisabled [^\n]+)$`, red),
		whole(`^(`+pIntShort+`[^\n]+ notconnect [^\n]+)$`, red),

		// "show interfaces description".
		whole(`^(`+pIntShort+`\s+up\s+up(?:\s+[^\n]+)?)$`, green),
		whole(`^(`+pIntShort+`\s+admin down\s+\S+(?:\s+[^\n]+)?)$`, orange),
		whole(`^(`+pIntShort+`\s+down\s+\S+(?:\s+[^\n]+)?)$`, red),

		// "show interfaces transceiver" (Arista). The receive level is the
		// last of the six readings, so a dim receiver is four readings in.
		whole(`^(`+pIntShort+`(?:\s+`+pLight+`){4}\s+`+pLowLight+`\s+\S+\sago)$`, red),
		whole(`^(`+pIntShort+`(?:\s+`+pLight+`){5}\s+\S+\sago)$`, info),
		whole(`^(`+pIntShort+`(?:\s+N/A){6}\s*)$`, orange),

		// "show interfaces transceiver detail" (Cisco). Trailing whitespace
		// is matched but not captured, so it is trimmed from the output.
		whole(`^(`+pIntShort+`(?:\s+`+pLight+`){3}\s+`+pLowLight+`)\s+$`, red),
		whole(`^(`+pIntShort+`(?:\s+`+pLight+`){4})\s+$`, info),

		// "show lldp neighbors detail".
		whole(`^(Interface\s\S+\sdetected\s`+pPosInt+` LLDP neighbors:)$`, info),
		whole(`^(  Neighbor \S+\sage\s`+pPosInt+`\sseconds)$`, info),
		whole(`^(  Discovered [^\n]+; Last changed [^\n]+\sago)$`, info),
		whole(`^(  - System Name: [^\n]+)$`, info),
		whole(`^(    Port ID     :[^\n]+)$`, info),
		whole(`^(    Management Address        : [^\n]+)$`, info),

		// "show spanning-tree".
		whole(`^(`+pIntShort+`\s+`+pSTPTypes+`\s+`+pSTPGood+`\s+[0-9]+\s+[0-9]+\.[0-9]+\s+P2p.*)$`, green),
		whole(`^(`+pIntShort+`\s+`+pSTPTypes+`\s+`+pSTPWarn+`\s+[0-9]+\s+[0-9]+\.[0-9]+\s+P2p.*)$`, orange),
		whole(`^(`+pIntShort+`\s+`+pSTPTypes+`\s+`+pSTPBad+`\s+[0-9]+\s+[0-9]+\.[0-9]+\s+P2p.*)$`, red),

		// "show bgp rpki cache" (Arista).
		whole(`^(State:\ssynced)$`, green),
		whole(`^(State:\s.*)$`, red),
		whole(`^(Connection:\sActive\s.*)$`, green),
		whole(`^(Connection:\s.*)$`, red),

		// Authentication error.
		whole(`^(Error in authentication)$`, red),

		// "show version" (Arista).
		whole(`^(Arista [0-9A-Z\-]+)$`, info),
		whole(`^(Serial number: [A-Z][A-Z0-9]+)$`, info),
		whole(`^(Software image version: [^\n]+)$`, info),
		whole(`^(Uptime: [^\n]+$)`, info),
		whole(`^((?:Total|Free) memory: [^\n]+)$`, info),

		// "show version" (Cisco).
		whole(`^(Cisco [A-Za-z ]+ Software, [^\n]+)$`, info),
		whole(`^([A-Za-z0-9_\-\.]+ uptime is [^\n]+)$`, info),
		whole(`^(System restarted at [^\n]+)$`, info),
		whole(`^(Last reload reason: [^\n]+)$`, info),
		whole(`^(License Level: [^\n]+)$`, info),
		whole(`^(Model number \s+: [^\n]+)$`, info),
		whole(`^(System serial number \s+: [^\n]+)$`, info),

		// Generic bad command error (Arista).
		whole(`^(% Invalid input)$`, red),
	}
)

// Interface counter lines, as emitted in the body of "show interfaces", are a
// comma-separated list of "<count> <name>" pairs:
//
//	0 input errors, 0 CRC, 0 alignment, 0 symbol, 0 input discards
//
// They are colorized as a unit by whether any counter is nonzero, which is a
// decision no single pattern can make.
var (
	interfaceCounters = regexp.MustCompile(`^     ((?:` + pInt + ` [^, ][^,]+, )*` + pInt + ` [^, ][^,]+)$`)

	// ttyLine matches the lines of "show users"-style output, which look
	// like counter lines but are not.
	ttyLine = regexp.MustCompile(`^\s+` + pInt + ` ` + pTTYTypes)

	// Counters that say nothing about the health of the interface.
	interfaceIgnores = []string{"bytes", "packets input", "packets output", "multicast"}

	// Counters that are noteworthy but not errors.
	interfaceInfos = []string{"PAUSE input", "PAUSE output", "pause input"}
)

func colorizeInterfaceCounters(line string) string {
	if ttyLine.MatchString(line) {
		return line
	}
	m := interfaceCounters.FindStringSubmatchIndex(line)
	if m == nil {
		return line
	}

	var ignored, noteworthy, faulted bool
	for rest := line[m[2]:m[3]]; rest != ""; {
		field, tail, _ := strings.Cut(rest, ", ")
		rest = tail

		count, name, ok := strings.Cut(field, " ")
		if !ok {
			// The pattern guarantees a space, but do not assume it.
			continue
		}
		switch {
		case slices.Contains(interfaceIgnores, name):
			ignored = true
		case slices.Contains(interfaceInfos, name):
			noteworthy = true
		}
		// A counter is nonzero if its value is one Perl would consider
		// true. Where a name is repeated, only its last value counts.
		if count != "" && count != "0" && !nameRepeats(rest, name) {
			faulted = true
		}
	}

	switch {
	case ignored:
		return line
	case noteworthy:
		return colorize(line, info)
	case faulted:
		return colorize(line, red)
	default:
		return colorize(line, green)
	}
}

// nameRepeats reports whether a counter called name appears in rest, the part
// of the counter list after the field being considered.
func nameRepeats(rest, name string) bool {
	for rest != "" {
		field, tail, _ := strings.Cut(rest, ", ")
		rest = tail
		if _, found, ok := strings.Cut(field, " "); ok && found == name {
			return true
		}
	}
	return false
}
