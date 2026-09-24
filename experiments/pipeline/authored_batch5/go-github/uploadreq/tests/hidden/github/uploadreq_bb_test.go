package github

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// TestDetail01: a urlStr containing ".." path segments is rejected before any
// request is built. (Inferable: partially — assert rejection, not which error.)
func TestDetail01(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	if _, err := c.NewUploadRequest(t.Context(), "repos/x/../../../admin", strings.NewReader("x"), 1, ""); err == nil {
		t.Fatal("NewUploadRequest with '..' path segments: err = nil, want rejection")
	}
	if _, err := c.NewUploadRequest(t.Context(), "a/%2e%2e/b", strings.NewReader("x"), 1, ""); err == nil {
		t.Fatal("NewUploadRequest with encoded '..' segments: err = nil, want rejection")
	}
}

// TestDetail02: an absolute urlStr naming a non-configured host is rejected;
// the configured upload origin is accepted.
func TestDetail02(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	_, err := c.NewUploadRequest(t.Context(), "https://untrusted.example.com/up", strings.NewReader("x"), 1, "")
	if !errors.Is(err, ErrUntrustedDestination) {
		t.Fatalf("NewUploadRequest foreign host: err = %v, want ErrUntrustedDestination", err)
	}

	req, err := c.NewUploadRequest(t.Context(), c.uploadURL.String()+"up/x", strings.NewReader("x"), 1, "")
	if err != nil {
		t.Fatalf("NewUploadRequest configured origin: %v", err)
	}
	if req.Method != "POST" {
		t.Fatalf("method = %v, want POST", req.Method)
	}
}

// bbReadSeekCloser is a caller-owned type the transport could observe if the
// reader were passed through unwrapped.
type bbReadSeekCloser struct{ *strings.Reader }

func (bbReadSeekCloser) Close() error { return nil }

// bbSeekReaderAt implements io.Seeker + io.ReaderAt — the rewindable pair.
type bbSeekReaderAt struct{ *strings.Reader }

// TestDetail03: the caller's reader is wrapped so the transport cannot
// observe concrete body types — a seekable input is hidden behind a
// non-seekable facade. (Inferable: no — asserted as observable behavior.)
func TestDetail03(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	req, err := c.NewUploadRequest(t.Context(), "up/x", bbReadSeekCloser{strings.NewReader("0123456789")}, 10, "")
	if err != nil {
		t.Fatalf("NewUploadRequest: %v", err)
	}
	if _, ok := req.Body.(io.Seeker); ok {
		t.Fatal("request body exposes io.Seeker: concrete reader type leaked to transport")
	}
	if _, ok := req.Body.(*bbReadSeekCloser); ok {
		t.Fatal("request body is the caller's concrete type, not a facade")
	}
}

// TestDetail04: GetBody is set when the reader supports rewinding, returning
// an independent view that starts at the offset the reader had at build time.
func TestDetail04(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	r := bbSeekReaderAt{strings.NewReader("hello world")}
	if _, err := r.Seek(6, io.SeekStart); err != nil {
		t.Fatalf("seek: %v", err)
	}
	req, err := c.NewUploadRequest(t.Context(), "up/x", r, 5, "")
	if err != nil {
		t.Fatalf("NewUploadRequest: %v", err)
	}
	if req.GetBody == nil {
		t.Fatal("GetBody not set for a seekable reader")
	}
	for i := 0; i < 2; i++ {
		rc, err := req.GetBody()
		if err != nil {
			t.Fatalf("GetBody: %v", err)
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("GetBody read: %v", err)
		}
		if string(b) != "world" {
			t.Fatalf("GetBody %d = %q, want %q (recorded offset)", i, b, "world")
		}
	}
}

// TestDetail05: GetBody is absent for a non-seekable reader.
func TestDetail05(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	r := io.MultiReader(strings.NewReader("abc"), strings.NewReader("def"))
	req, err := c.NewUploadRequest(t.Context(), "up/x", r, 6, "")
	if err != nil {
		t.Fatalf("NewUploadRequest: %v", err)
	}
	if req.GetBody != nil {
		t.Fatal("GetBody set for a non-seekable reader")
	}
}

// TestDetail06: ContentLength is set to size.
func TestDetail06(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	req, err := c.NewUploadRequest(t.Context(), "up/x", strings.NewReader("abc"), 3, "")
	if err != nil {
		t.Fatalf("NewUploadRequest: %v", err)
	}
	if req.ContentLength != 3 {
		t.Fatalf("ContentLength = %v, want 3", req.ContentLength)
	}
}

// TestDetail07: mediaType defaults when empty and is honored when given.
func TestDetail07(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	req, err := c.NewUploadRequest(t.Context(), "up/x", strings.NewReader("x"), 1, "")
	if err != nil {
		t.Fatalf("NewUploadRequest: %v", err)
	}
	if got := req.Header.Get("Content-Type"); got != defaultMediaType {
		t.Fatalf("empty mediaType: Content-Type = %q, want %q", got, defaultMediaType)
	}

	req, err = c.NewUploadRequest(t.Context(), "up/x", strings.NewReader("x"), 1, "text/plain")
	if err != nil {
		t.Fatalf("NewUploadRequest: %v", err)
	}
	if got := req.Header.Get("Content-Type"); got != "text/plain" {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}
}
