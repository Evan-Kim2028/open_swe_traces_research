// Package jsonutils_test is the hidden black-box suite for jsonstream.
// One TestDetailNN per DETAILS.md commitment. Exported API only:
// NewJSONStreamWriter, WriteToken, Path.
package jsonutils_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	ju "example.internal/clustkit/pkg/jsonutils"
)

func writeAll(t *testing.T, j interface {
	WriteToken(json.Token) error
	Path() string
}, toks ...json.Token) {
	t.Helper()
	for _, tok := range toks {
		if err := j.WriteToken(tok); err != nil {
			t.Fatalf("WriteToken(%v): %v", tok, err)
		}
	}
}

// Detail 1 (Inferable: yes): Path() returns enclosing field names dot-joined.
func TestDetail01(t *testing.T) {
	var buf bytes.Buffer
	j := ju.NewJSONStreamWriter(&buf)
	writeAll(t, j, json.Delim('{'), "a", json.Delim('{'), "b", json.Delim('{'))
	if got := j.Path(); got != "a.b" {
		t.Fatalf("Path = %q, want a.b", got)
	}
}

// Detail 2 (Inferable: yes): '{'/'[' push and grow the indent by two spaces;
// '}'/']' pop and shrink it. Observable in emitted whitespace.
func TestDetail02(t *testing.T) {
	var buf bytes.Buffer
	j := ju.NewJSONStreamWriter(&buf)
	writeAll(t, j,
		json.Delim('{'), "k", json.Delim('{'), "inner", float64(1),
		json.Delim('}'), json.Delim('}'))
	out := buf.String()
	if !strings.Contains(out, "\n  \"") {
		t.Fatalf("no 2-space indent at depth 1: %q", out)
	}
	if !strings.Contains(out, "\n    ") {
		t.Fatalf("no 4-space indent at depth 2: %q", out)
	}
	var v interface{}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("output not valid JSON: %q", out)
	}
}

// Detail 3 (Inferable: partially): element separation is a deferred comma —
// flushed before the next token, rewritten to newline on close; no trailing
// comma anywhere.
func TestDetail03(t *testing.T) {
	var buf bytes.Buffer
	j := ju.NewJSONStreamWriter(&buf)
	writeAll(t, j,
		json.Delim('{'), "a", float64(1), "b", float64(2), json.Delim('}'))
	out := buf.String()
	if !strings.Contains(out, ",") {
		t.Fatalf("sibling elements not separated: %q", out)
	}
	if strings.Contains(out, ",\n}") || strings.Contains(out, ",}") {
		t.Fatalf("trailing comma before close: %q", out)
	}
	var v map[string]interface{}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("invalid JSON: %q", out)
	}
	if len(v) != 2 || v["a"] != float64(1) || v["b"] != float64(2) {
		t.Fatalf("wrong content: %v", v)
	}
}

// Detail 4 (Inferable: partially): a scalar in object state becomes a field
// name — emitted, F pushed, recorded on the path stack.
func TestDetail04(t *testing.T) {
	var buf bytes.Buffer
	j := ju.NewJSONStreamWriter(&buf)
	writeAll(t, j, json.Delim('{'), "fieldname")
	if got := j.Path(); got != "fieldname" {
		t.Fatalf("Path after field name = %q", got)
	}
	if !strings.Contains(buf.String(), "fieldname") {
		t.Fatalf("field name not emitted: %q", buf.String())
	}
}

// Detail 5 (Inferable: partially): a scalar in F state is the field value —
// emitted, F and the path entry popped, comma deferred.
func TestDetail05(t *testing.T) {
	var buf bytes.Buffer
	j := ju.NewJSONStreamWriter(&buf)
	writeAll(t, j, json.Delim('{'), "k", float64(5))
	if got := j.Path(); got != "" {
		t.Fatalf("Path after value = %q, want empty", got)
	}
	out := buf.String()
	if !strings.Contains(out, "5") {
		t.Fatalf("value not emitted: %q", out)
	}
	if strings.Contains(out, ",\n}") {
		t.Fatal("premature trailing comma")
	}
}

// Detail 6 (Inferable: no): '}' while F is on the stack unwinds F rather
// than erroring or corrupting the state stack. Asserted shape only: the
// close returns no error, emits a '}', and the writer keeps accepting
// tokens. (The DETAILS claim that the path entry is also popped is NOT
// asserted: the reference implementation leaves it on Path() — reported as
// a DETAILS/gold divergence.)
func TestDetail06(t *testing.T) {
	var buf bytes.Buffer
	j := ju.NewJSONStreamWriter(&buf)
	writeAll(t, j, json.Delim('{'), "k")
	if err := j.WriteToken(json.Delim('}')); err != nil {
		t.Fatalf("close while F on stack errored: %v", err)
	}
	if !strings.Contains(buf.String(), "}") {
		t.Fatalf("close not emitted: %q", buf.String())
	}
	// subsequent writes must still work
	writeAll(t, j, json.Delim('{'), "x", float64(1), json.Delim('}'))
}

// Detail 7 (Inferable: no): scalar rendering — %g floats, verbatim
// json.Number, %v bools, null for nil, strings quoted with NO escaping.
// Asserted: values round-trip semantically; a newline inside a string value
// is NOT escaped (raw byte appears); json.Number keeps its literal digits.
func TestDetail07(t *testing.T) {
	var buf bytes.Buffer
	j := ju.NewJSONStreamWriter(&buf)
	writeAll(t, j,
		json.Delim('{'),
		"f", float64(1.5),
		"n", json.Number("3.140"),
		"b", true,
		"z", nil,
		"s", "a\nb",
		json.Delim('}'))
	out := buf.String()
	for _, want := range []string{"1.5", "3.140", "true", "null"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "3.14\"," ) && !strings.Contains(out, "3.140") {
		t.Fatal("json.Number digits not verbatim")
	}
	if !strings.Contains(out, "\"a\nb\"") {
		t.Fatalf("string value escaped or re-rendered: %q", out)
	}
}

// Detail 8 (Inferable: no): a scalar at top level (empty state) is an error.
func TestDetail08(t *testing.T) {
	for _, tok := range []json.Token{"x", float64(1), true, nil} {
		var buf bytes.Buffer
		j := ju.NewJSONStreamWriter(&buf)
		if err := j.WriteToken(tok); err == nil {
			t.Fatalf("top-level %v did not error", tok)
		}
	}
}

// Detail 9 (Inferable: partially): unrecognized delimiters / token types and
// unhandled state combos return an error rather than writing.
func TestDetail09(t *testing.T) {
	var buf bytes.Buffer
	j := ju.NewJSONStreamWriter(&buf)
	if err := j.WriteToken(json.Delim(')')); err == nil {
		t.Fatal("unknown delimiter did not error")
	}
	var buf2 bytes.Buffer
	j2 := ju.NewJSONStreamWriter(&buf2)
	writeAll(t, j2, json.Delim('{'), "k")
	if err := j2.WriteToken(struct{}{}); err == nil {
		t.Fatal("unknown token type did not error")
	}
}

// Detail 10 (Inferable: no): field-name emission records a stringified token
// on the path stack — a non-string field name lands in numeric/printed form.
// Asserted shape: it is recorded (non-empty path), no error, and writing
// continues.
func TestDetail10(t *testing.T) {
	var buf bytes.Buffer
	j := ju.NewJSONStreamWriter(&buf)
	writeAll(t, j, json.Delim('{'), float64(2))
	if got := j.Path(); got == "" {
		t.Fatal("non-string field name not recorded on path")
	}
	writeAll(t, j, float64(9), json.Delim('}'))
}
