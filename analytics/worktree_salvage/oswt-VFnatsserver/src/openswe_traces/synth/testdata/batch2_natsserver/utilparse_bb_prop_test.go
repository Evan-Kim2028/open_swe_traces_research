// Hidden black-box property suite for the utilparse unit.
// Drives only the API in api.md: version/parsers/hostport/url helpers,
// refCountedUrlSet, listen/dial, redact, copy, INFO, task queue, saturating
// math. One property per DETAILS.md line; seeded random inputs.
package server

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

const bbUtilHiddenSeed = 20260919

func bbUtilSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbUtilHiddenSeed
}

func bbUtilRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbUtilSeed()))
}

// Detail 1: versionComponents — strict major.minor.patch ints with optional
// valid -prerelease/+build; non-matching -> "invalid semver" error + zeros.
func TestDetail01_VersionComponents(t *testing.T) {
	valid := map[string][3]int{
		"0.0.0":                 {0, 0, 0},
		"1.2.3":                 {1, 2, 3},
		"10.20.30":              {10, 20, 30},
		"2.11.0-beta":           {2, 11, 0},
		"1.2.3-alpha.1":         {1, 2, 3},
		"1.2.3-beta+build.7":    {1, 2, 3},
		"1.2.3+exp.sha.5114f85": {1, 2, 3},
		"1.2.3-rc.1+meta":       {1, 2, 3},
	}
	for v, want := range valid {
		ma, mi, pa, err := versionComponents(v)
		if err != nil {
			t.Fatalf("%q: %v", v, err)
		}
		if [3]int{ma, mi, pa} != want {
			t.Fatalf("%q -> %d.%d.%d want %v", v, ma, mi, pa, want)
		}
	}
	invalid := []string{
		"", "1", "1.2", "1.2.3.4", "v1.2.3", "1.2.x", "abc", "1.2.3-",
		"01.2.3", "1.02.3", "1.2.03", "-1.2.3", "1.2.3-", "1.2.3+",
		"1.2.3-beta..x", "1..2.3", " 1.2.3", "1.2.3 ",
	}
	for _, v := range invalid {
		ma, mi, pa, err := versionComponents(v)
		if err == nil {
			t.Fatalf("%q should error", v)
		}
		if ma != 0 || mi != 0 || pa != 0 {
			t.Fatalf("%q -> %d.%d.%d want 0.0.0 on error", v, ma, mi, pa)
		}
	}
}

// Detail 2: versionAtLeastCheckError — numeric lexicographic compare;
// error -> false+err; versionAtLeast swallows err -> false.
func TestDetail02_VersionAtLeast(t *testing.T) {
	rng := bbUtilRng(t)
	for i := 0; i < 60; i++ {
		ma, mi, pa := rng.Intn(5), rng.Intn(5), rng.Intn(5)
		ea, ei, ep := rng.Intn(5), rng.Intn(5), rng.Intn(5)
		v := strconv.Itoa(ma) + "." + strconv.Itoa(mi) + "." + strconv.Itoa(pa)
		want := ma > ea || (ma == ea && (mi > ei || (mi == ei && pa >= ep)))
		got, err := versionAtLeastCheckError(v, ea, ei, ep)
		if err != nil {
			t.Fatalf("%v: %v", v, err)
		}
		if got != want {
			t.Fatalf("versionAtLeastCheckError(%s,%d,%d,%d)=%v want %v", v, ea, ei, ep, got, want)
		}
		if versionAtLeast(v, ea, ei, ep) != want {
			t.Fatalf("versionAtLeast(%s) mismatch", v)
		}
	}
	// Error path: invalid version -> false + error; AtLeast -> false.
	got, err := versionAtLeastCheckError("garbage", 1, 0, 0)
	if err == nil || got {
		t.Fatalf("invalid version: got=%v err=%v", got, err)
	}
	if versionAtLeast("garbage", 1, 0, 0) {
		t.Fatal("versionAtLeast on invalid should be false")
	}
	// Equal versions are "at least" at patch level.
	ok, err := versionAtLeastCheckError("2.3.4", 2, 3, 4)
	if err != nil || !ok {
		t.Fatalf("equal version should pass: %v %v", ok, err)
	}
}

