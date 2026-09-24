// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v92/github"
)

// captureRT records the request it is asked to send and replies 200.
type captureRT struct{ got *http.Request }

func (c *captureRT) RoundTrip(req *http.Request) (*http.Response, error) {
	c.got = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

// Detail 1: the credential-carrying request is a copy — the caller's request
// is never mutated. (Inferable: yes — RoundTripper contract.)
func TestDetail01(t *testing.T) {
	cap := &captureRT{}
	tr := &github.BasicAuthTransport{Username: "u", Password: "p", Transport: cap}

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Custom", "keep")
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("original request mutated: Authorization = %q", got)
	}
	if got := req.Header.Get("X-Custom"); got != "keep" {
		t.Errorf("original request lost X-Custom: %q", got)
	}
}

// Detail 2: the copy carries a Basic-Auth Authorization header built from the
// id/secret arguments. (Inferable: yes)
func TestDetail02(t *testing.T) {
	cap := &captureRT{}
	tr := &github.BasicAuthTransport{Username: "myid", Password: "mysecret", Transport: cap}

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("myid:mysecret"))
	if got := cap.got.Header.Get("Authorization"); got != want {
		t.Errorf("sent request Authorization = %q, want %q", got, want)
	}

	// Same via the OAuth-app transport.
	cap2 := &captureRT{}
	tr2 := &github.UnauthenticatedRateLimitedTransport{
		ClientID: "cid", ClientSecret: "csec", Transport: cap2,
	}
	req2, err := http.NewRequest(http.MethodGet, "https://api.github.com/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr2.RoundTrip(req2); err != nil {
		t.Fatalf("UnauthenticatedRateLimitedTransport.RoundTrip returned error: %v", err)
	}
	want2 := "Basic " + base64.StdEncoding.EncodeToString([]byte("cid:csec"))
	if got := cap2.got.Header.Get("Authorization"); got != want2 {
		t.Errorf("rate-limited transport Authorization = %q, want %q", got, want2)
	}
}

// Detail 3: the copied request's Header map is independent of the
// original's — the credential lands on the copy, not the caller's map.
// Inferable: partially — asserted that after RoundTrip the caller's Header
// carries no Authorization at all.
func TestDetail03(t *testing.T) {
	cap := &captureRT{}
	tr := &github.BasicAuthTransport{Username: "u", Password: "p", Transport: cap}

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	before := len(req.Header)
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	if len(req.Header) != before {
		t.Errorf("original Header map grew: %d -> %d entries", before, len(req.Header))
	}
	if _, present := req.Header["Authorization"]; present {
		t.Error("original Header map gained an Authorization key")
	}
	if cap.got == req {
		t.Error("transport sent the caller's *http.Request, not a copy")
	}
}

// Detail 4: transports consult an origin-scope check before attaching
// credentials — they flow only to requests targeting an allowed origin.
// Inferable: partially — asserted that an obviously foreign origin receives
// no credentials while the documented default origin does; the exact origin
// matching rule is the sibling unit's commitment and is not pinned here.
func TestDetail04(t *testing.T) {
	// Foreign origin: credentials must not be attached.
	cap := &captureRT{}
	tr := &github.BasicAuthTransport{Username: "u", Password: "p", Transport: cap}
	req, err := http.NewRequest(http.MethodGet, "https://evil.example.com/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	if got := cap.got.Header.Get("Authorization"); got != "" {
		t.Errorf("foreign origin received credentials: %q", got)
	}

	// Default allowed origin (documented on the field): credentials flow.
	cap = &captureRT{}
	tr = &github.BasicAuthTransport{Username: "u", Password: "p", Transport: cap}
	req, err = http.NewRequest(http.MethodGet, "https://api.github.com/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	if got := cap.got.Header.Get("Authorization"); got == "" {
		t.Error("default origin did not receive credentials")
	}
}
