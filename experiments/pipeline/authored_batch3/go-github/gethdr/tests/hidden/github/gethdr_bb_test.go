// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"testing"

	"github.com/google/go-github/v92/github"
)

// Detail 1: header lookup is case-insensitive — any letter-casing of the
// queried key matches the stored key. (Inferable: yes)
func TestDetail01(t *testing.T) {
	req := &github.HookRequest{Headers: map[string]string{
		"X-Github-Event": "push",
		"content-type":   "application/json",
	}}
	for _, key := range []string{"X-Github-Event", "x-github-event", "X-GITHUB-EVENT", "x-GitHub-Event"} {
		if got := req.GetHeader(key); got != "push" {
			t.Errorf("HookRequest.GetHeader(%q) = %q, want %q", key, got, "push")
		}
	}
	resp := &github.HookResponse{Headers: map[string]string{"Content-Type": "text/html"}}
	for _, key := range []string{"Content-Type", "content-type", "CONTENT-TYPE"} {
		if got := resp.GetHeader(key); got != "text/html" {
			t.Errorf("HookResponse.GetHeader(%q) = %q, want %q", key, got, "text/html")
		}
	}
}

// Detail 2: a key absent under every casing returns the empty string.
// (Inferable: yes)
func TestDetail02(t *testing.T) {
	req := &github.HookRequest{Headers: map[string]string{"X-A": "1"}}
	if got := req.GetHeader("X-Missing"); got != "" {
		t.Errorf("GetHeader(missing) = %q, want empty", got)
	}
	resp := &github.HookResponse{}
	if got := resp.GetHeader("X-Missing"); got != "" {
		t.Errorf("GetHeader on nil map = %q, want empty", got)
	}
}

// Detail 3: the stored value is returned verbatim — no trimming or
// rewriting. Inferable: partially — asserted that whitespace and casing of
// the value survive the lookup.
func TestDetail03(t *testing.T) {
	req := &github.HookRequest{Headers: map[string]string{
		"X-Padded": "  spaced value  ",
		"X-Case":   "MiXeD CaSe",
	}}
	if got := req.GetHeader("x-padded"); got != "  spaced value  " {
		t.Errorf("GetHeader(padded) = %q, want verbatim", got)
	}
	if got := req.GetHeader("x-case"); got != "MiXeD CaSe" {
		t.Errorf("GetHeader(case) = %q, want verbatim", got)
	}
}
