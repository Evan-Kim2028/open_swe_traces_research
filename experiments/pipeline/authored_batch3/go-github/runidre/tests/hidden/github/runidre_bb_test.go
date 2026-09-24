// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"testing"

	"github.com/google/go-github/v92/github"
)

// Detail 1: a callback URL containing
// repos/<org>/<repo>/actions/runs/<digits>/deployment_protection_rule yields
// that digit run as an int64. (Inferable: doc — the regexp constant survives
// and fixes the shape.)
func TestDetail01(t *testing.T) {
	e := &github.DeploymentProtectionRuleEvent{
		DeploymentCallbackURL: github.Ptr("https://api.github.com/repos/o/r/actions/runs/12345/deployment_protection_rule"),
	}
	got, err := e.GetRunID()
	if err != nil {
		t.Fatalf("GetRunID: %v", err)
	}
	if got != 12345 {
		t.Errorf("GetRunID = %d, want 12345", got)
	}
}

// Detail 2: both absolute and relative URLs satisfying the pattern work.
// (Inferable: yes — the regex is not anchored to a scheme.)
func TestDetail02(t *testing.T) {
	for _, u := range []string{
		"https://api.github.com/repos/myorg/myrepo/actions/runs/678/deployment_protection_rule",
		"repos/myorg/myrepo/actions/runs/678/deployment_protection_rule",
		"/api/v3/repos/a/b/actions/runs/42/deployment_protection_rule",
	} {
		e := &github.DeploymentProtectionRuleEvent{DeploymentCallbackURL: github.Ptr(u)}
		got, err := e.GetRunID()
		if err != nil {
			t.Errorf("GetRunID(%q): %v", u, err)
			continue
		}
		if got == -1 {
			t.Errorf("GetRunID(%q) = -1, want the run id", u)
		}
	}
	// Spot-check the digit value on a relative URL too.
	e := &github.DeploymentProtectionRuleEvent{
		DeploymentCallbackURL: github.Ptr("repos/o/r/actions/runs/678/deployment_protection_rule"),
	}
	if got, err := e.GetRunID(); err != nil || got != 678 {
		t.Errorf("relative URL: (%d, %v), want (678, nil)", got, err)
	}
}

// Detail 3: a URL that does not match the pattern returns -1 and an error.
// (Inferable: yes — signature contract; the error literal is arbitrary.)
func TestDetail03(t *testing.T) {
	for _, u := range []string{
		"https://api.github.com/repos/o/r/actions/runs/abc/deployment_protection_rule",
		"https://api.github.com/repos/o/r/actions/runs/123/other",
		"https://api.github.com/user",
		"",
	} {
		e := &github.DeploymentProtectionRuleEvent{DeploymentCallbackURL: github.Ptr(u)}
		got, err := e.GetRunID()
		if err == nil {
			t.Errorf("GetRunID(%q): expected error, got id %d", u, got)
		}
		if got != -1 {
			t.Errorf("GetRunID(%q) = %d, want -1 on failure", u, got)
		}
	}
}

// Detail 4: a matched capture that fails integer parsing returns -1 and the
// parse error. Inferable: no — asserted as shape only: an error is returned
// alongside -1; which error is not pinned.
func TestDetail04(t *testing.T) {
	// A digit run that overflows int64 still matches the regexp capture but
	// cannot parse to int64.
	e := &github.DeploymentProtectionRuleEvent{
		DeploymentCallbackURL: github.Ptr("repos/o/r/actions/runs/99999999999999999999999/deployment_protection_rule"),
	}
	got, err := e.GetRunID()
	if err == nil {
		t.Error("overflowing run id: expected error, got none")
	}
	if got != -1 {
		t.Errorf("overflowing run id: got %d, want -1", got)
	}
}
