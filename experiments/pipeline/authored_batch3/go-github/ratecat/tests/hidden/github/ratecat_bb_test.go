// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"testing"

	"github.com/google/go-github/v92/github"
)

// Detail 1: /search/code GET requests are their own category; other /search/
// requests share a search category regardless of method. Inferable:
// partially — asserted as relations only: the code-search bucket differs
// from the general search bucket, both differ from Core, and general search
// is method-insensitive. The exact constant bindings are not pinned.
func TestDetail01(t *testing.T) {
	code := github.GetRateLimitCategory("GET", "/search/code")
	search := github.GetRateLimitCategory("GET", "/search/repositories")

	if code == github.CoreCategory {
		t.Error("GET /search/code classified as Core")
	}
	if search == github.CoreCategory {
		t.Error("GET /search/repositories classified as Core")
	}
	if code == search {
		t.Error("GET /search/code and GET /search/repositories share a category")
	}
	if got := github.GetRateLimitCategory("POST", "/search/repositories"); got != search {
		t.Error("POST /search/repositories classified differently from GET")
	}
}

// Detail 2: /graphql is its own category. Inferable: partially — asserted
// as a distinct non-Core bucket.
func TestDetail02(t *testing.T) {
	g := github.GetRateLimitCategory("POST", "/graphql")
	if g == github.CoreCategory {
		t.Error("POST /graphql classified as Core")
	}
	if g == github.GetRateLimitCategory("GET", "/search/code") ||
		g == github.GetRateLimitCategory("GET", "/search/repositories") {
		t.Error("POST /graphql shares a category with search endpoints")
	}
}

// Detail 3: POSTs to /app-manifests/*/conversions and PUTs to
// /repos/*​/import are their own categories. Inferable: no — asserted as
// shape only: the named method+suffix forms land outside Core, and the
// routing is method-scoped (same path, other method, differs).
func TestDetail03(t *testing.T) {
	post := github.GetRateLimitCategory("POST", "/app-manifests/abc/conversions")
	if post == github.CoreCategory {
		t.Error("POST /app-manifests/*/conversions classified as Core")
	}
	if get := github.GetRateLimitCategory("GET", "/app-manifests/abc/conversions"); get == post {
		t.Error("GET /app-manifests/*/conversions shares the POST-only category")
	}

	put := github.GetRateLimitCategory("PUT", "/repos/o/r/import")
	if put == github.CoreCategory {
		t.Error("PUT /repos/*/import classified as Core")
	}
	if get := github.GetRateLimitCategory("GET", "/repos/o/r/import"); get == put {
		t.Error("GET /repos/*/import shares the PUT-only category")
	}
	if post == put {
		t.Error("manifest-conversion and source-import endpoints share a category")
	}
}

// Detail 4: */code-scanning/sarifs, /scim/*, and POST
// */dependency-graph/snapshots have their own categories. Inferable: no —
// asserted as shape only: non-Core, suffix/prefix-keyed (equal across
// prefixes), method-scoped where committed.
func TestDetail04(t *testing.T) {
	sarifA := github.GetRateLimitCategory("POST", "/repos/o/r/code-scanning/sarifs")
	sarifB := github.GetRateLimitCategory("POST", "/enterprises/e/code-scanning/sarifs")
	if sarifA == github.CoreCategory {
		t.Error("*/code-scanning/sarifs classified as Core")
	}
	if sarifA != sarifB {
		t.Error("*/code-scanning/sarifs not suffix-keyed across prefixes")
	}

	if scim := github.GetRateLimitCategory("GET", "/scim/v2/Users"); scim == github.CoreCategory {
		t.Error("/scim/* classified as Core")
	}

	snap := github.GetRateLimitCategory("POST", "/repos/o/r/dependency-graph/snapshots")
	if snap == github.CoreCategory {
		t.Error("POST */dependency-graph/snapshots classified as Core")
	}
	if get := github.GetRateLimitCategory("GET", "/repos/o/r/dependency-graph/snapshots"); get == snap {
		t.Error("GET */dependency-graph/snapshots shares the POST-only category")
	}
}

// Detail 5: */audit-log has its own category, and
// /repos/*​/dependency-graph/sbom[...] paths share one. Inferable: no —
// asserted as shape only: non-Core, suffix-keyed.
func TestDetail05(t *testing.T) {
	auditA := github.GetRateLimitCategory("GET", "/orgs/o/audit-log")
	auditB := github.GetRateLimitCategory("GET", "/enterprises/e/audit-log")
	if auditA == github.CoreCategory {
		t.Error("*/audit-log classified as Core")
	}
	if auditA != auditB {
		t.Error("*/audit-log not suffix-keyed across prefixes")
	}

	sbom := github.GetRateLimitCategory("GET", "/repos/o/r/dependency-graph/sbom")
	report := github.GetRateLimitCategory("GET", "/repos/o/r/dependency-graph/sbom/generate-report")
	if sbom == github.CoreCategory {
		t.Error("/repos/*/dependency-graph/sbom classified as Core")
	}
	if sbom != report {
		t.Error("sbom and sbom/generate-report under /repos/ do not share a category")
	}
}

// Detail 6: everything else — including runner-registration endpoints —
// falls through to Core. (Inferable: yes)
func TestDetail06(t *testing.T) {
	for _, mp := range [][2]string{
		{"GET", "/repos/o/r/issues"},
		{"POST", "/repos/o/r/actions/runners/registration-token"},
		{"POST", "/orgs/o/actions/runners/registration-token"},
		{"GET", "/user"},
		{"DELETE", "/repos/o/r"},
	} {
		if got := github.GetRateLimitCategory(mp[0], mp[1]); got != github.CoreCategory {
			t.Errorf("%s %s classified as %v, want CoreCategory", mp[0], mp[1], got)
		}
	}
}

// Detail 7: classification is a pure function of method and path — no client
// state is consulted. (Inferable: yes — asserted by repeatability and the
// absence of a receiver.)
func TestDetail07(t *testing.T) {
	a := github.GetRateLimitCategory("GET", "/search/code")
	b := github.GetRateLimitCategory("GET", "/search/code")
	if a != b {
		t.Error("repeated classification of the same input differs")
	}
}
