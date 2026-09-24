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

// apiVersionServer returns a client pointed at a test server plus a hit
// counter, so tests can tell whether a request reached the network.
func apiVersionServer(t *testing.T) (*github.Client, *atomic.Int64) {
	t.Helper()
	var hits atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/x", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
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
	return client, &hits
}

// Detail 1: a request carrying no version header proceeds unchecked — the
// guard only applies when the header is set. (Inferable: doc)
func TestDetail01(t *testing.T) {
	client, hits := apiVersionServer(t)

	// A request built by hand carries no X-Github-Api-Version header at all.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		client.BaseURL()+"x", nil)
	if err != nil {
		t.Fatalf("http.NewRequest returned error: %v", err)
	}
	if _, err := client.Do(req, nil); err != nil {
		t.Fatalf("Do on headerless request returned error: %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("headerless request did not reach the server (hits=%d)", hits.Load())
	}
}

// Detail 2: a set version is accepted iff it falls inside the client's
// [min, max] window. Inferable: partially — asserted that a clearly interior
// version is accepted and clearly exterior versions are rejected; the
// inclusivity of the exact boundary values is NOT pinned.
func TestDetail02(t *testing.T) {
	client, _ := apiVersionServer(t)
	ctx := context.Background()

	req, err := client.NewRequest(ctx, http.MethodGet, "x", nil,
		github.WithVersion("2024-06-15"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	if _, err := client.Do(req, nil); err != nil {
		t.Errorf("interior version 2024-06-15: Do returned error: %v", err)
	}

	for _, v := range []string{"1999-01-01", "2031-12-31"} {
		req, err := client.NewRequest(ctx, http.MethodGet, "x", nil,
			github.WithVersion(v))
		if err != nil {
			t.Fatalf("NewRequest returned error: %v", err)
		}
		if _, err := client.Do(req, nil); err == nil {
			t.Errorf("exterior version %s: Do succeeded, want an error", v)
		}
	}
}

// Detail 3: the comparison is plain string ordering, not date parsing.
// Inferable: no — asserted as shape only: a value that is not a valid date
// but orders inside the window is accepted; nothing about the mechanism is
// pinned beyond "ordering, not parsing".
func TestDetail03(t *testing.T) {
	client, hits := apiVersionServer(t)
	ctx := context.Background()

	// "2025-99-99" is not a real calendar date but sorts inside the window
	// advertised by the client constants; a date-parsing check would reject it.
	req, err := client.NewRequest(ctx, http.MethodGet, "x", nil,
		github.WithVersion("2025-99-99"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	if _, err := client.Do(req, nil); err != nil {
		t.Fatalf("ordered-but-invalid-date version: Do returned error: %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("ordered-but-invalid-date version did not reach the server")
	}
}

// Detail 4: out-of-range requests fail with the sentinel
// ErrUnsupportedAPIVersion (matching errors.Is). (Inferable: yes)
func TestDetail04(t *testing.T) {
	client, _ := apiVersionServer(t)

	req, err := client.NewRequest(context.Background(), http.MethodGet, "x", nil,
		github.WithVersion("1999-01-01"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	_, err = client.Do(req, nil)
	if err == nil {
		t.Fatal("out-of-range version: Do succeeded, want ErrUnsupportedAPIVersion")
	}
	if !errors.Is(err, github.ErrUnsupportedAPIVersion) {
		t.Fatalf("out-of-range version error = %v, want errors.Is ErrUnsupportedAPIVersion", err)
	}
}

// Detail 5: the check fires inside bareDo, so it gates requests that never
// went through NewRequest as well. (Inferable: yes — asserted via a raw
// http.Request carrying a bad version, which must error before any network
// call.)
func TestDetail05(t *testing.T) {
	client, hits := apiVersionServer(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		client.BaseURL()+"x", nil)
	if err != nil {
		t.Fatalf("http.NewRequest returned error: %v", err)
	}
	github.WithVersion("1999-01-01")(req)

	_, err = client.Do(req, nil)
	if err == nil {
		t.Fatal("raw request with out-of-range version: Do succeeded, want an error")
	}
	if hits.Load() != 0 {
		t.Fatalf("rejected request still reached the server (hits=%d)", hits.Load())
	}
}
