package github

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// TestDetail01: the request is made with BypassRateLimitCheck — an exhausted
// cached limit must not short-circuit Get (unless checks are disabled).
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var hits int32
	mux.HandleFunc("/rate_limit", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"resources":{"core":{"limit":60,"remaining":59,"reset":1}}}`))
	})

	// Seed an exhausted core limit in the future — without the bypass marker
	// this would short-circuit before the network.
	client.rateMu.Lock()
	client.rateLimits[CoreCategory] = Rate{Remaining: 0, Reset: Timestamp{time.Now().Add(time.Hour)}}
	client.rateMu.Unlock()

	if _, _, err := client.RateLimit.Get(t.Context()); err != nil {
		t.Fatalf("Get with exhausted cached limit: %v", err)
	}
	if n := atomic.LoadInt32(&hits); n != 1 {
		t.Fatalf("Get did not reach network: %d hits", n)
	}
}

// TestDetail02: populated Resources.<category> values are written back into
// client.rateLimits. (Assert the map updates, not the field list.)
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	reset := time.Now().Add(time.Hour).Unix()
	mux.HandleFunc("/rate_limit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"resources":{"core":{"limit":5000,"remaining":4999,"used":1,"reset":%d},"search":{"limit":30,"remaining":29,"used":1,"reset":%d}}}`, reset, reset)
	})

	if _, _, err := client.RateLimit.Get(t.Context()); err != nil {
		t.Fatalf("Get: %v", err)
	}

	client.rateMu.Lock()
	core := client.rateLimits[CoreCategory]
	search := client.rateLimits[SearchCategory]
	client.rateMu.Unlock()

	if core.Limit != 5000 || core.Remaining != 4999 || core.Reset.Unix() != reset {
		t.Fatalf("core rate not written back: %+v", core)
	}
	if search.Limit != 30 || search.Remaining != 29 {
		t.Fatalf("search rate not written back: %+v", search)
	}
}

// TestDetail03: nil Resources or nil category pointers leave the map
// untouched for that category.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	mux.HandleFunc("/rate_limit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"resources":null}`))
	})

	if _, _, err := client.RateLimit.Get(t.Context()); err != nil {
		t.Fatalf("Get: %v", err)
	}
	client.rateMu.Lock()
	got := client.rateLimits[CoreCategory]
	client.rateMu.Unlock()
	if got.Limit != 0 {
		t.Fatalf("nil Resources wrote a rate entry: %+v", got)
	}

	// Present resources but a nil category pointer.
	client2, muxA, _ := setup(t)
	muxA.HandleFunc("/rate_limit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"resources":{"search":{"limit":30,"remaining":30,"reset":1},"core":null}}`))
	})
	if _, _, err := client2.RateLimit.Get(t.Context()); err != nil {
		t.Fatalf("Get: %v", err)
	}
	client2.rateMu.Lock()
	got = client2.rateLimits[CoreCategory]
	search := client2.rateLimits[SearchCategory]
	client2.rateMu.Unlock()
	if got.Limit != 0 {
		t.Fatalf("nil category pointer wrote a rate entry: %+v", got)
	}
	if search.Limit != 30 {
		t.Fatalf("populated category not written: %+v", search)
	}
}

// TestDetail04: response.Resources is returned to the caller.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/rate_limit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"resources":{"core":{"limit":5000,"remaining":4999,"reset":1}}}`))
	})

	rl, _, err := client.RateLimit.Get(t.Context())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rl == nil || rl.Core == nil || rl.Core.Limit != 5000 {
		t.Fatalf("Resources = %+v, want populated", rl)
	}
}
