package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// bbTrackedRT returns responses whose body records Close.
type bbTrackedRT struct {
	closed *int32
	status int
	body   string
	header http.Header
}

func (rt *bbTrackedRT) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: rt.status,
		Status:     http.StatusText(rt.status),
		Header:     rt.header,
		Body:       &bbTrackBody{Reader: strings.NewReader(rt.body), closed: rt.closed},
		Request:    req,
	}, nil
}

type bbTrackBody struct {
	io.Reader
	closed *int32
}

func (b *bbTrackBody) Close() error {
	atomic.StoreInt32(b.closed, 1)
	return nil
}

// TestDetail01: an exhausted cached rate limit short-circuits before any
// network call with *RateLimitError carrying a populated Response;
// BypassRateLimitCheck in the context skips the check.
func TestDetail01(t *testing.T) {
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

	req, err := client.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	_, err = client.Do(req, nil)
	var rlerr *RateLimitError
	if !errors.As(err, &rlerr) {
		t.Fatalf("Do on exhausted limit: err = %v (%T), want *RateLimitError", err, err)
	}
	if rlerr.Response == nil {
		t.Fatal("RateLimitError.Response not populated")
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("network call made despite exhausted cached limit: %d hits", got)
	}

	// BypassRateLimitCheck skips the cached check entirely.
	req2, err := client.NewRequest(context.WithValue(t.Context(), BypassRateLimitCheck, true), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := client.Do(req2, nil); err != nil {
		t.Fatalf("Do with BypassRateLimitCheck: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("bypassed check did not reach network: %d hits", got)
	}
}

// TestDetail02: after a response the client's per-category rate state is
// updated from the response headers — unless rate checks are disabled.
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	resetAt := time.Now().Add(time.Hour).Unix()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderRateLimit, "5000")
		w.Header().Set(HeaderRateRemaining, "4999")
		w.Header().Set(HeaderRateUsed, "1")
		w.Header().Set(HeaderRateReset, strconv.FormatInt(resetAt, 10))
		w.WriteHeader(http.StatusOK)
	})

	req, err := client.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := client.Do(req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}

	client.rateMu.Lock()
	got := client.rateLimits[CoreCategory]
	client.rateMu.Unlock()
	if got.Limit != 5000 || got.Remaining != 4999 || got.Reset.Unix() != resetAt {
		t.Fatalf("rate state not updated from headers: %+v", got)
	}

	// With rate checks disabled the state is not tracked.
	client2, mux2, _ := setup(t)
	client2.disableRateLimitCheck = true
	mux2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderRateRemaining, "1")
		w.WriteHeader(http.StatusOK)
	})
	req, err = client2.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := client2.Do(req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	client2.rateMu.Lock()
	got = client2.rateLimits[CoreCategory]
	client2.rateMu.Unlock()
	if got.Remaining == 1 {
		t.Fatal("rate state updated while rate checks disabled")
	}
}

// TestDetail03: a secondary/abuse rate limit error is surfaced as
// *AbuseRateLimitError and the secondary limit is recorded so subsequent
// calls short-circuit without hitting the network. (The bounded retry
// mechanics are asserted at behavior level: the limit is honored, however it
// is enforced.)
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	client.maxSecondaryRateLimitRetryAfterDuration = 5 * time.Second

	var hits int32
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"abuse","documentation_url":"https://docs.github.com/free-pro-team@latest/rest/overview/resources-in-the-rest-api#secondary-rate-limits"}`))
	})

	req, err := client.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	_, err = client.Do(req, nil)
	var ab *AbuseRateLimitError
	if !errors.As(err, &ab) {
		t.Fatalf("secondary limit err = %v (%T), want *AbuseRateLimitError", err, err)
	}
	hitsAfterFirst := atomic.LoadInt32(&hits)

	// The recorded secondary limit short-circuits subsequent calls.
	for i := 0; i < 2; i++ {
		req2, err := client.NewRequest(t.Context(), "GET", ".", nil)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		_, err = client.Do(req2, nil)
		if !errors.As(err, &ab) {
			t.Fatalf("subsequent call err = %v (%T), want *AbuseRateLimitError", err, err)
		}
	}
	if got := atomic.LoadInt32(&hits); got != hitsAfterFirst {
		t.Fatalf("secondary limit not recorded/short-circuited: %d total requests", got)
	}
}

// TestDetail04: when the caller's context is cancelled mid-flight the returned
// error is the context's error, not a bare transport error.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	release := make(chan struct{})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	})
	defer close(release)

	ctx, cancel := context.WithCancel(t.Context())
	req, err := client.NewRequest(ctx, "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err = client.Do(req, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do on cancelled ctx: err = %v, want context.Canceled", err)
	}
}

// TestDetail05: a *url.Error surfaced through bareDo has its URL field
// sanitized — the sensitive query value does not survive on the error's URL.
// (Inferable: no — assert the error type + the field's shape, not the
// redaction literal.)
func TestDetail05(t *testing.T) {
	t.Parallel()
	client, _, _ := setup(t)

	// A permanently failing transport; http.Client.Do wraps its error in
	// *url.Error carrying the raw request URL.
	client.client = &http.Client{Transport: bbFailRT{}}

	req, err := client.NewRequest(t.Context(), "GET", "./x?access_token=SECRET_TOKEN_BB", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	_, err = client.Do(req, nil)
	var uerr *url.Error
	if !errors.As(err, &uerr) {
		t.Fatalf("transport error type = %T, want *url.Error", err)
	}
	if strings.Contains(uerr.URL, "SECRET_TOKEN_BB") {
		t.Fatalf("sensitive query leaked in url.Error.URL: %q", uerr.URL)
	}
}

type bbFailRT struct{}

func (bbFailRT) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

// TestDetail06: an *AcceptedError response has its raw body captured into
// AcceptedError.Raw and the original body closed.
func TestDetail06(t *testing.T) {
	t.Parallel()
	client, _, _ := setup(t)

	var closed int32
	client.client = &http.Client{Transport: &bbTrackedRT{closed: &closed, status: http.StatusAccepted, body: "raw-accepted-body", header: make(http.Header)}}

	req, err := client.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	_, err = client.BareDo(req)
	var ae *AcceptedError
	if !errors.As(err, &ae) {
		t.Fatalf("202 err = %v (%T), want *AcceptedError", err, err)
	}
	if string(ae.Raw) != "raw-accepted-body" {
		t.Fatalf("AcceptedError.Raw = %q, want the response body", ae.Raw)
	}
	if atomic.LoadInt32(&closed) != 1 {
		t.Fatal("original response body not closed on AcceptedError")
	}
}

// TestDetail07: on any error response the original network body is closed.
func TestDetail07(t *testing.T) {
	t.Parallel()
	client, _, _ := setup(t)

	var closed int32
	client.client = &http.Client{Transport: &bbTrackedRT{closed: &closed, status: http.StatusInternalServerError, body: `{"message":"boom"}`, header: make(http.Header)}}

	req, err := client.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := client.BareDo(req); err == nil {
		t.Fatal("500 returned nil error")
	}
	if atomic.LoadInt32(&closed) != 1 {
		t.Fatal("original response body not closed on error response")
	}
}
