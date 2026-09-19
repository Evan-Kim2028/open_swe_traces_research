// Black-box property suite for the bindingdispatch unit.
// Exported API only: binding.Default, binding.Validator, binding.JSON.Bind,
// MIME constants, binder vars (Form, JSON, XML, ...).
// Seed 20260919; >=10k cases; contract + binding_test.go coverage.
//
// Coverage table (contract sentence -> property):
//   "GET always returns the form binder regardless of content type"
//       -> TestBBDefaultGetAlwaysForm / TestBBDefaultGetOverridesMIME
//   "content type selects JSON/XML/ProtoBuf/MsgPack/YAML/TOML/multipart/BSON binders"
//       -> TestBBDefaultMIMETable / TestBBDefaultDualMIMEAliases
//   "urlencoded form type and empty content type fall back to plain form binder"
//       -> TestBBDefaultFormFallback / TestBBDefaultUnknownMIME
//   "validate hook is a no-op when no validator is installed"
//       -> TestBBValidationDisabledNilValidator / TestBBValidationDisabledRandomBodies
package binding_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"

	binding "example.internal/httprouter/binding"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

type bbFooRequired struct {
	Foo string `json:"foo" binding:"required"`
}

var bbDispatchMIME = []struct {
	mime   string
	expect binding.Binding
}{
	{binding.MIMEJSON, binding.JSON},
	{binding.MIMEXML, binding.XML},
	{binding.MIMEXML2, binding.XML},
	{binding.MIMEPROTOBUF, binding.ProtoBuf},
	{binding.MIMEMSGPACK, binding.MsgPack},
	{binding.MIMEMSGPACK2, binding.MsgPack},
	{binding.MIMEYAML, binding.YAML},
	{binding.MIMEYAML2, binding.YAML},
	{binding.MIMETOML, binding.TOML},
	{binding.MIMEMultipartPOSTForm, binding.FormMultipart},
	{binding.MIMEBSON, binding.BSON},
}

var bbFormFallbackMIME = []string{
	"",
	binding.MIMEPOSTForm,
	"text/plain",
	"application/octet-stream",
	"application/json; charset=utf-8",
	"multipart/mixed",
}

func bbRequestWithJSON(body string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", binding.MIMEJSON)
	return req
}

func TestBBDefaultGetAlwaysForm(t *testing.T) {
	methods := []string{http.MethodGet}
	contentTypes := []string{
		"",
		binding.MIMEJSON,
		binding.MIMEXML,
		binding.MIMEMultipartPOSTForm,
		binding.MIMEBSON,
		"application/unknown",
	}
	for _, method := range methods {
		for _, ct := range contentTypes {
			got := binding.Default(method, ct)
			if got != binding.Form {
				t.Fatalf("Default(%q,%q)=%v want Form", method, ct, got)
			}
			if got.Name() != "form" {
				t.Fatalf("Form binder Name()=%q want form", got.Name())
			}
		}
	}
}

func TestBBDefaultGetOverridesMIME(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		ct := bbDispatchMIME[rng.Intn(len(bbDispatchMIME))].mime
		if binding.Default(http.MethodGet, ct) != binding.Form {
			t.Fatalf("case %d: GET with %q must return Form", i, ct)
		}
	}
}

func TestBBDefaultMIMETable(t *testing.T) {
	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range methods {
		for _, row := range bbDispatchMIME {
			got := binding.Default(method, row.mime)
			if got != row.expect {
				t.Fatalf("Default(%q,%q)=%v want %v", method, row.mime, got, row.expect)
			}
		}
	}
}

func TestBBDefaultDualMIMEAliases(t *testing.T) {
	pairs := [][2]string{
		{binding.MIMEXML, binding.MIMEXML2},
		{binding.MIMEMSGPACK, binding.MIMEMSGPACK2},
		{binding.MIMEYAML, binding.MIMEYAML2},
	}
	for _, pair := range pairs {
		a := binding.Default(http.MethodPost, pair[0])
		b := binding.Default(http.MethodPost, pair[1])
		if a != b {
			t.Fatalf("aliases %q and %q returned different binders: %v vs %v", pair[0], pair[1], a, b)
		}
	}
}

func TestBBDefaultFormFallback(t *testing.T) {
	for _, ct := range bbFormFallbackMIME {
		got := binding.Default(http.MethodPost, ct)
		if got != binding.Form {
			t.Fatalf("Default(POST,%q)=%v want Form", ct, got)
		}
	}
}

func TestBBDefaultUnknownMIME(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		ct := fmt.Sprintf("application/x-rand-%d-%d", rng.Intn(1<<16), i)
		if strings.HasPrefix(ct, binding.MIMEJSON) {
			continue
		}
		got := binding.Default(http.MethodPost, ct)
		if got != binding.Form {
			t.Fatalf("case %d: unknown MIME %q should fall back to Form, got %v", i, ct, got)
		}
	}
}

func TestBBValidationDisabledNilValidator(t *testing.T) {
	backup := binding.Validator
	binding.Validator = nil
	defer func() { binding.Validator = backup }()

	var obj bbFooRequired
	req := bbRequestWithJSON(`{"bar":"only-bar"}`)
	if err := binding.JSON.Bind(req, &obj); err != nil {
		t.Fatalf("JSON.Bind with Validator=nil should succeed: %v", err)
	}
	if obj.Foo != "" {
		t.Fatalf("expected empty Foo, got %q", obj.Foo)
	}
}

func TestBBValidationDisabledRandomBodies(t *testing.T) {
	backup := binding.Validator
	binding.Validator = nil
	defer func() { binding.Validator = backup }()

	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		n := rng.Intn(8)
		parts := make([]string, n)
		for j := range parts {
			parts[j] = fmt.Sprintf(`"k%d":%d`, j, rng.Intn(1000))
		}
		body := "{" + strings.Join(parts, ",") + "}"
		var obj bbFooRequired
		req := bbRequestWithJSON(body)
		if err := binding.JSON.Bind(req, &obj); err != nil {
			t.Fatalf("case %d: Validator=nil bind failed: %v", i, err)
		}
	}
}

func TestBBValidationEnabledRequiresField(t *testing.T) {
	backup := binding.Validator
	defer func() { binding.Validator = backup }()

	var obj bbFooRequired
	req := bbRequestWithJSON(`{"bar":"x"}`)
	if err := binding.JSON.Bind(req, &obj); err == nil {
		t.Fatal("JSON.Bind with default Validator should fail when required field missing")
	}

	req2 := bbRequestWithJSON(`{"foo":"ok"}`)
	if err := binding.JSON.Bind(req2, &obj); err != nil {
		t.Fatalf("JSON.Bind with valid required field failed: %v", err)
	}
}
