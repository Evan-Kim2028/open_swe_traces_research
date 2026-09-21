package util_test

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/blang/semver/v4"

	kopsutil "example.internal/kops/pkg/apis/kops/util"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

// Detail 1: tolerant semver first — accepts v prefix, missing minor/patch.
func TestDetail01_TolerantParse(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 150; i++ {
		minor := uint64(rng.Intn(40))
		var s string
		var want semver.Version
		switch rng.Intn(4) {
		case 0:
			s = fmt.Sprintf("v1.%d", minor)
			want = semver.Version{Major: 1, Minor: minor}
		case 1:
			s = fmt.Sprintf("1.%d", minor)
			want = semver.Version{Major: 1, Minor: minor}
		case 2:
			patch := uint64(rng.Intn(10))
			s = fmt.Sprintf("v1.%d.%d", minor, patch)
			want = semver.Version{Major: 1, Minor: minor, Patch: patch}
		case 3:
			patch := uint64(rng.Intn(10))
			s = fmt.Sprintf("1.%d.%d", minor, patch)
			want = semver.Version{Major: 1, Minor: minor, Patch: patch}
		}
		v, err := kopsutil.ParseKubernetesVersion(s)
		if err != nil {
			t.Fatalf("i=%d ParseKubernetesVersion(%q) err %v", i, s, err)
		}
		if v.Major != want.Major || v.Minor != want.Minor || v.Patch != want.Patch {
			t.Fatalf("i=%d ParseKubernetesVersion(%q)=%v want %v", i, s, *v, want)
		}
	}
}

// Detail 2: on tolerant-parse failure the URL rule applies — extract
// /v1.<digits>. and synthesize {1, digits, 0}; patch and prerelease
// always dropped.
func TestDetail02_URLFallback(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		minor := uint64(rng.Intn(40))
		patch := uint64(rng.Intn(10))
		var s string
		switch rng.Intn(4) {
		case 0:
			s = fmt.Sprintf("https://dl.k8s.io/release/v1.%d.%d/bin/linux/amd64/kubectl", minor, patch)
		case 1:
			s = fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/v1.%d.%d/bin/linux/amd64/kubectl", minor, patch)
		case 2:
			s = fmt.Sprintf("https://example.com/v1.%d.%d-alpha.1/kubectl", minor, patch)
		case 3:
			s = fmt.Sprintf("/v1.%d.%d", minor, patch)
		}
		v, err := kopsutil.ParseKubernetesVersion(s)
		if err != nil {
			t.Fatalf("i=%d ParseKubernetesVersion(%q) err %v", i, s, err)
		}
		if v.Major != 1 || v.Minor != minor || v.Patch != 0 {
			t.Fatalf("i=%d ParseKubernetesVersion(%q)=%v want {1 %d 0}", i, s, *v, minor)
		}
		if len(v.Pre) != 0 || len(v.Build) != 0 {
			t.Fatalf("i=%d pre/build retained: %v", i, *v)
		}
	}
}

// Detail 3: the URL rule requires digits followed by "." — ".../v1.30/"
// fails.
func TestDetail03_URLRuleRequiresDotAfterDigits(t *testing.T) {
	for i, s := range []string{
		"https://example.com/v1.30/kubectl",
		"https://example.com/v1.30",
		"foo/v1.30x/kubectl",
	} {
		if v, err := kopsutil.ParseKubernetesVersion(s); err == nil {
			t.Fatalf("i=%d ParseKubernetesVersion(%q)=%v accepted", i, s, *v)
		}
	}
}

// Detail 4: unparseable input -> `unable to parse kubernetes version %q`.
func TestDetail04_ErrorFormat(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 60; i++ {
		s := fmt.Sprintf("garbage-%d-%s", rng.Intn(1000), strconv.Itoa(rng.Intn(100)))
		_, err := kopsutil.ParseKubernetesVersion(s)
		if err == nil {
			t.Fatalf("i=%d %q accepted", i, s)
		}
		want := fmt.Sprintf("unable to parse kubernetes version %q", s)
		if err.Error() != want {
			t.Fatalf("i=%d err %q want %q", i, err.Error(), want)
		}
	}
}

