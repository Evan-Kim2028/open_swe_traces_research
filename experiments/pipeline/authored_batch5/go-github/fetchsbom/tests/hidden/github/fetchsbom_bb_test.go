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

// bbCannedRT serves a fixed response whose body records Close.
type bbCannedRT struct {
	closed *int32
	status int
	body   string
}

func (rt *bbCannedRT) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: rt.status,
		Status:     http.StatusText(rt.status),
		Header:     make(http.Header),
		Body:       &bbSBOMTrackBody{Reader: strings.NewReader(rt.body), closed: rt.closed},
		Request:    req,
	}, nil
}

type bbSBOMTrackBody struct {
	io.Reader
	closed *int32
}

func (b *bbSBOMTrackBody) Close() error {
	atomic.StoreInt32(b.closed, 1)
	return nil
}

const bbSBOMPayload = `{"SPDXID":"SPDXRef-DOCUMENT","name":"o/r","packages":[{"name":"pkg-a"}]}`

// TestDetail01: with a non-nil followRedirectsClient the redirect target is
// fetched through that client — not the API client — and the decoded *SBOM
// is returned with an empty redirectURL.
func TestDetail01(t *testing.T) {
	t.Parallel()

	dl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bbSBOMPayload))
	}))
	defer dl.Close()

	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/dependency-graph/sbom/fetch-report/u1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", dl.URL+"/sbom")
		w.WriteHeader(http.StatusFound)
	})

	spy := &bbSpyRT{inner: http.DefaultTransport}
	sbom, redirectURL, _, err := client.DependencyGraph.FetchSBOM(t.Context(), "o", "r", "u1", &http.Client{Transport: spy})
	if err != nil {
		t.Fatalf("FetchSBOM: %v", err)
	}
	if redirectURL != "" {
		t.Fatalf("redirectURL = %q, want empty when followed", redirectURL)
	}
	if n := atomic.LoadInt32(&spy.hits); n != 1 {
		t.Fatalf("redirect not fetched through followRedirectsClient: %d hits", n)
	}
	if sbom == nil || sbom.SBOM == nil || sbom.SBOM.GetName() != "o/r" {
		t.Fatalf("SBOM not decoded: %+v", sbom)
	}
}

// TestDetail02: with a nil followRedirectsClient the redirectURL is returned
// and no fetch occurs.
func TestDetail02(t *testing.T) {
	t.Parallel()

	var dlHits int32
	dl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&dlHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer dl.Close()

	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/dependency-graph/sbom/fetch-report/u1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", dl.URL+"/sbom")
		w.WriteHeader(http.StatusFound)
	})

	sbom, redirectURL, _, err := client.DependencyGraph.FetchSBOM(t.Context(), "o", "r", "u1", nil)
	if err != nil {
		t.Fatalf("FetchSBOM: %v", err)
	}
	if sbom != nil {
		t.Fatal("sbom non-nil with nil followRedirectsClient")
	}
	if redirectURL != dl.URL+"/sbom" {
		t.Fatalf("redirectURL = %q, want %q", redirectURL, dl.URL+"/sbom")
	}
	if n := atomic.LoadInt32(&dlHits); n != 0 {
		t.Fatalf("redirect fetched despite nil client: %d hits", n)
	}
}

// TestDetail03: the redirect fetch is validated — an error HTTP status
// propagates as an error rather than decoding the error page.
// The error page here happens to be valid SBOM JSON, so only a status check
// can reject it.
func TestDetail03(t *testing.T) {
	t.Parallel()

	dl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(bbSBOMPayload))
	}))
	defer dl.Close()

	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/dependency-graph/sbom/fetch-report/u1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", dl.URL+"/sbom")
		w.WriteHeader(http.StatusFound)
	})

	sbom, _, _, err := client.DependencyGraph.FetchSBOM(t.Context(), "o", "r", "u1", &http.Client{})
	if err == nil {
		t.Fatalf("error status on redirect fetch decoded as success: sbom=%+v", sbom)
	}
}

// TestDetail04: the original network body is closed on every path — success
// and error.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var apiClosed, dlClosed int32

	// Success path: fetched body closed.
	dl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bbSBOMPayload))
	}))
	defer dl.Close()
	mux.HandleFunc("/repos/o/r/dependency-graph/sbom/fetch-report/u1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", dl.URL+"/sbom")
		w.WriteHeader(http.StatusFound)
	})

	client.clientIgnoreRedirects = &http.Client{
		Transport: &bbSBOMTrackRT{closed: &apiClosed, location: dl.URL + "/sbom"},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	follow := &http.Client{Transport: &bbCannedRT{closed: &dlClosed, status: http.StatusOK, body: bbSBOMPayload}}

	if _, _, _, err := client.DependencyGraph.FetchSBOM(t.Context(), "o", "r", "u1", follow); err != nil {
		t.Fatalf("FetchSBOM: %v", err)
	}
	if atomic.LoadInt32(&apiClosed) != 1 {
		t.Fatal("API response body not closed")
	}
	if atomic.LoadInt32(&dlClosed) != 1 {
		t.Fatal("download body not closed on success")
	}

	// Error path: fetched body closed even when the status is rejected.
	atomic.StoreInt32(&apiClosed, 0)
	atomic.StoreInt32(&dlClosed, 0)
	follow = &http.Client{Transport: &bbCannedRT{closed: &dlClosed, status: http.StatusInternalServerError, body: `{"message":"x"}`}}
	if _, _, _, err := client.DependencyGraph.FetchSBOM(t.Context(), "o", "r", "u1", follow); err == nil {
		t.Fatal("error status on redirect fetch: err = nil")
	}
	if atomic.LoadInt32(&apiClosed) != 1 {
		t.Fatal("API response body not closed on error path")
	}
	if atomic.LoadInt32(&dlClosed) != 1 {
		t.Fatal("download body not closed on error path")
	}
}

// bbSBOMTrackRT answers the API call with a 302 whose body records Close.
type bbSBOMTrackRT struct {
	closed   *int32
	location string
}

func (rt *bbSBOMTrackRT) RoundTrip(req *http.Request) (*http.Response, error) {
	h := make(http.Header)
	h.Set("Location", rt.location)
	return &http.Response{
		StatusCode: http.StatusFound,
		Status:     "302 Found",
		Header:     h,
		Body:       &bbSBOMTrackBody{Reader: strings.NewReader("redirect-page"), closed: rt.closed},
		Request:    req,
	}, nil
}

// TestDetail05: the downloaded payload decodes as a populated SBOM.
func TestDetail05(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	dl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bbSBOMPayload))
	}))
	defer dl.Close()
	mux.HandleFunc("/repos/o/r/dependency-graph/sbom/fetch-report/u1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", dl.URL+"/sbom")
		w.WriteHeader(http.StatusFound)
	})

	sbom, _, _, err := client.DependencyGraph.FetchSBOM(t.Context(), "o", "r", "u1", &http.Client{})
	if err != nil {
		t.Fatalf("FetchSBOM: %v", err)
	}
	if sbom == nil || sbom.SBOM == nil {
		t.Fatalf("SBOM = %+v, want populated", sbom)
	}
	if sbom.SBOM.GetSPDXID() != "SPDXRef-DOCUMENT" || len(sbom.SBOM.Packages) != 1 {
		t.Fatalf("SBOMInfo fields not decoded: %+v", sbom.SBOM)
	}
}
