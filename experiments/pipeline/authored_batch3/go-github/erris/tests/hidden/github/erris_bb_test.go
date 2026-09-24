// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-github/v92/github"
)

func bareResp(code int) *http.Response {
	return &http.Response{StatusCode: code, Header: make(http.Header)}
}

// Detail 1: each Is first errors.As the target into its own concrete type;
// a type mismatch returns false. (Inferable: yes)
func TestDetail01(t *testing.T) {
	if errors.Is(&github.ErrorResponse{Message: "x"}, &github.RateLimitError{}) {
		t.Error("ErrorResponse.Is accepted a *RateLimitError target")
	}
	if errors.Is(&github.RateLimitError{}, &github.AcceptedError{}) {
		t.Error("RateLimitError.Is accepted an *AcceptedError target")
	}
	if errors.Is(&github.AcceptedError{}, &github.ErrorResponse{}) {
		t.Error("AcceptedError.Is accepted an *ErrorResponse target")
	}
	// Same type must still be comparable.
	if !errors.Is(&github.AcceptedError{Raw: []byte("a")}, &github.AcceptedError{Raw: []byte("a")}) {
		t.Error("identical *AcceptedError pair not Is-equal")
	}
}

// Detail 2: response comparison treats nil==nil equal, nil vs non-nil
// unequal, and otherwise compares only StatusCode. (Inferable: doc)
func TestDetail02(t *testing.T) {
	a := &github.ErrorResponse{Message: "m", Response: nil}
	b := &github.ErrorResponse{Message: "m", Response: nil}
	if !errors.Is(a, b) {
		t.Error("nil Response pair not Is-equal")
	}
	c := &github.ErrorResponse{Message: "m", Response: bareResp(404)}
	if errors.Is(a, c) || errors.Is(c, a) {
		t.Error("nil vs non-nil Response compared equal")
	}
	// Same StatusCode but otherwise different responses compare equal.
	d := &github.ErrorResponse{Message: "m", Response: bareResp(404)}
	d.Response.Header.Set("X-Different", "1")
	if !errors.Is(c, d) {
		t.Error("responses differing only outside StatusCode not Is-equal")
	}
	e := &github.ErrorResponse{Message: "m", Response: bareResp(500)}
	if errors.Is(c, e) {
		t.Error("different StatusCode responses compared equal")
	}
}

// Detail 3: ErrorResponse.Is compares the documented field set. Inferable:
// partially — asserted that fully-equal values match and that differences in
// the headlining fields (Message, StatusCode, Errors) break equality; the
// full field list is not enumerated.
func TestDetail03(t *testing.T) {
	base := &github.ErrorResponse{
		Message:  "m",
		Response: bareResp(404),
		Errors:   []github.Error{{Resource: "Issue", Field: "f", Code: "c"}},
	}
	same := &github.ErrorResponse{
		Message:  "m",
		Response: bareResp(404),
		Errors:   []github.Error{{Resource: "Issue", Field: "f", Code: "c"}},
	}
	if !errors.Is(base, same) {
		t.Error("equal ErrorResponses not Is-equal")
	}
	diff := *base
	diff.Message = "other"
	if errors.Is(base, &diff) {
		t.Error("different Message compared equal")
	}
	diff = *base
	diff.Errors = []github.Error{{Resource: "Issue", Field: "f", Code: "other"}}
	if errors.Is(base, &diff) {
		t.Error("different Errors compared equal")
	}
	diff = *base
	diff.Response = bareResp(500)
	if errors.Is(base, &diff) {
		t.Error("different StatusCode compared equal")
	}
}

// Detail 4: RateLimitError.Is compares Rate (struct equality), Message and
// response StatusCode. (Inferable: partially)
func TestDetail04(t *testing.T) {
	base := &github.RateLimitError{
		Message:  "m",
		Response: bareResp(403),
		Rate:     github.Rate{Limit: 5000, Remaining: 0},
	}
	same := &github.RateLimitError{
		Message:  "m",
		Response: bareResp(403),
		Rate:     github.Rate{Limit: 5000, Remaining: 0},
	}
	if !errors.Is(base, same) {
		t.Error("equal RateLimitErrors not Is-equal")
	}
	diff := *base
	diff.Rate = github.Rate{Limit: 60, Remaining: 0}
	if errors.Is(base, &diff) {
		t.Error("different Rate compared equal")
	}
	diff = *base
	diff.Message = "other"
	if errors.Is(base, &diff) {
		t.Error("different Message compared equal")
	}
}

