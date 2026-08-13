// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package colorizer

import (
	"runtime"
	"strings"
	"sync"
)

// minChunk is the least amount of text worth handing to a goroutine of its
// own. Colorizing runs at a few megabytes a second, so a chunk this size is
// milliseconds of work -- thousands of times the cost of starting a goroutine
// and joining it, and small enough that a screenful of output still splits.
const minChunk = 8 << 10

// FormatTextParallel returns exactly what FormatText returns, dividing the
// work between up to workers goroutines. A workers of zero or less uses one
// per available CPU.
//
// This is worth reaching for when colorizing a large captured file. It is not
// worth reaching for on a live session: text arrives there far slower than a
// single core can colorize it, and the division is skipped anyway for anything
// small.
//
// Splitting is safe because lines are colorized independently of each other:
// text is only ever divided at a line ending, and colorizing the pieces and
// joining the results gives the same bytes as colorizing the whole. That
// property is checked against FormatText by the tests, including under fuzzing.
func FormatTextParallel(text string, workers int) string {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if n := len(text) / minChunk; workers > n {
		workers = n
	}
	if workers < 2 {
		return FormatText(text)
	}

	chunks := splitAtLines(text, workers)
	if len(chunks) < 2 {
		return FormatText(text)
	}

	results := make([]string, len(chunks))
	var wg sync.WaitGroup
	wg.Add(len(chunks))
	for i, chunk := range chunks {
		go func() {
			defer wg.Done()
			results[i] = FormatText(chunk)
		}()
	}
	wg.Wait()

	// Joining in index order is what keeps the output in input order.
	var b strings.Builder
	size := 0
	for _, r := range results {
		size += len(r)
	}
	b.Grow(size)
	for _, r := range results {
		b.WriteString(r)
	}
	return b.String()
}

// splitAtLines divides text into at most n pieces of roughly equal size, each
// piece ending just past a line feed so that no line is split across two of
// them. Text with no line feed left to cut at becomes a single final piece.
func splitAtLines(text string, n int) []string {
	if n < 2 || text == "" {
		return []string{text}
	}

	target := (len(text) + n - 1) / n
	chunks := make([]string, 0, n)
	for len(text) > target {
		// The line ending at or after the target, so that a target landing
		// exactly on one cuts there rather than swallowing the next line.
		nl := strings.IndexByte(text[target-1:], '\n')
		if nl < 0 {
			break
		}
		cut := target + nl
		chunks = append(chunks, text[:cut])
		text = text[cut:]
	}
	if text != "" {
		chunks = append(chunks, text)
	}
	return chunks
}
