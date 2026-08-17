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
	"os/exec"
	"sync"
	"time"

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
	update := flag.Bool("selfupdate", false, "update to the latest release and exit")
	cmdName := flag.String("cmd", "",
		"run this command, passing it any arguments not handled by router-colorizer, and colorize its output")
	paletteName := flag.String("palette", "default",
		`color palette to use: "default" or "deuteranopia" (red/green colorblind friendly)`)
	cb := flag.Bool("cb", false, "colorblind mode: shorthand for -palette deuteranopia")
	flag.Usage = usage
	flag.Parse()

	paletteSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "palette" {
			paletteSet = true
		}
	})
	name, err := resolvePalette(*paletteName, paletteSet, *cb)
	if err == nil {
		err = colorizer.SetPalette(name)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "router-colorizer: %v\n", err)
		os.Exit(2)
	}

	if *showVersion {
		fmt.Printf("router-colorizer %s\n", version)
		return
	}
	if *update {
		if err := selfUpdate(); err != nil {
			fmt.Fprintf(os.Stderr, "router-colorizer: %v\n", err)
			os.Exit(1)
		}
		return
	}

	notifyUpdateAvailable(os.Stderr)

	if *cmdName != "" {
		os.Exit(runCmd(*cmdName, flag.Args(), *delay))
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

// resolvePalette combines the -palette and -cb flags into the palette to use,
// where paletteSet says whether -palette was given explicitly. -cb is shorthand
// for the colorblind palette, so the two conflict when they disagree.
func resolvePalette(name string, paletteSet, cb bool) (string, error) {
	if cb && paletteSet && name != "deuteranopia" {
		return "", fmt.Errorf("-cb is shorthand for -palette deuteranopia and conflicts with -palette %s", name)
	}
	if cb {
		return "deuteranopia", nil
	}
	return name, nil
}

// runCmd runs name with args, colorizing its standard output and standard
// error as they come in, and returns the exit code to use.
func runCmd(name string, args []string, delay time.Duration) int {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "router-colorizer: %v\n", err)
		return 1
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "router-colorizer: %v\n", err)
		return 1
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "router-colorizer: %v\n", err)
		return 1
	}

	// Each stream gets its own Filter, run concurrently, so a quiet stderr
	// never waits behind a busy stdout (or vice versa).
	var wg sync.WaitGroup
	var stdoutErr, stderrErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		stdoutErr = colorizer.Filter(os.Stdout, stdout, delay)
	}()
	go func() {
		defer wg.Done()
		stderrErr = colorizer.Filter(os.Stderr, stderr, delay)
	}()
	wg.Wait()

	waitErr := cmd.Wait()

	for _, filterErr := range []error{stdoutErr, stderrErr} {
		if filterErr != nil && !errors.Is(filterErr, io.ErrClosedPipe) && !errors.Is(filterErr, os.ErrClosed) {
			fmt.Fprintf(os.Stderr, "router-colorizer: %v\n", filterErr)
			return 1
		}
	}

	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		return exitErr.ExitCode()
	}
	if waitErr != nil {
		fmt.Fprintf(os.Stderr, "router-colorizer: %v\n", waitErr)
		return 1
	}
	return 0
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprint(out, `Usage: router-colorizer [options]

Colorize router CLI output read from standard input, for example:

    ssh router.example.com | router-colorizer

Alternatively, run with -cmd to have router-colorizer launch and colorize a
command directly, forwarding any arguments router-colorizer does not
recognize to it:

    router-colorizer -cmd ssh router.example.com

Options:
`)
	flag.PrintDefaults()
	fmt.Fprintf(out, "\nThe default flush delay is %s.\n", colorizer.DefaultFlushDelay)
	fmt.Fprint(out, "\nRun with -selfupdate to fetch and install the latest release in place.\n")
}