// Detail 5: AcceptedError.Is compares the Raw byte payload only. Inferable:
// no — asserted as shape: identical Raw matches, differing Raw does not; the
// choice of Raw as sole identity is not itself pinned.
func TestDetail05(t *testing.T) {
	if !errors.Is(&github.AcceptedError{Raw: []byte("payload")}, &github.AcceptedError{Raw: []byte("payload")}) {
		t.Error("identical Raw payloads not Is-equal")
	}
	if errors.Is(&github.AcceptedError{Raw: []byte("a")}, &github.AcceptedError{Raw: []byte("b")}) {
		t.Error("different Raw payloads compared equal")
	}
}

// Detail 6: AbuseRateLimitError.Is compares Message, StatusCode and
// RetryAfter nil-safely. (Inferable: partially)
func TestDetail06(t *testing.T) {
	d90 := 90 * time.Second
	d91 := 91 * time.Second
	base := &github.AbuseRateLimitError{
		Message: "m", Response: bareResp(403), RetryAfter: &d90,
	}
	same := &github.AbuseRateLimitError{
		Message: "m", Response: bareResp(403), RetryAfter: &d90,
	}
	if !errors.Is(base, same) {
		t.Error("equal AbuseRateLimitErrors not Is-equal")
	}
	diff := *base
	diff.RetryAfter = &d91
	if errors.Is(base, &diff) {
		t.Error("different RetryAfter compared equal")
	}
	diff = *base
	diff.RetryAfter = nil
	if errors.Is(base, &diff) || errors.Is(&diff, base) {
		t.Error("nil vs set RetryAfter compared equal")
	}
	diff = *base
	diff.Message = "other"
	if errors.Is(base, &diff) {
		t.Error("different Message compared equal")
	}
}

// Detail 7: RedirectionError.Is compares StatusCode and Location — either
// both nil/same pointer, or both non-nil with equal String() forms.
// (Inferable: partially — asserted that two distinct *url.URL values with
// the same textual form count equal, and a nil-vs-set Location does not.)
func TestDetail07(t *testing.T) {
	locA, _ := url.Parse("https://api.github.com/x")
	locB, _ := url.Parse("https://api.github.com/x")
	base := &github.RedirectionError{Response: bareResp(302), StatusCode: 302, Location: locA}
	same := &github.RedirectionError{Response: bareResp(302), StatusCode: 302, Location: locB}
	if !errors.Is(base, same) {
		t.Error("equal RedirectionErrors (distinct URL pointers, same string) not Is-equal")
	}
	diff := *base
	diff.StatusCode = 307
	if errors.Is(base, &diff) {
		t.Error("different StatusCode compared equal")
	}
	diff = *base
	diff.Location = nil
	if errors.Is(base, &diff) || errors.Is(&diff, base) {
		t.Error("nil vs set Location compared equal")
	}
}

// Detail 8: duration-pointer equality treats nil==nil equal and otherwise
// compares pointed-to values — exercised through AbuseRateLimitError.
// (Inferable: yes)
func TestDetail08(t *testing.T) {
	a := &github.AbuseRateLimitError{Message: "m", Response: bareResp(403), RetryAfter: nil}
	b := &github.AbuseRateLimitError{Message: "m", Response: bareResp(403), RetryAfter: nil}
	if !errors.Is(a, b) {
		t.Error("nil+nil RetryAfter pair not Is-equal")
	}
	d := 5 * time.Second
	b.RetryAfter = &d
	if errors.Is(a, b) {
		t.Error("nil vs non-nil RetryAfter compared equal")
	}
	d2 := 5 * time.Second
	a.RetryAfter = &d2
	if !errors.Is(a, b) {
		t.Error("equal RetryAfter values behind different pointers not Is-equal")
	}
}