// Detail 3: parseSize — exactly 1-9 ASCII digits -> int; empty, >9 bytes,
// or any non-digit -> -1.
func TestDetail03_ParseSize(t *testing.T) {
	rng := bbUtilRng(t)
	for i := 0; i < 50; i++ {
		n := rng.Intn(999999999)
		s := strconv.Itoa(n)
		if got := parseSize([]byte(s)); got != n {
			t.Fatalf("parseSize(%q)=%d want %d", s, got, n)
		}
	}
	if got := parseSize([]byte("123456789")); got != 123456789 {
		t.Fatalf("9-digit boundary: %d", got)
	}
	for _, s := range []string{"", "1234567890", "12345678901", "12a45", " 123", "123 ", "-1", "+5", "1.5"} {
		if got := parseSize([]byte(s)); got != -1 {
			t.Fatalf("parseSize(%q)=%d want -1", s, got)
		}
	}
}

// Detail 4: parseInt64 — digit run -> int64 with silent overflow wrap;
// empty or non-digit -> -1.
func TestDetail04_ParseInt64(t *testing.T) {
	rng := bbUtilRng(t)
	for i := 0; i < 50; i++ {
		n := rng.Int63()
		s := strconv.FormatInt(n, 10)
		if got := parseInt64([]byte(s)); got != n {
			t.Fatalf("parseInt64(%q)=%d want %d", s, got, n)
		}
	}
	// Silent wrap: 2^64 -> 0.
	if got := parseInt64([]byte("18446744073709551616")); got != 0 {
		t.Fatalf("2^64 wrap=%d want 0", got)
	}
	// 2^63 wraps to MinInt64.
	if got := parseInt64([]byte("9223372036854775808")); got != math.MinInt64 {
		t.Fatalf("2^63 wrap=%d want MinInt64", got)
	}
	for _, s := range []string{"", "abc", "12a", "a12", "-5", " 42", "42 "} {
		if got := parseInt64([]byte(s)); got != -1 {
			t.Fatalf("parseInt64(%q)=%d want -1", s, got)
		}
	}
}

// Detail 5: parseUint64 — digit run -> (v,true); empty/non-digit/overflow
// -> (0,false).
func TestDetail05_ParseUint64(t *testing.T) {
	rng := bbUtilRng(t)
	for i := 0; i < 50; i++ {
		n := rng.Uint64()
		s := strconv.FormatUint(n, 10)
		v, ok := parseUint64([]byte(s))
		if !ok || v != n {
			t.Fatalf("parseUint64(%q)=(%d,%v) want (%d,true)", s, v, ok, n)
		}
	}
	// Overflow: 2^64.
	v, ok := parseUint64([]byte("18446744073709551616"))
	if ok || v != 0 {
		t.Fatalf("overflow=(%d,%v) want (0,false)", v, ok)
	}
	for _, s := range []string{"", "x", "12a", "-1", "1.5", " 9"} {
		v, ok := parseUint64([]byte(s))
		if ok || v != 0 {
			t.Fatalf("parseUint64(%q)=(%d,%v) want (0,false)", s, v, ok)
		}
	}
}

// Detail 6: secondsToDuration — float seconds -> nanosecond duration
// truncated to int64.
func TestDetail06_SecondsToDuration(t *testing.T) {
	rng := bbUtilRng(t)
	for i := 0; i < 40; i++ {
		f := rng.Float64() * 100
		got := secondsToDuration(f)
		want := time.Duration(f * float64(time.Second))
		if got != want {
			t.Fatalf("secondsToDuration(%v)=%v want %v", f, got, want)
		}
	}
	if got := secondsToDuration(1.5); got != 1500*time.Millisecond {
		t.Fatalf("1.5s=%v", got)
	}
	if got := secondsToDuration(0.001); got != time.Millisecond {
		t.Fatalf("1ms=%v", got)
	}
}

