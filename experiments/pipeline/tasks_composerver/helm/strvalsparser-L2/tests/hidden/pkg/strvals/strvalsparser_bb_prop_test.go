// Package strvals_test is a hidden black-box property suite for the
// strvalsparser unit. Exported API only (api.md): Parse, ParseString,
// ParseInto, ParseIntoString, ParseJSON, ParseFile, ParseIntoFile, ToYAML,
// ParseLiteral, ParseLiteralInto, RunesValueReader. Seed 20260919; >=10k cases.
//
// Contract (contract.md) -> property coverage table:
//
//	"no panic on arbitrary input" -> TestStrvalsNoPanicProperty
//	"literal mode keeps strings" -> TestStrvalsContractTableProperty, TestStrvalsLiteralProperty
//	"literal merge into dest" -> TestStrvalsLiteralIntoProperty
//	"nested level cap in literal mode" -> TestStrvalsNestedLevelProperty
//	"list grow/index assignment" -> TestStrvalsListGrowProperty
//	"set parsing incl. lists/escapes" -> TestStrvalsParseProperty, TestStrvalsContractTableProperty
//	"merge into existing map" -> TestStrvalsParseIntoProperty
//	"string mode: no inference" -> TestStrvalsStringModeProperty
//	"JSON value mode" -> TestStrvalsJSONProperty
//	"values via rune reader" -> TestStrvalsFileModeProperty
//	"file mode into dest" -> TestStrvalsFileModeProperty
//	"yaml rendering" -> TestStrvalsToYAMLProperty
//	"nested level cap" -> TestStrvalsNestedLevelProperty
package strvals_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	strvals "example.internal/chartkit/v4/pkg/strvals"

	"sigs.k8s.io/yaml"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

func yamlEq(t *testing.T, got, want map[string]any, label string) {
	t.Helper()
	y1, err := yaml.Marshal(want)
	if err != nil {
		t.Fatalf("%s: marshal want: %v", label, err)
	}
	y2, err := yaml.Marshal(got)
	if err != nil {
		t.Fatalf("%s: marshal got: %v", label, err)
	}
	if string(y1) != string(y2) {
		t.Fatalf("%s: yaml mismatch\ngot:  %s\nwant: %s", label, y2, y1)
	}
}

