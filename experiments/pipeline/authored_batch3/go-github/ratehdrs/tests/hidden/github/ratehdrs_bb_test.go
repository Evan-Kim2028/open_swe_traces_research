// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-github/v92/github"
)

// rateServer replies to any request with the given headers and a 200.
func rateServer(t *testing.T, hdr http.Header) *github.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, vs := range hdr {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client, err := github.NewClient(github.WithURLs(github.Ptr(srv.URL+"/"), github.Ptr(srv.URL+"/")))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func doGet(t *testing.T, client *github.Client) *github.Response {
	t.Helper()
	req, err := client.NewRequest(context.Background(), http.MethodGet, "", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

// Detail 1: limit, remaining, used and resource are read from their headers
// when present. (Inferable: yes — the header names are surviving constants.)
func TestDetail01(t *testing.T) {
	client := rateServer(t, http.Header{
		github.HeaderRateLimit:     {"60"},
		github.HeaderRateRemaining: {"42"},
		github.HeaderRateUsed:      {"18"},
		github.HeaderRateResource:  {"core"},
	})
	resp := doGet(t, client)
	if resp.Rate.Limit != 60 || resp.Rate.Remaining != 42 || resp.Rate.Used != 18 {
		t.Errorf("Rate = %+v, want Limit=60 Remaining=42 Used=18", resp.Rate)
	}
	if resp.Rate.Resource != "core" {
		t.Errorf("Rate.Resource = %q, want %q", resp.Rate.Resource, "core")
	}
}

// Detail 2: the reset header leaves Reset at the zero timestamp when absent
// or when it parses to 0. Inferable: no — asserted as shape only: the field
// stays unset (zero); a nonzero epoch parses normally as the committed
// complement.
func TestDetail02(t *testing.T) {
	// Absent.
	resp := doGet(t, rateServer(t, http.Header{github.HeaderRateLimit: {"60"}}))
	if !resp.Rate.Reset.Time.IsZero() {
		t.Errorf("absent reset header: Reset = %v, want zero", resp.Rate.Reset.Time)
	}
	// Parses to 0.
	resp = doGet(t, rateServer(t, http.Header{github.HeaderRateReset: {"0"}}))
	if !resp.Rate.Reset.Time.IsZero() {
		t.Errorf("reset header \"0\": Reset = %v, want zero", resp.Rate.Reset.Time)
	}
	// Nonzero epoch parses.
	resp = doGet(t, rateServer(t, http.Header{github.HeaderRateReset: {"1700000000"}}))
	if resp.Rate.Reset.Time.Unix() != 1700000000 {
		t.Errorf("reset header 1700000000: Reset = %v, want unix 1700000000", resp.Rate.Reset.Time)
	}
}

// Detail 3: the token-expiration header accepts the "YYYY-MM-DD HH:MM:SS
// MST" layout and the "YYYY-MM-DD HH:MM:SS -0700" numeric-offset layout,
// returning the instant. Inferable: partially — the two accepted layouts are
// forced by observed server formats; asserted with representative values.
func TestDetail03(t *testing.T) {
	const hdr = "Github-Authentication-Token-Expiration"

	client := rateServer(t, http.Header{hdr: {"2026-06-01 10:00:00 UTC"}})
	resp := doGet(t, client)
	want := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	if !resp.TokenExpiration.Time.Equal(want) {
		t.Errorf("zone-name layout: TokenExpiration = %v, want instant %v", resp.TokenExpiration.Time, want)
	}

	client = rateServer(t, http.Header{hdr: {"2026-06-01 10:00:00 +0230"}})
	resp = doGet(t, client)
	want = time.Date(2026, 6, 1, 7, 30, 0, 0, time.UTC)
	if !resp.TokenExpiration.Time.Equal(want) {
		t.Errorf("numeric-offset layout: TokenExpiration = %v, want instant %v", resp.TokenExpiration.Time, want)
	}
}

// Detail 4: a missing or unparseable expiration header yields the zero
// Timestamp. (Inferable: yes — documented above the function.)
func TestDetail04(t *testing.T) {
	const hdr = "Github-Authentication-Token-Expiration"

	resp := doGet(t, rateServer(t, http.Header{}))
	if !resp.TokenExpiration.Time.IsZero() {
		t.Errorf("missing header: TokenExpiration = %v, want zero", resp.TokenExpiration.Time)
	}
	for _, bad := range []string{"not-a-date", "2026/06/01", "soon"} {
		resp = doGet(t, rateServer(t, http.Header{hdr: {bad}}))
		if !resp.TokenExpiration.Time.IsZero() {
			t.Errorf("unparseable header %q: TokenExpiration = %v, want zero", bad, resp.TokenExpiration.Time)
		}
	}
}
