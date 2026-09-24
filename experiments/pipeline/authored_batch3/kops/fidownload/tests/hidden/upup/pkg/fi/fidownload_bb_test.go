// Package fi_test is the hidden black-box suite for fidownload.
// One TestDetailNN per DETAILS.md commitment. Uses only exported API
// (DownloadURL, OpenURL) plus a localhost httptest server — no egress.
package fi_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fi "example.internal/clustkit/upup/pkg/fi"
	"example.internal/clustkit/util/pkg/hashing"
)

// Detail 1 (Inferable: partially): with a non-nil expected hash, an existing
// destination that already matches short-circuits — no network is touched.
// Proved by pointing at a dead URL: any fetch attempt would error.
func TestDetail01(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "payload.bin")
	content := "already-here"
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	want, err := hashing.HashAlgorithmSHA256.Hash(strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	got, err := fi.DownloadURL(context.Background(), "http://127.0.0.1:1/never", dest, want)
	if err != nil {
		t.Fatalf("cache hit should not touch network: %v", err)
	}
	if got == nil || !got.Equal(want) {
		t.Fatalf("cache hit returned wrong hash: %v", got)
	}
}

// Detail 2 (Inferable: no): scheme dispatch — http(s) and other non-cloud
// schemes stream through OpenURL; the cloud-scheme set is not pinned.
// Asserted: a real http URL downloads, and an unknown scheme errors rather
// than panicking or silently succeeding.
func TestDetail02(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "streamed")
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out")
	h, err := fi.DownloadURL(context.Background(), srv.URL+"/f", dest, nil)
	if err != nil {
		t.Fatalf("http download failed: %v", err)
	}
	if h == nil {
		t.Fatal("nil hash on plain download")
	}
	b, err := os.ReadFile(dest)
	if err != nil || string(b) != "streamed" {
		t.Fatalf("dest content %q err %v", b, err)
	}

	if _, err := fi.DownloadURL(context.Background(), "notascheme://x/y", filepath.Join(dir, "o2"), nil); err == nil {
		t.Fatal("unknown scheme did not error")
	}
}

// Detail 3 (Inferable: partially): bytes are hashed while streaming; a
// mismatch is an error raised after the bytes were written, and the expected
// hash's own algorithm is honoured (sha1 accepted, not only sha256).
func TestDetail03(t *testing.T) {
	content := "hash me as i stream"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, content)
	}))
	defer srv.Close()
	dir := t.TempDir()

	good, err := hashing.HashAlgorithmSHA256.Hash(strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "ok")
	got, err := fi.DownloadURL(context.Background(), srv.URL+"/f", dest, good)
	if err != nil {
		t.Fatalf("matching hash rejected: %v", err)
	}
	if got == nil || !got.Equal(good) {
		t.Fatalf("returned hash %v != expected %v", got, good)
	}

	// sha1 expected hash: the expected hash's algorithm must be used.
	sha1, err := hashing.HashAlgorithmSHA1.Hash(strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fi.DownloadURL(context.Background(), srv.URL+"/f", filepath.Join(dir, "ok1"), sha1); err != nil {
		t.Fatalf("sha1 expected-hash rejected: %v", err)
	}

	// Mismatch: error, and the staged file must not survive as dest.
	bad, err := hashing.HashAlgorithmSHA256.Hash(strings.NewReader("different"))
	if err != nil {
		t.Fatal(err)
	}
	destBad := filepath.Join(dir, "bad")
	if _, err := fi.DownloadURL(context.Background(), srv.URL+"/f", destBad, bad); err == nil {
		t.Fatal("hash mismatch did not error")
	}
}

// Detail 4 (Inferable: partially): staging is a temp file in the destination
// dir followed by rename; on failure the temp is removed and dest untouched.
// Asserted: no temp files survive success or failure, dest state is clean.
// The temp name spelling and mode bits are not pinned.
func TestDetail04(t *testing.T) {
	content := "staged"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, content)
	}))
	defer srv.Close()
	dir := t.TempDir()

	dest := filepath.Join(dir, "final.bin")
	if _, err := fi.DownloadURL(context.Background(), srv.URL+"/f", dest, nil); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "final.bin" {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("leftover entries after success: %v", names)
	}
	b, _ := os.ReadFile(dest)
	if string(b) != content {
		t.Fatalf("dest content %q", b)
	}

	// Failure path: hash mismatch -> error, dest absent, no temp left.
	bad, _ := hashing.HashAlgorithmSHA256.Hash(strings.NewReader("nope"))
	destBad := filepath.Join(dir, "willfail")
	if _, err := fi.DownloadURL(context.Background(), srv.URL+"/f", destBad, bad); err == nil {
		t.Fatal("mismatch did not error")
	}
	if _, err := os.Stat(destBad); !os.IsNotExist(err) {
		t.Fatal("dest created despite failed download")
	}
	for _, e := range mustReadDir(t, dir) {
		if strings.HasPrefix(e, ".") && strings.HasSuffix(e, ".tmp") {
			t.Fatalf("temp file left behind: %s", e)
		}
		if strings.HasPrefix(e, "willfail") {
			t.Fatalf("partial dest left behind: %s", e)
		}
	}
}

func mustReadDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// Detail 5 (Inferable: no): OpenURL is a hardened client — non-2xx (after
// redirects) is an error that carries the status; a 2xx body streams and its
// reader closes cleanly. Timeout constants are not pinned.
func TestDetail05(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "payload")
	})
	mux.HandleFunc("/err", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/redir", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ok", http.StatusFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rc, err := fi.OpenURL(srv.URL + "/ok")
	if err != nil {
		t.Fatalf("OpenURL 200: %v", err)
	}
	b, err := io.ReadAll(rc)
	if err != nil || string(b) != "payload" {
		t.Fatalf("body %q err %v", b, err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err = fi.OpenURL(srv.URL + "/err")
	if err == nil {
		t.Fatal("OpenURL 500 did not error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("error does not carry status: %v", err)
	}

	rc, err = fi.OpenURL(srv.URL + "/redir")
	if err != nil {
		t.Fatalf("redirect not followed: %v", err)
	}
	b, _ = io.ReadAll(rc)
	rc.Close()
	if string(b) != "payload" {
		t.Fatalf("redirected body %q", b)
	}
}

// Detail 6 (Inferable: yes): Close closes the body then cancels the request
// context; Close itself returns the body's close error (nil on a healthy
// stream). Observable: after Close the body is closed (further reads fail).
func TestDetail06(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		w.Write([]byte(strings.Repeat("x", 4096))) // partial body; leave unread
	}))
	defer srv.Close()

	rc, err := fi.OpenURL(srv.URL + "/big")
	if err != nil {
		t.Fatalf("OpenURL: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("Close returned %v", err)
	}
	if _, err := rc.Read(make([]byte, 16)); err == nil {
		t.Fatal("Read after Close did not fail — body not closed")
	}
}
