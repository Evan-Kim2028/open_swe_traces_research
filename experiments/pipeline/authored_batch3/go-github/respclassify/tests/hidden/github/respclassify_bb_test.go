// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v92/github"
)

func respWith(code int, body string, hdr http.Header) *http.Response {
	if hdr == nil {
		hdr = make(http.Header)
	}
	return &http.Response{
		StatusCode: code,
		Status:     strconv.Itoa(code),
		Header:     hdr,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    httptest.NewRequest(http.MethodGet, "https://api.github.com/x", nil),
	}
}

// Detail 1: 202 Accepted yields *AcceptedError; the rest of 2xx yields nil.
// (Inferable: doc)
func TestDetail01(t *testing.T) {
	err := github.CheckResponse(respWith(http.StatusAccepted, `{"msg":"async"}`, nil))
	var aerr *github.AcceptedError
	if !errors.As(err, &aerr) {
		t.Fatalf("202 response: error type %T, want *AcceptedError", err)
	}
	for _, code := range []int{200, 201, 204, 299} {
		if err := github.CheckResponse(respWith(code, ``, nil)); err != nil {
			t.Errorf("%d response: error = %v, want nil", code, err)
		}
	}
}

// Detail 2: the response body is decoded into ErrorResponse under a size
// cap, and the body is restored afterward so callers can re-read it.
// Inferable: partially — body restoration is the documented quirk; asserted
// by re-reading the full body after CheckResponse.
func TestDetail02(t *testing.T) {
	body := `{"message":"oops","documentation_url":"https://docs.example.com"}`
	r := respWith(http.StatusBadRequest, body, nil)
	err := github.CheckResponse(r)
	var eerr *github.ErrorResponse
	if !errors.As(err, &eerr) {
		t.Fatalf("400 response: error type %T, want *ErrorResponse", err)
	}
	if eerr.Message != "oops" {
		t.Errorf("Message = %q, want %q", eerr.Message, "oops")
	}
	rest, rerr := io.ReadAll(r.Body)
	if rerr != nil {
		t.Fatalf("re-reading response body: %v", rerr)
	}
	if string(rest) != body {
		t.Errorf("restored body = %q, want %q", rest, body)
	}
}

// Detail 3: 401 with an OTP-required header yields *TwoFactorAuthError, a
// cast of the parsed ErrorResponse. Inferable: partially — the header
// constant survives; asserted with the conventional "required" value.
func TestDetail03(t *testing.T) {
	hdr := http.Header{"X-Github-Otp": {"required"}}
	err := github.CheckResponse(respWith(http.StatusUnauthorized, `{"message":"otp needed"}`, hdr))
	var tfa *github.TwoFactorAuthError
	if !errors.As(err, &tfa) {
		t.Fatalf("401 + OTP-required: error type %T, want *TwoFactorAuthError", err)
	}
	if tfa.Message != "otp needed" {
		t.Errorf("Message = %q, want %q", tfa.Message, "otp needed")
	}
	// A plain 401 stays a plain ErrorResponse.
	err = github.CheckResponse(respWith(http.StatusUnauthorized, `{"message":"bad creds"}`, nil))
	var eerr *github.ErrorResponse
	if !errors.As(err, &eerr) || errors.As(err, &tfa) {
		t.Errorf("plain 401: error type %T, want *ErrorResponse (not *TwoFactorAuthError)", err)
	}
}

// Detail 4: 403 or 429 with rate-limit-remaining 0 yields *RateLimitError
// carrying the parsed rate and message. (Inferable: doc)
func TestDetail04(t *testing.T) {
	for _, code := range []int{http.StatusForbidden, http.StatusTooManyRequests} {
		hdr := http.Header{
			"X-Ratelimit-Remaining": {"0"},
			"X-Ratelimit-Limit":     {"60"},
		}
		err := github.CheckResponse(respWith(code, `{"message":"rate limited"}`, hdr))
		var rl *github.RateLimitError
		if !errors.As(err, &rl) {
			t.Fatalf("%d + remaining=0: error type %T, want *RateLimitError", code, err)
		}
		if rl.Rate.Limit != 60 {
			t.Errorf("%d: Rate.Limit = %d, want 60", code, rl.Rate.Limit)
		}
		if rl.Message != "rate limited" {
			t.Errorf("%d: Message = %q, want %q", code, rl.Message, "rate limited")
		}
	}
}

