// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// releasesURL points at the GitHub API's "latest release" endpoint for this
// project. Overridden by tests so they never hit the network.
var releasesURL = "https://api.github.com/repos/jmaslak/go-router-colorizer/releases/latest"

// httpClient is shared so tests can point it at a local server via a custom
// Transport instead of touching the network.
var httpClient = &http.Client{Timeout: 30 * time.Second}

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (r release) assetURL(name string) (string, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a.BrowserDownloadURL, true
		}
	}
	return "", false
}

// selfUpdate replaces the running executable with the latest release build
// for the current OS and architecture, verifying it against the release's
// published checksums first.
func selfUpdate() error {
	rel, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	if version != "dev" && latest == strings.TrimPrefix(version, "v") {
		fmt.Printf("router-colorizer %s is already the latest version\n", version)
		return nil
	}

	assetName := fmt.Sprintf("router-colorizer_%s_%s", runtime.GOOS, runtime.GOARCH)
	binURL, ok := rel.assetURL(assetName)
	if !ok {
		return fmt.Errorf("release %s has no build for %s/%s", rel.TagName, runtime.GOOS, runtime.GOARCH)
	}
	checksumsURL, ok := rel.assetURL("checksums.txt")
	if !ok {
		return fmt.Errorf("release %s is missing checksums.txt", rel.TagName)
	}

	checksums, err := download(checksumsURL)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}
	wantSum, err := checksumFor(checksums, assetName)
	if err != nil {
		return err
	}

	bin, err := download(binURL)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", assetName, err)
	}
	if gotSum := sha256.Sum256(bin); hex.EncodeToString(gotSum[:]) != wantSum {
		return fmt.Errorf("checksum mismatch for %s: the download is corrupt or the release was tampered with", assetName)
	}

	if err := installExecutable(bin); err != nil {
		return err
	}

	fmt.Printf("router-colorizer updated to %s\n", rel.TagName)
	return nil
}

func fetchLatestRelease() (release, error) {
	resp, err := httpClient.Get(releasesURL)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("unexpected status from GitHub: %s", resp.Status)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return release{}, fmt.Errorf("decoding release metadata: %w", err)
	}
	return rel, nil
}

func download(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s from %s", resp.Status, url)
	}
	return io.ReadAll(resp.Body)
}

// checksumFor finds name's checksum in a checksums.txt produced by
// `goreleaser`, formatted as one "<sha256>  <filename>" pair per line.
func checksumFor(checksums []byte, name string) (string, error) {
	for line := range strings.SplitSeq(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %s", name)
}

// installExecutable overwrites the running binary with bin. It is a package
// variable so tests can redirect it at a throwaway file instead of the test
// binary itself.
var installExecutable = replaceExecutable

// replaceExecutable atomically overwrites the running binary with bin. It
// writes to a temporary file in the same directory first, so the rename that
// replaces the live executable is a same-filesystem, single-syscall swap.
func replaceExecutable(bin []byte) error {
	target, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating the running executable: %w", err)
	}
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return fmt.Errorf("resolving %s: %w", target, err)
	}
	return installOver(target, bin)
}

// installOver atomically overwrites target with bin. Split out from
// replaceExecutable so tests can exercise it against a throwaway file instead
// of the test binary itself.
func installOver(target string, bin []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".router-colorizer-update-*")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return fmt.Errorf("writing new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing new binary: %w", err)
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return fmt.Errorf("making new binary executable: %w", err)
	}

	if err := os.Rename(tmp.Name(), target); err != nil {
		return fmt.Errorf("installing new binary over %s: %w", target, err)
	}
	return nil
}
