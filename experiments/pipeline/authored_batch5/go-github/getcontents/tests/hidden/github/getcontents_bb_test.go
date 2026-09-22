package github

import (
	"net/http"
	"strings"
	"testing"
)

// TestDetail01: the path is escaped as a URL path segment and a trailing "/"
// is trimmed, so path never breaks the route. A literal '?' must stay in the
// path (not become a query) and 'dir/' must arrive as 'dir'.
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	type captured struct {
		escaped string
		query   string
	}
	got := make(chan captured, 4)
	mux.HandleFunc("/repos/o/r/contents/", func(w http.ResponseWriter, r *http.Request) {
		got <- captured{escaped: r.URL.EscapedPath(), query: r.URL.RawQuery}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"file","name":"x"}`))
	})

	// '?' is data inside the segment, not a query delimiter.
	if _, _, _, err := client.Repositories.GetContents(t.Context(), "o", "r", "a?x=1", nil); err != nil {
		t.Fatalf("GetContents: %v", err)
	}
	c := <-got
	if !strings.HasSuffix(c.escaped, "/contents/a%3Fx=1") {
		t.Fatalf("path not segment-escaped: %q", c.escaped)
	}
	if c.query != "" {
		t.Fatalf("path leaked into query: %q", c.query)
	}

	// Trailing slash is trimmed.
	if _, _, _, err := client.Repositories.GetContents(t.Context(), "o", "r", "dir/", nil); err != nil {
		t.Fatalf("GetContents: %v", err)
	}
	c = <-got
	if !strings.HasSuffix(c.escaped, "/contents/dir") {
		t.Fatalf("trailing slash not trimmed: %q", c.escaped)
	}
	if strings.HasSuffix(c.escaped, "/contents/dir/") {
		t.Fatalf("trailing slash survived: %q", c.escaped)
	}
}

// TestDetail02: the response body is decoded as object → fileContent, else
// array → directoryContent. Exactly one is non-nil on success.
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/contents/f.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"file","name":"f.txt"}`))
	})
	mux.HandleFunc("/repos/o/r/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"type":"file","name":"a"},{"type":"dir","name":"b"}]`))
	})

	fc, dc, _, err := client.Repositories.GetContents(t.Context(), "o", "r", "f.txt", nil)
	if err != nil {
		t.Fatalf("GetContents file: %v", err)
	}
	if fc == nil || dc != nil {
		t.Fatalf("file response: fileContent=%v directoryContent=%v, want exactly fileContent", fc, dc)
	}
	if fc.GetName() != "f.txt" {
		t.Fatalf("fileContent.Name = %v", fc.GetName())
	}

	fc, dc, _, err = client.Repositories.GetContents(t.Context(), "o", "r", "dir", nil)
	if err != nil {
		t.Fatalf("GetContents dir: %v", err)
	}
	if dc == nil || fc != nil {
		t.Fatalf("dir response: fileContent=%v directoryContent=%v, want exactly directoryContent", fc, dc)
	}
	if len(dc) != 2 {
		t.Fatalf("directoryContent len = %d, want 2", len(dc))
	}
}

// TestDetail03: when neither shape decodes, an error surfaces.
// (Inferable: no — assert an error, not the combined message wording.)
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/contents/weird", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`"just a string"`))
	})

	_, _, _, err := client.Repositories.GetContents(t.Context(), "o", "r", "weird", nil)
	if err == nil {
		t.Fatal("GetContents on undecodable body: err = nil")
	}
}

// TestDetail04: opts.Ref applies to the request URL.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var gotRef string
	mux.HandleFunc("/repos/o/r/contents/f.txt", func(w http.ResponseWriter, r *http.Request) {
		gotRef = r.URL.Query().Get("ref")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"file","name":"f.txt"}`))
	})

	if _, _, _, err := client.Repositories.GetContents(t.Context(), "o", "r", "f.txt", &RepositoryContentGetOptions{Ref: "my-branch"}); err != nil {
		t.Fatalf("GetContents: %v", err)
	}
	if gotRef != "my-branch" {
		t.Fatalf("ref param = %q, want my-branch", gotRef)
	}
}
