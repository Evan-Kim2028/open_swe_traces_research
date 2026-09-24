package server

import (
	"net/http"
	"testing"
)

func ssClient(h http.Header) *client {
	c := &client{}
	c.parseState.header = h
	return c
}

var (
	ssUberSampled = http.Header{trcUber: {"a:b:c:1"}}
	ssUberDenied  = http.Header{trcUber: {"a:b:c:0"}}
	ssTpSampled   = http.Header{trcCtx: {"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"}}
)

// TestDetail01 (yes): a nil *serviceLatency returns (false, nil).
func TestDetail01(t *testing.T) {
	ok, hdr := shouldSample(nil, ssClient(ssUberSampled))
	if ok || hdr != nil {
		t.Fatalf("shouldSample(nil) = (%v, %v), want (false, nil)", ok, hdr)
	}
}

// TestDetail02 (yes): sampling < 0 returns (false, nil) without inspecting
// headers — even a header that would force sampling does not matter.
func TestDetail02(t *testing.T) {
	ok, hdr := shouldSample(&serviceLatency{sampling: -1}, ssClient(ssUberSampled))
	if ok || hdr != nil {
		t.Fatalf("sampling=-1 = (%v, %v), want (false, nil)", ok, hdr)
	}
}

// TestDetail03 (yes): sampling >= 100 returns (true, nil) without inspecting
// headers.
func TestDetail03(t *testing.T) {
	ok, hdr := shouldSample(&serviceLatency{sampling: 100}, ssClient(nil))
	if !ok || hdr != nil {
		t.Fatalf("sampling=100 = (%v, %v), want (true, nil)", ok, hdr)
	}
	ok, _ = shouldSample(&serviceLatency{sampling: 100}, ssClient(ssUberDenied))
	if !ok {
		t.Fatal("sampling=100 consulted headers")
	}
}

// TestDetail04 (shape — Inferable: no): the random branch samples when
// rand.Int32N(100) <= int32(sampling) — a configured 1 covers both 0 and 1,
// i.e. fires at roughly 2%, not 1%.
func TestDetail04(t *testing.T) {
	l := &serviceLatency{sampling: 1}
	c := ssClient(nil)
	const n = 20000
	hits := 0
	for i := 0; i < n; i++ {
		if ok, _ := shouldSample(l, c); ok {
			hits++
		}
	}
	// p=2% gives mean 400, sd ~20. Exclusive-bound (<) would give p=1% (~200).
	if hits <= 250 || hits >= 800 {
		t.Fatalf("sampling=1 hit %d of %d — not the committed ~2%% inclusive bound", hits, n)
	}
}

// TestDetail05 (partially): when the random branch does not sample,
// header-driven sampling is still consulted. With sampling=1 and an
// always-sampling header, every call must return true.
func TestDetail05(t *testing.T) {
	l := &serviceLatency{sampling: 1}
	c := ssClient(ssUberSampled)
	for i := 0; i < 200; i++ {
		if ok, _ := shouldSample(l, c); !ok {
			t.Fatal("sampling=1 with a sampled uber header returned false — headers not consulted after a random miss")
		}
	}
}

// TestDetail06 (yes): sampling == 0 skips the random branch and goes straight
// to headers.
func TestDetail06(t *testing.T) {
	l := &serviceLatency{sampling: 0}
	if ok, hdr := shouldSample(l, ssClient(nil)); ok || hdr != nil {
		t.Fatal("sampling=0 with no headers sampled")
	}
	if ok, _ := shouldSample(l, ssClient(ssUberSampled)); !ok {
		t.Fatal("sampling=0 with a sampled uber header did not sample")
	}
}

// TestDetail07 (yes): missing or empty parsed headers return (false, nil).
func TestDetail07(t *testing.T) {
	l := &serviceLatency{sampling: 0}
	for _, h := range []http.Header{nil, {}} {
		if ok, hdr := shouldSample(l, ssClient(h)); ok || hdr != nil {
			t.Fatalf("header %v -> (%v, %v), want (false, nil)", h, ok, hdr)
		}
	}
}

