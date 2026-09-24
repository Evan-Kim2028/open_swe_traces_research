package github

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// bbAccept captures the Accept header of the request it serves.
type bbAccept struct{ got atomic.Value }

func (a *bbAccept) handler(w http.ResponseWriter, r *http.Request) {
	a.got.Store(r.Header.Get("Accept"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
}

// TestDetail01: opts.TextMatch adds a text-match media type to Accept.
// (Assert a text-match value is present, not the exact literal.)
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	rec := &bbAccept{}
	mux.HandleFunc("/search/repositories", rec.handler)

	if _, _, err := client.Search.Repositories(t.Context(), "q", &SearchOptions{TextMatch: true}); err != nil {
		t.Fatalf("Search: %v", err)
	}
	accept, _ := rec.got.Load().(string)
	if !strings.Contains(accept, "text-match") {
		t.Fatalf("Accept = %q, want a text-match media type", accept)
	}
}

// TestDetail02: each search type contributes its preview media type to
// Accept — the header carries more than just the base media type.
// (Inferable: no — assert Accept is assembled from media types, not names.)
func TestDetail02(t *testing.T) {
	t.Parallel()
	for _, typ := range []string{"repositories", "commits", "topics", "issues"} {
		client, mux, _ := setup(t)
		rec := &bbAccept{}
		mux.HandleFunc("/search/"+typ, rec.handler)

		var err error
		switch typ {
		case "repositories":
			_, _, err = client.Search.Repositories(t.Context(), "q", nil)
		case "commits":
			_, _, err = client.Search.Commits(t.Context(), "q", nil)
		case "topics":
			_, _, err = client.Search.Topics(t.Context(), "q", nil)
		case "issues":
			_, _, err = client.Search.Issues(t.Context(), "q", nil)
		}
		if err != nil {
			t.Fatalf("Search %s: %v", typ, err)
		}
		accept, _ := rec.got.Load().(string)
		if accept == "" || accept == mediaTypeV3 {
			t.Fatalf("%s: Accept = %q, want a preview media type contributed", typ, accept)
		}
		if !strings.Contains(accept, "vnd.github") {
			t.Fatalf("%s: Accept = %q, want vnd.github media type(s)", typ, accept)
		}
		for _, p := range strings.Split(accept, ",") {
			if strings.TrimSpace(p) == "" {
				t.Fatalf("%s: Accept %q has an empty media-type slot", typ, accept)
			}
		}
	}
}

// TestDetail03: parameters.RepositoryID becomes the repository_id query
// param.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var got atomic.Value
	mux.HandleFunc("/search/labels", func(w http.ResponseWriter, r *http.Request) {
		got.Store(r.URL.Query().Get("repository_id"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	})

	if _, _, err := client.Search.Labels(t.Context(), 4242, "bug", nil); err != nil {
		t.Fatalf("Labels: %v", err)
	}
	if v := got.Load(); v != "4242" {
		t.Fatalf("repository_id param = %v, want 4242", v)
	}
}

// TestDetail04: Accept values are joined into a single comma-separated
// header — not repeated header lines.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var lines atomic.Value
	mux.HandleFunc("/search/commits", func(w http.ResponseWriter, r *http.Request) {
		lines.Store(r.Header.Values("Accept"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	})

	if _, _, err := client.Search.Commits(t.Context(), "q", &SearchOptions{TextMatch: true}); err != nil {
		t.Fatalf("Commits: %v", err)
	}
	vals, _ := lines.Load().([]string)
	if len(vals) != 1 {
		t.Fatalf("Accept sent as %d header lines, want 1 comma-joined", len(vals))
	}
	if !strings.Contains(vals[0], ",") {
		t.Fatalf("Accept = %q, want comma-joined media types", vals[0])
	}
}
