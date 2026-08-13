// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

// Command router-colorizer colorizes router and switch CLI output read from
// standard input.
//
// It is meant to sit at the end of a pipe:
//
//	ssh router.example.com | router-colorizer
//
// Colorization is line at a time and passes anything it does not recognize
// through unchanged, so it is safe to leave in a pipeline permanently. It
// works on an interactive session too: a prompt, which is a line that is never
// finished, is written out after a brief pause rather than held back.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jmaslak/go-router-colorizer/colorizer"
)

// version is the release this was built from. Release builds overwrite it with
//
//	-ldflags "-X main.version=$(git describe --tags)"
var version = "dev"

func main() {
	delay := flag.Duration("flush-delay", colorizer.DefaultFlushDelay,
		"how long to wait for the rest of a partial line before writing it")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("router-colorizer %s\n", version)
		return
	}
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "router-colorizer: unexpected argument %q; input is read from standard input\n",
			flag.Arg(0))
		flag.Usage()
		os.Exit(2)
	}

	if err := colorizer.Filter(os.Stdout, os.Stdin, *delay); err != nil {
		// A closed pipe is how the reader downstream says it has had
		// enough, and is not worth reporting.
		if errors.Is(err, io.ErrClosedPipe) || errors.Is(err, os.ErrClosed) {
			return
		}
		fmt.Fprintf(os.Stderr, "router-colorizer: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprint(out, `Usage: router-colorizer [options]

Colorize router CLI output read from standard input, for example:

    ssh router.example.com | router-colorizer

Options:
`)
	flag.PrintDefaults()
	fmt.Fprintf(out, "\nThe default flush delay is %s.\n", colorizer.DefaultFlushDelay)
}
