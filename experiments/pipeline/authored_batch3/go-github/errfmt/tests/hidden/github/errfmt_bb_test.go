// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v92/github"
)

func respWithReq(code int, rawurl string) *http.Response {
	u, _ := url.Parse(rawurl)
	req := &http.Request{Method: http.MethodGet, URL: u}
	return &http.Response{StatusCode: code, Request: req, Header: make(http.Header)}
}

// Detail 1: ErrorResponse.Error renders method+URL+code+message when the
// request is known, code+message when only the response is known, and
// message otherwise. Inferable: partially — asserted that each level renders
// the components it names; the exact format verbs are not pinned.
func TestDetail01(t *testing.T) {
	full := &github.ErrorResponse{
		Response: respWithReq(404, "https://api.github.com/x/y"),
		Message:  "boom happened",
	}
	s := full.Error()
	for _, want := range []string{"GET", "api.github.com/x/y", "404", "boom happened"} {
		if !strings.Contains(s, want) {
			t.Errorf("full ErrorResponse.Error() = %q, missing %q", s, want)
		}
	}

	noReq := &github.ErrorResponse{
		Response: &http.Response{StatusCode: 403, Header: make(http.Header)},
		Message:  "forbidden thing",
	}
	s = noReq.Error()
	for _, want := range []string{"403", "forbidden thing"} {
		if !strings.Contains(s, want) {
			t.Errorf("response-only Error() = %q, missing %q", s, want)
		}
	}

	bare := &github.ErrorResponse{Message: "just the message"}
	s = bare.Error()
	if !strings.Contains(s, "just the message") {
		t.Errorf("bare Error() = %q, missing message", s)
	}
}

// Detail 2: RateLimitError.Error renders the request line plus a reset
// suffix derived from the rate reset time. Inferable: no — asserted as
// shape: method, URL, code and message are present, and a future Reset adds
// output beyond the bare message.
func TestDetail02(t *testing.T) {
	e := &github.RateLimitError{
		Response: respWithReq(403, "https://api.github.com/rl"),
		Message:  "rate limited",
		Rate: github.Rate{
			Remaining: 0,
			Reset:     github.Timestamp{Time: time.Now().Add(2 * time.Minute)},
		},
	}
	s := e.Error()
	for _, want := range []string{"GET", "api.github.com/rl", "403", "rate limited"} {
		if !strings.Contains(s, want) {
			t.Errorf("RateLimitError.Error() = %q, missing %q", s, want)
		}
	}
	plain := &github.RateLimitError{
		Response: respWithReq(403, "https://api.github.com/rl"),
		Message:  "rate limited",
	}.Error()
	if s == plain {
		t.Errorf("future Reset added nothing to the rendered error: %q", s)
	}
}

// Detail 3: AbuseRateLimitError.Error appends a retry-after clause only when
// RetryAfter is set and positive, rendered in seconds. Inferable: no —
// asserted as shape: a positive RetryAfter lengthens the output and carries
// the second count; a nil RetryAfter does not.
func TestDetail03(t *testing.T) {
	mk := func(ra *time.Duration) *github.AbuseRateLimitError {
		return &github.AbuseRateLimitError{
			Response:   respWithReq(403, "https://api.github.com/ab"),
			Message:    "abuse detected",
			RetryAfter: ra,
		}
	}
	nilOut := mk(nil).Error()
	d := 90 * time.Second
	setOut := mk(&d).Error()
	if !strings.Contains(nilOut, "abuse detected") {
		t.Errorf("nil RetryAfter Error() = %q, missing message", nilOut)
	}
	if len(setOut) <= len(nilOut) {
		t.Errorf("RetryAfter=90s added no suffix: %q vs %q", setOut, nilOut)
	}
	if !strings.Contains(setOut, "90") {
		t.Errorf("RetryAfter=90s output %q does not carry the seconds count", setOut)
	}
}

// Detail 4: AcceptedError.Error returns a fixed message independent of
// payload. Inferable: no — asserted as shape only: non-empty, and identical
// for different Raw payloads.
func TestDetail04(t *testing.T) {
	a := (&github.AcceptedError{Raw: []byte("one")}).Error()
	b := (&github.AcceptedError{Raw: []byte("two two two")}).Error()
	if a == "" {
		t.Error("AcceptedError.Error() is empty")
	}
	if a != b {
		t.Errorf("AcceptedError.Error() varies with payload: %q vs %q", a, b)
	}
}

