// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

// junosRules cover Juniper Junos output.
var junosRules = []rule{
	// "show interfaces": physical link state.
	whole(`^(Physical interface: \S+ Enabled, Physical link is Up)$`, green),
	whole(`^(Physical interface: \S+ Enabled, Physical link is Down)$`, red),
	whole(`^(Physical interface: \S+\s\S+ Physical link is Down)$`, orange),
	whole(`^(Physical interface: \S+)$`, info),
	whole(`^(  Logical interface [^\n]+)$`, info),
	whole(`^(  Last flapped   : [^\n]+)$`, info),

	// "show interfaces": rates and counters.
	whole(`^(  Input rate     : `+pNumStart+`[^\n]*)$`, info),
	whole(`^(  Output rate    : `+pNumStart+`[^\n]*)$`, info),
	whole(`^(    Input packets : `+pNumStart+`[^\n]*)$`, info),
	whole(`^(    Output packets: `+pNumStart+`[^\n]*)$`, info),

	// "show interfaces": alarms and defects.
	whole(`^(  Active alarms  : None)$`, green),
	whole(`^(  Active alarms  : [^\n]+)$`, red),
	whole(`^(  Active defects : None)$`, green),
	whole(`^(  Active defects : [^\n]+)$`, red),

	// "show interfaces terse".
	whole(`^((?:`+pJunosIfaces+`)\s+up\s+up[^\n]*)$`, green),
	whole(`^(`+pJunosEth+`\s+VCP)$`, green),
	whole(`^((?:`+pJunosIfaces+`)\s+up\s+down[^\n]*)$`, red),
	whole(`^((?:`+pJunosIfaces+`)\s+down\s+[^\n]*)$`, orange),

	// "show interfaces extensive": error counters. Note that a single
	// errored block reads as clean: only a run of them is called out.
	whole(`^(    Bit errors \s+0)$`, green),
	whole(`^(    Errored blocks \s+[1-9][0-9]+)$`, red),
	whole(`^(    Errored blocks \s+0)$`, green),
	whole(`^(    FEC \S+\sErrors(?:\sRate)?\s+0)$`, green),
	whole(`^(    FEC \S+\sErrors(?:\sRate)?\s+[^\n]+)$`, red),

	// "show interfaces diagnostics optics". The first rule only reaches
	// readings with a single space after the colon, which is how this output
	// reports an out-of-range value; the milliwatt/dBm pairs below carry the
	// usual two spaces.
	whole(`^(    Laser output power \s+:\s`+pNumStart+`[^\n]*)$`, red),
	whole(`^(    Laser output power \s+:\s+`+pNum+` mW / `+pLowLight+`\sdBm)$`, red),
	whole(`^(    Laser output power \s+:\s+`+pNum+` mW / `+pLight+`\sdBm)$`, info),
	whole(`^(    Laser receiver power \s+:\s+`+pNum+` mW / `+pLowLight+`\sdBm)$`, red),
	whole(`^(    Laser receiver power \s+:\s+`+pNum+` mW / `+pLight+`\sdBm)$`, info),
	whole(`^(    Receiver signal average optical power \s+:\s+`+pNum+` mW / `+pLowLight+`\sdBm)$`, red),
	whole(`^(    Receiver signal average optical power \s+:\s+`+pNum+` mW / `+pLight+`\sdBm)$`, info),
}

// vyosRules cover the VyOS/FRR output that the Arista and Cisco rules do not
// already handle.
var vyosRules = []rule{
	// BGP neighbor state. The Established rule runs first so that the
	// general rule below, which the Perl original guards with a negative
	// lookahead, cannot reach an established session: by the time it runs,
	// the line already starts with an escape sequence.
	whole(`^(  BGP state = Established[^\n]*)$`, green),
	whole(`^(  BGP state = [^\n]*)$`, red),

	whole(`^(  Route map for (?:incoming|outgoing) advertisements is [^\n]*)$`, info),
	whole(`^(  \S+ peer-group member[^\n]+)$`, info),
	whole(`^(  `+pInt+` accepted prefixes[^\n]+)$`, info),
	whole(`^(Local host: [^\n]+)$`, info),
	whole(`^(Foreign host: [^\n]+)$`, info),

	// Show version
	whole(`^(Version:          [^\n]+)$`, info),
}
