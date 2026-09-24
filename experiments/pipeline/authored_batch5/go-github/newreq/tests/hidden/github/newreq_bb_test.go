package github

import (
	"testing"
)

// TestDetail01: every NewRequest carries X-Github-Api-Version equal to the
// client's configured default API version.
func TestDetail01(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)

	req, err := c.NewRequest(t.Context(), "GET", "foo", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	got := req.Header.Get(headerAPIVersion)
	if got == "" {
		t.Fatalf("NewRequest did not set %v", headerAPIVersion)
	}
	if got != c.apiVersionDefault {
		t.Fatalf("%v = %q, want client default %q", headerAPIVersion, got, c.apiVersionDefault)
	}
}

// TestDetail02: when the client's user-agent is empty the User-Agent header is
// omitted entirely rather than sent empty.
func TestDetail02(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t)
	c.userAgent = ""

	req, err := c.NewRequest(t.Context(), "GET", "foo", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, ok := req.Header["User-Agent"]; ok {
		t.Fatalf("User-Agent header present with empty value: %q", req.Header.Get("User-Agent"))
	}
}

// TestDetail03: a non-empty user-agent is sent verbatim.
func TestDetail03(t *testing.T) {
	t.Parallel()
	c := mustNewClient(t, WithUserAgent("bb-agent/1.0"))

	req, err := c.NewRequest(t.Context(), "GET", "foo", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if got := req.Header.Get("User-Agent"); got != "bb-agent/1.0" {
		t.Fatalf("User-Agent = %q, want %q", got, "bb-agent/1.0")
	}
}
