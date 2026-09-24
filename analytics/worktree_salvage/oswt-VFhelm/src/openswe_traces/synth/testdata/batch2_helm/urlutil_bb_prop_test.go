// Hidden black-box suite for urlutil. Exported API only (api.md):
// URLJoin, Equal, ExtractHostname. One TestDetailNN per DETAILS.md line.
package urlutil_test

import (
	"math/rand"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"example.internal/helm/internal/urlutil"
)

const HiddenSeed int64 = 20260919

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return HiddenSeed
}

func TestDetail01_URLJoinPathOnlyPreservesSchemeHostQuery(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	schemes := []string{"http", "https", "oci"}
	for i := 0; i < 400; i++ {
		scheme := schemes[rng.Intn(len(schemes))]
		host := "h" + strconv.Itoa(rng.Intn(10000)) + ".example.test"
		qkey := "k" + strconv.Itoa(rng.Intn(50))
		qval := "v" + strconv.Itoa(rng.Intn(50))
		basePath := "/" + randSeg(rng) + "/" + randSeg(rng)
		base := scheme + "://" + host + basePath + "?" + qkey + "=" + qval
		extra := []string{randSeg(rng), randSeg(rng)}
		got, err := urlutil.URLJoin(base, extra...)
		if err != nil {
			t.Fatalf("i=%d URLJoin(%q, %v): %v", i, base, extra, err)
		}
		u, err := url.Parse(got)
		if err != nil {
			t.Fatalf("i=%d parse got %q: %v", i, got, err)
		}
		if u.Scheme != scheme {
			t.Fatalf("i=%d scheme: got %q want %q (out=%q)", i, u.Scheme, scheme, got)
		}
		if u.Host != host {
			t.Fatalf("i=%d host: got %q want %q (out=%q)", i, u.Host, host, got)
		}
		if u.RawQuery != qkey+"="+qval {
			t.Fatalf("i=%d query: got %q (out=%q)", i, u.RawQuery, got)
		}
		wantPath := path.Join(append([]string{basePath}, extra...)...)
		if u.Path != wantPath && u.EscapedPath() != wantPath {
			t.Fatalf("i=%d path: got %q want POSIX join %q (out=%q)", i, u.Path, wantPath, got)
		}
	}
}

func TestDetail02_URLJoinPathishBaseAndUnparseableError(t *testing.T) {
	got, err := urlutil.URLJoin("../example.com", "charts", "a.tgz")
	if err != nil {
		t.Fatalf("pathish base should still join, err=%v", err)
	}
	if !strings.Contains(got, "example.com") {
		t.Fatalf("pathish join lost host-like segment: %q", got)
	}
	if !strings.Contains(got, "charts") {
		t.Fatalf("pathish join dropped extra: %q", got)
	}

	rng := rand.New(rand.NewSource(hiddenSeed() + 1))
	bads := []string{
		"http://[::1",
		"://",
		"http://[",
		string([]byte{0xff, 0xfe}),
	}
	for i := 0; i < 80; i++ {
		base := bads[rng.Intn(len(bads))]
		_, err := urlutil.URLJoin(base, "x")
		if err == nil {
			// some inputs url.Parse accepts; only assert on ones that fail parse
			if _, perr := url.Parse(base); perr != nil && err == nil {
				t.Fatalf("unparseable base %q joined without error", base)
			}
		}
	}
	if _, jerr := urlutil.URLJoin("http://[::1", "x"); jerr == nil {
		t.Fatal("unparseable http://[::1 must error")
	}
}

