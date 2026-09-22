// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/google/go-github/v92/github"
)

// patServer returns a client plus a pointer to the RawQuery of the last
// request, and a hit counter.
func patServer(t *testing.T) (*github.Client, *atomic.Value, *atomic.Int64) {
	t.Helper()
	var rawq atomic.Value
	var hits atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/orgs/o/personal-access-tokens", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		rawq.Store(r.URL.RawQuery)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[]`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	base := srv.URL + "/"
	client, err := github.NewClient(github.WithURLs(&base, &base))
	if err != nil {
		t.Fatalf("github.NewClient returned error: %v", err)
	}
	return client, &rawq, &hits
}

func lastQuery(t *testing.T, rawq *atomic.Value) url.Values {
	t.Helper()
	v := rawq.Load()
	if v == nil {
		t.Fatal("no request reached the server")
	}
	q, err := url.ParseQuery(v.(string))
	if err != nil {
		t.Fatalf("query %q did not parse: %v", v.(string), err)
	}
	return q
}

// Detail 1: each Owner entry becomes its own owner[]= pair, order preserved.
// Inferable: partially — the repeated-pair shape and the documented
// owner[]= spelling (kept in the excised doc comment) are asserted.
func TestDetail01(t *testing.T) {
	client, rawq, _ := patServer(t)
	opts := &github.ListFineGrainedPATOptions{Owner: []string{"alice", "bob"}}
	if _, _, err := client.Organizations.ListFineGrainedPersonalAccessTokens(context.Background(), "o", opts); err != nil {
		t.Fatalf("ListFineGrainedPersonalAccessTokens returned error: %v", err)
	}
	q := lastQuery(t, rawq)
	got := q["owner[]"]
	if len(got) != 2 || got[0] != "alice" || got[1] != "bob" {
		t.Errorf("owner[] values = %v, want [alice bob] in order", got)
	}
}

// Detail 2: each TokenID entry becomes its own token_id[]= pair.
// (Inferable: partially — same convention, documented in the surviving
// comment.)
func TestDetail02(t *testing.T) {
	client, rawq, _ := patServer(t)
	opts := &github.ListFineGrainedPATOptions{TokenID: []int64{123, 456}}
	if _, _, err := client.Organizations.ListFineGrainedPersonalAccessTokens(context.Background(), "o", opts); err != nil {
		t.Fatalf("ListFineGrainedPersonalAccessTokens returned error: %v", err)
	}
	q := lastQuery(t, rawq)
	got := q["token_id[]"]
	if len(got) != 2 || got[0] != "123" || got[1] != "456" {
		t.Errorf("token_id[] values = %v, want [123 456] in order", got)
	}
}

// Detail 3: the array params join the existing query rather than replacing
// it. (Inferable: yes — asserted that array params and generic option params
// coexist in the final query.)
func TestDetail03(t *testing.T) {
	client, rawq, _ := patServer(t)
	opts := &github.ListFineGrainedPATOptions{
		Owner:  []string{"alice"},
		Sort:   "created_at",
		Repository: "repo1",
	}
	if _, _, err := client.Organizations.ListFineGrainedPersonalAccessTokens(context.Background(), "o", opts); err != nil {
		t.Fatalf("ListFineGrainedPersonalAccessTokens returned error: %v", err)
	}
	q := lastQuery(t, rawq)
	if q.Get("sort") != "created_at" || q.Get("repository") != "repo1" {
		t.Errorf("generic options missing from query: %v", q)
	}
	if len(q["owner[]"]) != 1 || q["owner[]"][0] != "alice" {
		t.Errorf("owner[] missing from query: %v", q)
	}
}

// Detail 4: a nil opts returns an error rather than a bare listing URL.
// Inferable: no — asserted as shape only: an error surfaces and no request
// is made.
func TestDetail04(t *testing.T) {
	client, _, hits := patServer(t)
	_, _, err := client.Organizations.ListFineGrainedPersonalAccessTokens(context.Background(), "o", nil)
	if err == nil {
		t.Error("nil opts: ListFineGrainedPersonalAccessTokens returned nil error")
	}
	if hits.Load() != 0 {
		t.Error("nil-opts request reached the server")
	}
}

// Detail 5: other option fields still flow through the generic addOptions
// encoder. (Inferable: yes)
func TestDetail05(t *testing.T) {
	client, rawq, _ := patServer(t)
	opts := &github.ListFineGrainedPATOptions{
		Direction:      "asc",
		Permission:     "contents",
		LastUsedBefore: "2026-01-01T00:00:00Z",
	}
	if _, _, err := client.Organizations.ListFineGrainedPersonalAccessTokens(context.Background(), "o", opts); err != nil {
		t.Fatalf("ListFineGrainedPersonalAccessTokens returned error: %v", err)
	}
	q := lastQuery(t, rawq)
	for k, want := range map[string]string{
		"direction":        "asc",
		"permission":       "contents",
		"last_used_before": "2026-01-01T00:00:00Z",
	} {
		if q.Get(k) != want {
			t.Errorf("query param %s = %q, want %q (raw %v)", k, q.Get(k), want, q)
		}
	}
}
