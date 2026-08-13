// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

// Regular expression fragments shared by the rule tables. Each one is wrapped
// in a non-capturing group so that it can be interpolated into a larger
// pattern without its alternations escaping, and none of them contain a
// capture group, so that the submatch numbering of a rule's pattern is exactly
// the number of parentheses written in the rule itself.
const (
	// pNum is a real number: an optional sign, digits with an optional
	// fractional part (or a bare fractional part), and an optional exponent.
	pNum = `(?:[-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[Ee][-+]?[0-9]+)?)`
	// pInt is an unsigned integer, pPosInt one without a leading zero.
	pInt    = `(?:[0-9]+)`
	pPosInt = `(?:[1-9][0-9]*)`
	// pNumStart matches just the start of a number, for fields that are
	// documented to hold one but whose value is not otherwise inspected.
	pNumStart = `(?:[-+]?\.?[0-9])`

	// Optical receive levels, in dBm. pLowLight is dim enough to be a
	// problem, pVeryLowLight is dim enough to be a problem on a device that
	// reports a floor rather than "no signal", and pLight is any reading at
	// all (including a port that reports N/A).
	pLowLight     = `(?:-[3-9][0-9]\.[0-9]{1,2}|-\sInf|-2[5-9]\.[0-9]{1,2})`
	pVeryLowLight = `(?:-[4-9][0-9]\.[0-9]{1,2})`
	pLight        = `(?:` + pNum + `|N/A)`

	// Optical return loss, in dB. More loss is better: a reflection coming
	// back down the fiber is what these numbers measure.
	pGoodReturnLoss = `(?:29\.[0-9]+|[3-9][0-9]+\.[0-9]+)`
	pWarnReturnLoss = `(?:2[0-8]\.[0-9]+|1[6-9]\.[0-9]+)`
	pBadReturnLoss  = `(?:1[0-5]\.[0-9]+|[0-9]\.[0-9]+)`

	// Ciena alarm severities, and the severities that are nominally serious
	// but common enough on a healthy device to be worth downgrading.
	pBigAlarms    = `(?:critical|major|minor|warning)`
	pLittleAlarms = `(?:info)`
	pBigOverrides = `(?:Rx Remote Fault|Far End Client Signal Fail)`

	// Spanning tree port states and roles.
	pSTPGood  = `(?:forwarding|FWD)`
	pSTPWarn  = `(?:learning|LRN)`
	pSTPBad   = `(?:discarding|BLK)`
	pSTPTypes = `(?:designated|root|alternate|Desg|Root|Altn)`

	// pTTYTypes are the line types in "show users"-style output, which look
	// enough like interface statistics to need excluding.
	pTTYTypes = `(?:AUX|CTY|LPR|TTY|VTY)`

	// Interface names as Arista and Cisco render them: "Ethernet1" in full
	// output, "Et1" in tabular output.
	pInterface = `(?:[A-Z]\S+)`
	pIntShort  = `(?:[A-Z][a-z][0-9]\S*)`
)

// Interface names as Junos renders them.
const (
	// pJunosEth covers ge-, xe-, and et- names, including a channelized
	// port's ":channel" and the logical unit that may follow it, as in
	// "et-0/0/1:0.0".
	pJunosEth    = `(?:(?:[gx]e|et)-[0-9/.]+(?::[0-9]+(?:\.[0-9]+)?)?)`
	pJunosIfaces = `(?:` +
		`ae[0-9.]+` + // aggregated ethernet
		`|bme[0-9.]+` + // internal management
		`|cbp[0-9.]+` + // customer backbone port
		`|` + pJunosEth + // ge-/xe-/et- ethernet
		`|irb[0-9/.]*` + // integrated routing and bridging
		`|lo[0-9.]+` + // loopback
		`|me[0-9.]+` + // management ethernet
		`|jsrv[0-9.]*` + // internal services
		`|pf[eh]-[0-9/.]+` + // packet forwarding engine / host
		`|pip[0-9/.]+` + // provider instance port
		`|vcp-[0-9/.]+` + // virtual chassis port
		`|vlan\.[0-9]+` +
		// The remainder, including the tunnel and internal interfaces.
		`|(?:fti|fxp|gr|ip|lsq|lt|mt|sp|pp|ppd|ppe|st)-?[0-9/.]+` +
		`|gr-\S+` +
		`|dsc|esi|gre|ipip|jsrv|lsi` +
		`|mtun|pimd|pime|rbeb|tap|vlan|vme|vtep` +
		`)`
)
