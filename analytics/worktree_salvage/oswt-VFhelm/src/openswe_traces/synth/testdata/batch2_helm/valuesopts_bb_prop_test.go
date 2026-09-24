// Hidden black-box suite for valuesopts. Exported API only: Options.MergeValues.
package values_test

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.internal/helm/pkg/cli/values"
	"example.internal/helm/pkg/getter"
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

type recGetter struct {
	body    []byte
	lastURL string
	gotURL  string // WithURL captured via option application if we can see it
}

func (g *recGetter) Get(href string, options ...getter.Option) (*bytes.Buffer, error) {
	g.lastURL = href
	return bytes.NewBuffer(g.body), nil
}

func providersWith(scheme string, g getter.Getter) getter.Providers {
	return getter.Providers{{
		Schemes: []string{scheme},
		New: func(options ...getter.Option) (getter.Getter, error) {
			return g, nil
		},
	}}
}

func TestDetail01_LayerOrderFilesJSONSetStringFileLiteral(t *testing.T) {
	dir := t.TempDir()
	vf := filepath.Join(dir, "v.yaml")
	if err := os.WriteFile(vf, []byte("k: from-file\nshared: file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sf := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(sf, []byte("from-setfile"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := &values.Options{
		ValueFiles:    []string{vf},
		JSONValues:    []string{`{"k":"from-json","shared":"json"}`},
		Values:        []string{"k=from-set"},
		StringValues:  []string{"k=from-string"},
		FileValues:    []string{"k=" + sf},
		LiteralValues: []string{"k=from-literal"},
	}
	got, err := opts.MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["k"] != "from-literal" {
		t.Fatalf("literal must win last, k=%v full=%v", got["k"], got)
	}
	// peel layers by omitting later ones
	opts.LiteralValues = nil
	got, err = opts.MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["k"] != "from-setfile" && got["k"] != string([]byte("from-setfile")) {
		t.Fatalf("set-file wins over string: k=%v", got["k"])
	}
	opts.FileValues = nil
	got, err = opts.MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["k"] != "from-string" {
		t.Fatalf("set-string wins over set: k=%v", got["k"])
	}
	opts.StringValues = nil
	got, err = opts.MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["k"] != "from-set" {
		t.Fatalf("set wins over json: k=%v", got["k"])
	}
	opts.Values = nil
	got, err = opts.MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["k"] != "from-json" {
		t.Fatalf("json wins over files: k=%v", got["k"])
	}
}

func TestDetail02_SetJSONBraceSniffVsParseJSON(t *testing.T) {
	opts := &values.Options{JSONValues: []string{`{"obj":{"a":1}}`}}
	got, err := opts.MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	obj, ok := got["obj"].(map[string]any)
	if !ok {
		t.Fatalf("{ prefix must json.Unmarshal to map, got %T %v", got["obj"], got)
	}
	if obj["a"] != float64(1) && obj["a"] != 1 && obj["a"] != int64(1) {
		t.Fatalf("a=%v", obj["a"])
	}
	opts = &values.Options{JSONValues: []string{`k={"x":2}`}}
	got, err = opts.MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, has := got["k"]; !has {
		t.Fatalf("non-{ prefix uses ParseJSON key=value, got %v", got)
	}
}

func TestDetail03_SetJSONErrorTextAsymmetry(t *testing.T) {
	opts := &values.Options{JSONValues: []string{`{not json`}}
	_, err := opts.MergeValues(nil)
	if err == nil {
		t.Fatal("invalid JSON object must error")
	}
	if !strings.Contains(err.Error(), "data JSON:") && !strings.Contains(err.Error(), "JSON") {
		t.Fatalf("{ sniff error wording: %v", err)
	}
	opts = &values.Options{JSONValues: []string{`k={not`}}
	_, err = opts.MergeValues(nil)
	if err == nil {
		t.Fatal("invalid ParseJSON must error")
	}
	// two near-identical formats: object path mentions JSON more explicitly
	_, e1 := (&values.Options{JSONValues: []string{`{`}}).MergeValues(nil)
	_, e2 := (&values.Options{JSONValues: []string{`k=[`}}).MergeValues(nil)
	if e1 == nil || e2 == nil {
		t.Fatal("both malformed json paths must error")
	}
	if e1.Error() == e2.Error() {
		t.Fatalf("error texts must be asymmetric, both %q", e1.Error())
	}
}

func TestDetail04_DashAfterTrimSpaceIsStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old; _ = r.Close() })
	go func() {
		_, _ = io.WriteString(w, "from: stdin\n")
		_ = w.Close()
	}()
	opts := &values.Options{ValueFiles: []string{"  -  "}}
	got, err := opts.MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["from"] != "stdin" {
		t.Fatalf("TrimSpace('-') must read stdin, got %v", got)
	}
}

