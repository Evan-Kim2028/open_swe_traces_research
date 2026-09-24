package server

import (
	"net/url"
	"testing"
)

func damURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	return u
}

// TestDetail01 (yes): a nil URL in the slice is skipped.
func TestDetail01(t *testing.T) {
	san := dnsAltNameLabels("a.b")
	if dnsAltNameMatches(san, []*url.URL{nil}) {
		t.Fatal("nil-only URL slice matched")
	}
	if !dnsAltNameMatches(san, []*url.URL{nil, damURL(t, "nats://a.b")}) {
		t.Fatal("nil URL was not skipped; reachable match behind it missed")
	}
}

// TestDetail02 (yes): the URL hostname is split on "." after lowering.
func TestDetail02(t *testing.T) {
	san := dnsAltNameLabels("x.a.b")
	if !dnsAltNameMatches(san, []*url.URL{damURL(t, "nats://X.A.B")}) {
		t.Fatal("uppercase URL hostname did not match lowercase SAN")
	}
}

// TestDetail03 (doc): the SAN and the hostname must have the same number of
// labels; otherwise that URL cannot match.
func TestDetail03(t *testing.T) {
	if dnsAltNameMatches(dnsAltNameLabels("a.b"), []*url.URL{damURL(t, "nats://x.a.b")}) {
		t.Fatal("2-label SAN matched 3-label hostname")
	}
	if dnsAltNameMatches(dnsAltNameLabels("x.a.b"), []*url.URL{damURL(t, "nats://a.b")}) {
		t.Fatal("3-label SAN matched 2-label hostname")
	}
	if !dnsAltNameMatches(dnsAltNameLabels("a.b"), []*url.URL{damURL(t, "nats://a.b")}) {
		t.Fatal("equal-label SAN did not match")
	}
}

// TestDetail04 (doc): a SAN whose first label is exactly "*" may skip
// comparing label 0 and compare the rest.
func TestDetail04(t *testing.T) {
	if !dnsAltNameMatches(dnsAltNameLabels("*.a.b"), []*url.URL{damURL(t, "nats://x.a.b")}) {
		t.Fatal("leading-* SAN did not match a hostname differing only in label 0")
	}
	if dnsAltNameMatches(dnsAltNameLabels("*.a.b"), []*url.URL{damURL(t, "nats://x.b.b")}) {
		t.Fatal("leading-* SAN matched a hostname differing in a non-skipped label")
	}
}

// TestDetail05 (doc): a "*" in any later SAN label is compared as a literal,
// not a wildcard.
func TestDetail05(t *testing.T) {
	if dnsAltNameMatches(dnsAltNameLabels("a.*"), []*url.URL{damURL(t, "nats://a.b")}) {
		t.Fatal("non-leading * matched as a wildcard")
	}
	// Literal: the * label compares equal only to a literal "*" label.
	if !dnsAltNameMatches(dnsAltNameLabels("a.*"), []*url.URL{damURL(t, "nats://a.*")}) {
		t.Fatal("non-leading * did not compare equal to a literal * label")
	}
	if dnsAltNameMatches(dnsAltNameLabels("a.*.c"), []*url.URL{damURL(t, "nats://a.b.c")}) {
		t.Fatal("interior * label matched as a wildcard")
	}
}

// TestDetail06 (doc): "*" never spans more than one label, so *.a.b cannot
// match x.y.a.b.
func TestDetail06(t *testing.T) {
	if dnsAltNameMatches(dnsAltNameLabels("*.a.b"), []*url.URL{damURL(t, "nats://x.y.a.b")}) {
		t.Fatal("leading * spanned more than one label")
	}
}

// TestDetail07 (yes): remaining labels are compared with != after both sides
// are already lowercase.
func TestDetail07(t *testing.T) {
	if dnsAltNameMatches(dnsAltNameLabels("a.b"), []*url.URL{damURL(t, "nats://a.c")}) {
		t.Fatal("differing final label matched")
	}
	if !dnsAltNameMatches(dnsAltNameLabels("a.b"), []*url.URL{damURL(t, "nats://a.B")}) {
		t.Fatal("case-differing final label did not match")
	}
}

// TestDetail08 (yes): the first URL that survives the label walk returns true.
func TestDetail08(t *testing.T) {
	san := dnsAltNameLabels("a.b")
	if !dnsAltNameMatches(san, []*url.URL{damURL(t, "nats://x.y"), damURL(t, "nats://a.b"), damURL(t, "nats://q.r")}) {
		t.Fatal("no match reported although a later URL matched")
	}
}

// TestDetail09 (yes): if no URL matches, the result is false.
func TestDetail09(t *testing.T) {
	san := dnsAltNameLabels("a.b")
	if dnsAltNameMatches(san, []*url.URL{damURL(t, "nats://x.y"), damURL(t, "nats://c.d")}) {
		t.Fatal("reported match with no matching URL")
	}
	if dnsAltNameMatches(san, nil) {
		t.Fatal("reported match on nil URL slice")
	}
}

// TestDetail10 (partially): an empty hostname still participates in the
// equal-length check — it is one empty label.
func TestDetail10(t *testing.T) {
	empty := &url.URL{Scheme: "nats"} // Hostname() == ""
	// One-label SAN against a one-label (empty) hostname goes through the
	// label walk: wildcard label 0 is skipped -> match; literal mismatch -> no.
	if !dnsAltNameMatches(dnsAltNameLabels("*"), []*url.URL{empty}) {
		t.Fatal("single-* SAN did not match a single empty hostname label")
	}
	if dnsAltNameMatches(dnsAltNameLabels("a"), []*url.URL{empty}) {
		t.Fatal("literal SAN label matched an empty hostname label")
	}
	if dnsAltNameMatches(dnsAltNameLabels("a.b"), []*url.URL{empty}) {
		t.Fatal("2-label SAN matched a 1-label empty hostname")
	}
}
