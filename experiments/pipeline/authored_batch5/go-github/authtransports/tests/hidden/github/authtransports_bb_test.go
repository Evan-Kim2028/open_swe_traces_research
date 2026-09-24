package github

import (
	"io"
	"net/http"
	"net/url"
	"testing"
)

// bbRecordRT records the last request it saw and answers 200 with an empty body.
type bbRecordRT struct{ last *http.Request }

func (rt *bbRecordRT) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.last = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(nil),
		Request:    req,
	}, nil
}

func bbAuthedReq(t *testing.T, rt http.RoundTripper, origins []*url.URL, rawurl string) *http.Request {
	t.Helper()
	tp := &UnauthenticatedRateLimitedTransport{
		ClientID:       "id",
		ClientSecret:   "secret",
		AllowedOrigins: origins,
		Transport:      rt,
	}
	req, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		t.Fatalf("bad test url: %v", err)
	}
	if _, err := tp.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	rec := rt.(*bbRecordRT)
	if rec.last == nil {
		t.Fatal("transport never received the request")
	}
	return rec.last
}

// TestDetail01: credentials are attached only when the request origin is
// allowed — matching AllowedOrigins, or the default GitHub origins when the
// list is empty.
func TestDetail01(t *testing.T) {
	t.Parallel()
	allowed := mustParseURL(t, "https://ghe.example.com/api/")

	rt := &bbRecordRT{}
	got := bbAuthedReq(t, rt, []*url.URL{allowed}, "https://ghe.example.com/api/x")
	if _, _, ok := got.BasicAuth(); !ok {
		t.Fatal("credentials not attached to an allowed origin")
	}

	rt = &bbRecordRT{}
	got = bbAuthedReq(t, rt, nil, "https://api.github.com/x")
	if _, _, ok := got.BasicAuth(); !ok {
		t.Fatal("empty AllowedOrigins: credentials not attached to a default GitHub origin")
	}
}

// TestDetail02: requests to a disallowed origin pass through with no credential
// material added.
func TestDetail02(t *testing.T) {
	t.Parallel()
	allowed := mustParseURL(t, "https://ghe.example.com/api/")

	rt := &bbRecordRT{}
	got := bbAuthedReq(t, rt, []*url.URL{allowed}, "https://evil.example.com/x?client_id=keep")
	if _, _, ok := got.BasicAuth(); ok {
		t.Fatal("basic auth attached to a disallowed origin")
	}
	q := got.URL.Query()
	if q.Get("client_secret") != "" {
		t.Fatal("client_secret attached to a disallowed origin")
	}
	if q.Get("client_id") != "keep" {
		t.Fatalf("existing query value not preserved: %v", got.URL.RawQuery)
	}

	rt = &bbRecordRT{}
	ba := &BasicAuthTransport{Username: "u", Password: "p", AllowedOrigins: []*url.URL{allowed}, Transport: rt}
	req, _ := http.NewRequest("GET", "https://evil.example.com/x", nil)
	if _, err := ba.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if _, _, ok := rt.last.BasicAuth(); ok {
		t.Fatal("basic auth attached to a disallowed origin")
	}
}

// TestDetail03: the OTP header is sent only when OTP is non-empty.
func TestDetail03(t *testing.T) {
	t.Parallel()
	allowed := mustParseURL(t, "https://api.github.com/")

	rt := &bbRecordRT{}
	ba := &BasicAuthTransport{Username: "u", Password: "p", OTP: "123456", AllowedOrigins: []*url.URL{allowed}, Transport: rt}
	req, _ := http.NewRequest("GET", "https://api.github.com/x", nil)
	if _, err := ba.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if got := rt.last.Header.Get(headerOTP); got != "123456" {
		t.Fatalf("OTP header = %q, want %q", got, "123456")
	}

	rt = &bbRecordRT{}
	ba = &BasicAuthTransport{Username: "u", Password: "p", AllowedOrigins: []*url.URL{allowed}, Transport: rt}
	req, _ = http.NewRequest("GET", "https://api.github.com/x", nil)
	if _, err := ba.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if _, ok := rt.last.Header[headerOTP]; ok {
		t.Fatalf("OTP header present with empty OTP: %q", rt.last.Header.Get(headerOTP))
	}
}

// TestDetail04: credentials are conveyed while preserving existing query
// values on the request URL.
func TestDetail04(t *testing.T) {
	t.Parallel()
	allowed := mustParseURL(t, "https://api.github.com/")

	rt := &bbRecordRT{}
	got := bbAuthedReq(t, rt, []*url.URL{allowed}, "https://api.github.com/x?keep=1&other=two")
	q := got.URL.Query()
	if q.Get("keep") != "1" || q.Get("other") != "two" {
		t.Fatalf("existing query values not preserved: %v", got.URL.RawQuery)
	}
	if id, secret, ok := got.BasicAuth(); !ok || id != "id" || secret != "secret" {
		t.Fatalf("credentials not conveyed to an allowed origin: ok=%v", ok)
	}
}
