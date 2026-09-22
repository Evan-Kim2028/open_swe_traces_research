// Package terraformWriter_test is the hidden black-box suite for tfliterals.
// One TestDetailNN per DETAILS.md commitment. Literal.String + MarshalJSON +
// Write are the observation surface; dedupLiterals is observed through
// GetOutputs.
package terraformWriter_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	tw "example.internal/clustkit/upup/pkg/fi/cloudup/terraformWriter"
)

// Detail 1 (Inferable: yes — sanitizeName is kept in this package):
// LiteralProperty renders <type>.<sanitized-name>.<prop>; LiteralData adds a
// data. prefix.
func TestDetail01(t *testing.T) {
	if got := tw.LiteralProperty("aws_instance", "my.web", "id").String; got != "aws_instance.my-web.id" {
		t.Fatalf("LiteralProperty = %q", got)
	}
	if got := tw.LiteralData("aws_ami", "my.ami", "id").String; got != "data.aws_ami.my-ami.id" {
		t.Fatalf("LiteralData = %q", got)
	}
}

// Detail 2 (Inferable: yes): LiteralSelfLink = LiteralProperty(.., "self_link").
func TestDetail02(t *testing.T) {
	got := tw.LiteralSelfLink("aws_vpc", "main").String
	want := tw.LiteralProperty("aws_vpc", "main", "self_link").String
	if got != want || !strings.HasSuffix(got, "self_link") {
		t.Fatalf("LiteralSelfLink = %q", got)
	}
}

// Detail 3 (Inferable: yes): LiteralFunctionExpression renders fn(a, b).
func TestDetail03(t *testing.T) {
	got := tw.LiteralFunctionExpression("fn",
		tw.LiteralTokens("a"), tw.LiteralTokens("b")).String
	if got != "fn(a, b)" {
		t.Fatalf("LiteralFunctionExpression = %q", got)
	}
}

// Detail 4 (Inferable: yes): LiteralListExpression renders [a, b].
func TestDetail04(t *testing.T) {
	got := tw.LiteralListExpression(
		tw.LiteralTokens("a"), tw.LiteralTokens("b")).String
	if got != "[a, b]" {
		t.Fatalf("LiteralListExpression = %q", got)
	}
}

// Detail 5 (Inferable: no): LiteralFromStringValue wraps verbatim — content
// NOT escaped. Asserted shape: output is exactly quote + input + quote.
func TestDetail05(t *testing.T) {
	if got := tw.LiteralFromStringValue("plain").String; got != `"plain"` {
		t.Fatalf("LiteralFromStringValue = %q", got)
	}
	// verbatim: an embedded quote is passed through unescaped
	got := tw.LiteralFromStringValue(`a"b`).String
	if got != `"a"b"` {
		t.Fatalf("input quote escaped: %q", got)
	}
}

// Detail 6 (Inferable: yes): LiteralFromIntValue renders %d; LiteralTokens
// dot-joins.
func TestDetail06(t *testing.T) {
	if got := tw.LiteralFromIntValue(42).String; got != "42" {
		t.Fatalf("int = %q", got)
	}
	if got := tw.LiteralTokens("a", "b", "c").String; got != "a.b.c" {
		t.Fatalf("tokens = %q", got)
	}
}

// Detail 7 (Inferable: partially): LiteralWithIndex renders the string with
// an index-expression suffix inside quotes.
func TestDetail07(t *testing.T) {
	got := tw.LiteralWithIndex("s").String
	if !strings.HasPrefix(got, `"s-`) || !strings.Contains(got, "index") || !strings.HasSuffix(got, `"`) {
		t.Fatalf("LiteralWithIndex = %q", got)
	}
}

// Detail 8 (Inferable: yes): binary expression space-separated; index
// expression tight.
func TestDetail08(t *testing.T) {
	l := tw.LiteralTokens("a")
	r := tw.LiteralTokens("b")
	if got := tw.LiteralBinaryExpression(l, "==", r).String; got != "a == b" {
		t.Fatalf("binary = %q", got)
	}
	if got := tw.LiteralIndexExpression(tw.LiteralTokens("c"), tw.LiteralTokens("i")).String; got != "c[i]" {
		t.Fatalf("index = %q", got)
	}
}

// Detail 9 (Inferable: partially): empty-string conditional renders
// `<e> == "" ? null : <v>` — a ternary shape.
func TestDetail09(t *testing.T) {
	got := tw.LiteralEmptyStrConditionalExpression(
		tw.LiteralTokens("e"), tw.LiteralTokens("v")).String
	for _, part := range []string{"e", "==", `""`, "?", ":", "null", "v"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in %q", part, got)
		}
	}
	if strings.Index(got, "?") > strings.Index(got, ":") {
		t.Fatalf("ternary order wrong: %q", got)
	}
}

// Detail 10 (Inferable: no): Write emits `= <String>` + newline, ignoring the
// indent and key arguments.
func TestDetail10(t *testing.T) {
	l := tw.LiteralTokens("x", "y")
	var buf bytes.Buffer
	l.Write(&buf, 99, "ignoredkey")
	got := buf.String()
	if got != "= x.y\n" {
		t.Fatalf("Write emitted %q", got)
	}
	if strings.Contains(got, "ignoredkey") {
		t.Fatal("Write did not ignore the key argument")
	}
}

// Detail 11 (Inferable: partially): MarshalJSON marshals the String field —
// a bare JSON string.
func TestDetail11(t *testing.T) {
	l := tw.LiteralTokens("a", "b")
	b, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("not a bare JSON string: %s", b)
	}
	if s != "a.b" {
		t.Fatalf("marshaled %q, want literal string", s)
	}
}

// Detail 12 (Inferable: doc — doc comment states the sorted side effect):
// dedupLiterals drops duplicates and returns sorted order; nil in, nil out.
// Observed through GetOutputs ValueArray (dedup+sort on read).
func TestDetail12(t *testing.T) {
	w := &tw.TerraformWriter{}
	w.InitTerraformWriter()
	// insert out of order, with a duplicate
	if err := w.AddOutputVariableArray("k", tw.LiteralTokens("b")); err != nil {
		t.Fatal(err)
	}
	if err := w.AddOutputVariableArray("k", tw.LiteralTokens("a")); err != nil {
		t.Fatal(err)
	}
	if err := w.AddOutputVariableArray("k", tw.LiteralTokens("a")); err != nil {
		t.Fatal(err)
	}
	outs, err := w.GetOutputs()
	if err != nil {
		t.Fatal(err)
	}
	arr := outs["k"].ValueArray
	if len(arr) != 2 || arr[0].String != "a" || arr[1].String != "b" {
		t.Fatalf("ValueArray not deduped+sorted: %v", arr)
	}
}

// Detail 13 (Inferable: yes): SortLiterals sorts ascending by .String in
// place.
func TestDetail13(t *testing.T) {
	v := []*tw.Literal{tw.LiteralTokens("z"), tw.LiteralTokens("a"), tw.LiteralTokens("m")}
	tw.SortLiterals(v)
	if v[0].String != "a" || v[1].String != "m" || v[2].String != "z" {
		t.Fatalf("not sorted: %v", v)
	}
}
