// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/go-github/v92/github"
)

// comfortFadeServer returns a client, a pointer to the Accept header observed
// on the last request, and a hit counter.
func comfortFadeServer(t *testing.T) (*github.Client, *atomic.Value, *atomic.Int64) {
	t.Helper()
	var accept atomic.Value
	var hits atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		accept.Store(r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id":1}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	base := srv.URL + "/"
	client, err := github.NewClient(github.WithURLs(&base, &base))
	if err != nil {
		t.Fatalf("github.NewClient returned error: %v", err)
	}
	return client, &accept, &hits
}

func strp(s string) *string { return &s }
func intp(i int) *int       { return &i }

// Detail 1: a comment is comfort-fade style iff a comfort-fade field is set;
// Position means legacy style. Inferable: partially — asserted that a Line
// comment selects the preview Accept header and a Position comment does not;
// the exact member list of the comfort-fade field set is not enumerated.
func TestDetail01(t *testing.T) {
	ctx := context.Background()

	client, accept, _ := comfortFadeServer(t)
	review := &github.PullRequestReviewRequest{
		Comments: []*github.DraftReviewComment{
			{Path: strp("f.go"), Line: intp(3), Body: strp("hi")},
		},
	}
	if _, _, err := client.PullRequests.CreateReview(ctx, "o", "r", 1, review); err != nil {
		t.Fatalf("CreateReview (line comment) returned error: %v", err)
	}
	if got := accept.Load().(string); got != "application/vnd.github.comfort-fade-preview+json" {
		t.Errorf("line comment Accept = %q, want comfort-fade preview", got)
	}

	client, accept, _ = comfortFadeServer(t)
	review = &github.PullRequestReviewRequest{
		Comments: []*github.DraftReviewComment{
			{Path: strp("f.go"), Position: intp(3), Body: strp("hi")},
		},
	}
	if _, _, err := client.PullRequests.CreateReview(ctx, "o", "r", 1, review); err != nil {
		t.Fatalf("CreateReview (position comment) returned error: %v", err)
	}
	if got := accept.Load().(string); got == "application/vnd.github.comfort-fade-preview+json" {
		t.Errorf("position comment Accept = %q, want non-preview default", got)
	}
}

// Detail 2: a single comment carrying both styles produces
// ErrMixedCommentStyles. (Inferable: yes)
func TestDetail02(t *testing.T) {
	client, _, hits := comfortFadeServer(t)
	review := &github.PullRequestReviewRequest{
		Comments: []*github.DraftReviewComment{
			{Path: strp("f.go"), Position: intp(1), Line: intp(2), Body: strp("x")},
		},
	}
	_, _, err := client.PullRequests.CreateReview(context.Background(), "o", "r", 1, review)
	if !errors.Is(err, github.ErrMixedCommentStyles) {
		t.Fatalf("mixed single comment: error = %v, want errors.Is ErrMixedCommentStyles", err)
	}
	if hits.Load() != 0 {
		t.Error("mixed-style request reached the network")
	}
}

// Detail 3: styles must be consistent across the whole comment list — a
// Position comment followed by a Line comment is also an error before any
// network call. Inferable: no — asserted as shape only: an error surfaces and
// the request never reaches the server.
func TestDetail03(t *testing.T) {
	client, _, hits := comfortFadeServer(t)
	review := &github.PullRequestReviewRequest{
		Comments: []*github.DraftReviewComment{
			{Path: strp("a.go"), Position: intp(1), Body: strp("legacy")},
			{Path: strp("b.go"), Line: intp(2), Body: strp("fade")},
		},
	}
	_, _, err := client.PullRequests.CreateReview(context.Background(), "o", "r", 1, review)
	if err == nil {
		t.Fatal("cross-comment mixed styles: CreateReview returned nil error")
	}
	if hits.Load() != 0 {
		t.Error("cross-comment mixed styles request reached the network")
	}
}

// Detail 4: nil comments in the slice are skipped. Inferable: no — asserted
// as shape only: nil entries neither panic nor poison detection.
func TestDetail04(t *testing.T) {
	ctx := context.Background()

	client, accept, _ := comfortFadeServer(t)
	review := &github.PullRequestReviewRequest{
		Comments: []*github.DraftReviewComment{
			nil,
			{Path: strp("f.go"), Line: intp(3), Body: strp("hi")},
		},
	}
	if _, _, err := client.PullRequests.CreateReview(ctx, "o", "r", 1, review); err != nil {
		t.Fatalf("nil + line comment: CreateReview returned error: %v", err)
	}
	if got := accept.Load().(string); got != "application/vnd.github.comfort-fade-preview+json" {
		t.Errorf("nil + line comment Accept = %q, want comfort-fade preview", got)
	}

	client, _, _ = comfortFadeServer(t)
	review = &github.PullRequestReviewRequest{Comments: []*github.DraftReviewComment{nil}}
	if _, _, err := client.PullRequests.CreateReview(ctx, "o", "r", 1, review); err != nil {
		t.Fatalf("all-nil comments: CreateReview returned error: %v", err)
	}
}

// Detail 5: an all-comfort-fade review returns true; an all-position or
// commentless review returns false. (Inferable: yes — asserted through the
// emitted Accept header.)
func TestDetail05(t *testing.T) {
	ctx := context.Background()

	client, accept, _ := comfortFadeServer(t)
	review := &github.PullRequestReviewRequest{
		Comments: []*github.DraftReviewComment{
			{Path: strp("f.go"), Side: strp("RIGHT"), Line: intp(3), Body: strp("a")},
			{Path: strp("g.go"), StartSide: strp("LEFT"), StartLine: intp(1), Line: intp(4), Body: strp("b")},
		},
	}
	if _, _, err := client.PullRequests.CreateReview(ctx, "o", "r", 1, review); err != nil {
		t.Fatalf("all-fade review: CreateReview returned error: %v", err)
	}
	if got := accept.Load().(string); got != "application/vnd.github.comfort-fade-preview+json" {
		t.Errorf("all-fade review Accept = %q, want comfort-fade preview", got)
	}

	client, accept, _ = comfortFadeServer(t)
	review = &github.PullRequestReviewRequest{
		Comments: []*github.DraftReviewComment{
			{Path: strp("f.go"), Position: intp(1), Body: strp("a")},
			{Path: strp("g.go"), Position: intp(2), Body: strp("b")},
		},
	}
	if _, _, err := client.PullRequests.CreateReview(ctx, "o", "r", 1, review); err != nil {
		t.Fatalf("all-position review: CreateReview returned error: %v", err)
	}
	if got := accept.Load().(string); got == "application/vnd.github.comfort-fade-preview+json" {
		t.Errorf("all-position review Accept = %q, want non-preview default", got)
	}

	client, _, _ = comfortFadeServer(t)
	review = &github.PullRequestReviewRequest{Body: strp("no comments")}
	if _, _, err := client.PullRequests.CreateReview(ctx, "o", "r", 1, review); err != nil {
		t.Fatalf("commentless review: CreateReview returned error: %v", err)
	}
}

// Detail 6: a true result sets Accept to mediaTypeMultiLineCommentsPreview;
// an error aborts the request before any network call. (Inferable: yes)
func TestDetail06(t *testing.T) {
	// The preview header literal is covered by Detail 1/5; this test pins the
	// pre-network abort for the error path.
	client, _, hits := comfortFadeServer(t)
	review := &github.PullRequestReviewRequest{
		Comments: []*github.DraftReviewComment{
			{Path: strp("f.go"), Position: intp(1), Side: strp("RIGHT"), Body: strp("x")},
		},
	}
	_, _, err := client.PullRequests.CreateReview(context.Background(), "o", "r", 1, review)
	if err == nil {
		t.Fatal("mixed comment: CreateReview returned nil error")
	}
	if hits.Load() != 0 {
		t.Error("erroring review still reached the network")
	}
}
