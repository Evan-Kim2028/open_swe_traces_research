package server

import (
	"strings"
	"testing"
	"time"
)

type osFakeTk struct {
	v   any
	used bool
}

func (t osFakeTk) Value() any         { return t.v }
func (t osFakeTk) Line() int          { return 1 }
func (t osFakeTk) IsUsedVariable() bool { return t.used }
func (t osFakeTk) SourceFile() string { return "f" }
func (t osFakeTk) Position() int      { return 1 }

// TestDetail01: parseDuration — string via time.ParseDuration (error →
// configErr, 0); bare int = seconds with a warning, not an error.
func TestDetail01(t *testing.T) {
	var errs, warns []error
	if d := parseDuration("wd", osFakeTk{v: "2s"}, "2s", &errs, &warns); d != 2*time.Second {
		t.Fatalf("parseDuration(2s) = %v", d)
	}
	if len(errs) != 0 || len(warns) != 0 {
		t.Fatalf("clean parse should record nothing, errs=%v warns=%v", errs, warns)
	}
	errs, warns = nil, nil
	if d := parseDuration("wd", osFakeTk{v: "bogus"}, "bogus", &errs, &warns); d != 0 || len(errs) != 1 {
		t.Fatalf("bad string should append a configErr and return 0, got %v errs=%v", d, errs)
	}
	errs, warns = nil, nil
	if d := parseDuration("wd", osFakeTk{v: int64(5)}, int64(5), &errs, &warns); d != 5*time.Second {
		t.Fatalf("bare int should scale as seconds, got %v", d)
	}
	if len(errs) != 0 || len(warns) != 1 {
		t.Fatalf("bare int should warn not error, errs=%v warns=%v", errs, warns)
	}
}

// TestDetail02: parseWriteDeadlinePolicy accepts only
// default|close|retry; bad input appends a configErr and falls back to
// WriteTimeoutPolicyDefault.
func TestDetail02(t *testing.T) {
	var errs []error
	for s, want := range map[string]WriteTimeoutPolicy{
		"default": WriteTimeoutPolicyDefault,
		"close":   WriteTimeoutPolicyClose,
		"retry":   WriteTimeoutPolicyRetry,
	} {
		if p := parseWriteDeadlinePolicy(osFakeTk{v: s}, s, &errs); p != want {
			t.Fatalf("policy %q = %d, want %d", s, p, want)
		}
	}
	if len(errs) != 0 {
		t.Fatalf("valid policies should not error, got %v", errs)
	}
	errs = nil
	if p := parseWriteDeadlinePolicy(osFakeTk{v: "bogus"}, "bogus", &errs); p != WriteTimeoutPolicyDefault || len(errs) != 1 {
		t.Fatalf("bad policy should fall back to default after a configErr, got %d errs=%v", p, errs)
	}
}

// TestDetail03: parseListen — int64 → port with empty host; string goes
// through SplitHostPort so a bare port string fails; port must be numeric;
// other types error.
func TestDetail03(t *testing.T) {
	hp, err := parseListen(int64(8922))
	if err != nil || hp.host != "" || hp.port != 8922 {
		t.Fatalf("int form: %+v err=%v", hp, err)
	}
	hp, err = parseListen("host:8922")
	if err != nil || hp.host != "host" || hp.port != 8922 {
		t.Fatalf("host:port form: %+v err=%v", hp, err)
	}
	if _, err := parseListen("8922"); err == nil {
		t.Fatal("bare port string must fail the host:port split")
	}
	if _, err := parseListen("host:abc"); err == nil {
		t.Fatal("non-numeric port must error")
	}
	for _, v := range []any{1.5, true} {
		if _, err := parseListen(v); err == nil {
			t.Fatalf("type %T must error", v)
		}
	}
}

// TestDetail04: parseURL trims then parses; parseURLs dedupes exact
// strings with a warning and accumulates per-entry errors without
// aborting.
func TestDetail04(t *testing.T) {
	u, err := parseURL("  nats://h:4222 ", "route")
	if err != nil || u.Host != "h:4222" {
		t.Fatalf("parseURL should trim then parse: %v err=%v", u, err)
	}
	if _, err := parseURL("::bad", "route"); err == nil {
		t.Fatal("unparsable URL must error")
	}
	var warns []error
	urls, errs := parseURLs([]any{
		osFakeTk{v: "nats://a:1"},
		osFakeTk{v: "nats://a:1"},
		osFakeTk{v: "nats://b:2"},
		osFakeTk{v: "::bad"},
	}, "route", &warns)
	if len(urls) != 2 {
		t.Fatalf("exact duplicates should dedupe, got %d urls", len(urls))
	}
	if len(errs) != 1 {
		t.Fatalf("one bad entry should produce one error without aborting, got %v", errs)
	}
	if len(warns) != 1 {
		t.Fatalf("a duplicate should produce a warning, got %v", warns)
	}
}

