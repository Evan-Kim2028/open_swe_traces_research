package util

import (
	"context"
	"strings"
	"testing"
)

// Hidden suite for unit reqsource. One TestDetailNN per DETAILS.md line.

// TestDetail01: nil receiver or both type fields empty yields "unknown", even
// when the internal flag is set.
func TestDetail01(t *testing.T) {
	if got := (*RequestSource)(nil).GetRequestSource(); got != SourceUnknown {
		t.Fatalf("nil receiver: %q", got)
	}
	if got := (&RequestSource{}).GetRequestSource(); got != SourceUnknown {
		t.Fatalf("empty source: %q", got)
	}
	if got := (&RequestSource{RequestSourceInternal: true}).GetRequestSource(); got != SourceUnknown {
		t.Fatalf("internal-but-typeless source: %q", got)
	}
}

// TestDetail02: the label is {internal|external}_{type} with an "unknown"
// type fallback, plus _{explicitType} when non-empty and different.
func TestDetail02(t *testing.T) {
	cases := []struct {
		rs   RequestSource
		want string
	}{
		{RequestSource{RequestSourceInternal: true, RequestSourceType: "test", ExplicitRequestSourceType: "lightning"}, "internal_test_lightning"},
		{RequestSource{RequestSourceInternal: false, ExplicitRequestSourceType: "lightning"}, "external_unknown_lightning"},
		{RequestSource{RequestSourceInternal: true, RequestSourceType: "gc"}, "internal_gc"},
		{RequestSource{RequestSourceInternal: false, RequestSourceType: "test"}, "external_test"},
	}
	for _, c := range cases {
		if got := c.rs.GetRequestSource(); got != c.want {
			t.Fatalf("GetRequestSource(%+v) = %q, want %q", c.rs, got, c.want)
		}
	}
	// explicit == type: the dedup is a choice — assert only the committed
	// shape that the label carries the scope and type.
	got := (&RequestSource{RequestSourceType: "test", ExplicitRequestSourceType: "test"}).GetRequestSource()
	if !strings.HasPrefix(got, "external_test") {
		t.Fatalf("explicit==type label = %q, want external_test prefix", got)
	}
}

// TestDetail03: IsInternalRequest recognizes the "internal" prefix; external
// and empty sources are not internal.
func TestDetail03(t *testing.T) {
	for _, s := range []string{"internal", "internal_gc", "internal_test_lightning"} {
		if !IsInternalRequest(s) {
			t.Fatalf("IsInternalRequest(%q) = false", s)
		}
	}
	for _, s := range []string{"", "external", "external_test", SourceUnknown} {
		if IsInternalRequest(s) {
			t.Fatalf("IsInternalRequest(%q) = true", s)
		}
	}
}

// TestDetail04: IsRequestSourceInternal is nil-safe and defers to the label;
// BuildRequestSource composes GetRequestSource.
func TestDetail04(t *testing.T) {
	if IsRequestSourceInternal(nil) {
		t.Fatalf("nil source reported internal")
	}
	in := &RequestSource{RequestSourceInternal: true, RequestSourceType: "gc"}
	if !IsRequestSourceInternal(in) {
		t.Fatalf("internal source not recognized")
	}
	out := &RequestSource{RequestSourceInternal: false, RequestSourceType: "test"}
	if IsRequestSourceInternal(out) {
		t.Fatalf("external source reported internal")
	}
	if got, want := BuildRequestSource(true, "test", "lightning"),
		(&RequestSource{RequestSourceInternal: true, RequestSourceType: "test", ExplicitRequestSourceType: "lightning"}).GetRequestSource(); got != want {
		t.Fatalf("BuildRequestSource = %q, want %q", got, want)
	}
}

// TestDetail05: the With* helpers store a RequestSource value under
// RequestSourceKey; RequestSourceFromCtx returns "unknown" when absent.
func TestDetail05(t *testing.T) {
	if got := RequestSourceFromCtx(context.Background()); got != SourceUnknown {
		t.Fatalf("absent ctx: %q", got)
	}
	ctx := WithInternalSourceType(context.Background(), InternalTxnGC)
	v, ok := ctx.Value(RequestSourceKey).(RequestSource)
	if !ok {
		t.Fatalf("stored value is %T, want RequestSource value", ctx.Value(RequestSourceKey))
	}
	if !v.RequestSourceInternal || v.RequestSourceType != InternalTxnGC {
		t.Fatalf("stored source = %+v", v)
	}
	if got := RequestSourceFromCtx(ctx); got == SourceUnknown {
		t.Fatalf("RequestSourceFromCtx after With returned unknown")
	}
	ctx2 := WithInternalSourceAndTaskType(context.Background(), InternalTxnOthers, "task")
	v2, ok := ctx2.Value(RequestSourceKey).(RequestSource)
	if !ok || !v2.RequestSourceInternal || v2.RequestSourceType != InternalTxnOthers {
		t.Fatalf("WithInternalSourceAndTaskType stored %+v (ok=%v)", v2, ok)
	}
}

// TestDetail06: resource group name uses its own context key; absent -> "".
func TestDetail06(t *testing.T) {
	if got := ResourceGroupNameFromCtx(context.Background()); got != "" {
		t.Fatalf("absent group name: %q", got)
	}
	ctx := WithResourceGroupName(context.Background(), "rg1")
	if got := ResourceGroupNameFromCtx(ctx); got != "rg1" {
		t.Fatalf("group name = %q", got)
	}
	// Distinct key: setting the group name must not plant a request source.
	if got := RequestSourceFromCtx(ctx); got != SourceUnknown {
		t.Fatalf("group ctx leaked request source %q", got)
	}
}
