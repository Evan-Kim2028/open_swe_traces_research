package github

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// TestDetail01: a directory passed as file is rejected before any request —
// the error is a pre-request rejection, not a transport failure.
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	var hits int32
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	})

	dir, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open dir: %v", err)
	}
	defer dir.Close()

	_, _, err = client.Repositories.UploadReleaseAsset(t.Context(), "o", "r", 1, nil, dir)
	if err == nil {
		t.Fatal("UploadReleaseAsset with directory: err = nil")
	}
	var uerr *url.Error
	if errors.As(err, &uerr) {
		t.Fatalf("directory rejection surfaced as transport error (request was attempted): %v", err)
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("request attempted despite directory input: %d hits", n)
	}
}

// TestDetail02: nil reader, negative size, and empty UploadURL are all
// rejected.
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	var hits int32
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	})

	release := &RepositoryRelease{UploadURL: "uploads/rel/assets"}

	if _, _, err := client.Repositories.UploadReleaseAssetFromRelease(t.Context(), release, nil, nil, 10); err == nil {
		t.Fatal("nil reader: err = nil")
	}
	if _, _, err := client.Repositories.UploadReleaseAssetFromRelease(t.Context(), release, nil, strings.NewReader("x"), -1); err == nil {
		t.Fatal("negative size: err = nil")
	}
	if _, _, err := client.Repositories.UploadReleaseAssetFromRelease(t.Context(), &RepositoryRelease{}, nil, strings.NewReader("x"), 1); err == nil {
		t.Fatal("empty UploadURL: err = nil")
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("request attempted despite invalid input: %d hits", n)
	}
}

// TestDetail03: release.UploadURL's {?name,label} URI-template suffix is
// stripped before the upload URL is used.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, serverURL := setup(t)

	var gotPath, gotName, gotLabel atomic.Value
	mux.HandleFunc("/upload-here/assets", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		gotPath.Store(r.URL.Path)
		gotName.Store(r.URL.Query().Get("name"))
		gotLabel.Store(r.URL.Query().Get("label"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":7}`))
	})

	release := &RepositoryRelease{UploadURL: serverURL + "/api-v3/upload-here/assets{?name,label}"}
	_, _, err := client.Repositories.UploadReleaseAssetFromRelease(t.Context(), release, &UploadOptions{Name: "n.txt", Label: "lbl"}, strings.NewReader("data"), 4)
	if err != nil {
		t.Fatalf("UploadReleaseAssetFromRelease: %v", err)
	}
	if v := gotPath.Load(); v != "/upload-here/assets" {
		t.Fatalf("upload path = %v, want /upload-here/assets (template stripped)", v)
	}
	if v := gotName.Load(); v != "n.txt" {
		t.Fatalf("name param = %v", v)
	}
	if v := gotLabel.Load(); v != "lbl" {
		t.Fatalf("label param = %v", v)
	}
}

// TestDetail04: a relative upload URL is normalized (leading "/" trimmed) so
// it composes with the client's URL path prefix; absolute URLs pass through
// the destination gate.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var got atomic.Value
	mux.HandleFunc("/rel/path/assets", func(w http.ResponseWriter, r *http.Request) {
		got.Store(r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	})

	// Leading "/" must not escape the configured upload path prefix.
	release := &RepositoryRelease{UploadURL: "/rel/path/assets"}
	_, _, err := client.Repositories.UploadReleaseAssetFromRelease(t.Context(), release, nil, strings.NewReader("data"), 4)
	if err != nil {
		t.Fatalf("relative upload URL rejected/failed: %v", err)
	}
	if v := got.Load(); v != "/rel/path/assets" {
		t.Fatalf("upload path = %v, want /rel/path/assets", v)
	}
}

// TestDetail05: a media type is always produced for the upload request.
// (Inferable: no — assert Content-Type is set, not which value.)
func TestDetail05(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var got atomic.Value
	mux.HandleFunc("/repos/o/r/releases/1/assets", func(w http.ResponseWriter, r *http.Request) {
		got.Store(r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	})

	f := openTestFile(t, "asset.zip", "payload")
	defer f.Close()

	if _, _, err := client.Repositories.UploadReleaseAsset(t.Context(), "o", "r", 1, nil, f); err != nil {
		t.Fatalf("UploadReleaseAsset: %v", err)
	}
	if v, _ := got.Load().(string); v == "" {
		t.Fatal("no Content-Type produced for upload")
	}
}