// Detail 5: IsKubernetesGTE(candidate, current) evaluates current >=
// candidate — the argument order is reversed from intuition.
func TestDetail05_ReversedComparison(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		a := fmt.Sprintf("1.%d.0", rng.Intn(40))
		b := fmt.Sprintf("1.%d.0", rng.Intn(40))
		va, _ := semver.ParseTolerant(a)
		vb, _ := semver.ParseTolerant(b)
		got := kopsutil.IsKubernetesGTE(a, vb)
		want := vb.GTE(va) // current >= candidate
		if got != want {
			t.Fatalf("i=%d IsKubernetesGTE(%q, %v)=%v want %v", i, a, vb, got, want)
		}
	}
	// Deterministic: candidate 1.30, current 1.29 -> false; reversed arg
	// intuition would say true.
	if kopsutil.IsKubernetesGTE("1.30.0", semver.MustParse("1.29.0")) {
		t.Fatalf("1.30.0 candidate vs 1.29.0 current returned true")
	}
	if !kopsutil.IsKubernetesGTE("1.29.0", semver.MustParse("1.30.0")) {
		t.Fatalf("1.29.0 candidate vs 1.30.0 current returned false")
	}
}

// Detail 6: strips Pre and Build from the CURRENT argument only; the
// candidate's prerelease still counts.
func TestDetail06_StripsPreFromCurrentOnly(t *testing.T) {
	// 1.6.0-alpha.1 current satisfies candidate "1.6" (pre stripped).
	if !kopsutil.IsKubernetesGTE("1.6", semver.MustParse("1.6.0-alpha.1")) {
		t.Fatalf("current prerelease not stripped")
	}
	// 1.5.9-alpha.1 current does NOT satisfy "1.6.0".
	if kopsutil.IsKubernetesGTE("1.6.0", semver.MustParse("1.5.9-alpha.1")) {
		t.Fatalf("1.5.9-alpha.1 satisfied 1.6.0")
	}
	// Candidate WITH prerelease still counts: "1.6.0-alpha.1" vs current
	// 1.6.0 -> true (pre on current stripped; candidate 1.6.0-a1 <= 1.6.0).
	if !kopsutil.IsKubernetesGTE("1.6.0-alpha.1", semver.MustParse("1.6.0")) {
		t.Fatalf("prerelease candidate not counted")
	}
	// And against current 1.5.9 the prerelease candidate fails.
	if kopsutil.IsKubernetesGTE("1.6.0-alpha.1", semver.MustParse("1.5.9")) {
		t.Fatalf("prerelease candidate satisfied by older current")
	}
}

// Detail 7: IsKubernetesGTE panics on an unparseable candidate.
func TestDetail07_PanicsOnBadCandidate(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("no panic on unparseable candidate")
		}
		if !strings.Contains(fmt.Sprint(r), "Error parsing version") {
			t.Fatalf("panic %v lacks 'Error parsing version'", r)
		}
	}()
	kopsutil.IsKubernetesGTE("not-a-version", semver.MustParse("1.30.0"))
}

// Detail 8: ParseVersion is strict semver (no v prefix, no missing
// components); errors wrap `error parsing version %q: %w`.
func TestDetail08_ParseVersionStrict(t *testing.T) {
	v, err := kopsutil.ParseVersion("1.30.2")
	if err != nil {
		t.Fatalf("ParseVersion(1.30.2) err %v", err)
	}
	if v.String() != "1.30.2" {
		t.Fatalf("String()=%q", v.String())
	}
	for i, s := range []string{"v1.30.2", "1.30", "1", "1.30.x", ""} {
		_, err := kopsutil.ParseVersion(s)
		if err == nil {
			t.Fatalf("i=%d ParseVersion(%q) accepted", i, s)
		}
		if !strings.HasPrefix(err.Error(), fmt.Sprintf("error parsing version %q: ", s)) {
			t.Fatalf("i=%d err %q lacks wrap prefix", i, err.Error())
		}
	}
	// IsInRange delegates to semver ranges.
	v2, _ := kopsutil.ParseVersion("1.30.0")
	r := semver.MustParseRange(">=1.29.0 <1.31.0")
	if !v2.IsInRange(r) {
		t.Fatalf("1.30.0 not in >=1.29.0 <1.31.0")
	}
}
