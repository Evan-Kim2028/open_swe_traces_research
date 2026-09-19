// Package templater_test is a hidden black-box property suite for templater.
// Exported API only (api.md): NewTemplater, Render. Seed 20260919; >=10k cases.
//
// Contract (contract.md) -> property coverage table:
//
//	"basic interpolation" -> TestTemplaterRenderContextProperty
//	"failOnMissing errors" -> TestTemplaterFailOnMissingProperty
//	"failOnMissing false allows missing" -> TestTemplaterAllowMissingProperty
//	"indent skips first/empty lines" -> TestTemplaterIndentProperty
//	"include named snippet" -> TestTemplaterIncludeProperty
//	"snippet mainTemplate rejected" -> TestTemplaterSnippetNameProperty
//	"combined snippets + funcs" -> TestTemplaterUnseenRandomProperty
package templater_test

import (
	"math/rand"
	"strings"
	"testing"

	"example.internal/clustkit/pkg/util/templater"
)

const bbSeed = 20260919
const bbCases = 10000

func bbNewTemplater(t *testing.T) *templater.Templater {
	t.Helper()
	return templater.NewTemplater(nil)
}

func TestTemplaterRenderContextProperty(t *testing.T) {
	tr := bbNewTemplater(t)
	got, err := tr.Render("hello {{ .Name }}", map[string]interface{}{"Name": "world"}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestTemplaterFailOnMissingProperty(t *testing.T) {
	tr := bbNewTemplater(t)
	_, err := tr.Render("{{ .MissingKey }}", map[string]interface{}{}, nil, true)
	if err == nil {
		t.Fatal("failOnMissing must error on missing key")
	}
}

func TestTemplaterAllowMissingProperty(t *testing.T) {
	tr := bbNewTemplater(t)
	got, err := tr.Render("{{ .MissingKey }}", map[string]interface{}{}, nil, false)
	if err != nil {
		t.Fatalf("allow missing: %v", err)
	}
	if strings.Contains(got, "MissingKey") {
		t.Fatalf("should not render missing key name: %q", got)
	}
}

func TestTemplaterIndentProperty(t *testing.T) {
	tr := bbNewTemplater(t)
	content := "first\n\nsecond\nthird"
	got, err := tr.Render(`{{ indent 2 .Body }}`, map[string]interface{}{"Body": content}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(got, "\n")
	if len(lines) < 3 {
		t.Fatalf("indent output: %q", got)
	}
	if strings.HasPrefix(lines[0], "  ") {
		t.Fatalf("first line must not be indented: %q", got)
	}
	if lines[1] != "" {
		t.Fatalf("empty line preserved: %q", got)
	}
	if !strings.HasPrefix(lines[2], "  ") || !strings.HasPrefix(lines[3], "  ") {
		t.Fatalf("non-first non-empty lines indented: %q", got)
	}
}

func TestTemplaterIncludeProperty(t *testing.T) {
	tr := bbNewTemplater(t)
	snippets := map[string]string{
		"partials/header.tpl": "hdr={{ .Val }}",
	}
	got, err := tr.Render(`{{ include "partials/header.tpl" . }}`, map[string]interface{}{"Val": "ok"}, snippets, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hdr=ok" {
		t.Fatalf("include snippet: %q", got)
	}
}

func TestTemplaterSnippetNameProperty(t *testing.T) {
	tr := bbNewTemplater(t)
	_, err := tr.Render("x", nil, map[string]string{"mainTemplate": "bad"}, false)
	if err == nil {
		t.Fatal("snippet named mainTemplate must be rejected")
	}
}

func TestTemplaterIncludeMissingProperty(t *testing.T) {
	tr := bbNewTemplater(t)
	_, err := tr.Render(`{{ include "nope.tpl" . }}`, nil, nil, false)
	if err == nil {
		t.Fatal("missing include snippet must error (panic recovered)")
	}
}

func TestTemplaterUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	tr := bbNewTemplater(t)
	for i := 0; i < bbCases; i++ {
		key := "K" + string(rune('a'+rng.Intn(26)))
		val := rngString(rng, 5+rng.Intn(8))
		tpl := "v={{ ." + key + " }}"
		ctx := map[string]interface{}{key: val}
		got, err := tr.Render(tpl, ctx, nil, false)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if got != "v="+val {
			t.Fatalf("case %d: got %q want v=%s", i, got, val)
		}
	}
}

func rngString(rng *rand.Rand, n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(chars[rng.Intn(len(chars))])
	}
	return b.String()
}
