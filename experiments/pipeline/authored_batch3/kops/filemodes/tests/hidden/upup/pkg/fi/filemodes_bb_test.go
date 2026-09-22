// Package fi_test is the hidden black-box suite for filemodes.
// One TestDetailNN per DETAILS.md commitment. Exported API only;
// fileHasHash is exercised through DownloadURL + a localhost server.
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

// Detail 1 (Inferable: partially): ParseFileMode returns the default on empty
// input AND on parse failure (with an error); base-8 otherwise.
func TestDetail01(t *testing.T) {
	m, err := fi.ParseFileMode("", 0o640)
	if err != nil || m != 0o640 {
		t.Fatalf("empty input: got %v err %v, want default 0640", m, err)
	}
	m, err = fi.ParseFileMode("755", 0)
	if err != nil || m != 0o755 {
		t.Fatalf("755: got %o err %v", m, err)
	}
	for _, bad := range []string{"zzz", "888", "12x9", " "} {
		m, err := fi.ParseFileMode(bad, 0o600)
		if err == nil {
			t.Fatalf("%q: expected error", bad)
		}
		if m != 0o600 {
			t.Fatalf("%q: got mode %o, want default 0600 alongside error", bad, m)
		}
	}
}

// Detail 2 (Inferable: partially): FileModeToString renders octal. The
// leading-zero spelling is not pinned; what is asserted is a lossless
// round-trip through ParseFileMode (which pins octal-ness) and non-emptiness.
func TestDetail02(t *testing.T) {
	for _, m := range []os.FileMode{0o7, 0o644, 0o600, 0o755, 0o400, 0o1, 0o777} {
		s := fi.FileModeToString(m)
		if s == "" {
			t.Fatalf("FileModeToString(%o) empty", m)
		}
		back, err := fi.ParseFileMode(s, 0o123)
		if err != nil {
			t.Fatalf("FileModeToString(%o)=%q does not parse: %v", m, s, err)
		}
		if back != m&os.ModePerm {
			t.Fatalf("round-trip %o -> %q -> %o", m, s, back)
		}
	}
}

// Detail 3 (Inferable: partially): EnsureFileMode compares stat.Mode() &
// os.ModePerm via Lstat, chmods on drift, and reports changed honestly.
func TestDetail03(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := fi.EnsureFileMode(f, 0o644)
	if err != nil || !changed {
		t.Fatalf("drift: changed=%v err=%v", changed, err)
	}
	st, _ := os.Stat(f)
	if st.Mode()&os.ModePerm != 0o644 {
		t.Fatalf("mode now %o", st.Mode())
	}
	changed, err = fi.EnsureFileMode(f, 0o644)
	if err != nil || changed {
		t.Fatalf("no drift: changed=%v err=%v", changed, err)
	}

	// Type/setid bits ignored: a file whose perm bits already match reports
	// no drift even though raw mode differs.
	if err := os.Chmod(f, 0o4644); err != nil {
		t.Fatal(err)
	}
	changed, err = fi.EnsureFileMode(f, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("setuid-bit file with matching perm reported changed")
	}

	// Missing file: error, changed=false — not a panic, not silent.
	changed, err = fi.EnsureFileMode(filepath.Join(dir, "missing"), 0o644)
	if err == nil {
		t.Fatal("missing file: expected error")
	}
	if changed {
		t.Fatal("missing file reported changed")
	}
}

// Detail 4 (Inferable: partially): fileHasHash — missing file is (false,nil)
// and a hash mismatch is (false,nil): both must lead DownloadURL to fetch.
// Exercised black-box through the intact DownloadURL caller.
func TestDetail04(t *testing.T) {
	content := "server-bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, content)
	}))
	defer srv.Close()
	dir := t.TempDir()
	good, err := hashing.HashAlgorithmSHA256.Hash(strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}

	// Missing dest + expected hash: (false,nil) -> proceeds to download.
	dest := filepath.Join(dir, "new")
	if _, err := fi.DownloadURL(context.Background(), srv.URL+"/f", dest, good); err != nil {
		t.Fatalf("missing dest + hash: %v", err)
	}
	if b, _ := os.ReadFile(dest); string(b) != content {
		t.Fatalf("dest %q", b)
	}

	// Wrong content at dest: mismatch is (false,nil) -> re-download overwrites.
	if err := os.WriteFile(dest, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fi.DownloadURL(context.Background(), srv.URL+"/f", dest, good); err != nil {
		t.Fatalf("mismatched dest + hash: %v", err)
	}
	if b, _ := os.ReadFile(dest); string(b) != content {
		t.Fatalf("mismatched dest not overwritten: %q", b)
	}
}
