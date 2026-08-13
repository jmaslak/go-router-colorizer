// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer is a bytes.Buffer that can be read while Filter is writing to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestFilter(t *testing.T) {
	in := "Status: Ready\nplain text\nStatus: Busy\n"
	var out bytes.Buffer
	if err := Filter(&out, strings.NewReader(in), DefaultFlushDelay); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), FormatText(in); got != want {
		t.Errorf("Filter wrote %q, want %q", got, want)
	}
}

func TestFilterFlushesTrailingPartialLine(t *testing.T) {
	in := "Status: Ready\nno newline here"
	var out bytes.Buffer
	if err := Filter(&out, strings.NewReader(in), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), FormatText(in); got != want {
		t.Errorf("Filter wrote %q, want %q", got, want)
	}
}

// A device prompt is a line that is never finished. It has to be written out
// anyway, or an interactive session appears to hang.
func TestFilterFlushesIdlePartialLine(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()

	var out syncBuffer
	done := make(chan error, 1)
	go func() { done <- Filter(&out, reader, time.Millisecond) }()

	if _, err := writer.Write([]byte("router# ")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for out.String() != "router# " {
		if time.Now().After(deadline) {
			t.Fatalf("partial line was not written; output is %q", out.String())
		}
		time.Sleep(time.Millisecond)
	}

	writer.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// A lone character is an echo of a keystroke, and waiting even briefly on it
// would be visible as lag.
func TestFilterWritesSingleCharacterImmediately(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()

	var out syncBuffer
	done := make(chan error, 1)
	go func() { done <- Filter(&out, reader, time.Hour) }()

	if _, err := writer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for out.String() != "x" {
		if time.Now().After(deadline) {
			t.Fatalf("single character was not written; output is %q", out.String())
		}
		time.Sleep(time.Millisecond)
	}

	writer.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestFilterReportsReadError(t *testing.T) {
	want := errors.New("device went away")
	src := io.MultiReader(strings.NewReader("Status: Ready\n"), errReader{want})

	var out bytes.Buffer
	err := Filter(&out, src, time.Millisecond)
	if !errors.Is(err, want) {
		t.Errorf("Filter returned %v, want %v", err, want)
	}
	// Whatever arrived before the failure is still colorized and written.
	if got := out.String(); got != FormatText("Status: Ready\n") {
		t.Errorf("Filter wrote %q, want the line it managed to read", got)
	}
}

func TestFilterReportsWriteError(t *testing.T) {
	want := errors.New("pipe closed")
	err := Filter(errWriter{want}, strings.NewReader("Status: Ready\n"), time.Millisecond)
	if !errors.Is(err, want) {
		t.Errorf("Filter returned %v, want %v", err, want)
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }
