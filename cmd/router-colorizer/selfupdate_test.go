// Copyright (C) 2021-2026 Joelle Maslak
// SPDX-License-Identifier: Artistic-2.0

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestChecksumFor(t *testing.T) {
	checksums := []byte("aaa  router-colorizer_linux_amd64\nbbb  router-colorizer_darwin_arm64\n")

	got, err := checksumFor(checksums, "router-colorizer_darwin_arm64")
	if err != nil {
		t.Fatalf("checksumFor: %v", err)
	}
	if got != "bbb" {
		t.Errorf("checksumFor: got %q, want %q", got, "bbb")
	}

	if _, err := checksumFor(checksums, "router-colorizer_windows_amd64"); err == nil {
		t.Error("checksumFor: expected an error for a missing entry, got nil")
	}
}

func TestInstallOver(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "router-colorizer")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	if err := installOver(target, []byte("new")); err != nil {
		t.Fatalf("installOver: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("installOver: target contains %q, want %q", got, "new")
	}

	if runtime.GOOS == "windows" {
		// Windows has no POSIX executable bit: os.Chmod only toggles the
		// read-only attribute, so os.Stat always reports 0666 for a
		// writable file regardless of the mode passed to Chmod.
		return
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("installOver: target is not executable, mode %v", info.Mode())
	}
}

func TestSelfUpdate(t *testing.T) {
	assetName := fmt.Sprintf("router-colorizer_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}
	const binContent = "pretend-binary"
	// sha256("pretend-binary")
	const binSum = "45a99ba009be499ca30636f95de8bd2f743a4b800015c4b4a812484b9cc02de4"

	var baseURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v9.9.9","assets":[
			{"name":%q,"browser_download_url":"%s/bin"},
			{"name":"checksums.txt","browser_download_url":"%s/sums"}
		]}`, assetName, baseURL, baseURL)
	})
	mux.HandleFunc("/bin", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, binContent)
	})
	mux.HandleFunc("/sums", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", binSum, assetName)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	baseURL = srv.URL

	origURL, origClient, origVersion := releasesURL, httpClient, version
	releasesURL = srv.URL + "/latest"
	httpClient = srv.Client()
	version = "0.0.1"
	defer func() {
		releasesURL, httpClient, version = origURL, origClient, origVersion
	}()

	dir := t.TempDir()
	target := filepath.Join(dir, "router-colorizer")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	origInstall := installExecutable
	installExecutable = func(bin []byte) error { return installOver(target, bin) }
	defer func() { installExecutable = origInstall }()

	if err := selfUpdate(); err != nil {
		t.Fatalf("selfUpdate: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != binContent {
		t.Errorf("selfUpdate: target contains %q, want %q", got, binContent)
	}
}

func withUpdateCachePath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "update-check.json")
	orig := updateCachePath
	updateCachePath = func() (string, error) { return path, nil }
	t.Cleanup(func() { updateCachePath = orig })
	return path
}

func TestNotifyUpdateAvailable_NoCacheNoNotice(t *testing.T) {
	withUpdateCachePath(t)
	origVersion := version
	version = "1.0.0"
	defer func() { version = origVersion }()

	origSchedule := scheduleRefresh
	refreshed := false
	scheduleRefresh = func() { refreshed = true }
	defer func() { scheduleRefresh = origSchedule }()

	var buf bytes.Buffer
	notifyUpdateAvailable(&buf)

	if !refreshed {
		t.Error("notifyUpdateAvailable: expected a refresh to be scheduled for a cold cache")
	}

	if buf.Len() != 0 {
		t.Errorf("notifyUpdateAvailable: got %q, want no output on a cold cache", buf.String())
	}
}

func TestNotifyUpdateAvailable_DevVersionSkipsCheck(t *testing.T) {
	path := withUpdateCachePath(t)
	origVersion := version
	version = "dev"
	defer func() { version = origVersion }()

	var buf bytes.Buffer
	notifyUpdateAvailable(&buf)

	if buf.Len() != 0 {
		t.Errorf("notifyUpdateAvailable: got %q, want no output for a dev build", buf.String())
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("notifyUpdateAvailable: expected no cache file to be written for a dev build")
	}
}

func TestNotifyUpdateAvailable_StaleCacheNotifies(t *testing.T) {
	path := withUpdateCachePath(t)
	origVersion := version
	version = "1.0.0"
	defer func() { version = origVersion }()

	data, err := json.Marshal(updateCache{CheckedAt: time.Now(), Latest: "v9.9.9"})
	if err != nil {
		t.Fatalf("marshaling cache: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("seeding cache: %v", err)
	}

	var buf bytes.Buffer
	notifyUpdateAvailable(&buf)

	if !bytes.Contains(buf.Bytes(), []byte("v9.9.9")) {
		t.Errorf("notifyUpdateAvailable: got %q, want it to mention v9.9.9", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("-selfupdate")) {
		t.Errorf("notifyUpdateAvailable: got %q, want it to mention -selfupdate", buf.String())
	}
}

func TestNotifyUpdateAvailable_UpToDateCacheIsSilent(t *testing.T) {
	path := withUpdateCachePath(t)
	origVersion := version
	version = "9.9.9"
	defer func() { version = origVersion }()

	data, err := json.Marshal(updateCache{CheckedAt: time.Now(), Latest: "v9.9.9"})
	if err != nil {
		t.Fatalf("marshaling cache: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("seeding cache: %v", err)
	}

	var buf bytes.Buffer
	notifyUpdateAvailable(&buf)

	if buf.Len() != 0 {
		t.Errorf("notifyUpdateAvailable: got %q, want no output when already up to date", buf.String())
	}
}

func TestRefreshUpdateCache(t *testing.T) {
	path := withUpdateCachePath(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v2.3.4","assets":[]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	origURL, origClient := releasesURL, httpClient
	releasesURL = srv.URL + "/latest"
	httpClient = srv.Client()
	defer func() { releasesURL, httpClient = origURL, origClient }()

	refreshUpdateCache()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading cache: %v", err)
	}
	var c updateCache
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("unmarshaling cache: %v", err)
	}
	if c.Latest != "v2.3.4" {
		t.Errorf("refreshUpdateCache: cached latest = %q, want %q", c.Latest, "v2.3.4")
	}
}
