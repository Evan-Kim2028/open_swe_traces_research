package github

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// bbSpyRT counts RoundTrips then delegates to its inner transport.
type bbSpyRT struct {
	inner http.RoundTripper
	hits  int32
}

func (rt *bbSpyRT) RoundTrip(req *http.Request) (*http.Response, error) {
	atomic.AddInt32(&rt.hits, 1)
	return rt.inner.RoundTrip(req)
}

// bbTracked302RT answers the API call with a 302 whose body records Close.
type bbTracked302RT struct {
	closed   *int32
	location string
}

func (rt *bbTracked302RT) RoundTrip(req *http.Request) (*http.Response, error) {
	h := make(http.Header)
	h.Set("Location", rt.location)
	return &http.Response{
		StatusCode: http.StatusFound,
		Status:     "302 Found",
		Header:     h,
		Body:       &bbAssetTrackBody{Reader: strings.NewReader("redirect-page"), closed: rt.closed},
		Request:    req,
	}, nil
}

type bbAssetTrackBody struct {
	io.Reader
	closed *int32
}

func (b *bbAssetTrackBody) Close() error {
	atomic.StoreInt32(b.closed, 1)
	return nil
}

// TestDetail01: with no redirect the API response body is returned directly.
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/releases/assets/99", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("asset-bytes"))
	})

	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(t.Context(), "o", "r", 99, nil)
	if err != nil {
		t.Fatalf("DownloadReleaseAsset: %v", err)
	}
	if redirectURL != "" {
		t.Fatalf("redirectURL = %q, want empty on direct response", redirectURL)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	if string(b) != "asset-bytes" {
		t.Fatalf("asset body = %q, want asset-bytes", b)
	}
}

// TestDetail02: on redirect with a non-nil followRedirectsClient the target is
// fetched through that client — not the API client — and the fetched body is
// returned with an empty redirectURL.
func TestDetail02(t *testing.T) {
	t.Parallel()

	var s3Hits int32
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&s3Hits, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("presigned-asset"))
	}))
	defer s3.Close()

	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/releases/assets/99", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", s3.URL+"/file")
		w.WriteHeader(http.StatusFound)
	})

	spy := &bbSpyRT{inner: http.DefaultTransport}
	follow := &http.Client{Transport: spy}

	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(t.Context(), "o", "r", 99, follow)
	if err != nil {
		t.Fatalf("DownloadReleaseAsset: %v", err)
	}
	if redirectURL != "" {
		t.Fatalf("redirectURL = %q, want empty when followed", redirectURL)
	}
	if n := atomic.LoadInt32(&spy.hits); n != 1 {
		t.Fatalf("redirect not fetched through followRedirectsClient: %d hits", n)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	if string(b) != "presigned-asset" {
		t.Fatalf("fetched body = %q, want presigned-asset", b)
	}
}

// TestDetail03: on redirect with a nil followRedirectsClient the redirectURL
// is returned and no fetch occurs.
func TestDetail03(t *testing.T) {
	t.Parallel()

	var s3Hits int32
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&s3Hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer s3.Close()

	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/releases/assets/99", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", s3.URL+"/file")
		w.WriteHeader(http.StatusFound)
	})

	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(t.Context(), "o", "r", 99, nil)
	if err != nil {
		t.Fatalf("DownloadReleaseAsset: %v", err)
	}
	if rc != nil {
		t.Fatal("rc non-nil with nil followRedirectsClient")
	}
	if redirectURL != s3.URL+"/file" {
		t.Fatalf("redirectURL = %q, want %q", redirectURL, s3.URL+"/file")
	}
	if n := atomic.LoadInt32(&s3Hits); n != 0 {
		t.Fatalf("redirect target fetched despite nil client: %d hits", n)
	}
}

// TestDetail04: the redirect fetch is validated — an error status propagates
// and no body is returned.
func TestDetail04(t *testing.T) {
	t.Parallel()

	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"s3 broken"}`))
	}))
	defer s3.Close()

	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/releases/assets/99", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", s3.URL+"/file")
		w.WriteHeader(http.StatusFound)
	})

	rc, _, err := client.Repositories.DownloadReleaseAsset(t.Context(), "o", "r", 99, &http.Client{})
	if err == nil {
		t.Fatal("error status on redirect fetch: err = nil")
	}
	if rc != nil {
		t.Fatal("rc non-nil on failed redirect fetch")
	}
}

// TestDetail05: when a redirect is followed, the original API response body
// is closed first.
func TestDetail05(t *testing.T) {
	t.Parallel()

	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("presigned-asset"))
	}))
	defer s3.Close()

	client, _, _ := setup(t)

	var closed int32
	client.clientIgnoreRedirects = &http.Client{
		Transport: &bbTracked302RT{closed: &closed, location: s3.URL + "/file"},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	rc, _, err := client.Repositories.DownloadReleaseAsset(t.Context(), "o", "r", 99, &http.Client{})
	if err != nil {
		t.Fatalf("DownloadReleaseAsset: %v", err)
	}
	defer rc.Close()
	if atomic.LoadInt32(&closed) != 1 {
		t.Fatal("original API response body not closed before following redirect")
	}
}
