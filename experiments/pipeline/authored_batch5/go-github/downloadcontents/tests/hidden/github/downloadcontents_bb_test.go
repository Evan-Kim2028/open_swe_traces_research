package github

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestDetail01: a populated download_url is fetched with a plain GET and its
// bytes are returned.
func TestDetail01(t *testing.T) {
	t.Parallel()

	var dlHits int32
	dl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&dlHits, 1)
		testMethod(t, r, "GET")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("downloaded-bytes"))
	}))
	defer dl.Close()

	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/contents/f.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"file","name":"f.txt","download_url":"` + dl.URL + `/dl"}`))
	})

	rc, meta, _, err := client.Repositories.DownloadContentsWithMeta(t.Context(), "o", "r", "f.txt", nil)
	if err != nil {
		t.Fatalf("DownloadContentsWithMeta: %v", err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(b) != "downloaded-bytes" {
		t.Fatalf("download body = %q, want downloaded-bytes", b)
	}
	if meta == nil || meta.GetName() != "f.txt" {
		t.Fatalf("metadata not populated: %+v", meta)
	}
	if atomic.LoadInt32(&dlHits) != 1 {
		t.Fatalf("download_url fetched %d times, want 1", dlHits)
	}
}

// TestDetail02: no download_url and no inline Content →
// ErrContentsNoDownloadURL with the metadata still populated.
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/contents/f.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"file","name":"f.txt"}`))
	})

	rc, meta, _, err := client.Repositories.DownloadContentsWithMeta(t.Context(), "o", "r", "f.txt", nil)
	if !errors.Is(err, ErrContentsNoDownloadURL) {
		t.Fatalf("err = %v, want ErrContentsNoDownloadURL", err)
	}
	if rc != nil {
		t.Fatal("rc non-nil on ErrContentsNoDownloadURL")
	}
	if meta == nil || meta.GetName() != "f.txt" {
		t.Fatalf("metadata not populated on error: %+v", meta)
	}
}

// TestDetail03: a directory or submodule path returns early without any
// download.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/contents/dir", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"type":"file","name":"a"}]`))
	})
	mux.HandleFunc("/repos/o/r/contents/sub", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"submodule","name":"sub","submodule_git_url":"https://x"}`))
	})

	rc, _, _, err := client.Repositories.DownloadContentsWithMeta(t.Context(), "o", "r", "dir", nil)
	if !errors.Is(err, ErrContentsDirectory) {
		t.Fatalf("directory err = %v, want ErrContentsDirectory", err)
	}
	if rc != nil {
		t.Fatal("rc non-nil for directory")
	}

	rc, meta, _, err := client.Repositories.DownloadContentsWithMeta(t.Context(), "o", "r", "sub", nil)
	if !errors.Is(err, ErrContentsSubmodule) {
		t.Fatalf("submodule err = %v, want ErrContentsSubmodule", err)
	}
	if rc != nil {
		t.Fatal("rc non-nil for submodule")
	}
	if meta == nil || meta.GetType() != "submodule" {
		t.Fatalf("submodule metadata not populated: %+v", meta)
	}
}

// TestDetail04: a file with inline Content returns that content wrapped as a
// reader without a second request.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var hits int32
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"file","name":"f.txt","content":"aW5saW5l","encoding":"base64"}`))
	})

	rc, meta, _, err := client.Repositories.DownloadContentsWithMeta(t.Context(), "o", "r", "f.txt", nil)
	if err != nil {
		t.Fatalf("DownloadContentsWithMeta: %v", err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(b) != "inline" {
		t.Fatalf("inline content = %q, want decoded inline", b)
	}
	if meta == nil {
		t.Fatal("metadata not populated")
	}
	if n := atomic.LoadInt32(&hits); n != 1 {
		t.Fatalf("inline content made %d requests, want 1", n)
	}
}

// TestDetail05: on download transport error a response is returned and the
// error is the transport failure, not the no-download-url sentinel.
// (Inferable: no — assert shape, not which response object.)
func TestDetail05(t *testing.T) {
	t.Parallel()

	// An unroutable listener guarantees a transport-level failure.
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/contents/f.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"file","name":"f.txt","download_url":"` + deadURL + `/dl"}`))
	})

	rc, _, resp, err := client.Repositories.DownloadContentsWithMeta(t.Context(), "o", "r", "f.txt", nil)
	if err == nil {
		t.Fatal("download transport error: err = nil")
	}
	if errors.Is(err, ErrContentsNoDownloadURL) {
		t.Fatal("transport error reported as ErrContentsNoDownloadURL")
	}
	if rc != nil {
		t.Fatal("rc non-nil on transport error")
	}
	if resp == nil {
		t.Fatal("no *Response returned on transport error")
	}
}