// TestDetail08 (shape — Inferable: no): Uber-Trace-Id is four
// colon-separated fields, the last 1-2 hex digits; sampling is on iff the
// decoded byte's low bit is set.
func TestDetail08(t *testing.T) {
	l := &serviceLatency{sampling: 0}
	for _, c := range []struct {
		val  string
		want bool
	}{
		{"a:b:c:1", true},   // 0x01 -> sampled
		{"a:b:c:0", false},  // low bit clear
		{"a:b:c:01", true},  // two hex digits
		{"a:b:c:2", false},  // even
		{"a:b:c:ff", true},  // low bit set
		{"a:b:c:001", false},// three digits — not the committed shape
		{"a:b:c", false},    // three fields
		{"a:b:c:1:e", false},// five fields
		{"a:b:c:zz", false}, // not hex
	} {
		ok, _ := shouldSample(l, ssClient(http.Header{trcUber: {c.val}}))
		if ok != c.want {
			t.Fatalf("Uber-Trace-Id %q -> %v, want %v", c.val, ok, c.want)
		}
	}
}

// TestDetail09 (yes): a sampled Uber header returns newUberHeader(h, tId).
func TestDetail09(t *testing.T) {
	tId := []string{"a:b:c:1"}
	h := http.Header{trcUber: tId, trcUberCtxPrefix + "Bag": {"v1"}}
	ok, hdr := shouldSample(&serviceLatency{sampling: 0}, ssClient(h))
	if !ok {
		t.Fatal("sampled uber header denied")
	}
	if got := hdr[trcUber]; len(got) != 1 || got[0] != "a:b:c:1" {
		t.Fatalf("returned header trcUber = %v", got)
	}
	if got := hdr[trcUberCtxPrefix+"Bag"]; len(got) != 1 || got[0] != "v1" {
		t.Fatalf("uberctx baggage not propagated: %v", got)
	}
}

// TestDetail10 (yes): X-B3-Sampled "1" samples and returns newB3Header(h).
func TestDetail10(t *testing.T) {
	h := http.Header{trcB3Sm: {"1"}, trcB3Id: {"abc"}}
	ok, hdr := shouldSample(&serviceLatency{sampling: 0}, ssClient(h))
	if !ok {
		t.Fatal("X-B3-Sampled=1 denied")
	}
	if got := hdr[trcB3Sm]; len(got) != 1 || got[0] != "1" {
		t.Fatalf("returned header lacks X-B3-Sampled: %v", hdr)
	}
	if got := hdr[trcB3Id]; len(got) != 1 || got[0] != "abc" {
		t.Fatalf("returned header lacks X-B3-TraceId: %v", hdr)
	}
}

// TestDetail11 (yes): X-B3-Sampled "0" denies even if a B3 trace id is
// present.
func TestDetail11(t *testing.T) {
	h := http.Header{trcB3Sm: {"0"}, trcB3Id: {"abc"}}
	if ok, _ := shouldSample(&serviceLatency{sampling: 0}, ssClient(h)); ok {
		t.Fatal("X-B3-Sampled=0 with a trace id sampled")
	}
}

// TestDetail12 (doc): presence of X-B3-TraceId without a deny samples and
// returns newB3Header(h).
func TestDetail12(t *testing.T) {
	h := http.Header{trcB3Id: {"abc123"}}
	ok, hdr := shouldSample(&serviceLatency{sampling: 0}, ssClient(h))
	if !ok {
		t.Fatal("lone X-B3-TraceId denied")
	}
	if got := hdr[trcB3Id]; len(got) != 1 || got[0] != "abc123" {
		t.Fatalf("returned header lacks X-B3-TraceId: %v", hdr)
	}
}

// TestDetail13 (shape — Inferable: no): a single-field B3 value "0" denies.
func TestDetail13(t *testing.T) {
	ok, _ := shouldSample(&serviceLatency{sampling: 0}, ssClient(http.Header{trcB3: {"0"}}))
	if ok {
		t.Fatal("single-field B3 \"0\" sampled")
	}
}

// TestDetail14 (shape — Inferable: no): a multi-field B3 denies when the
// third "-" field is "0"; any other third field samples.
func TestDetail14(t *testing.T) {
	l := &serviceLatency{sampling: 0}
	if ok, _ := shouldSample(l, ssClient(http.Header{trcB3: {"x-y-0-z"}})); ok {
		t.Fatal("B3 x-y-0-z sampled on third field 0")
	}
	for _, v := range []string{"x-y-d-z", "x-y-1-z", "x-y-s-z"} {
		if ok, _ := shouldSample(l, ssClient(http.Header{trcB3: {v}})); !ok {
			t.Fatalf("B3 %q denied although third field is not 0", v)
		}
	}
}

