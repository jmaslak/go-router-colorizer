// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmaslak/go-router-colorizer/colorizer"
)

func ExampleFormatText() {
	out := colorizer.FormatText("Ethernet1 is up, line protocol is up\n")

	// Escapes are shown here as "ESC" so that the example output is
	// readable; a terminal would render the line in green.
	fmt.Println(strings.ReplaceAll(out, "\x1b", "ESC"))
	// Output: ESC[32mEthernet1 is up, line protocol is upESC[0m
}

func ExampleFilter() {
	// Colorize a device session as it streams past, which is what the
	// router-colorizer command does.
	if err := colorizer.Filter(os.Stdout, os.Stdin, colorizer.DefaultFlushDelay); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
