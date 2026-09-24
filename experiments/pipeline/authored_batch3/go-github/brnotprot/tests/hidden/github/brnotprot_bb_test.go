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
	"strings"
	"testing"

	"github.com/google/go-github/v92/github"
)

// brNotProtServer returns a client whose branch-protection endpoint answers
// with the given status and body.
func brNotProtServer(t *testing.T, status int, body string) *github.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/branches/b/protection", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	base := srv.URL + "/"
	client, err := github.NewClient(github.WithURLs(&base, &base))
	if err != nil {
		t.Fatalf("github.NewClient returned error: %v", err)
	}
	return client
}

// Detail 1: the predicate fires only on errors carrying the documented
// "not protected" message. Inferable: partially — asserted that the named
// message triggers translation while a different message on the same status
// does not; whether matching keys on message alone (vs status) is not pinned.
func TestDetail01(t *testing.T) {
	ctx := context.Background()

	match := brNotProtServer(t, http.StatusNotFound, `{"message":"Branch not protected"}`)
	_, _, err := match.Repositories.GetBranchProtection(ctx, "o", "r", "b")
	if err == nil {
		t.Fatal("documented message: GetBranchProtection returned nil error")
	}

	nomatch := brNotProtServer(t, http.StatusNotFound, `{"message":"some other failure"}`)
	_, _, err = nomatch.Repositories.GetBranchProtection(ctx, "o", "r", "b")
	if err == nil {
		t.Fatal("different message: GetBranchProtection returned nil error")
	}
	if errors.Is(err, github.ErrBranchNotProtected) {
		t.Error("different message: error mapped to ErrBranchNotProtected, want propagation")
	}
}

// Detail 2: the match reads the API error's message after unwrapping —
// non-API errors never match. Inferable: partially — asserted that a
// transport-level failure propagates without translation.
func TestDetail02(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	base := srv.URL + "/"
	client, err := github.NewClient(github.WithURLs(&base, &base))
	if err != nil {
		t.Fatalf("github.NewClient returned error: %v", err)
	}
	srv.Close()

	_, _, err = client.Repositories.GetBranchProtection(context.Background(), "o", "r", "b")
	if err == nil {
		t.Fatal("transport failure: GetBranchProtection returned nil error")
	}
	if errors.Is(err, github.ErrBranchNotProtected) {
		t.Error("transport failure mapped to ErrBranchNotProtected, want propagation")
	}
}

// Detail 3: a match makes the caller return ErrBranchNotProtected in place of
// the raw API error. (Inferable: yes)
func TestDetail03(t *testing.T) {
	ctx := context.Background()
	client := brNotProtServer(t, http.StatusNotFound, `{"message":"Branch not protected"}`)

	_, _, err := client.Repositories.GetBranchProtection(ctx, "o", "r", "b")
	if !errors.Is(err, github.ErrBranchNotProtected) {
		t.Errorf("GetBranchProtection error = %v, want errors.Is ErrBranchNotProtected", err)
	}
}

// Detail 4: any other error — same status, different message — is returned
// unchanged. (Inferable: yes — asserted that the original message survives.)
func TestDetail04(t *testing.T) {
	client := brNotProtServer(t, http.StatusNotFound, `{"message":"a different rejection"}`)
	_, _, err := client.Repositories.GetBranchProtection(context.Background(), "o", "r", "b")
	if err == nil {
		t.Fatal("GetBranchProtection returned nil error")
	}
	if errors.Is(err, github.ErrBranchNotProtected) {
		t.Fatal("unrelated message mapped to ErrBranchNotProtected, want propagation")
	}
	if !strings.Contains(err.Error(), "a different rejection") {
		t.Errorf("propagated error %q does not carry the original message", err.Error())
	}
}
