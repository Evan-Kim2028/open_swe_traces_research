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
	"testing"

	"github.com/google/go-github/v92/github"
)

// doWithLink performs a client.Do against a test server that answers with the
// given Link response header, returning the populated *github.Response.
func doWithLink(t *testing.T, linkHeader string) *github.Response {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/x", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", linkHeader)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	base := srv.URL + "/"
	client, err := github.NewClient(github.WithURLs(&base, &base))
	if err != nil {
		t.Fatalf("github.NewClient returned error: %v", err)
	}
	req, err := client.NewRequest(context.Background(), http.MethodGet, "x", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	resp, err := client.Do(req, nil)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	return resp
}

// Detail 1: a link carrying a cursor= param populates Response.Cursor on the
// next link. Inferable: no — asserted as shape only: a cursor= link is
// honored (the cursor value reaches the Cursor field) and a cursor-only link
// is not consulted for the integer page fields. The "only rel=next picks up
// cursor" exclusion is NOT pinned: a cursor on a non-next rel is left
// unasserted.
func TestDetail01(t *testing.T) {
	resp := doWithLink(t, `<https://api.github.com/x?cursor=abc123>; rel="next"`)
	if resp.Cursor != "abc123" {
		t.Errorf("cursor= link: Response.Cursor = %q, want %q", resp.Cursor, "abc123")
	}
	if resp.NextPage != 0 {
		t.Errorf("cursor-only link must not set NextPage, got %d", resp.NextPage)
	}

	// A cursor on a non-next rel must not corrupt the page fields; whether it
	// populates Cursor is left unasserted (the rel-scoping rule is arbitrary).
	resp = doWithLink(t, `<https://api.github.com/x?cursor=zzz>; rel="prev"`)
	if resp.PrevPage != 0 {
		t.Errorf("cursor link with no page param set PrevPage = %d, want 0", resp.PrevPage)
	}
}

// Detail 2: a since= param acts as a page= fallback. Inferable: no — the
// since→page alias is arbitrary, but the bug report states since-style links
// must paginate, so a numeric since on a rel link must reach that rel's page
// field; page= precedence over since= is asserted as "the explicit page
// value wins".
func TestDetail02(t *testing.T) {
	resp := doWithLink(t, `<https://api.github.com/x?since=2>; rel="prev"`)
	if resp.PrevPage != 2 {
		t.Errorf("since=2 rel=prev: Response.PrevPage = %d, want 2", resp.PrevPage)
	}

	resp = doWithLink(t, `<https://api.github.com/x?since=2021-12-04T10:43:42Z&page=4>; rel="next"`)
	if resp.NextPage != 4 {
		t.Errorf("since+page rel=next: Response.NextPage = %d, want 4 (explicit page wins)", resp.NextPage)
	}
}

// Detail 3: links where none of the pagination params is present are skipped
// entirely. Inferable: partially — asserted that a link carrying no
// recognized pagination param leaves every page field at its zero value;
// which params qualify is not pinned beyond the param-free case.
func TestDetail03(t *testing.T) {
	resp := doWithLink(t, `<https://api.github.com/x>; rel="next"`)
	if resp.NextPage != 0 || resp.Cursor != "" || resp.Before != "" || resp.After != "" || resp.NextPageToken != "" {
		t.Errorf("param-free link set fields: %+v", resp)
	}

	resp = doWithLink(t, `<https://api.github.com/x?unrelated=1>; rel="prev"`)
	if resp.PrevPage != 0 || resp.Cursor != "" || resp.Before != "" || resp.After != "" || resp.NextPageToken != "" {
		t.Errorf("link with unrelated param set fields: %+v", resp)
	}
}

// Detail 4: a page= value that fails integer parse populates NextPageToken
// instead of NextPage. Inferable: partially — the field is derivable from the
// exported API surface.
func TestDetail04(t *testing.T) {
	resp := doWithLink(t, `<https://api.github.com/x?page=abc>; rel="next"`)
	if resp.NextPage != 0 {
		t.Errorf("non-integer page set NextPage = %d, want 0", resp.NextPage)
	}
	if resp.NextPageToken != "abc" {
		t.Errorf("non-integer page: NextPageToken = %q, want %q", resp.NextPageToken, "abc")
	}
}

// Detail 5: malformed links are skipped, not errors. Inferable: partially —
// asserted that malformed segments in a Link header do not disturb the
// parsing of well-formed links; the exact validity rules are not pinned.
func TestDetail05(t *testing.T) {
	link := `garbage-segment, <https://api.github.com/x?page=2>; rel="next", <missing-close, ` +
		`<https://api.github.com/y?before=B1>; rel="prev"`
	resp := doWithLink(t, link)
	if resp.NextPage != 2 {
		t.Errorf("valid next link amid malformed segments: NextPage = %d, want 2", resp.NextPage)
	}
	if resp.Before != "B1" {
		t.Errorf("valid prev link amid malformed segments: Before = %q, want %q", resp.Before, "B1")
	}
}

// Detail 6: before=/after= params land on Response.Before/Response.After from
// their respective prev/next rels regardless of numeric parsing. Inferable:
// partially — the named pairings are asserted (superset implementations that
// read the params from additional rels still satisfy them); exclusivity of
// the pairing is not pinned.
func TestDetail06(t *testing.T) {
	resp := doWithLink(t, `<https://api.github.com/x?before=B1>; rel="prev", <https://api.github.com/x?after=A2&page=3>; rel="next"`)
	if resp.Before != "B1" {
		t.Errorf("before= on prev rel: Before = %q, want %q", resp.Before, "B1")
	}
	if resp.After != "A2" {
		t.Errorf("after= on next rel: After = %q, want %q", resp.After, "A2")
	}
	if resp.NextPage != 3 {
		t.Errorf("page= on next rel: NextPage = %d, want 3", resp.NextPage)
	}
}
