// Black-box property suite for jsonrenders (render package).
// Exported API only: JSON, IndentedJSON, SecureJSON, JsonpJSON, AsciiJSON, PureJSON, WriteJSON.
// Seed bbSeed=20260919; bbCases=10000 adversarial draws per random property.
//
// Coverage table (contract.md → property):
// | contract sentence | property |
// |---|---|
// | WriteJSON marshals and writes bytes with JSON content type | TestJSONWriteJSONContentTypeProperty |
// | marshal errors propagate after content type is set | TestJSONMarshalErrorProperty |
// | IndentedJSON uses 4-space indent | TestIndentedJSONIndentProperty |
// | SecureJSON prefix only for top-level JSON arrays | TestSecureJSONPrefixProperty |
// | JsonpJSON javascript content type and callback wrapping | TestJsonpJSONCallbackProperty |
// | empty JsonpJSON callback writes raw JSON | TestJsonpJSONEmptyCallbackProperty |
// | AsciiJSON bare application/json and \\uXXXX lowercase escapes | TestAsciiJSONEscapeProperty |
// | PureJSON encoder HTML escape off with trailing newline | TestPureJSONNoHTMLEscapeProperty |
// | pre-set Content-Type left alone on all JSON renderers | TestJSONPreservesPresetContentTypeProperty |
package render_test

