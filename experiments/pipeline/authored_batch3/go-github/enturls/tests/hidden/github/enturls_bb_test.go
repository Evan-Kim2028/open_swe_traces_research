// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v92/github"
)

func mustNewEnterprise(t *testing.T, base, upload string) *github.Client {
	t.Helper()
	c, err := github.NewClient(github.WithEnterpriseURLs(base, upload))
	if err != nil {
		t.Fatalf("NewClient(WithEnterpriseURLs(%q, %q)) returned error: %v", base, upload, err)
	}
	return c
}

// Detail 1: parseURL("") is an error; parse failures surface as errors.
// (Inferable: doc — asserted through WithURLs/WithEnterpriseURLs.)
func TestDetail01(t *testing.T) {
	empty := ""
	if _, err := github.NewClient(github.WithURLs(&empty, nil)); err == nil {
		t.Error("WithURLs(\"\") succeeded, want error")
	}
	if _, err := github.NewClient(github.WithEnterpriseURLs("", "https://up.example.com")); err == nil {
		t.Error("WithEnterpriseURLs(\"\") succeeded, want error")
	}
	// A control: a non-empty URL parses fine.
	ok := "https://ghe.example.com"
	if _, err := github.NewClient(github.WithURLs(&ok, nil)); err != nil {
		t.Errorf("WithURLs(valid) returned error: %v", err)
	}
}

// Detail 2: parseURL ensures the path ends in "/" — appending one when
// missing. (Inferable: doc)
func TestDetail02(t *testing.T) {
	base := "https://ghe.example.com"
	c, err := github.NewClient(github.WithURLs(&base, nil))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if !strings.HasSuffix(c.BaseURL(), "/") {
		t.Errorf("BaseURL %q does not end in '/'", c.BaseURL())
	}
	base = "https://ghe.example.com/v1/api"
	c, err = github.NewClient(github.WithURLs(&base, nil))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if !strings.HasSuffix(c.BaseURL(), "/") {
		t.Errorf("BaseURL %q does not end in '/'", c.BaseURL())
	}
}

// Detail 3: WithEnterpriseURLs appends the enterprise API path convention to
// the base URL's path unless the path already ends with it or the host is
// api-prefixed. Inferable: no — asserted as shape only: a plain enterprise
// host has its path extended (ending in '/'), an api.* host is exempted, and
// a path already carrying the convention is not doubled. The literal suffix
// is never asserted.
func TestDetail03(t *testing.T) {
	// Plain host: path must be extended beyond the input and end in '/'.
	c := mustNewEnterprise(t, "https://ghe.example.com", "https://ghe.example.com/up")
	u, err := url.Parse(c.BaseURL())
	if err != nil {
		t.Fatalf("BaseURL %q did not parse: %v", c.BaseURL(), err)
	}
	if u.Path == "" || u.Path == "/" {
		t.Errorf("plain enterprise host: BaseURL path = %q, want an extended path", u.Path)
	}
	if !strings.HasSuffix(u.Path, "/") {
		t.Errorf("extended BaseURL path %q does not end in '/'", u.Path)
	}

	// api.* host: exempt — path must not gain extra segments.
	c = mustNewEnterprise(t, "https://api.ghe.example.com", "https://api.ghe.example.com/up")
	u, _ = url.Parse(c.BaseURL())
	if u.Path != "/" {
		t.Errorf("api.* host: BaseURL path = %q, want no added segments", u.Path)
	}

	// *.api.* host: also exempt.
	c = mustNewEnterprise(t, "https://ghe.api.example.com", "https://ghe.example.com/up")
	u, _ = url.Parse(c.BaseURL())
	if u.Path != "/" {
		t.Errorf("*.api.* host: BaseURL path = %q, want no added segments", u.Path)
	}

	// A base path that already carries the convention must not be doubled:
	// re-run WithEnterpriseURLs on the already-extended path and require the
	// extension not repeat.
	c = mustNewEnterprise(t, "https://ghe.example.com", "https://ghe.example.com/up")
	once, _ := url.Parse(c.BaseURL())
	c2 := mustNewEnterprise(t, "https://ghe.example.com"+once.Path,
		"https://ghe.example.com/up")
	twice, _ := url.Parse(c2.BaseURL())
	if once.Path != twice.Path {
		t.Errorf("re-applying WithEnterpriseURLs changed path: %q -> %q (extension not idempotent)",
			once.Path, twice.Path)
	}
}

// Detail 4: the upload URL is extended under the same rule, with an
// uploads.* host exemption. Inferable: no — shape only, as in Detail 3.
func TestDetail04(t *testing.T) {
	c := mustNewEnterprise(t, "https://ghe.example.com", "https://ghe.example.com/up")
	u, err := url.Parse(c.UploadURL())
	if err != nil {
		t.Fatalf("UploadURL %q did not parse: %v", c.UploadURL(), err)
	}
	if u.Path == "" || u.Path == "/up/" {
		t.Errorf("plain enterprise host: UploadURL path = %q, want an extended path", u.Path)
	}
	if !strings.HasSuffix(u.Path, "/") {
		t.Errorf("extended UploadURL path %q does not end in '/'", u.Path)
	}

	c = mustNewEnterprise(t, "https://ghe.example.com", "https://uploads.ghe.example.com")
	u, _ = url.Parse(c.UploadURL())
	if u.Path != "/" {
		t.Errorf("uploads.* host: UploadURL path = %q, want no added segments", u.Path)
	}
}

// Detail 5: URL mutation happens inside the option closure at
// client-construction time — the effect is visible on BaseURL()/UploadURL()
// before any request is made. (Inferable: yes — all assertions above read
// the client fields directly; this test makes it explicit.)
func TestDetail05(t *testing.T) {
	c := mustNewEnterprise(t, "https://ghe.example.com", "https://ghe.example.com/up")
	if c.BaseURL() == "https://ghe.example.com/" {
		t.Error("BaseURL unchanged after construction — option did not run at build time")
	}
}

// Detail 6: WithURLs performs only the parseURL normalization — no
// enterprise rewriting. Inferable: partially — asserted that the base URL is
// preserved verbatim apart from the trailing-slash normalization.
func TestDetail06(t *testing.T) {
	base := "https://ghe.example.com/foo"
	up := "https://up.example.com/bar"
	c, err := github.NewClient(github.WithURLs(&base, &up))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if c.BaseURL() != "https://ghe.example.com/foo/" {
		t.Errorf("WithURLs BaseURL = %q, want verbatim + trailing slash", c.BaseURL())
	}
	if c.UploadURL() != "https://up.example.com/bar/" {
		t.Errorf("WithURLs UploadURL = %q, want verbatim + trailing slash", c.UploadURL())
	}
}
