package config

import (
	"testing"

	"example.internal/kvstore/v2/oracle"
)

// Hidden suite for unit configpath. One TestDetailNN per DETAILS.md line.
//
// The accepted URL scheme is not derivable: the doc comment shows one
// spelling while DETAILS commits to another, so the suite discovers the
// working scheme instead of pinning it.

// bbScheme returns a URL scheme ParsePath accepts.
func bbScheme(t *testing.T) string {
	t.Helper()
	for _, s := range []string{"tikv", "TIKV", "kvstore", "KVSTORE", "tikV"} {
		if _, _, _, err := ParsePath(s + "://h1:2379"); err == nil {
			return s
		}
	}
	t.Fatal("ParsePath accepts no scheme")
	return ""
}

// TestDetail01: a scheme gate exists — an unrelated scheme is rejected while
// the accepted scheme parses.
func TestDetail01(t *testing.T) {
	s := bbScheme(t)
	if _, _, _, err := ParsePath(s + "://h1:2379"); err != nil {
		t.Fatalf("accepted scheme %q rejected: %v", s, err)
	}
	if _, _, _, err := ParsePath("bogus-scheme://h1:2379"); err == nil {
		t.Fatalf("unrelated scheme accepted")
	}
}

// TestDetail02: disableGC accepts true/false (case-insensitive) and empty;
// anything else errors.
func TestDetail02(t *testing.T) {
	s := bbScheme(t)
	if _, d, _, err := ParsePath(s + "://h1?disableGC=true"); err != nil || !d {
		t.Fatalf("disableGC=true -> %v %v", d, err)
	}
	if _, d, _, err := ParsePath(s + "://h1?disableGC=false"); err != nil || d {
		t.Fatalf("disableGC=false -> %v %v", d, err)
	}
	if _, d, _, err := ParsePath(s + "://h1?disableGC=TRUE"); err != nil || !d {
		t.Fatalf("disableGC=TRUE -> %v %v", d, err)
	}
	if _, d, _, err := ParsePath(s + "://h1?disableGC="); err != nil || d {
		t.Fatalf("disableGC= (empty) -> %v %v", d, err)
	}
	if _, _, _, err := ParsePath(s + "://h1?disableGC=foo"); err == nil {
		t.Fatalf("disableGC=foo accepted")
	}
}

// TestDetail03: keyspaceName passes through verbatim (default ""), etcdAddrs
// is the comma-split host list, and an empty host yields a single empty
// element.
func TestDetail03(t *testing.T) {
	s := bbScheme(t)
	a, _, k, err := ParsePath(s + "://h1:2379,h2:2380?keyspaceName=abc")
	if err != nil || k != "abc" {
		t.Fatalf("keyspaceName: %q %v", k, err)
	}
	if len(a) != 2 || a[0] != "h1:2379" || a[1] != "h2:2380" {
		t.Fatalf("etcdAddrs = %v", a)
	}
	_, _, k, err = ParsePath(s + "://h1")
	if err != nil || k != "" {
		t.Fatalf("absent keyspaceName: %q %v", k, err)
	}
	a, _, _, err = ParsePath(s + "://")
	if err != nil || len(a) != 1 || a[0] != "" {
		t.Fatalf("empty host etcdAddrs = %v, want [\"\"]", a)
	}
}

// TestDetail04: TxnLocalLatches.Valid errors only when Enabled && Capacity==0.
func TestDetail04(t *testing.T) {
	if err := (&TxnLocalLatches{Enabled: true, Capacity: 0}).Valid(); err == nil {
		t.Fatalf("Valid() on enabled+zero capacity must error")
	}
	for _, c := range []TxnLocalLatches{
		{Enabled: true, Capacity: 5},
		{Enabled: false, Capacity: 0},
		{Enabled: false, Capacity: 5},
	} {
		if err := c.Valid(); err != nil {
			t.Fatalf("Valid() on %+v errored: %v", c, err)
		}
	}
}

// TestDetail05: GetTxnScopeFromConfig returns the configured TxnScope when
// non-empty and falls back to oracle.GlobalTxnScope otherwise.
func TestDetail05(t *testing.T) {
	orig := GetGlobalConfig()
	defer StoreGlobalConfig(orig)
	conf := *GetGlobalConfig()
	conf.TxnScope = ""
	StoreGlobalConfig(&conf)
	if got := GetTxnScopeFromConfig(); got != oracle.GlobalTxnScope {
		t.Fatalf("empty TxnScope -> %q, want the global-scope default", got)
	}
	conf.TxnScope = "local-test-scope"
	StoreGlobalConfig(&conf)
	if got := GetTxnScopeFromConfig(); got != "local-test-scope" {
		t.Fatalf("configured TxnScope -> %q", got)
	}
}
