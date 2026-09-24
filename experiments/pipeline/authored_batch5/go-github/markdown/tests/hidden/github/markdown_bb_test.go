package github

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestDetail01: a non-empty opts.Mode is sent as the request's mode field.
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var got map[string]any
	mux.HandleFunc("/markdown", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<p>hi</p>"))
	})

	_, _, err := client.Markdown.Render(t.Context(), "hello", &MarkdownOptions{Mode: "gfm"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got["mode"] != "gfm" {
		t.Fatalf("request mode = %v, want %q", got["mode"], "gfm")
	}
}

// TestDetail02: a non-empty opts.Context is sent as the request's context field.
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var got map[string]any
	mux.HandleFunc("/markdown", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<p>hi</p>"))
	})

	_, _, err := client.Markdown.Render(t.Context(), "hello", &MarkdownOptions{Mode: "gfm", Context: "o/r"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got["context"] != "o/r" {
		t.Fatalf("request context = %v, want %q", got["context"], "o/r")
	}
}

// TestDetail03: empty option fields are omitted, and opts == nil sends only text.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var got map[string]any
	mux.HandleFunc("/markdown", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<p>hi</p>"))
	})

	if _, _, err := client.Markdown.Render(t.Context(), "hello", &MarkdownOptions{}); err != nil {
		t.Fatalf("Render empty opts: %v", err)
	}
	if _, ok := got["mode"]; ok {
		t.Fatalf("empty Mode was sent as %v", got["mode"])
	}
	if _, ok := got["context"]; ok {
		t.Fatalf("empty Context was sent as %v", got["context"])
	}

	got = nil
	if _, _, err := client.Markdown.Render(t.Context(), "hello", nil); err != nil {
		t.Fatalf("Render nil opts: %v", err)
	}
	if len(got) != 1 || got["text"] != "hello" {
		t.Fatalf("nil opts body = %v, want only text field", got)
	}
}

// TestDetail04: the rendered HTML body is returned as a string.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	mux.HandleFunc("/markdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<h1>Title</h1>"))
	})

	out, _, err := client.Markdown.Render(t.Context(), "# Title", nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if out != "<h1>Title</h1>" {
		t.Fatalf("Render returned %q, want %q", out, "<h1>Title</h1>")
	}
}
