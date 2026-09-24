package github

import (
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
)

// TestDetail01: CreateRef rejects an empty Ref and an empty SHA before any
// request is built. (Shape: an error surfaces; literals not pinned.)
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	var hits int32
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	})

	if _, _, err := client.Git.CreateRef(t.Context(), "o", "r", CreateRef{Ref: "", SHA: "abc"}); err == nil {
		t.Fatal("CreateRef with empty Ref: err = nil")
	}
	if _, _, err := client.Git.CreateRef(t.Context(), "o", "r", CreateRef{Ref: "heads/x", SHA: ""}); err == nil {
		t.Fatal("CreateRef with empty SHA: err = nil")
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("request built despite invalid input: %d hits", n)
	}
}

// TestDetail02: CreateRef normalizes body.Ref to begin with "refs/"
// (idempotent for already-prefixed refs).
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var gotRef atomic.Value
	mux.HandleFunc("/repos/o/r/git/refs", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		raw, _ := io.ReadAll(r.Body)
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Errorf("body not JSON: %v", err)
		}
		gotRef.Store(m["ref"])
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ref":"refs/heads/x"}`))
	})

	if _, _, err := client.Git.CreateRef(t.Context(), "o", "r", CreateRef{Ref: "heads/x", SHA: "abc"}); err != nil {
		t.Fatalf("CreateRef: %v", err)
	}
	if v := gotRef.Load(); v != "refs/heads/x" {
		t.Fatalf("body ref = %v, want refs/heads/x", v)
	}

	if _, _, err := client.Git.CreateRef(t.Context(), "o", "r", CreateRef{Ref: "refs/heads/y", SHA: "abc"}); err != nil {
		t.Fatalf("CreateRef: %v", err)
	}
	if v := gotRef.Load(); v != "refs/heads/y" {
		t.Fatalf("already-prefixed ref changed: %v", v)
	}
}

// TestDetail03: UpdateRef rejects empty ref and empty body.SHA before
// building the request.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	var hits int32
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	})

	if _, _, err := client.Git.UpdateRef(t.Context(), "o", "r", "", UpdateRef{SHA: "abc"}); err == nil {
		t.Fatal("UpdateRef with empty ref: err = nil")
	}
	if _, _, err := client.Git.UpdateRef(t.Context(), "o", "r", "heads/x", UpdateRef{SHA: ""}); err == nil {
		t.Fatal("UpdateRef with empty SHA: err = nil")
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("request built despite invalid input: %d hits", n)
	}
}

// TestDetail04: UpdateRef and DeleteRef strip a leading "refs/" before
// escaping into the route — "refs/heads/x" and "heads/x" resolve identically.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var paths []string
	mux.HandleFunc("/repos/o/r/git/refs/", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.EscapedPath())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	if _, _, err := client.Git.UpdateRef(t.Context(), "o", "r", "refs/heads/x", UpdateRef{SHA: "abc"}); err != nil {
		t.Fatalf("UpdateRef refs/heads/x: %v", err)
	}
	if _, _, err := client.Git.UpdateRef(t.Context(), "o", "r", "heads/y", UpdateRef{SHA: "abc"}); err != nil {
		t.Fatalf("UpdateRef heads/y: %v", err)
	}
	if _, err := client.Git.DeleteRef(t.Context(), "o", "r", "refs/heads/z"); err != nil {
		t.Fatalf("DeleteRef refs/heads/z: %v", err)
	}

	want := []string{
		"PATCH /repos/o/r/git/refs/heads/x",
		"PATCH /repos/o/r/git/refs/heads/y",
		"DELETE /repos/o/r/git/refs/heads/z",
	}
	if len(paths) != len(want) {
		t.Fatalf("got %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("request %d = %q, want %q", i, paths[i], want[i])
		}
	}
}

// TestDetail05: the ref is path-escaped so a ref containing a space can't
// break the URL.
func TestDetail05(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var got string
	mux.HandleFunc("/repos/o/r/git/refs/", func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	if _, _, err := client.Git.UpdateRef(t.Context(), "o", "r", "refs/heads/foo bar", UpdateRef{SHA: "abc"}); err != nil {
		t.Fatalf("UpdateRef: %v", err)
	}
	want := "/repos/o/r/git/refs/heads/foo%20bar"
	if got != want {
		t.Fatalf("escaped route = %q, want %q", got, want)
	}
}
