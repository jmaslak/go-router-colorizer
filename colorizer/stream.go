// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"bytes"
	"errors"
	"io"
	"time"
)

// DefaultFlushDelay is how long Filter waits for the rest of a line before
// colorizing and writing what it has. It is short enough not to be felt while
// typing at an interactive prompt.
const DefaultFlushDelay = 10 * time.Millisecond

// readSize is the largest amount read from src at once.
const readSize = 65536

// Filter copies src to dst, colorizing the text as it goes, and returns when
// src reaches end of file.
//
// Colorization needs whole lines, but a device prompt is a line that is never
// finished, so a partial line that has been idle for flushAfter is colorized
// and written anyway. A zero or negative flushAfter uses DefaultFlushDelay.
//
// Filter reads src in a goroutine so that it can wait on the clock and on the
// input at the same time. That goroutine stays blocked in a read until src
// yields or fails, so a caller that needs Filter to return promptly should
// close src.
func Filter(dst io.Writer, src io.Reader, flushAfter time.Duration) error {
	if flushAfter <= 0 {
		flushAfter = DefaultFlushDelay
	}

	reads, done := make(chan []byte), make(chan struct{})
	defer close(done)
	readErr := make(chan error, 1)
	go func() {
		defer close(reads)
		for {
			buf := make([]byte, readSize)
			n, err := src.Read(buf)
			if n > 0 {
				select {
				case reads <- buf[:n]:
				case <-done:
					return
				}
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					readErr <- err
				}
				return
			}
		}
	}()

	// Blocks large enough to be worth dividing are colorized in parallel;
	// anything a live session produces falls below that and is colorized on
	// this goroutine, so interactive latency is unaffected.
	write := func(text string) error {
		_, err := io.WriteString(dst, FormatTextParallel(text, 0))
		return err
	}

	// The timer starts disarmed: there is nothing pending to flush yet.
	// Stopping it is enough to discard a pending tick as of Go 1.23.
	var pending []byte
	idle := time.NewTimer(flushAfter)
	idle.Stop()
	defer idle.Stop()

	for {
		select {
		case data, ok := <-reads:
			if !ok {
				if len(pending) > 0 {
					if err := write(string(pending)); err != nil {
						return err
					}
				}
				select {
				case err := <-readErr:
					return err
				default:
					return nil
				}
			}

			pending = append(pending, data...)
			switch end := bytes.LastIndexByte(pending, '\n') + 1; {
			case end > 0:
				// Everything through the last complete line.
				if err := write(string(pending[:end])); err != nil {
					return err
				}
				pending = append(pending[:0], pending[end:]...)
			case len(pending) == 1:
				// A single character is almost certainly an echo of
				// something just typed, and waiting on it would be
				// visible as lag.
				if err := write(string(pending)); err != nil {
					return err
				}
				pending = pending[:0]
			}

			idle.Stop()
			if len(pending) > 0 {
				idle.Reset(flushAfter)
			}

		case <-idle.C:
			if len(pending) > 0 {
				if err := write(string(pending)); err != nil {
					return err
				}
				pending = pending[:0]
			}
		}
	}
}
