package github

import (
	"errors"

	"strings"
	"testing"
)

// TestDetail01: a urlStr resolving to a host outside the configured origins is
// rejected with ErrUntrustedDestination before the body is posted.
func TestDetail01(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	_, err := c.NewFormRequest(t.Context(), "https://untrusted.example.com/collect", strings.NewReader("a=b"))
	if !errors.Is(err, ErrUntrustedDestination) {
		t.Fatalf("NewFormRequest foreign host: err = %v, want ErrUntrustedDestination", err)
	}
}

// TestDetail02: a urlStr that resolves to a configured destination is accepted.
func TestDetail02(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	req, err := c.NewFormRequest(t.Context(), "hub", strings.NewReader("a=b"))
	if err != nil {
		t.Fatalf("NewFormRequest relative: %v", err)
	}
	if req.Method != "POST" {
		t.Fatalf("method = %v, want POST", req.Method)
	}

	// Absolute URL at the client's own configured origin is accepted.
	req, err = c.NewFormRequest(t.Context(), c.baseURL.String()+"hub", strings.NewReader("a=b"))
	if err != nil {
		t.Fatalf("NewFormRequest same-origin absolute: %v", err)
	}
	if req.Method != "POST" {
		t.Fatalf("method = %v, want POST", req.Method)
	}
}

// TestDetail03: X-Github-Api-Version is set to the client's default API version.
func TestDetail03(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	req, err := c.NewFormRequest(t.Context(), "hub", strings.NewReader("a=b"))
	if err != nil {
		t.Fatalf("NewFormRequest: %v", err)
	}
	if got := req.Header.Get(headerAPIVersion); got != c.apiVersionDefault {
		t.Fatalf("%v = %q, want %q", headerAPIVersion, got, c.apiVersionDefault)
	}
}

// TestDetail04: User-Agent is omitted entirely when the client's is empty.
func TestDetail04(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)
	c.userAgent = ""

	req, err := c.NewFormRequest(t.Context(), "hub", strings.NewReader("a=b"))
	if err != nil {
		t.Fatalf("NewFormRequest: %v", err)
	}
	if _, ok := req.Header["User-Agent"]; ok {
		t.Fatalf("User-Agent header present with empty value: %q", req.Header.Get("User-Agent"))
	}
}

// TestDetail05: Content-Type conveys the encoded form produced by the caller's
// body — per the method's documented contract, urlencoded form data.
func TestDetail05(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	req, err := c.NewFormRequest(t.Context(), "hub", strings.NewReader("login=l"))
	if err != nil {
		t.Fatalf("NewFormRequest: %v", err)
	}
	if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		t.Fatalf("Content-Type = %q, want application/x-www-form-urlencoded", got)
	}
}