// TestDetail05: getStorageSize — int64 passthrough, "" → 0, K/M/G/T
// suffixes map to powers of two; non-numeric prefix or unknown suffix →
// error.
func TestDetail05(t *testing.T) {
	if n, err := getStorageSize(int64(99)); err != nil || n != 99 {
		t.Fatalf("int64 passthrough: %d %v", n, err)
	}
	if n, err := getStorageSize(""); err != nil || n != 0 {
		t.Fatalf("empty string: %d %v", n, err)
	}
	for s, want := range map[string]int64{
		"1K": 1 << 10, "1M": 1 << 20, "1G": 1 << 30, "1T": 1 << 40,
	} {
		if n, err := getStorageSize(s); err != nil || n != want {
			t.Fatalf("%s = %d %v, want %d", s, n, err, want)
		}
	}
	for _, s := range []string{"1L", "TT", "2k", "12x"} {
		if _, err := getStorageSize(s); err == nil {
			t.Fatalf("%q should error", s)
		}
	}
}

// TestDetail06: parseCompression — string mode verbatim, bool →
// chosenModeForOn / CompressionOff, map with `mode` and an rtt-threshold
// synonym list; unknown keys error unless the value is a used variable.
func TestDetail06(t *testing.T) {
	co := &CompressionOpts{}
	if err := parseCompression(co, CompressionS2Fast, osFakeTk{v: true}, "compression", true); err != nil || co.Mode != CompressionS2Fast {
		t.Fatalf("bool true should pick chosen mode: %+v err=%v", co, err)
	}
	co = &CompressionOpts{}
	if err := parseCompression(co, CompressionS2Fast, osFakeTk{v: false}, "compression", false); err != nil || co.Mode != CompressionOff {
		t.Fatalf("bool false should be off: %+v err=%v", co, err)
	}
	co = &CompressionOpts{}
	if err := parseCompression(co, CompressionS2Fast, osFakeTk{v: "s2_best"}, "compression", "s2_best"); err != nil || co.Mode != "s2_best" {
		t.Fatalf("string should be verbatim: %+v err=%v", co, err)
	}
	for _, k := range []string{"rtt_thresholds", "thresholds", "rtts", "rtt"} {
		co = &CompressionOpts{}
		err := parseCompression(co, CompressionS2Fast, osFakeTk{}, "compression",
			map[string]any{k: []any{"5ms"}})
		if err != nil || len(co.RTTThresholds) != 1 || co.RTTThresholds[0] != 5*time.Millisecond {
			t.Fatalf("key %q should fill thresholds: %+v err=%v", k, co, err)
		}
	}
	co = &CompressionOpts{}
	if err := parseCompression(co, CompressionS2Fast, osFakeTk{}, "compression",
		map[string]any{"mode": "s2_auto"}); err != nil || co.Mode != "s2_auto" {
		t.Fatalf("mode key should set Mode: %+v err=%v", co, err)
	}
	co = &CompressionOpts{}
	if err := parseCompression(co, CompressionS2Fast, osFakeTk{}, "compression",
		map[string]any{"bogus_key": int64(1)}); err == nil {
		t.Fatal("unknown key should error")
	}
	co = &CompressionOpts{}
	if err := parseCompression(co, CompressionS2Fast, osFakeTk{}, "compression",
		map[string]any{"bogus_key": osFakeTk{v: int64(1), used: true}}); err != nil {
		t.Fatalf("used-variable key should not error: %v", err)
	}
	co = &CompressionOpts{}
	if err := parseCompression(co, CompressionS2Fast, osFakeTk{v: int64(4)}, "compression", int64(4)); err == nil {
		t.Fatal("scalar non-bool should error")
	}
}

// TestDetail07: trackExplicitVal lazily allocates and records even false.
func TestDetail07(t *testing.T) {
	var pm map[string]bool
	trackExplicitVal(&pm, "a", false)
	trackExplicitVal(&pm, "b", true)
	if pm == nil || len(pm) != 2 {
		t.Fatalf("map should be allocated with both entries, got %v", pm)
	}
	if v, ok := pm["a"]; !ok || v {
		t.Fatal("explicit false must be recorded")
	}
	if !pm["b"] {
		t.Fatal("explicit true must be recorded")
	}
}

// TestDetail08: parsers take tokens and append to errors/warnings slices
// rather than returning errors — a bad duration surfaces through the
// slice as a positional config error, not a return value.
func TestDetail08(t *testing.T) {
	var errs []error
	parseDuration("wd", osFakeTk{v: "bad"}, "bad", &errs, &[]error{})
	if len(errs) != 1 {
		t.Fatalf("error must be appended, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "1:1") {
		t.Fatalf("config error should carry the token position, got %v", errs[0])
	}
}