// Detail 5: 403 or 429 whose documentation_url ends in the
// abuse/secondary-limit fragment yields *AbuseRateLimitError, with
// RetryAfter filled when derivable. Inferable: partially — the fragment is
// spelled out by the surviving AbuseRateLimitError doc comment.
func TestDetail05(t *testing.T) {
	abuseURL := "https://docs.github.com/rest/overview/rate-limits-for-the-rest-api?apiVersion=2022-11-28#about-secondary-rate-limits"
	hdr := http.Header{"Retry-After": {"120"}}
	body := `{"message":"slow down","documentation_url":"` + abuseURL + `"}`

	err := github.CheckResponse(respWith(http.StatusForbidden, body, hdr))
	var abuse *github.AbuseRateLimitError
	if !errors.As(err, &abuse) {
		t.Fatalf("403 + secondary-limit doc url: error type %T, want *AbuseRateLimitError", err)
	}
	if abuse.RetryAfter == nil || *abuse.RetryAfter != 120*time.Second {
		t.Errorf("RetryAfter = %v, want 120s", abuse.RetryAfter)
	}
	if abuse.Message != "slow down" {
		t.Errorf("Message = %q, want %q", abuse.Message, "slow down")
	}

	// The same status without the fragment does not classify as abuse.
	err = github.CheckResponse(respWith(http.StatusForbidden, `{"message":"plain","documentation_url":"https://docs.example.com/other"}`, nil))
	if errors.As(err, &abuse) {
		t.Error("403 without the fragment classified as *AbuseRateLimitError")
	}
}

// Detail 6: redirect statuses yield *RedirectionError carrying the status
// code and parsed Location header. Inferable: partially — asserted on the
// documented set members 302 and 307.
func TestDetail06(t *testing.T) {
	for _, code := range []int{http.StatusFound, http.StatusTemporaryRedirect} {
		hdr := http.Header{"Location": {"https://api.github.com/target"}}
		err := github.CheckResponse(respWith(code, ``, hdr))
		var redir *github.RedirectionError
		if !errors.As(err, &redir) {
			t.Fatalf("%d: error type %T, want *RedirectionError", code, err)
		}
		if redir.StatusCode != code {
			t.Errorf("StatusCode = %d, want %d", redir.StatusCode, code)
		}
		if redir.Location == nil || redir.Location.String() != "https://api.github.com/target" {
			t.Errorf("Location = %v, want https://api.github.com/target", redir.Location)
		}
	}
}

// Detail 7: the secondary-rate retry delay prefers the Retry-After header
// (seconds) and falls back to the rate-reset epoch header as a
// duration-until. (Inferable: doc — both headers are described in comments.)
func TestDetail07(t *testing.T) {
	abuseURL := "https://docs.github.com/rest/overview/rate-limits-for-the-rest-api?apiVersion=2022-11-28#about-secondary-rate-limits"
	body := `{"message":"slow down","documentation_url":"` + abuseURL + `"}`

	// Preference: Retry-After wins over the reset epoch.
	reset := time.Now().Add(10 * time.Minute).Unix()
	hdr := http.Header{
		"Retry-After":        {"45"},
		"X-Ratelimit-Reset": {strconv.FormatInt(reset, 10)},
	}
	err := github.CheckResponse(respWith(http.StatusForbidden, body, hdr))
	var abuse *github.AbuseRateLimitError
	if !errors.As(err, &abuse) || abuse.RetryAfter == nil {
		t.Fatalf("expected *AbuseRateLimitError with RetryAfter, got %T %v", err, err)
	}
	if *abuse.RetryAfter != 45*time.Second {
		t.Errorf("RetryAfter = %v, want Retry-After's 45s preferred over reset epoch", *abuse.RetryAfter)
	}

	// Fallback: only the reset epoch is present.
	hdr = http.Header{"X-Ratelimit-Reset": {strconv.FormatInt(time.Now().Add(5 * time.Minute).Unix(), 10)}}
	err = github.CheckResponse(respWith(http.StatusForbidden, body, hdr))
	if !errors.As(err, &abuse) || abuse.RetryAfter == nil {
		t.Fatalf("expected *AbuseRateLimitError with RetryAfter, got %T %v", err, err)
	}
	if *abuse.RetryAfter <= 0 || *abuse.RetryAfter > 5*time.Minute {
		t.Errorf("RetryAfter = %v, want a positive duration-until-reset <= 5m", *abuse.RetryAfter)
	}
}

// Detail 8: anything else yields the plain *ErrorResponse. (Inferable: yes)
func TestDetail08(t *testing.T) {
	for _, code := range []int{400, 404, 409, 422, 500} {
		err := github.CheckResponse(respWith(code, `{"message":"generic"}`, nil))
		var eerr *github.ErrorResponse
		if !errors.As(err, &eerr) {
			t.Fatalf("%d: error type %T, want *ErrorResponse", code, err)
		}
		var (
			aerr  *github.AcceptedError
			tfa   *github.TwoFactorAuthError
			rl    *github.RateLimitError
			abuse *github.AbuseRateLimitError
			redir *github.RedirectionError
		)
		for target, into := range map[string]any{
			"AcceptedError": &aerr, "TwoFactorAuthError": &tfa,
			"RateLimitError": &rl, "AbuseRateLimitError": &abuse,
			"RedirectionError": &redir,
		} {
			if errors.As(err, into) {
				t.Errorf("%d classified as %s", code, target)
			}
		}
	}
}