// Detail 5: RedirectionError.Error renders method+URL+code and the
// (sanitized) location. Inferable: no — asserted as shape: status code and
// the location target are present in the output.
func TestDetail05(t *testing.T) {
	loc, _ := url.Parse("https://api.github.com/moved/here")
	e := &github.RedirectionError{
		Response:   respWithReq(301, "https://api.github.com/orig"),
		StatusCode: 301,
		Location:   loc,
	}
	s := e.Error()
	for _, want := range []string{"301", "moved/here"} {
		if !strings.Contains(s, want) {
			t.Errorf("RedirectionError.Error() = %q, missing %q", s, want)
		}
	}
}

// Detail 6: Error.Error renders the code, field and resource values.
// Inferable: no — asserted as shape: each populated field's value appears.
func TestDetail06(t *testing.T) {
	e := &github.Error{Resource: "Issue", Field: "title", Code: "missing_field"}
	s := e.Error()
	for _, want := range []string{"Issue", "title", "missing_field"} {
		if !strings.Contains(s, want) {
			t.Errorf("Error.Error() = %q, missing %q", s, want)
		}
	}
}

// Detail 7: Error.UnmarshalJSON retries a bare JSON string into Message when
// the structured object decode fails. Inferable: no — the retry rule itself
// is arbitrary; asserted that a bare string decodes without error and lands
// in the message channel the doc comment reserves for it.
func TestDetail07(t *testing.T) {
	var e github.Error
	if err := json.Unmarshal([]byte(`"plain failure"`), &e); err != nil {
		t.Fatalf("UnmarshalJSON(bare string) returned error: %v", err)
	}
	if e.Message != "plain failure" {
		t.Errorf("bare-string decode: Message = %q, want %q", e.Message, "plain failure")
	}

	var e2 github.Error
	err := json.Unmarshal([]byte(`{"resource":"Issue","field":"title","code":"missing_field","message":"m"}`), &e2)
	if err != nil {
		t.Fatalf("UnmarshalJSON(object) returned error: %v", err)
	}
	if e2.Resource != "Issue" || e2.Field != "title" || e2.Code != "missing_field" {
		t.Errorf("object decode = %+v", e2)
	}
}

// Detail 8: sanitizeURL rewrites sensitive query values and leaves other
// params untouched. Inferable: partially — the sensitiveParams list survives
// in the tree; asserted through Error() that the secret VALUE is gone, the
// param name remains, and an unrelated param's value is untouched. The
// redaction marker literal is not pinned.
func TestDetail08(t *testing.T) {
	e := &github.ErrorResponse{
		Response: respWithReq(400,
			"https://api.github.com/x?client_secret=TOPSECRET123&harmless=keepme"),
		Message: "bad request",
	}
	s := e.Error()
	if strings.Contains(s, "TOPSECRET123") {
		t.Errorf("Error() leaked secret value: %q", s)
	}
	if !strings.Contains(s, "harmless=keepme") && !strings.Contains(s, "keepme") {
		t.Errorf("Error() dropped unrelated param: %q", s)
	}
	if !strings.Contains(s, "api.github.com") {
		t.Errorf("Error() did not render the request URL at all: %q", s)
	}
}

// Detail 9: formatRateReset renders a future reset differently from a past
// one and rounds to seconds. Inferable: no — asserted as shape via
// RateLimitError.Error: a past Reset and a future Reset render differently,
// both carry a reset indication, and sub-second differences round away.
func TestDetail09(t *testing.T) {
	mk := func(reset time.Time) string {
		return (&github.RateLimitError{
			Response: respWithReq(403, "https://api.github.com/rl"),
			Message:  "rate limited",
			Rate:     github.Rate{Reset: github.Timestamp{Time: reset}},
		}).Error()
	}
	future := mk(time.Now().Add(45 * time.Second))
	past := mk(time.Now().Add(-45 * time.Second))
	if future == past {
		t.Error("future and past resets render identically")
	}
	if !strings.Contains(future, "reset") || !strings.Contains(past, "reset") {
		t.Errorf("reset indication missing: future=%q past=%q", future, past)
	}

	// Sub-second differences round to the same rendering.
	a := mk(time.Now().Add(45100 * time.Millisecond))
	b := mk(time.Now().Add(45400 * time.Millisecond))
	if a != b {
		t.Errorf("sub-second reset difference changed output: %q vs %q", a, b)
	}
}