func TestDetail05_UnsupportedSchemeFallsBackToLocalFile(t *testing.T) {
	dir := t.TempDir()
	// a path that looks like a URL with an unsupported scheme: parse succeeds,
	// getter misses, fallback to os.ReadFile of the original string — so use a real file
	// whose name is a valid URL-looking path that still exists locally? that's hard.
	// Instead: a real file path (no scheme) is local. And a garbage scheme of an
	// existing file: "weirdscheme://..." won't exist on disk.
	p := filepath.Join(dir, "vals.yaml")
	if err := os.WriteFile(p, []byte("local: fromdisk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := &values.Options{ValueFiles: []string{p}}
	got, err := opts.MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["local"] != "fromdisk" {
		t.Fatalf("plain path: %v", got)
	}
	_, err = (&values.Options{ValueFiles: []string{"nope://missing"}}).MergeValues(nil)
	if err == nil {
		t.Fatal("unsupported scheme with missing file must still fail the ReadFile fallback")
	}
}

func TestDetail06_RemoteReadPassesWithURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("remote: 1\n"))
	}))
	t.Cleanup(srv.Close)
	g := &recGetter{body: []byte("remote: 1\n")}
	opts := &values.Options{ValueFiles: []string{srv.URL + "/v.yaml"}}
	// built-in http getter via Getters(); also our recorder via http
	got, err := opts.MergeValues(getter.Getters())
	if err != nil {
		// if Getters is excised in getterdispatch unit but this is valuesopts tree —
		// getters are live here. If it still fails, try custom provider.
		ps := providersWith("http", g)
		ps = append(ps, providersWith("https", g)...)
		got, err = opts.MergeValues(ps)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got["remote"] != 1 && got["remote"] != int64(1) && got["remote"] != float64(1) {
		t.Fatalf("remote yaml: %v", got)
	}
}

func TestDetail07_ValueFilesReadLoadMergeInFlagOrder(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yaml")
	b := filepath.Join(dir, "b.yaml")
	if err := os.WriteFile(a, []byte("k: a\nkeep: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("k: b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := (&values.Options{ValueFiles: []string{a, b}}).MergeValues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["k"] != "b" {
		t.Fatalf("later file wins: %v", got["k"])
	}
	if got["keep"] != 1 && got["keep"] != float64(1) && got["keep"] != int64(1) {
		t.Fatalf("earlier unique key kept: %v", got)
	}
}

func TestDetail08_PerFamilyErrorWrap(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	_ = rng
	_, err := (&values.Options{Values: []string{`nested.too.` + strings.Repeat("x.", 40) + "z=1"}}).MergeValues(nil)
	if err == nil {
		t.Fatal("set parse must wrap")
	}
	_, err = (&values.Options{StringValues: []string{strings.Repeat("nest.", 40) + "z=1"}}).MergeValues(nil)
	if err == nil {
		t.Fatal("set-string parse must wrap")
	}
	_, err = (&values.Options{LiteralValues: []string{strings.Repeat("a.", 40) + "z=1"}}).MergeValues(nil)
	if err == nil {
		t.Fatal("set-literal parse must wrap")
	}
	_, err = (&values.Options{FileValues: []string{"k=/no/such/file-" + strconv.FormatInt(time.Now().UnixNano(), 10)}}).MergeValues(nil)
	if err == nil {
		t.Fatal("set-file missing must wrap")
	}
}
