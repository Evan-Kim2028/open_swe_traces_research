// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v92/github"
)

// upCapture records the request it is asked to send and replies 200.
type upCapture struct{ got *http.Request }

func (c *upCapture) RoundTrip(req *http.Request) (*http.Response, error) {
	c.got = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("{}")),
		Request:    req,
	}, nil
}

func upClient(t *testing.T, base, upload string, cap *upCapture) *github.Client {
	t.Helper()
	client, err := github.NewClient(
		github.WithHTTPClient(&http.Client{Transport: cap}),
		github.WithAuthToken("tok123"),
		github.WithURLs(github.Ptr(base), github.Ptr(upload)),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

// doAbs issues a GET to an absolute URL through the client and reports the
// Authorization header observed by the underlying transport.
func doAbs(t *testing.T, client *github.Client, cap *upCapture, absURL string) string {
	t.Helper()
	req, err := client.NewRequest(context.Background(), http.MethodGet, absURL, nil)
	if err != nil {
		t.Fatalf("NewRequest %q: %v", absURL, err)
	}
	if _, err := client.Do(req, nil); err != nil {
		t.Fatalf("Do %q: %v", absURL, err)
	}
	return cap.got.Header.Get("Authorization")
}

// Detail 1: a literal ".." path segment is refused with ErrPathForbidden;
// ".." inside a segment or in the query is fine. (Inferable: doc)
func TestDetail01(t *testing.T) {
	cap := &upCapture{}
	client := upClient(t, "https://api.example.com/", "https://uploads.example.com/", cap)

	for _, u := range []string{"a/../b", "../x", "a/../../b"} {
		if _, err := client.NewRequest(context.Background(), http.MethodGet, u, nil); !errors.Is(err, github.ErrPathForbidden) {
			t.Errorf("NewRequest(%q) error = %v, want ErrPathForbidden", u, err)
		}
		if _, err := client.NewUploadRequest(context.Background(), u, strings.NewReader("x"), 1, "text/plain"); !errors.Is(err, github.ErrPathForbidden) {
			t.Errorf("NewUploadRequest(%q) error = %v, want ErrPathForbidden", u, err)
		}
	}
	for _, u := range []string{"file..txt", "a/file..txt/b", "x?next=..", "a/b"} {
		if _, err := client.NewRequest(context.Background(), http.MethodGet, u, nil); err != nil {
			t.Errorf("NewRequest(%q) error = %v, want nil", u, err)
		}
	}
}

// Detail 2: origins compare equal case-insensitively on scheme and hostname,
// with explicit ports normalized against scheme defaults. Inferable:
// partially — the normalization rule is documented; asserted via whether
// credentials flow to each URL, not via any error literal.
func TestDetail02(t *testing.T) {
	cap := &upCapture{}
	client := upClient(t, "https://api.example.com/", "https://uploads.example.com/", cap)

	for _, u := range []string{
		"https://API.EXAMPLE.COM/x",
		"https://api.example.com:443/x",
	} {
		if got := doAbs(t, client, cap, u); got == "" {
			t.Errorf("credentials withheld from %q, want authorized", u)
		}
	}
	for _, u := range []string{
		"http://api.example.com/x",
		"https://api.example.com:444/x",
		"https://other.example.com/x",
	} {
		if got := doAbs(t, client, cap, u); got != "" {
			t.Errorf("credentials sent to %q, want withheld", u)
		}
	}
}

// Detail 3: an empty allow-list on the credential transports means the
// package default origins, not "allow all". (Inferable: doc)
func TestDetail03(t *testing.T) {
	cap := &upCapture{}
	tr := &github.BasicAuthTransport{Username: "u", Password: "p", Transport: cap}

	for _, u := range []string{"https://api.github.com/x", "https://uploads.github.com/x"} {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tr.RoundTrip(req); err != nil {
			t.Fatalf("RoundTrip %q: %v", u, err)
		}
		if got := cap.got.Header.Get("Authorization"); got == "" {
			t.Errorf("credentials withheld from default origin %q", u)
		}
	}

	req, err := http.NewRequest(http.MethodGet, "https://other.example.com/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if got := cap.got.Header.Get("Authorization"); got != "" {
		t.Errorf("credentials sent to https://other.example.com with empty allow-list")
	}
}

// Detail 4: credentials may flow to the configured base or upload origin;
// anything else is out of scope. (Inferable: doc)
func TestDetail04(t *testing.T) {
	cap := &upCapture{}
	client := upClient(t, "https://api.example.com/", "https://uploads.example.com/", cap)

	for _, u := range []string{"https://api.example.com/x", "https://uploads.example.com/x"} {
		if got := doAbs(t, client, cap, u); got == "" {
			t.Errorf("credentials withheld from configured origin %q", u)
		}
	}
	if got := doAbs(t, client, cap, "https://elsewhere.example.com/x"); got != "" {
		t.Error("credentials sent to an unconfigured origin")
	}
}

// Detail 5: sending a body to an out-of-scope destination is refused with
// ErrUntrustedDestination carrying the (redacted) URL. Inferable: partially —
// the refusal and the error identity are documented; asserted via errors.Is
// plus the destination host appearing in the message, not an exact literal.
func TestDetail05(t *testing.T) {
	cap := &upCapture{}
	client := upClient(t, "https://api.example.com/", "https://uploads.example.com/", cap)

	_, err := client.NewUploadRequest(context.Background(), "https://evil.example.com/x", strings.NewReader("x"), 1, "text/plain")
	if !errors.Is(err, github.ErrUntrustedDestination) {
		t.Fatalf("NewUploadRequest to foreign origin: error = %v, want ErrUntrustedDestination", err)
	}
	if !strings.Contains(err.Error(), "evil.example.com") {
		t.Errorf("refusal does not name the destination: %q", err.Error())
	}

	// Configured and relative destinations are accepted.
	if _, err := client.NewUploadRequest(context.Background(), "https://uploads.example.com/x", strings.NewReader("x"), 1, "text/plain"); err != nil {
		t.Errorf("NewUploadRequest to upload origin: %v", err)
	}
	if _, err := client.NewUploadRequest(context.Background(), "rel/path", strings.NewReader("x"), 1, "text/plain"); err != nil {
		t.Errorf("NewUploadRequest relative: %v", err)
	}
	if _, err := client.NewFormRequest(context.Background(), "https://evil.example.com/x", strings.NewReader("a=b")); !errors.Is(err, github.ErrUntrustedDestination) {
		t.Errorf("NewFormRequest to foreign origin: error = %v, want ErrUntrustedDestination", err)
	}
}
