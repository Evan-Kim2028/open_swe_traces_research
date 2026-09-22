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

// boolRespServer returns a client pointed at a test server whose /user/starred
// endpoint answers with the given status and body.
func boolRespServer(t *testing.T, status int, body string) *github.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/user/starred/o/r", func(w http.ResponseWriter, r *http.Request) {
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

// Detail 1: nil error in -> (true, nil) out. Asserted through a predicate
// consumer: a 200 response yields (true, nil). (Inferable: yes)
func TestDetail01(t *testing.T) {
	client := boolRespServer(t, http.StatusOK, `{}`)
	starred, _, err := client.Activity.IsStarred(context.Background(), "o", "r")
	if err != nil {
		t.Fatalf("IsStarred on 200 returned error: %v", err)
	}
	if !starred {
		t.Error("IsStarred on 200 = false, want true")
	}
}

// Detail 2: an *ErrorResponse carrying HTTP 404 is swallowed to (false, nil).
// Inferable: partially — asserted that the named status maps to plain false;
// no other status is special-cased here.
func TestDetail02(t *testing.T) {
	client := boolRespServer(t, http.StatusNotFound, `{"message":"Not Found"}`)
	starred, _, err := client.Activity.IsStarred(context.Background(), "o", "r")
	if err != nil {
		t.Fatalf("IsStarred on 404 returned error: %v", err)
	}
	if starred {
		t.Error("IsStarred on 404 = true, want false")
	}
}

// Detail 3: any other error — non-404 API error or a non-API error — is
// returned unchanged as (false, err). (Inferable: yes)
func TestDetail03(t *testing.T) {
	client := boolRespServer(t, http.StatusInternalServerError, `{"message":"boom"}`)
	starred, _, err := client.Activity.IsStarred(context.Background(), "o", "r")
	if err == nil {
		t.Error("IsStarred on 500 returned nil error, want propagation")
	}
	if starred {
		t.Error("IsStarred on 500 = true, want false")
	}

	// Non-API error: kill the server so Do fails with a transport error.
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	base := srv.URL + "/"
	dead, err := github.NewClient(github.WithURLs(&base, &base))
	if err != nil {
		t.Fatalf("github.NewClient returned error: %v", err)
	}
	srv.Close()
	starred, _, err = dead.Activity.IsStarred(context.Background(), "o", "r")
	if err == nil {
		t.Error("IsStarred on transport failure returned nil error, want propagation")
	}
	if starred {
		t.Error("IsStarred on transport failure = true, want false")
	}
}

// Detail 4: the 404 case is detected on the error value, so it is robust to
// the error payload. Inferable: no — asserted as shape only: a 404 response
// still maps to (false, nil) when the body is not a decodable error payload
// and when it carries no message; the wrapped-error edge itself is not
// reachable through the exported surface and is left unasserted.
func TestDetail04(t *testing.T) {
	for _, body := range []string{`page not found`, ``, `{}`} {
		client := boolRespServer(t, http.StatusNotFound, body)
		starred, _, err := client.Activity.IsStarred(context.Background(), "o", "r")
		if err != nil {
			t.Errorf("IsStarred on 404 body %q returned error: %v", body, err)
		}
		if starred {
			t.Errorf("IsStarred on 404 body %q = true, want false", body)
		}
	}
}