// Detail 7: parseHostPort — trims spaces on host AND port; missing port ->
// default; explicit 0 or -1 -> default; non-numeric/malformed/empty ->
// error with port -1.
func TestDetail07_ParseHostPort(t *testing.T) {
	const def = 4222
	cases := []struct {
		in   string
		host string
		port int
		err  bool
	}{
		{"localhost:1234", "localhost", 1234, false},
		{" localhost : 1234 ", "localhost", 1234, false},
		{"localhost", "localhost", def, false},
		{"localhost:0", "localhost", def, false},
		{"localhost:-1", "localhost", def, false},
		{"10.0.0.1:80", "10.0.0.1", 80, false},
		{"  spaced-host ", "spaced-host", def, false},
		{"localhost:abc", "", -1, true},
		{"", "", -1, true},
		{"localhost:1.5", "", -1, true},
	}
	for _, c := range cases {
		host, port, err := parseHostPort(c.in, def)
		if c.err {
			if err == nil {
				t.Fatalf("%q should error", c.in)
			}
			if port != -1 {
				t.Fatalf("%q port=%d want -1 on error", c.in, port)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if host != c.host || port != c.port {
			t.Fatalf("%q -> (%q,%d) want (%q,%d)", c.in, host, port, c.host, c.port)
		}
	}
}

// Detail 8: urlsAreEqual — DeepEqual; distinguishes "user:" (empty password
// set) from "user" (no password); nil==nil true.
func TestDetail08_URLsAreEqual(t *testing.T) {
	if !urlsAreEqual(nil, nil) {
		t.Fatal("nil==nil should be true")
	}
	mk := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	a := mk("nats://user:pass@host:4222")
	if !urlsAreEqual(a, a) {
		t.Fatal("same pointer not equal")
	}
	b := mk("nats://user:pass@host:4222")
	if !urlsAreEqual(a, b) {
		t.Fatal("identical URLs not equal")
	}
	c := mk("nats://user:pass@host:4223")
	if urlsAreEqual(a, c) {
		t.Fatal("different ports equal")
	}
	// user: (empty password set) != user (no password).
	d := mk("nats://user:@host")
	e := mk("nats://user@host")
	if urlsAreEqual(d, e) {
		t.Fatal("'user:' should differ from 'user'")
	}
	if urlsAreEqual(a, nil) || urlsAreEqual(nil, a) {
		t.Fatal("nil vs non-nil equal")
	}
}

// Detail 9: comma — groups of 3, zero-padded interior groups, sign
// preserved; MinInt64 special-cased.
func TestDetail09_Comma(t *testing.T) {
	cases := map[int64]string{
		0:                  "0",
		1:                  "1",
		12:                 "12",
		123:                "123",
		1234:               "1,234",
		12345:              "12,345",
		123456:             "123,456",
		1234567:            "1,234,567",
		-1:                 "-1",
		-1234:              "-1,234",
		-1234567:           "-1,234,567",
		1000:               "1,000",
		1000000:            "1,000,000",
		math.MaxInt64:      "9,223,372,036,854,775,807",
		math.MinInt64:      "-9,223,372,036,854,775,808",
		1000000000:         "1,000,000,000",
		-92233720368547758: "-92,233,720,368,547,758",
	}
	for v, want := range cases {
		if got := comma(v); got != want {
			t.Fatalf("comma(%d)=%q want %q", v, got, want)
		}
	}
	// Random values: strip commas and reparse to check correctness.
	rng := bbUtilRng(t)
	for i := 0; i < 30; i++ {
		v := rng.Int63()
		got := comma(v)
		back, err := strconv.ParseInt(strings.ReplaceAll(got, ",", ""), 10, 64)
		if err != nil || back != v {
			t.Fatalf("comma(%d)=%q does not round-trip", v, got)
		}
		// Interior groups are zero-padded to 3.
		parts := strings.Split(got, ",")
		for j, p := range parts {
			if j == 0 {
				continue
			}
			if len(p) != 3 {
				t.Fatalf("comma(%d)=%q group %d %q not 3 digits", v, got, j, p)
			}
		}
	}
}

// Detail 10: refCountedUrlSet — addUrl true iff first reference; removeUrl
// true iff count reaches zero (key deleted); absent -> false no-op;
// getAsStringSlice unique keys.
func TestDetail10_RefCountedURLSet(t *testing.T) {
	rng := bbUtilRng(t)
	m := make(refCountedUrlSet)
	add := m.addUrl
	rm := m.removeUrl
	keys := m.getAsStringSlice
	if !add("nats://a:1") {
		t.Fatal("first add should be true")
	}
	if add("nats://a:1") {
		t.Fatal("second add should be false")
	}
	if !add("nats://b:2") {
		t.Fatal("new key add should be true")
	}
	got := keys()
	if len(got) != 2 {
		t.Fatalf("keys=%v want 2 unique", got)
	}
	seen := map[string]bool{}
	for _, k := range got {
		seen[k] = true
	}
	if !seen["nats://a:1"] || !seen["nats://b:2"] {
		t.Fatalf("keys=%v missing entries", got)
	}
	// removeUrl: count 2->1 false, 1->0 true (key deleted).
	if rm("nats://a:1") {
		t.Fatal("remove to count 1 should be false")
	}
	if !rm("nats://a:1") {
		t.Fatal("remove to count 0 should be true")
	}
	// Absent key -> false no-op.
	if rm("nats://absent:9") {
		t.Fatal("absent remove should be false")
	}
	if rm("nats://a:1") {
		t.Fatal("already-deleted remove should be false")
	}
	// Re-add after delete counts as first reference again.
	if !add("nats://a:1") {
		t.Fatal("re-add after delete should be true")
	}
	// Randomized add/remove sequence vs oracle.
	m2 := make(refCountedUrlSet)
	add2, rm2 := m2.addUrl, m2.removeUrl
	oracle := map[string]int{}
	for i := 0; i < 200; i++ {
		k := "nats://h:" + strconv.Itoa(rng.Intn(8))
		if rng.Intn(2) == 0 {
			got := add2(k)
			want := oracle[k] == 0
			oracle[k]++
			if got != want {
				t.Fatalf("iter %d add(%s)=%v want %v (count was %d)", i, k, got, want, oracle[k]-1)
			}
		} else {
			got := rm2(k)
			var want bool
			if oracle[k] > 0 {
				oracle[k]--
				want = oracle[k] == 0
				if want {
					delete(oracle, k)
				}
			}
			if got != want {
				t.Fatalf("iter %d rm(%s)=%v want %v", i, k, got, want)
			}
		}
	}
}

// Detail 11: natsListen/natsDialTimeout — TCP keepalive disabled (-1).
func TestDetail11_KeepaliveDisabled(t *testing.T) {
	// The shared listen config disables keepalive.
	if natsListenConfig.KeepAlive != -1 {
		t.Fatalf("natsListenConfig.KeepAlive=%v want -1", natsListenConfig.KeepAlive)
	}
	ln, err := natsListen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	// Accept a dial and check SO_KEEPALIVE off on both ends.
	done := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err == nil {
			done <- c
		}
	}()
	conn, err := natsDialTimeout("tcp", ln.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ka := func(c net.Conn) int {
		tc, ok := c.(*net.TCPConn)
		if !ok {
			return -99
		}
		var val int
		sc, err := tc.SyscallConn()
		if err != nil {
			return -98
		}
		sc.Control(func(fd uintptr) {
			v, err := syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
			if err == nil {
				val = v
			}
		})
		return val
	}
	if v := ka(conn); v != 0 {
		t.Fatalf("dial conn SO_KEEPALIVE=%d want 0", v)
	}
	select {
	case srv := <-done:
		defer srv.Close()
		if v := ka(srv); v != 0 {
			t.Fatalf("accepted conn SO_KEEPALIVE=%d want 0", v)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("accept timeout")
	}
	// The shared config also yields keepalive-off listeners directly.
	ln2, err := natsListenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ln2.Close()
}

// Detail 12: redactURLList — nil entries stay nil; passwordless entries
// keep original pointers; no-entry-needed-redaction returns the ORIGINAL
// slice; else copies with password "xxxxx".
func TestDetail12_RedactURLList(t *testing.T) {
	mk := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	withpw := mk("nats://user:secret@host1:4222")
	nopw := mk("nats://user@host2:4222")
	nouser := mk("nats://host3:4222")
	in := []*url.URL{withpw, nopw, nil, nouser}
	out := redactURLList(in)
	if len(out) != 4 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0] == withpw {
		t.Fatal("password URL was not copied")
	}
	if !strings.Contains(out[0].String(), "xxxxx") || strings.Contains(out[0].String(), "secret") {
		t.Fatalf("redacted=%q want xxxxx", out[0])
	}
	if out[1] != nopw {
		t.Fatal("passwordless URL should keep original pointer")
	}
	if out[2] != nil {
		t.Fatal("nil entry not preserved")
	}
	if out[3] != nouser {
		t.Fatal("userless URL should keep original pointer")
	}
	// No redaction needed -> SAME backing array returned.
	in2 := []*url.URL{nopw, nouser, nil}
	out2 := redactURLList(in2)
	if len(out2) != len(in2) || &out2[0] != &in2[0] {
		t.Fatal("all-clean input should return the original slice")
	}
	// 'user:' with empty password counts as having a password? It has
	// password SET — empty string is still a password field; redaction
	// behavior: verify contract is applied consistently (copy iff
	// password set).
	emptypw := mk("nats://user:@host4")
	out3 := redactURLList([]*url.URL{emptypw})
	if out3[0] == emptypw {
		t.Fatal("empty-password URL treated as passwordless — password is SET")
	}
}

// Detail 13: redactURLString — no '@' -> verbatim; parse failure ->
// verbatim; else URL Redacted().
func TestDetail13_RedactURLString(t *testing.T) {
	if got := redactURLString("nats://host:4222"); got != "nats://host:4222" {
		t.Fatalf("no @ verbatim: %q", got)
	}
	if got := redactURLString("not-a-url"); got != "not-a-url" {
		t.Fatalf("plain verbatim: %q", got)
	}
	got := redactURLString("nats://user:secret@host:4222")
	if !strings.Contains(got, "xxxxx") || strings.Contains(got, "secret") {
		t.Fatalf("redacted=%q", got)
	}
	// Parse failure with '@' -> verbatim.
	bad := "://user:pw@"
	if got := redactURLString(bad); got != bad {
		t.Fatalf("unparseable should be verbatim: %q", got)
	}
}

// Detail 14: getURLsAsString — each URL's Host.
func TestDetail14_GetURLsAsString(t *testing.T) {
	mk := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	urls := []*url.URL{mk("nats://u:p@h1:1/x"), mk("nats://h2:2"), mk("nats://h3")}
	got := getURLsAsString(urls)
	if len(got) != 3 || got[0] != "h1:1" || got[1] != "h2:2" || got[2] != "h3" {
		t.Fatalf("getURLsAsString=%v", got)
	}
}

// Detail 15: copyBytes — nil OR empty -> nil. copyStrings — nil -> nil;
// empty non-nil -> empty non-nil.
func TestDetail15_CopyAsymmetry(t *testing.T) {
	if got := copyBytes(nil); got != nil {
		t.Fatalf("copyBytes(nil)=%v want nil", got)
	}
	if got := copyBytes([]byte{}); got != nil {
		t.Fatalf("copyBytes(empty)=%v want nil", got)
	}
	src := []byte{1, 2, 3}
	cp := copyBytes(src)
	if len(cp) != 3 || cp[0] != 1 || cp[2] != 3 {
		t.Fatalf("copyBytes=%v", cp)
	}
	src[0] = 99
	if cp[0] != 1 {
		t.Fatal("copyBytes shares backing")
	}
	if got := copyStrings(nil); got != nil {
		t.Fatalf("copyStrings(nil)=%v want nil", got)
	}
	empty := copyStrings([]string{})
	if empty == nil || len(empty) != 0 {
		t.Fatalf("copyStrings(empty)=%v want non-nil empty", empty)
	}
	ss := []string{"a", "b"}
	cs := copyStrings(ss)
	if len(cs) != 2 || cs[0] != "a" {
		t.Fatalf("copyStrings=%v", cs)
	}
}

// Detail 16: generateInfoJSON — "INFO"+" "+json+" "+CRLF (space before the
// line terminator).
func TestDetail16_GenerateInfoJSON(t *testing.T) {
	info := &Info{ID: "TEST-ID", Version: "9.9.9", Port: 4222, MaxPayload: 1024}
	got := generateInfoJSON(info)
	s := string(got)
	if !strings.HasPrefix(s, "INFO ") {
		t.Fatalf("prefix=%q want \"INFO \"", s[:10])
	}
	if !strings.HasSuffix(s, " \r\n") {
		t.Fatalf("suffix=%q want \" \\r\\n\"", s[len(s)-4:])
	}
	body := s[len("INFO ") : len(s)-len(" \r\n")]
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if m["server_id"] != "TEST-ID" || m["version"] != "9.9.9" {
		t.Fatalf("json=%v", m)
	}
	// Cross-check with json.Marshal — body must equal the marshaled info.
	jb, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if body != string(jb) {
		t.Fatalf("body=%q want %q", body, jb)
	}
}

// Detail 17: parallelTaskQueue — mp<=0 -> GOMAXPROCS workers; mp>0 ->
// max(mp, GOMAXPROCS) — never fewer than GOMAXPROCS.
func TestDetail17_ParallelTaskQueue(t *testing.T) {
	gomax := runtime.GOMAXPROCS(0)
	barrier := func(n int, ch chan<- func()) bool {
		arrived := make(chan struct{}, n)
		release := make(chan struct{})
		var count atomic.Int32
		for i := 0; i < n; i++ {
			ch <- func() {
				count.Add(1)
				arrived <- struct{}{}
				<-release
			}
		}
		timer := time.After(10 * time.Second)
		for i := 0; i < n; i++ {
			select {
			case <-arrived:
			case <-timer:
				close(release)
				return false
			}
		}
		close(release)
		return int(count.Load()) == n
	}
	// mp=1 must still get GOMAXPROCS workers.
	ch := parallelTaskQueue(1)
	if !barrier(gomax, ch) {
		t.Fatalf("mp=1: fewer than GOMAXPROCS(%d) workers", gomax)
	}
	close(ch)
	// mp=0 same.
	ch = parallelTaskQueue(0)
	if !barrier(gomax, ch) {
		t.Fatalf("mp=0: fewer than GOMAXPROCS(%d) workers", gomax)
	}
	close(ch)
	// mp>GOMAXPROCS scales up.
	extra := gomax + 4
	ch = parallelTaskQueue(extra)
	if !barrier(extra, ch) {
		t.Fatalf("mp=%d: fewer than %d workers", extra, extra)
	}
	close(ch)
}

// Detail 18: addSaturate — a+b clamped at type max; mulSaturate — a*b
// clamped at MaxInt64.
func TestDetail18_SaturatingMath(t *testing.T) {
	rng := bbUtilRng(t)
	if got := addSaturate[int64](math.MaxInt64-1, 5); got != math.MaxInt64 {
		t.Fatalf("signed saturate=%d", got)
	}
	if got := addSaturate[uint64](math.MaxUint64-1, 5); got != math.MaxUint64 {
		t.Fatalf("unsigned saturate=%d", got)
	}
	if got := addSaturate[int64](3, 4); got != 7 {
		t.Fatalf("add=%d", got)
	}
	if got := addSaturate[uint64](3, 4); got != 7 {
		t.Fatalf("uadd=%d", got)
	}
	for i := 0; i < 40; i++ {
		a, b := rng.Int63n(1<<40), rng.Int63n(1<<40)
		got := addSaturate[int64](a, b)
		want := a + b
		if a > math.MaxInt64-b {
			want = math.MaxInt64
		}
		if got != want {
			t.Fatalf("addSaturate(%d,%d)=%d want %d", a, b, got, want)
		}
	}
	if got := mulSaturate(math.MaxInt64-1, 3); got != math.MaxInt64 {
		t.Fatalf("mul saturate=%d", got)
	}
	if got := mulSaturate(3, 4); got != 12 {
		t.Fatalf("mul=%d", got)
	}
	if got := mulSaturate(math.MaxInt64, 2); got != math.MaxInt64 {
		t.Fatalf("mul max=%d", got)
	}
	if got := mulSaturate(1<<32, 1<<32); got != math.MaxInt64 {
		t.Fatalf("mul 2^32*2^32=%d want MaxInt64", got)
	}
}