func TestDetail03_EqualNormalizesEmptyPathAndCleansDots(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed() + 2))
	if !urlutil.Equal("http://ex.test", "http://ex.test/") {
		t.Fatal("empty path must normalize to /")
	}
	if !urlutil.Equal("http://ex.test/a/./b", "http://ex.test/a/b") {
		t.Fatal("dot segment must collapse")
	}
	if !urlutil.Equal("http://ex.test/a/foo/../b", "http://ex.test/a/b") {
		t.Fatal("dotdot must collapse")
	}
	if !urlutil.Equal("http://ex.test/a//b", "http://ex.test/a/b") {
		t.Fatal("dup slash must collapse")
	}
	if !urlutil.Equal("http://ex.test/a/b/", "http://ex.test/a/b") {
		t.Fatal("trailing slash must collapse on path")
	}
	for i := 0; i < 200; i++ {
		host := "h" + strconv.Itoa(rng.Intn(9999)) + ".t"
		seg := randSeg(rng)
		a := "https://" + host + "/" + seg + "/./x"
		b := "https://" + host + "/" + seg + "/x"
		if !urlutil.Equal(a, b) {
			t.Fatalf("i=%d Equal(%q,%q) = false", i, a, b)
		}
	}
}

func TestDetail04_EqualFallbackAsymmetry(t *testing.T) {
	unp := "http://[::1"
	if _, err := url.Parse(unp); err == nil {
		t.Fatalf("precondition: %q should be unparseable", unp)
	}
	cleaned := filepath.Clean(unp)
	if !urlutil.Equal(unp, cleaned) && !urlutil.Equal(unp, unp) {
		t.Fatalf("first-arg unparseable should filepath.Clean-compare; Equal(%q,%q)=false Equal(%q,%q)=false", unp, cleaned, unp, unp)
	}
	if !urlutil.Equal(unp, unp) {
		t.Fatal("first unparseable vs identical string must be true (clean-compare)")
	}
	if urlutil.Equal("http://ok.example/a", unp) {
		t.Fatal("second arg unparseable must yield false")
	}
	if urlutil.Equal("https://z.test/foo", "::::not-a-url") {
		t.Fatal("second unparseable must yield false even vs a valid first URL")
	}
	rng := rand.New(rand.NewSource(hiddenSeed() + 3))
	for i := 0; i < 120; i++ {
		second := "http://[" + strconv.Itoa(rng.Intn(99))
		if urlutil.Equal("http://good.test/p", second) {
			t.Fatalf("i=%d second unparseable %q must be false", i, second)
		}
	}
}

func TestDetail05_ExtractHostnameStripsPort(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed() + 4))
	for i := 0; i < 300; i++ {
		host := "n" + strconv.Itoa(rng.Intn(100000)) + ".example"
		port := 1 + rng.Intn(65534)
		addr := "https://" + host + ":" + strconv.Itoa(port) + "/x"
		got, err := urlutil.ExtractHostname(addr)
		if err != nil {
			t.Fatalf("i=%d ExtractHostname(%q): %v", i, addr, err)
		}
		if got != host {
			t.Fatalf("i=%d got %q want %q (port must strip)", i, got, host)
		}
	}
	got, err := urlutil.ExtractHostname("http://[2001:db8::1]:443/p")
	if err != nil {
		t.Fatal(err)
	}
	if got != "2001:db8::1" {
		t.Fatalf("ipv6 hostname got %q", got)
	}
	if _, err := urlutil.ExtractHostname("http://[::1"); err == nil {
		t.Fatal("parse error must propagate")
	}
}

func TestDetail06_URLJoinZeroComponentsUnchanged(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed() + 5))
	bases := []string{
		"http://ex.test/a/b?q=1",
		"https://h.test/",
		"oci://reg.test/ns/chart",
	}
	for _, b := range bases {
		got, err := urlutil.URLJoin(b)
		if err != nil {
			t.Fatalf("URLJoin(%q): %v", b, err)
		}
		if got != b {
			t.Fatalf("zero extras: got %q want %q", got, b)
		}
	}
	for i := 0; i < 80; i++ {
		b := "https://z.test/" + randSeg(rng) + "?x=" + strconv.Itoa(i)
		got, err := urlutil.URLJoin(b)
		if err != nil {
			t.Fatal(err)
		}
		if got != b {
			t.Fatalf("i=%d zero extras mutated %q -> %q", i, b, got)
		}
	}
}

func randSeg(rng *rand.Rand) string {
	n := 1 + rng.Intn(8)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(byte('a' + rng.Intn(26)))
	}
	return b.String()
}