// TestDetail15 (shape — Inferable: no): a sampled combined B3 header comes
// back as http.Header{trcB3: b3} — the single combined header, not the split
// B3 fields.
func TestDetail15(t *testing.T) {
	v := "80f198ee56343ba864fe8b2a57d3eff7-e457b5a2e4d86bd1-1-05e3ac9a4f6e3b90"
	h := http.Header{trcB3: {v}}
	ok, hdr := shouldSample(&serviceLatency{sampling: 0}, ssClient(h))
	if !ok {
		t.Fatal("combined B3 denied")
	}
	if len(hdr) != 1 || len(hdr[trcB3]) != 1 || hdr[trcB3][0] != v {
		t.Fatalf("combined B3 returned %v, want just the B3 key with the original value", hdr)
	}
}

// TestDetail16 (shape — Inferable: no): W3C traceparent samples when it has
// four "-" fields, the flags field is two characters, and the parsed hex has
// bit 0x1 set.
func TestDetail16(t *testing.T) {
	l := &serviceLatency{sampling: 0}
	base := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7"
	for _, c := range []struct {
		val  string
		want bool
	}{
		{base + "-01", true},
		{base + "-00", false},  // bit clear
		{base + "-02", false},  // different bit
		{base + "-1", false},   // flags not 2 chars
		{base + "-011", false}, // flags 3 chars
		{"00-abc-def-01", true},// four fields + 2-char flags with the bit set — field lengths are not part of the rule
		{"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7", false}, // 3 fields
		{base + "-01-extra", false},                                    // 5 fields
	} {
		ok, _ := shouldSample(l, ssClient(http.Header{trcCtx: {c.val}}))
		if ok != c.want {
			t.Fatalf("traceparent %q -> %v, want %v", c.val, ok, c.want)
		}
	}
}

// TestDetail17 (yes): a sampled W3C header returns newTraceCtxHeader(h, tId).
func TestDetail17(t *testing.T) {
	tp := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	h := http.Header{trcCtx: {tp}, trcCtxSt: {"rojo=00f067aa0ba902b7"}}
	ok, hdr := shouldSample(&serviceLatency{sampling: 0}, ssClient(h))
	if !ok {
		t.Fatal("traceparent denied")
	}
	if got := hdr[trcCtx]; len(got) != 1 || got[0] != tp {
		t.Fatalf("returned header trcCtx = %v", got)
	}
	if got := hdr[trcCtxSt]; len(got) != 1 || got[0] != "rojo=00f067aa0ba902b7" {
		t.Fatalf("tracestate not propagated: %v", got)
	}
}

// TestDetail18 (yes): header checks run in order — Uber, then
// X-B3-Sampled/TraceId, then combined B3, then traceparent — and the first
// decision wins.
func TestDetail18(t *testing.T) {
	l := &serviceLatency{sampling: 0}
	tp := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	// Uber deny beats B3 sample and traceparent sample.
	h := http.Header{trcUber: {"a:b:c:0"}, trcB3Sm: {"1"}, trcCtx: {tp}}
	if ok, _ := shouldSample(l, ssClient(h)); ok {
		t.Fatal("uber deny lost to a later header")
	}
	// X-B3-Sampled wins over the combined B3 deny.
	h = http.Header{trcB3Sm: {"1"}, trcB3: {"x-y-0-z"}}
	ok, hdr := shouldSample(l, ssClient(h))
	if !ok {
		t.Fatal("X-B3-Sampled=1 lost to a later combined-B3 deny")
	}
	if len(hdr[trcB3Sm]) == 0 {
		t.Fatalf("expected the split-B3 header shape, got %v", hdr)
	}
	// Combined B3 deny beats traceparent sample.
	h = http.Header{trcB3: {"x-y-0-z"}, trcCtx: {tp}}
	if ok, _ := shouldSample(l, ssClient(h)); ok {
		t.Fatal("combined-B3 deny lost to traceparent")
	}
	// Combined B3 sample beats traceparent deny — and returns the combined form.
	h = http.Header{trcB3: {"x-y-1-z"}, trcCtx: {"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00"}}
	ok, hdr = shouldSample(l, ssClient(h))
	if !ok || len(hdr[trcB3]) == 0 {
		t.Fatalf("combined-B3 sample lost to traceparent deny: ok=%v hdr=%v", ok, hdr)
	}
}