func bbNestedKey(depth int) string {
	var b strings.Builder
	for i := 1; i <= depth; i++ {
		if i > 1 {
			b.WriteByte('.')
		}
		fmt.Fprintf(&b, "n%d", i)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// contract table
// ---------------------------------------------------------------------------

func TestStrvalsContractTableProperty(t *testing.T) {
	type row struct {
		in   string
		want map[string]any
		err  bool
		mode string // parse, string, literal
	}
	rows := []row{
		{"name1=null,f=false,t=true", map[string]any{"name1": nil, "f": false, "t": true}, false, "parse"},
		{"name1=value1", map[string]any{"name1": "value1"}, false, "parse"},
		{"name1=value1,name2=value2,", map[string]any{"name1": "value1", "name2": "value2"}, false, "parse"},
		{"name1=,name2=value2", map[string]any{"name1": "", "name2": "value2"}, false, "parse"},
		{"leading_zeros=00009", map[string]any{"leading_zeros": "00009"}, false, "parse"},
		{"zero_int=0", map[string]any{"zero_int": 0}, false, "parse"},
		{"boolean=true", map[string]any{"boolean": true}, false, "parse"},
		{"name1={value1,value2}", map[string]any{"name1": []string{"value1", "value2"}}, false, "parse"},
		{"list[0]=foo,list[3]=bar", map[string]any{"list": []any{"foo", nil, nil, "bar"}}, false, "parse"},
		{"name1=one\\,two,name2=three\\,four", map[string]any{"name1": "one,two", "name2": "three,four"}, false, "parse"},
		{"outer.inner=value", map[string]any{"outer": map[string]any{"inner": "value"}}, false, "parse"},
		{"long_int_string=1234567890", map[string]any{"long_int_string": "1234567890"}, false, "string"},
		{"boolean=true", map[string]any{"boolean": "true"}, false, "string"},
		{"is_null=null", map[string]any{"is_null": "null"}, false, "string"},
		{"name=value", map[string]any{"name": "value"}, false, "literal"},
		{"boolean=true", map[string]any{"boolean": "true"}, false, "literal"},
		{"name1={value1,value2}", map[string]any{"name1": "{value1,value2}"}, false, "literal"},
		{"list[3]=bar", map[string]any{"list": []any{nil, nil, nil, "bar"}}, false, "literal"},
	}
	for i, tc := range rows {
		var got map[string]any
		var err error
		switch tc.mode {
		case "parse":
			got, err = strvals.Parse(tc.in)
		case "string":
			got, err = strvals.ParseString(tc.in)
		case "literal":
			got, err = strvals.ParseLiteral(tc.in)
		default:
			t.Fatalf("row %d: bad mode %q", i, tc.mode)
		}
		if tc.err {
			if err == nil {
				t.Fatalf("row %d: expected error for %q", i, tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("row %d: %q errored: %v", i, tc.in, err)
		}
		yamlEq(t, got, tc.want, fmt.Sprintf("row %d", i))
	}

	for _, bad := range []string{"name1,name2=value2", "name1,,,,name2=value2", "name1.name2", "list[0]=foo,list[-20]=bar"} {
		if _, err := strvals.Parse(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

// ---------------------------------------------------------------------------
// property loops (>=10k total across file)
// ---------------------------------------------------------------------------

func TestStrvalsNoPanicProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789=,\\.[]{}\"'_-!@#$%"
	for i := 0; i < bbCases/2; i++ {
		n := 1 + rng.Intn(40)
		var b strings.Builder
		for j := 0; j < n; j++ {
			b.WriteByte(alphabet[rng.Intn(len(alphabet))])
		}
		s := b.String()
		func() {
			defer func() {
				if recover() != nil {
					t.Fatalf("case %d: Parse panicked on %q", i, s)
				}
			}()
			_, _ = strvals.Parse(s)
		}()
		func() {
			defer func() {
				if recover() != nil {
					t.Fatalf("case %d: ParseLiteral panicked on %q", i, s)
				}
			}()
			_, _ = strvals.ParseLiteral(s)
		}()
	}
}

func TestStrvalsParseSimpleProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases/2; i++ {
		key := fmt.Sprintf("k%d", rng.Intn(500))
		val := fmt.Sprintf("v%x", rng.Uint32())
		in := key + "=" + val
		got, err := strvals.Parse(in)
		if err != nil {
			t.Fatalf("case %d: Parse(%q): %v", i, in, err)
		}
		want := map[string]any{key: val}
		yamlEq(t, got, want, fmt.Sprintf("case %d", i))
		g2, err := strvals.Parse(in)
		if err != nil || !reflect.DeepEqual(got, g2) {
			t.Fatalf("case %d: nondeterministic Parse", i)
		}
	}
}

func TestStrvalsListGrowProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases/2; i++ {
		idx := rng.Intn(8)
		val := fmt.Sprintf("x%d", rng.Intn(1000))
		in := fmt.Sprintf("lst[%d]=%s", idx, val)
		got, err := strvals.Parse(in)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		lst, ok := got["lst"].([]any)
		if !ok || len(lst) != idx+1 {
			t.Fatalf("case %d: list len %d want %d", i, len(lst), idx+1)
		}
		for j := 0; j < idx; j++ {
			if lst[j] != nil {
				t.Fatalf("case %d: index %d padded with %v, want nil", i, j, lst[j])
			}
		}
		if lst[idx] != val {
			t.Fatalf("case %d: index %d = %v want %q", i, idx, lst[idx], val)
		}
	}
}

func TestStrvalsStringModeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	scalars := []string{"true", "false", "0", "42", "null", "1234567890"}
	for i := 0; i < bbCases/4; i++ {
		key := fmt.Sprintf("s%d", i%50)
		val := scalars[rng.Intn(len(scalars))]
		got, err := strvals.ParseString(key + "=" + val)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if got[key] != val {
			t.Fatalf("case %d: string mode inferred %v, want %q", i, got[key], val)
		}
	}
}

func TestStrvalsLiteralProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases/4; i++ {
		key := fmt.Sprintf("L%d", rng.Intn(200))
		val := fmt.Sprintf("raw_%d_%x", rng.Intn(9), rng.Uint32())
		if rng.Intn(3) == 0 {
			val = "{" + val + ",tail}"
		}
		got, err := strvals.ParseLiteral(key + "=" + val)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if got[key] != val {
			t.Fatalf("case %d: literal got %v want %q", i, got[key], val)
		}
	}
}

func TestStrvalsParseIntoProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	for i := 0; i < bbCases/4; i++ {
		dest := map[string]any{
			"outer": map[string]any{
				"keep": "stays",
			},
		}
		k := fmt.Sprintf("inner%d", rng.Intn(20))
		v := fmt.Sprintf("v%d", rng.Intn(100))
		in := "outer." + k + "=" + v
		if err := strvals.ParseInto(in, dest); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		outer, ok := dest["outer"].(map[string]any)
		if !ok || outer["keep"] != "stays" || outer[k] != v {
			t.Fatalf("case %d: merge wrong: %+v", i, dest)
		}
	}
}

func TestStrvalsLiteralIntoProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 6))
	for i := 0; i < bbCases/4; i++ {
		dest := map[string]any{"a": "old"}
		payload := fmt.Sprintf("comma,inside_%d", rng.Intn(50))
		in := "a=" + payload
		if err := strvals.ParseLiteralInto(in, dest); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if dest["a"] != payload {
			t.Fatalf("case %d: got %v want %q", i, dest["a"], payload)
		}
	}
}

func TestStrvalsJSONProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 7))
	for i := 0; i < bbCases/4; i++ {
		dest := map[string]any{"outer": map[string]any{"inner2": "keep"}}
		in := `outer.inner1="1",outer.inner3=3,outer.inner4=true`
		if err := strvals.ParseJSON(in, dest); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		outer := dest["outer"].(map[string]any)
		inner1 := outer["inner1"]
		if inner1 != float64(1) && inner1 != 1 && inner1 != "1" {
			t.Fatalf("case %d: inner1 got %T %v", i, inner1, inner1)
		}
		inner3 := outer["inner3"]
		if inner3 != float64(3) && inner3 != 3 {
			t.Fatalf("case %d: inner3 got %T %v", i, inner3, inner3)
		}
		if outer["inner4"] != true || outer["inner2"] != "keep" {
			t.Fatalf("case %d: json merge wrong: %+v", i, outer)
		}
		if rng.Intn(5) == 0 {
			if err := strvals.ParseJSON(`outer.bad={not json}`, map[string]any{}); err == nil {
				t.Fatalf("case %d: bad json should error", i)
			}
		}
	}
}

func TestStrvalsFileModeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 8))
	reader := strvals.RunesValueReader(func(rs []rune) (any, error) {
		return "file-" + string(rs), nil
	})
	for i := 0; i < bbCases/4; i++ {
		path := fmt.Sprintf("/tmp/p%d", rng.Intn(500))
		in := fmt.Sprintf("name=%s", path)
		got, err := strvals.ParseFile(in, reader)
		if err != nil {
			t.Fatalf("case %d ParseFile: %v", i, err)
		}
		want := map[string]any{"name": "file-" + path}
		yamlEq(t, got, want, fmt.Sprintf("ParseFile %d", i))

		dest := map[string]any{}
		if err := strvals.ParseIntoFile(in, dest, reader); err != nil {
			t.Fatalf("case %d ParseIntoFile: %v", i, err)
		}
		yamlEq(t, dest, want, fmt.Sprintf("ParseIntoFile %d", i))
	}
}

func TestStrvalsToYAMLProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 9))
	for i := 0; i < bbCases/4; i++ {
		key := fmt.Sprintf("y%d", rng.Intn(100))
		val := fmt.Sprintf("val%d", rng.Intn(1000))
		out, err := strvals.ToYAML(key + "=" + val)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if !strings.Contains(out, key) || !strings.Contains(out, val) {
			t.Fatalf("case %d: yaml %q missing key/val", i, out)
		}
	}
}

func TestStrvalsNestedLevelProperty(t *testing.T) {
	okKey := bbNestedKey(3)
	got, err := strvals.Parse(okKey + "=value")
	if err != nil {
		t.Fatalf("3-level nested: %v", err)
	}
	want := map[string]any{"n1": map[string]any{"n2": map[string]any{"n3": "value"}}}
	yamlEq(t, got, want, "3-level")

	tooDeep := bbNestedKey(strvals.MaxNestedNameLevel + 2)
	if _, err := strvals.Parse(tooDeep + "=value"); err == nil {
		t.Fatal("Parse: expected nested level error")
	}
	if _, err := strvals.ParseLiteral(tooDeep + "=value"); err == nil {
		t.Fatal("ParseLiteral: expected nested level error")
	}
}

func TestStrvalsUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 10))
	// adversarial keys/values not present in contract examples
	lits := []string{"qzx", "wobble", "kr", "blob.bin", "ut", "em%GT"}
	for i := 0; i < bbCases/2; i++ {
		key := lits[rng.Intn(len(lits))]
		if rng.Intn(2) == 0 {
			key = fmt.Sprintf("pre.%s", key)
		}
		val := lits[rng.Intn(len(lits))]
		in := key + "=" + val
		got, err := strvals.ParseLiteral(in)
		if err != nil {
			t.Fatalf("case %d: literal %q: %v", i, in, err)
		}
		if got[key] != val && !strings.Contains(key, ".") {
			t.Fatalf("case %d: got %v", i, got)
		}
		// list literal form name[]={a,b} in normal parse mode
		if rng.Intn(4) == 0 {
			a, b := fmt.Sprintf("a%d", i), fmt.Sprintf("b%d", i)
			in2 := fmt.Sprintf("items[0]=%s,items[1]=%s", a, b)
			m, err := strvals.Parse(in2)
			if err != nil {
				t.Fatalf("case %d list literal: %v", i, err)
			}
			lst, ok := m["items"].([]any)
			if !ok {
				if ss, ok2 := m["items"].([]string); ok2 && len(ss) == 2 && ss[0] == a && ss[1] == b {
					continue
				}
				t.Fatalf("case %d: list literal got %+v", i, m["items"])
			}
			if len(lst) != 2 || lst[0] != a || lst[1] != b {
				t.Fatalf("case %d: list literal got %+v", i, m["items"])
			}
		}
	}
}
