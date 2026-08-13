// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

// Column fragments for the tabular output of the Ciena CLI, where every row is
// bounded and separated by "|" characters:
//
//	| Admin State        | Enabled        |
//
// cell is one column body, index a leading row-number column.
const (
	cienaCell  = `([^|]+)`
	cienaIndex = `(\|\s+[0-9]*\s*\|)`
	cienaBar   = `(\|)`
)

// cienaRules cover Ciena WDM and packet-optical devices.
var cienaRules = []rule{
	// Admin and operational state, as reported by several "show" commands.
	group(`^(\|\s?Admin State \s+\|)(\s?Enabled )([^\n]+)$`, "", green, ""),
	group(`^(\|\s?Admin State \s+\|)(\s?Disabled )([^\n]+)$`, "", orange, ""),
	group(`^(\|\s?Admin State \s+\|)`+cienaCell+`([^\n]+)$`, "", red, ""),

	group(`^(\|\s?Operational State \s+\|)(\s?Up           )([^\n]+)$`, "", green, ""),
	group(`^(\|\s?Operational State \s+\|)(\s?Initializing )([^\n]+)$`, "", orange, ""),
	group(`^(\|\s?Operational State \s+\|)`+cienaCell+`([^\n]+)$`, "", red, ""),

	// "xcvr show xcvr N/N".
	group(`^`+cienaBar+`( Tx Power \(dBm\)\s*)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)([^\n]+)$`,
		"", info, "", red, ""),
	group(`^`+cienaBar+`( Tx Power \(dBm\)\s*)`+cienaBar+`(\s+`+pLight+`\s+)([^\n]+)$`,
		"", info, "", info, ""),
	group(`^`+cienaBar+`( Rx Power \(dBm\)\s*)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)([^\n]+)$`,
		"", info, "", red, ""),
	group(`^`+cienaBar+`( Rx Power \(dBm\)\s*)`+cienaBar+`(\s+`+pLight+`\s+)([^\n]+)$`,
		"", info, "", info, ""),

	// "xcvr show xcvr status", which adds a port number column.
	group(`^`+cienaIndex+`( Tx Power \(dBm\)\s*)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)([^\n]+)$`,
		"", info, "", red, ""),
	group(`^`+cienaIndex+`( Tx Power \(dBm\)\s*)`+cienaBar+`(\s+`+pLight+`\s+)([^\n]+)$`,
		"", info, "", info, ""),
	group(`^`+cienaIndex+`( Rx Power \(dBm\)\s*)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)([^\n]+)$`,
		"", info, "", red, ""),
	group(`^`+cienaIndex+`( Rx Power \(dBm\)\s*)`+cienaBar+`(\s+`+pLight+`\s+)([^\n]+)$`,
		"", info, "", info, ""),

	// "ptp show ptp N/N [status]".
	group(`^(\|(?:\sTransmitter)? State \s+\|)( Enabled \s+)`+cienaBar+`( Up | Enabled )([^\n]+)$`,
		"", green, "", green, ""),
	group(`^(\|(?:\sTransmitter)? State \s+\|)( Enabled \s+)`+cienaBar+cienaCell+`([^\n]+)$`,
		"", green, "", red, ""),
	group(`^(\|(?:\sTransmitter)? State \s+\|)( Disabled \s+)`+cienaBar+cienaCell+`([^\n]+)$`,
		"", orange, "", orange, ""),

	// Amplifier and channel power, reported as a pair of readings.
	group(`^`+cienaIndex+`( Actual Power \(dBm\)\s*)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)([^\n]+)$`,
		"", info, "", red, "", red, ""),
	group(`^`+cienaIndex+`( Actual Power \(dBm\)\s*)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)`+cienaBar+`(\s+`+pLight+`\s+)([^\n]+)$`,
		"", info, "", red, "", info, ""),
	group(`^`+cienaIndex+`( Actual Power \(dBm\)\s*)`+cienaBar+`(\s+`+pLight+`\s+)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)([^\n]+)$`,
		"", info, "", info, "", red, ""),
	group(`^`+cienaIndex+`( Actual Power \(dBm\)\s*)`+cienaBar+`(\s+`+pLight+`\s+)`+cienaBar+`(\s+`+pLight+`\s+)([^\n]+)$`,
		"", info, "", info, "", info, ""),

	group(`^`+cienaIndex+`( Actual Aggregate Power \(dBm\)\s*)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)([^\n]+)$`,
		"", info, "", red, "", red, ""),
	group(`^`+cienaIndex+`( Actual Aggregate Power \(dBm\)\s*)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)`+cienaBar+`(\s+`+pLight+`\s+)([^\n]+)$`,
		"", info, "", red, "", info, ""),
	group(`^`+cienaIndex+`( Actual Aggregate Power \(dBm\)\s*)`+cienaBar+`(\s+`+pLight+`\s+)`+cienaBar+`(\s+`+pVeryLowLight+`\s+)([^\n]+)$`,
		"", info, "", info, "", red, ""),
	group(`^`+cienaIndex+`( Actual Aggregate Power \(dBm\)\s*)`+cienaBar+`(\s+`+pLight+`\s+)`+cienaBar+`(\s+`+pLight+`\s+)([^\n]+)$`,
		"", info, "", info, "", info, ""),

	alternate(`^`+cienaIndex+`( Span Loss \(dB\)\s*)`+cienaBar+`(\s+`+pNum+`\s+)`+cienaBar+`(\s+`+pNum+`\s+)`+cienaBar+`$`, info),

	// Optical return loss: more loss is better, so the thresholds run the
	// opposite way from the power readings above.
	group(`^(\|\s+\|)( Optical Return Loss \(dB\))(\s+\|\s+)(`+pGoodReturnLoss+`)(\s+\|)$`,
		"", green, "", green, ""),
	group(`^(\|\s+\|)( Optical Return Loss \(dB\))(\s+\|\s+)(`+pWarnReturnLoss+`)(\s+\|)$`,
		"", orange, "", orange, ""),
	group(`^(\|\s+\|)( Optical Return Loss \(dB\))(\s+\|\s+)(`+pBadReturnLoss+`)(\s+\|)$`,
		"", red, "", red, ""),

	// "alarm show". An alarm whose description is one of the overrides is
	// downgraded to a warning: those are routine on a link whose far end is
	// down, and are already reported by the interface rules.
	alternate(`^`+cienaBar+`(\s*[0-9]*)`+cienaBar+`(\s+)`+cienaBar+cienaCell+cienaBar+`(\s+[0-9]+)`+
		cienaBar+`(`+cienaSeverity+`)`+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+`(\s+`+pBigOverrides+`\s+)`+cienaBar+`$`, orange),
	alternate(`^`+cienaBar+`(\s*[0-9]*)`+cienaBar+`(\s+)`+cienaBar+cienaCell+cienaBar+`(\s+[0-9]+)`+
		cienaBar+`(`+cienaSeverity+`)`+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+`$`, red),
	// Acknowledged alarms ("Y" in the second column) are less urgent.
	alternate(`^`+cienaBar+`(\s+[0-9]*)`+cienaBar+`(\s+Y\s*)`+cienaBar+cienaCell+cienaBar+`(\s+[0-9]+)`+
		cienaBar+`(\s*`+pBigAlarms+`\s*)`+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+`$`, orange),
	alternate(`^`+cienaBar+`(\s+[0-9]*)`+cienaBar+`(\s+Y\s*)`+cienaBar+cienaCell+cienaBar+`(\s+[0-9]+)`+
		cienaBar+`(\s*`+pLittleAlarms+`\s*)`+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+`$`, info),

	whole(`^(SHELL \S+ FAILURE[^\n]+)$`, red),

	// "ptp show". These rows are wider than the terminal, so the pattern is
	// not anchored at the end of the line.
	alternate(`^`+cienaBar+cienaCell+cienaBar+`( Ena )`+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`(Up\s+)`+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar, green),
	alternate(`^`+cienaBar+cienaCell+cienaBar+`( Ena )`+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar, red),
	alternate(`^`+cienaBar+cienaCell+cienaBar+`( Dis )`+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar, orange),

	// "module show".
	alternate(`^`+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`( Enabled \s+)`+cienaBar+cienaCell+
		cienaBar+`( Up \s+)`+cienaBar+cienaCell+cienaBar+`$`, green),
	alternate(`^`+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`( Enabled \s+)`+cienaBar+cienaCell+
		cienaBar+`( Initializing \s+)`+cienaBar+cienaCell+cienaBar+`$`, orange),
	alternate(`^`+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`( Enabled \s+)`+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`$`, red),
	alternate(`^`+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`( Disabled \s+)`+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`$`, orange),

	// "chassis power show".
	alternate(`^`+cienaBar+cienaCell+cienaBar+`(\s+Disabled\s+)`+cienaBar+cienaCell+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`$`, orange),
	alternate(`^`+cienaBar+cienaCell+cienaBar+`(\s+Enabled\s+)`+cienaBar+`(\s+Up\s+)`+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`$`, green),
	alternate(`^`+cienaBar+cienaCell+cienaBar+`(\s+Enabled\s+)`+cienaBar+`(\s+Faulted\s+)`+cienaBar+cienaCell+
		cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+cienaCell+cienaBar+`$`, red),

	whole(`^(Success! Preparing state dump\.\.\.\s*)$`, green),

	// Session and operator warnings. These are not anchored at the end of
	// the line because the device pads them.
	whole(`^(Terminal will be disconnected in 1 minute if it remains inactive\.)`, red),
	whole(`^(WARNING:)`, orange),
	whole(`^(  You cannot abort the restart operation once it has started\.)$`, orange),

	// Refusing an operation because a prerequisite is still enabled.
	whole(`^(ERROR: .*disabled)$`, red),
}

// cienaSeverity matches the severity column of "alarm show", padded however
// the device chose to pad it.
const cienaSeverity = `\s*(?:` + pBigAlarms + `|` + pLittleAlarms + `)\s*`