import (
	"bytes"
	"encoding/json"
	"html/template"
	"math/rand"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode"

	render "example.internal/httprouter/render"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

func bbJSONRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func bbRandString(rng *rand.Rand, max int) string {
	const alphabet = "abcXYZ0123<>&\"'\\/\n\r\t"
	n := 1 + rng.Intn(max)
	var b strings.Builder
	for i := 0; i < n; i++ {
		if rng.Intn(20) == 0 {
			b.WriteRune(rune(0x4e00 + rng.Intn(0x5000)))
		} else {
			b.WriteByte(alphabet[rng.Intn(len(alphabet))])
		}
	}
	return b.String()
}

func bbRandJSONValue(rng *rand.Rand, depth int) any {
	switch rng.Intn(8) {
	case 0:
		return bbRandString(rng, 12)
	case 1:
		return rng.Intn(1000) - 500
	case 2:
		return rng.Float64()
	case 3:
		return rng.Intn(2) == 0
	case 4:
		if depth > 2 {
			return bbRandString(rng, 8)
		}
		m := make(map[string]any, 1+rng.Intn(4))
		for i := 0; i < 1+rng.Intn(4); i++ {
			m[bbRandString(rng, 6)] = bbRandJSONValue(rng, depth+1)
		}
		return m
	case 5:
		if depth > 2 {
			return bbRandString(rng, 8)
		}
		n := 1 + rng.Intn(4)
		s := make([]any, n)
		for i := range s {
			s[i] = bbRandJSONValue(rng, depth+1)
		}
		return s
	case 6:
		return nil
	default:
		return map[string]any{"html": "<b>", "tag": "&", "mix": "GO语言"}
	}
}

func bbOracleAsciiJSONFixed(raw []byte) string {
	var buf strings.Builder
	for _, r := range string(raw) {
		if r > unicode.MaxASCII {
			var h [6]byte
			h[0], h[1] = '\\', 'u'
			for i, shift := 0, 12; i < 4; i, shift = i+1, shift-4 {
				d := (int(r) >> shift) & 0xf
				if d < 10 {
					h[2+i] = byte('0' + d)
				} else {
					h[2+i] = byte('a' + d - 10)
				}
			}
			buf.Write(h[:6])
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

func bbIsJSONArray(b []byte) bool {
	b = bytes.TrimSpace(b)
	return len(b) >= 2 && b[0] == '[' && b[len(b)-1] == ']'
}

func TestJSONWriteJSONContentTypeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		data := bbRandJSONValue(rng, 0)
		w := bbJSONRecorder()
		if err := render.WriteJSON(w, data); err != nil {
			t.Fatalf("case %d WriteJSON: %v", i, err)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
		want, err := json.Marshal(data)
		if err != nil {
			continue
		}
		if !bytes.Equal(w.Body.Bytes(), want) {
			t.Fatalf("case %d body mismatch", i)
		}
		w2 := bbJSONRecorder()
		(render.JSON{Data: data}).WriteContentType(w2)
		if ct := w2.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("case %d JSON.WriteContentType %q", i, ct)
		}
	}
}

func TestJSONMarshalErrorProperty(t *testing.T) {
	bad := make(chan int)
	renderers := []render.Render{
		render.JSON{Data: bad},
		render.IndentedJSON{Data: bad},
		render.SecureJSON{Prefix: "while(1);", Data: bad},
		render.JsonpJSON{Callback: "cb", Data: bad},
		render.AsciiJSON{Data: bad},
		render.PureJSON{Data: bad},
	}
	for _, r := range renderers {
		w := bbJSONRecorder()
		_ = r.Render(w)
		if ct := w.Header().Get("Content-Type"); ct == "" {
			t.Fatalf("%T set no content type before error", r)
		}
	}
	w := bbJSONRecorder()
	if err := render.WriteJSON(w, bad); err == nil {
		t.Fatal("WriteJSON expected error")
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("WriteJSON content-type on error %q", ct)
	}
}

func TestIndentedJSONIndentProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		data := bbRandJSONValue(rng, 0)
		want, err := json.MarshalIndent(data, "", "    ")
		if err != nil {
			continue
		}
		w := bbJSONRecorder()
		if err := (render.IndentedJSON{Data: data}).Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if !bytes.Equal(w.Body.Bytes(), want) {
			t.Fatalf("case %d indent mismatch", i)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
	}
	// fixed oracle from render_test.go
	w := bbJSONRecorder()
	data := map[string]any{"foo": "bar", "bar": "foo"}
	if err := (render.IndentedJSON{Data: data}).Render(w); err != nil {
		t.Fatal(err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want = data
	if got["foo"] != want["foo"] || got["bar"] != want["bar"] {
		t.Fatalf("fixed indent data mismatch")
	}
	if !strings.Contains(w.Body.String(), "\n    ") {
		t.Fatal("expected four-space indent lines")
	}
}

func TestSecureJSONPrefixProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	prefixes := []string{"while(1);", ")]}'\n", "", "/*x*/"}
	for i := 0; i < bbCases; i++ {
		prefix := prefixes[rng.Intn(len(prefixes))]
		var data any
		if rng.Intn(3) == 0 {
			n := 1 + rng.Intn(3)
			arr := make([]map[string]any, n)
			for j := range arr {
				arr[j] = map[string]any{bbRandString(rng, 4): bbRandString(rng, 6)}
			}
			data = arr
		} else {
			data = map[string]any{"k": bbRandString(rng, 8), "n": rng.Intn(100)}
		}
		raw, err := json.Marshal(data)
		if err != nil {
			continue
		}
		w := bbJSONRecorder()
		if err := (render.SecureJSON{Prefix: prefix, Data: data}).Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		body := w.Body.Bytes()
		if bbIsJSONArray(raw) {
			if !bytes.HasPrefix(body, []byte(prefix)) {
				t.Fatalf("case %d array missing prefix %q", i, prefix)
			}
			if !bytes.Equal(body[len(prefix):], raw) {
				t.Fatalf("case %d array payload mismatch", i)
			}
		} else if !bytes.Equal(body, raw) {
			t.Fatalf("case %d non-array must not get prefix", i)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
	}
	// adversarial: JSON string value that looks like an array
	w := bbJSONRecorder()
	strData := "[not-really-array]"
	if err := (render.SecureJSON{Prefix: "P:", Data: strData}).Render(w); err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(strData)
	if !bytes.Equal(w.Body.Bytes(), want) {
		t.Fatal("string looking like array must not get prefix")
	}
}

func TestJsonpJSONCallbackProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		data := bbRandJSONValue(rng, 0)
		raw, err := json.Marshal(data)
		if err != nil {
			continue
		}
		cb := bbRandString(rng, 8)
		if rng.Intn(5) == 0 {
			cb = "fn<script>\n\"\\"
		}
		w := bbJSONRecorder()
		if err := (render.JsonpJSON{Callback: cb, Data: data}).Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/javascript; charset=utf-8" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
		escaped := template.JSEscapeString(cb)
		want := escaped + "(" + string(raw) + ");"
		if w.Body.String() != want {
			t.Fatalf("case %d jsonp wrap mismatch", i)
		}
	}
}

func TestJsonpJSONEmptyCallbackProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases; i++ {
		data := bbRandJSONValue(rng, 0)
		raw, err := json.Marshal(data)
		if err != nil {
			continue
		}
		w := bbJSONRecorder()
		if err := (render.JsonpJSON{Callback: "", Data: data}).Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/javascript; charset=utf-8" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
		if !bytes.Equal(w.Body.Bytes(), raw) {
			t.Fatalf("case %d empty callback must passthrough JSON", i)
		}
	}
}

func TestAsciiJSONEscapeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	for i := 0; i < bbCases; i++ {
		data := bbRandJSONValue(rng, 0)
		raw, err := json.Marshal(data)
		if err != nil {
			continue
		}
		w := bbJSONRecorder()
		if err := (render.AsciiJSON{Data: data}).Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Fatalf("case %d content-type %q want bare application/json", i, ct)
		}
		if strings.Contains(ct, "charset") {
			t.Fatalf("case %d charset must be omitted", i)
		}
		want := bbOracleAsciiJSONFixed(raw)
		if w.Body.String() != want {
			t.Fatalf("case %d ascii escape mismatch", i)
		}
	}
	// fixed oracle from render_test.go
	w := bbJSONRecorder()
	data1 := map[string]any{"lang": "GO语言", "tag": "<br>"}
	if err := (render.AsciiJSON{Data: data1}).Render(w); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["lang"] != "GO语言" || got["tag"] != "<br>" {
		t.Fatal("ascii fixed roundtrip failed")
	}
	if !strings.Contains(w.Body.String(), `\u8bed`) || !strings.Contains(w.Body.String(), `\u003c`) {
		t.Fatal("expected lowercase unicode escapes")
	}
}

func TestPureJSONNoHTMLEscapeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 6))
	for i := 0; i < bbCases; i++ {
		data := map[string]any{
			"a": bbRandString(rng, 6),
			"html": "<script>&\"",
		}
		w := bbJSONRecorder()
		if err := (render.PureJSON{Data: data}).Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
		body := w.Body.String()
		if !strings.HasSuffix(body, "\n") {
			t.Fatalf("case %d encoder must end with newline", i)
		}
		if strings.Contains(body, `\u003c`) || strings.Contains(body, `\u003e`) || strings.Contains(body, `\u0026`) {
			t.Fatalf("case %d must not HTML-escape", i)
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(data); err != nil {
			t.Fatalf("case %d oracle encode: %v", i, err)
		}
		if body != buf.String() {
			t.Fatalf("case %d pure json mismatch", i)
		}
	}
}

func TestJSONPreservesPresetContentTypeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 7))
	preset := "application/vnd.test+json"
	data := map[string]any{"x": 1}
	renderers := []func(httpW *httptest.ResponseRecorder) error{
		func(w *httptest.ResponseRecorder) error { return render.WriteJSON(w, data) },
		func(w *httptest.ResponseRecorder) error { return (render.JSON{Data: data}).Render(w) },
		func(w *httptest.ResponseRecorder) error { return (render.IndentedJSON{Data: data}).Render(w) },
		func(w *httptest.ResponseRecorder) error { return (render.SecureJSON{Data: data}).Render(w) },
		func(w *httptest.ResponseRecorder) error { return (render.JsonpJSON{Data: data}).Render(w) },
		func(w *httptest.ResponseRecorder) error { return (render.AsciiJSON{Data: data}).Render(w) },
		func(w *httptest.ResponseRecorder) error { return (render.PureJSON{Data: data}).Render(w) },
	}
	for i := 0; i < bbCases; i++ {
		r := renderers[rng.Intn(len(renderers))]
		w := bbJSONRecorder()
		w.Header().Set("Content-Type", preset)
		if err := r(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if ct := w.Header().Get("Content-Type"); ct != preset {
			t.Fatalf("case %d preset content-type overwritten to %q", i, ct)
		}
	}
}
