// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestHelperProcess is not a real test. It is invoked as a subprocess by the
// runCmd tests below (the standard library's own os/exec tests use this same
// pattern) so those tests don't depend on any external executable being
// present, which keeps them portable to Windows.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) == 0 {
		os.Exit(2)
	}

	switch args[0] {
	case "stdout":
		fmt.Fprint(os.Stdout, "hello stdout\n")
	case "stderr":
		fmt.Fprint(os.Stderr, "hello stderr\n")
	case "echostdin":
		io.Copy(os.Stdout, os.Stdin)
	case "exit":
		code := 0
		if len(args) > 1 {
			switch args[1] {
			case "1":
				code = 1
			case "3":
				code = 3
			}
		}
		os.Exit(code)
	}
}

func withCapturedStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	os.Stdout = orig
	w.Close()
	return <-done
}

func withStdin(t *testing.T, content string, fn func()) {
	t.Helper()

	orig := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdin = r

	go func() {
		io.WriteString(w, content)
		w.Close()
	}()

	fn()

	os.Stdin = orig
}

func helperArgs(subcommand string, extra ...string) []string {
	args := []string{"-test.run=TestHelperProcess", "--", subcommand}
	return append(args, extra...)
}

func TestRunCmd_Success(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	var code int
	out := withCapturedStdout(t, func() {
		code = runCmd(os.Args[0], helperArgs("exit"), time.Millisecond)
	})

	if code != 0 {
		t.Errorf("runCmd: got exit code %d, want 0", code)
	}
	if out != "" {
		t.Errorf("runCmd: got stdout %q, want empty", out)
	}
}

func TestRunCmd_PassesThroughExitCode(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	code := runCmd(os.Args[0], helperArgs("exit", "3"), time.Millisecond)
	if code != 3 {
		t.Errorf("runCmd: got exit code %d, want 3", code)
	}
}

func TestRunCmd_ForwardsStdout(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	var code int
	out := withCapturedStdout(t, func() {
		code = runCmd(os.Args[0], helperArgs("stdout"), time.Millisecond)
	})

	if code != 0 {
		t.Errorf("runCmd: got exit code %d, want 0", code)
	}
	if !strings.Contains(out, "hello stdout") {
		t.Errorf("runCmd: got stdout %q, want it to contain %q", out, "hello stdout")
	}
}

func TestRunCmd_ForwardsStdin(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	var code int
	var out string
	withStdin(t, "piped input\n", func() {
		out = withCapturedStdout(t, func() {
			code = runCmd(os.Args[0], helperArgs("echostdin"), time.Millisecond)
		})
	})

	if code != 0 {
		t.Errorf("runCmd: got exit code %d, want 0", code)
	}
	if !strings.Contains(out, "piped input") {
		t.Errorf("runCmd: got stdout %q, want it to contain %q", out, "piped input")
	}
}

func TestRunCmd_CommandNotFound(t *testing.T) {
	code := runCmd("router-colorizer-does-not-exist", nil, time.Millisecond)
	if code != 1 {
		t.Errorf("runCmd: got exit code %d, want 1", code)
	}
}
