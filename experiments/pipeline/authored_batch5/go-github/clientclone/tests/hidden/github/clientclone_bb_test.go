package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

type bbCountJar struct{ seen int32 }

func (j *bbCountJar) SetCookies(*url.URL, []*http.Cookie) {}
func (j *bbCountJar) Cookies(*url.URL) []*http.Cookie {
	atomic.AddInt32(&j.seen, 1)
	return nil
}

// TestDetail01: Clone on a client whose inner http.Client is nil returns
// errUninitialized instead of panicking.
func TestDetail01(t *testing.T) {
	t.Parallel()
	c := &Client{}
	if _, err := c.Clone(); !errors.Is(err, errUninitialized) {
		t.Fatalf("Clone on uninitialized client: err = %v, want errUninitialized", err)
	}
}

// TestDetail02: all configuration fields carry over to the clone unless
// overridden by opts.
func TestDetail02(t *testing.T) {
	t.Parallel()
	parent, err := newClient(clientOptions{
		userAgent:             bbStr("ua-parent/9.9"),
		token:                 bbStr("tok-parent"),
		apiVersionMin:         bbStr("2022-11-28"),
		apiVersionMax:         bbStr("2030-01-01"),
		disableRateLimitCheck: true,
		marketplaceStubbed:    true,
	})
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	parent.baseURL = mustParseURL(t, "https://ghe.example.com/api-v3/")
	parent.uploadURL = mustParseURL(t, "https://ghe.example.com/api/uploads/")

	clone, err := parent.Clone()
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if clone.baseURL.String() != parent.baseURL.String() {
		t.Fatalf("baseURL not carried: %v", clone.baseURL)
	}
	if clone.uploadURL.String() != parent.uploadURL.String() {
		t.Fatalf("uploadURL not carried: %v", clone.uploadURL)
	}
	if clone.userAgent != "ua-parent/9.9" {
		t.Fatalf("userAgent not carried: %v", clone.userAgent)
	}
	if clone.apiVersionMin != "2022-11-28" || clone.apiVersionMax != "2030-01-01" {
		t.Fatalf("api version range not carried: %v..%v", clone.apiVersionMin, clone.apiVersionMax)
	}
	if clone.authToken == nil || *clone.authToken != "tok-parent" {
		t.Fatalf("token not carried: %v", clone.authToken)
	}
	if !clone.disableRateLimitCheck {
		t.Fatal("disableRateLimitCheck not carried")
	}
	if clone.Marketplace == nil || !clone.Marketplace.Stubbed {
		t.Fatal("marketplace stub not carried")
	}

	// opts override the carried values.
	clone2, err := parent.Clone(WithUserAgent("ua-override/1.0"))
	if err != nil {
		t.Fatalf("Clone with opts: %v", err)
	}
	if clone2.userAgent != "ua-override/1.0" {
		t.Fatalf("opts did not override: %v", clone2.userAgent)
	}
}

// TestDetail03: a token-bearing clone installs its auth transport against the
// clone's own origins — the clone is not layered on the parent's already
// wrapped transport.
func TestDetail03(t *testing.T) {
	t.Parallel()

	muxA := http.NewServeMux()
	muxA.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	serverA := httptest.NewServer(muxA)
	defer serverA.Close()

	var hitsB int32
	var authAtB atomic.Value
	muxB := http.NewServeMux()
	muxB.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hitsB, 1)
		authAtB.Store(r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	})
	serverB := httptest.NewServer(muxB)
	defer serverB.Close()

	parent := mustNewClient(t, WithAuthToken("tok-parent"))
	parent.baseURL = mustParseURL(t, serverA.URL+"/api/")
	parent.uploadURL = parent.baseURL

	bURL := serverB.URL + "/api/"
	clone, err := parent.Clone(WithURLs(&bURL, &bURL))
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}

	// A request from the clone to the clone's own origin must carry the token.
	req, err := clone.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := clone.Do(req, nil); err != nil {
		t.Fatalf("clone.Do: %v", err)
	}
	if got := authAtB.Load(); got != "Bearer tok-parent" {
		t.Fatalf("clone origin request Authorization = %v, want Bearer tok-parent", got)
	}

	// The token is not leaked to an origin the clone was not configured for.
	var authForeign atomic.Value
	muxF := http.NewServeMux()
	muxF.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		authForeign.Store(r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	})
	serverF := httptest.NewServer(muxF)
	defer serverF.Close()

	req, err = clone.NewRequest(t.Context(), "GET", serverF.URL+"/x", nil)
	if err != nil {
		t.Fatalf("NewRequest foreign: %v", err)
	}
	if _, err := clone.Do(req, nil); err != nil {
		t.Fatalf("clone.Do foreign: %v", err)
	}
	if got := authForeign.Load(); got != "" {
		t.Fatalf("token leaked to foreign origin: Authorization = %v", got)
	}
}

// TestDetail04: the clone sees the rate-limit state the parent already holds —
// a limit learned by the parent short-circuits the clone's calls.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var hits int32
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	})

	client.rateMu.Lock()
	client.rateLimits[CoreCategory] = Rate{Remaining: 0, Reset: Timestamp{time.Now().Add(time.Hour)}}
	client.rateMu.Unlock()

	clone, err := client.Clone()
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	req, err := clone.NewRequest(ctx, "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	_, err = clone.Do(req, nil)
	var rlerr *RateLimitError
	if !errors.As(err, &rlerr) {
		t.Fatalf("clone.Do under exhausted parent limit: err = %v (%T), want *RateLimitError", err, err)
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("clone made a network call despite inherited exhausted limit: %d", got)
	}
}

// TestDetail05: CheckRedirect, Jar, and Timeout are copied to the clone's HTTP
// client.
func TestDetail05(t *testing.T) {
	t.Parallel()
	parent := mustNewClient(t)
	jar := &bbCountJar{}
	redirect := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	parent.client.CheckRedirect = redirect
	parent.client.Jar = jar
	parent.client.Timeout = 42 * time.Second

	clone, err := parent.Clone()
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if clone.client == nil {
		t.Fatal("clone has no http client")
	}
	if reflect.ValueOf(clone.client.CheckRedirect).Pointer() != reflect.ValueOf(redirect).Pointer() {
		t.Fatal("CheckRedirect not copied to clone")
	}
	if clone.client.Jar != jar {
		t.Fatal("Jar not copied to clone")
	}
	if clone.client.Timeout != 42*time.Second {
		t.Fatalf("Timeout not copied: %v", clone.client.Timeout)
	}
}

// TestDetail06: when rate checks are disabled the clone does not attach the
// shared rate state (flag carries; no rate state shared).
func TestDetail06(t *testing.T) {
	t.Parallel()
	parent := mustNewClient(t, WithDisableRateLimitCheck())

	clone, err := parent.Clone()
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if !clone.disableRateLimitCheck {
		t.Fatal("disableRateLimitCheck not carried to clone")
	}
}

func bbStr(s string) *string { return &s }
