// Black-box property suite for htmlrender (render package).
// Exported API only: HTMLProduction, HTMLDebug, HTML, Delims, HTMLRender via Instance/Render.
// Seed bbSeed=20260919; bbCases=10000 adversarial draws per random property.
//
// Coverage table (contract.md → property):
// | contract sentence | property |
// |---|---|
// | production Instance executes named template | TestHTMLProductionNamedTemplateProperty |
// | empty Name executes template root | TestHTMLProductionEmptyNameProperty |
// | nil template returns not-configured error | TestHTMLNotConfiguredProperty |
// | debug renderer re-parses from Files | TestHTMLDebugFilesProperty |
// | debug renderer re-parses from Glob | TestHTMLDebugGlobProperty |
// | debug renderer re-parses from FileSystem+Patterns | TestHTMLDebugFSProperty |
// | debug renderer with no source configured panics | TestHTMLDebugPanicProperty |
// | Render writes HTML content type first | TestHTMLContentTypeProperty |
// | custom Delims applied on debug parse | TestHTMLDelimsProperty |
// | execution errors propagate | TestHTMLExecuteErrorProperty |
package render_test

import (
	"html/template"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	render "example.internal/httprouter/render"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

const (
	bbHelloTemplateFile = "../testdata/template/hello.tmpl"
	bbHelloGlob         = "../testdata/template/hello*"
	bbHelloFSDir        = "../testdata/template"
	bbHelloPattern      = "hello.tmpl"
)

func bbHTMLRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func bbRandName(rng *rand.Rand) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	n := 1 + rng.Intn(12)
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

func TestHTMLProductionNamedTemplateProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		name := bbRandName(rng)
		tplText := "Hello {{" + ".name" + "}}"
		templ := template.Must(template.New(name).Parse(tplText))
		dataName := bbRandName(rng)
		inst := (render.HTMLProduction{Template: templ}).Instance(name, map[string]any{"name": dataName})
		w := bbHTMLRecorder()
		if err := inst.Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		want := "Hello " + dataName
		if w.Body.String() != want {
			t.Fatalf("case %d body %q want %q", i, w.Body.String(), want)
		}
		if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
	}
}

func TestHTMLProductionEmptyNameProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		dataName := bbRandName(rng)
		templ := template.Must(template.New("").Parse(`Hello {{.name}}`))
		inst := (render.HTMLProduction{Template: templ}).Instance("", map[string]any{"name": dataName})
		w := bbHTMLRecorder()
		if err := inst.Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if w.Body.String() != "Hello "+dataName {
			t.Fatalf("case %d empty name must execute root", i)
		}
	}
}

func TestHTMLNotConfiguredProperty(t *testing.T) {
	for i := 0; i < 100; i++ {
		w := bbHTMLRecorder()
		err := (render.HTML{}).Render(w)
		if err == nil || err.Error() != "html renderer is not configured" {
			t.Fatalf("case %d err=%v", i, err)
		}
		if w.Body.Len() != 0 {
			t.Fatalf("case %d body must stay empty", i)
		}
		if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
	}
	var _ render.HTMLRender = render.HTMLProduction{}
	var _ render.HTMLRender = render.HTMLDebug{}
}

func TestHTMLDebugFilesProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		name := bbRandName(rng)
		htmlRender := render.HTMLDebug{
			Files:  []string{bbHelloTemplateFile},
			Delims: render.Delims{Left: "{[{", Right: "}]}"},
		}
		inst := htmlRender.Instance(bbHelloPattern, map[string]any{"name": name})
		w := bbHTMLRecorder()
		if err := inst.Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		want := "<h1>Hello " + name + "</h1>"
		if w.Body.String() != want {
			t.Fatalf("case %d body %q want %q", i, w.Body.String(), want)
		}
	}
}

func TestHTMLDebugGlobProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		name := bbRandName(rng)
		htmlRender := render.HTMLDebug{
			Glob:   bbHelloGlob,
			Delims: render.Delims{Left: "{[{", Right: "}]}"},
		}
		inst := htmlRender.Instance(bbHelloPattern, map[string]any{"name": name})
		w := bbHTMLRecorder()
		if err := inst.Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if w.Body.String() != "<h1>Hello "+name+"</h1>" {
			t.Fatalf("case %d glob body mismatch", i)
		}
	}
}

func TestHTMLDebugFSProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases; i++ {
		name := bbRandName(rng)
		htmlRender := render.HTMLDebug{
			FileSystem: http.Dir(bbHelloFSDir),
			Patterns:   []string{bbHelloPattern},
			Delims:     render.Delims{Left: "{[{", Right: "}]}"},
		}
		inst := htmlRender.Instance(bbHelloPattern, map[string]any{"name": name})
		w := bbHTMLRecorder()
		if err := inst.Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if w.Body.String() != "<h1>Hello "+name+"</h1>" {
			t.Fatalf("case %d fs body mismatch", i)
		}
	}
}

func TestHTMLDebugPanicProperty(t *testing.T) {
	htmlRender := render.HTMLDebug{
		Delims: render.Delims{"{{", "}}"},
	}
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		htmlRender.Instance("", nil)
	}()
	if !panicked {
		t.Fatal("debug renderer without sources must panic")
	}
}

func TestHTMLContentTypeProperty(t *testing.T) {
	templ := template.Must(template.New("t").Parse("x"))
	for i := 0; i < bbCases; i++ {
		inst := (render.HTMLProduction{Template: templ}).Instance("t", nil)
		w := bbHTMLRecorder()
		inst.WriteContentType(w)
		if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Fatalf("case %d WriteContentType %q", i, ct)
		}
		w = bbHTMLRecorder()
		w.Header().Set("Content-Type", "text/custom")
		if err := inst.Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if ct := w.Header().Get("Content-Type"); ct != "text/custom" {
			t.Fatalf("case %d preset content-type overwritten", i)
		}
	}
}

func TestHTMLDelimsProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 6))
	lefts := []string{"[[", "{[{", "{{"}
	rights := []string{"]]", "}]}", "}}"}
	for i := 0; i < bbCases; i++ {
		li := rng.Intn(len(lefts))
		ri := rng.Intn(len(rights))
		name := bbRandName(rng)
		htmlRender := render.HTMLDebug{
			Files:  []string{bbHelloTemplateFile},
			Delims: render.Delims{Left: lefts[li], Right: rights[ri]},
		}
		// hello.tmpl uses {[{ .name }]} — only matching delims succeed
		if lefts[li] != "{[{" || rights[ri] != "}]}" {
			continue
		}
		inst := htmlRender.Instance(bbHelloPattern, map[string]any{"name": name})
		w := bbHTMLRecorder()
		if err := inst.Render(w); err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if !strings.Contains(w.Body.String(), name) {
			t.Fatalf("case %d delims failed to render name", i)
		}
	}
}

func TestHTMLExecuteErrorProperty(t *testing.T) {
	templ := template.Must(template.New("t").Parse(`Hello {{.name.invalid}}`))
	inst := (render.HTMLProduction{Template: templ}).Instance("t", map[string]any{"name": "x"})
	w := bbHTMLRecorder()
	if err := inst.Render(w); err == nil {
		t.Fatal("expected execute error")
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type on error %q", ct)
	}
}
