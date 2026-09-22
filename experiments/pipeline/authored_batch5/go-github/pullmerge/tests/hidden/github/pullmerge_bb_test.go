package github

import (
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
)

// TestDetail01: empty commitMessage + nil options (or DontDefaultIfBlank
// false) → the request body omits commit_message.
func TestDetail01(t *testing.T) {
	t.Parallel()
	for _, opts := range []*PullRequestOptions{nil, {DontDefaultIfBlank: false}} {
		var present atomic.Bool
		var val atomic.Value
		client, mux, _ := setup(t)
		mux.HandleFunc("/repos/o/r/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, "PUT")
			raw, _ := io.ReadAll(r.Body)
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Errorf("body not JSON: %v", err)
			}
			v, ok := m["commit_message"]
			present.Store(ok)
			val.Store(v)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"merged":true}`))
		})

		if _, _, err := client.PullRequests.Merge(t.Context(), "o", "r", 1, "", opts); err != nil {
			t.Fatalf("Merge: %v", err)
		}
		if present.Load() {
			t.Fatalf("commit_message present (%v) with empty message and blank-default allowed", val.Load())
		}
	}
}

// TestDetail02: empty commitMessage + DontDefaultIfBlank true → the body
// carries commit_message as an explicit empty string. (Inferable: no —
// asserted: present-and-empty.)
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var present atomic.Bool
	var val atomic.Value
	mux.HandleFunc("/repos/o/r/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Errorf("body not JSON: %v", err)
		}
		v, ok := m["commit_message"]
		present.Store(ok)
		val.Store(v)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"merged":true}`))
	})

	if _, _, err := client.PullRequests.Merge(t.Context(), "o", "r", 1, "", &PullRequestOptions{DontDefaultIfBlank: true}); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !present.Load() {
		t.Fatal("commit_message omitted despite DontDefaultIfBlank")
	}
	if v := val.Load(); v != "" {
		t.Fatalf("commit_message = %v, want explicit empty string", v)
	}
}

// TestDetail03: a non-empty commitMessage is sent regardless of the flag.
func TestDetail03(t *testing.T) {
	t.Parallel()
	for _, opts := range []*PullRequestOptions{nil, {DontDefaultIfBlank: true}} {
		var val atomic.Value
		client, mux, _ := setup(t)
		mux.HandleFunc("/repos/o/r/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Errorf("body not JSON: %v", err)
			}
			val.Store(m["commit_message"])
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"merged":true}`))
		})

		if _, _, err := client.PullRequests.Merge(t.Context(), "o", "r", 1, "my message", opts); err != nil {
			t.Fatalf("Merge: %v", err)
		}
		if v := val.Load(); v != "my message" {
			t.Fatalf("commit_message = %v, want \"my message\"", v)
		}
	}
}
