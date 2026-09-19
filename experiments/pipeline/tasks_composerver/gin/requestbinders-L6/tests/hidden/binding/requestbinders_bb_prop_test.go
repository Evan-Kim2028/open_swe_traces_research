// Black-box property suite for the requestbinders unit.
// Exported API only: binding.Form, binding.FormPost, binding.Query,
// binding.Header, binding.Uri, binding.FormMultipart.
// Seed 20260919; >=10k cases; contract + binding_test.go coverage.
//
// Coverage table (contract sentence -> property):
//   "urlencoded+multipart binder maps parsed form incl. query values"
//       -> TestBBFormBinderQueryAndBody / TestBBFormBinderToleratesNonMultipart
//   "post-form binder reads body values only"
//       -> TestBBFormPostBodyOnly / TestBBFormPostIgnoresQuery
//   "query binder maps URL values and propagates conversion errors"
//       -> TestBBQueryBinderURLOnly / TestBBQueryConversionError
//   "header binder maps canonicalized header keys via the header tag"
//       -> TestBBHeaderBinderCanonical / TestBBHeaderBinderCaseInsensitive
//   "uri binder maps the params map via the uri tag, nested structs included"
//       -> TestBBUriBinderParams / TestBBUriNestedStruct
//   "every binder reports a short lowercase Name()"
//       -> TestBBBinderNamesLowercase
//   "validation runs after mapping in all binders"
//       -> TestBBValidationAfterMapping / TestBBValidationAfterMappingRandom
package binding_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	binding "example.internal/httprouter/binding"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

type bbFooBar struct {
	Foo string `form:"foo" query:"foo" header:"foo" uri:"foo" binding:"required"`
	Bar string `form:"bar" query:"bar" header:"bar" uri:"bar"`
}

type bbUriNested struct {
	Name string `uri:"name"`
	Inner struct {
		Age int `uri:"age"`
	} `uri:""`
}

func bbPostFormRequest(path string, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", binding.MIMEPOSTForm)
	return req
}

func TestBBFormBinderQueryAndBody(t *testing.T) {
	req := bbPostFormRequest("/?foo=qbar&extra=1", "foo=bar&bar=foo")
	var obj bbFooBar
	if err := binding.Form.Bind(req, &obj); err != nil {
		t.Fatalf("Form.Bind: %v", err)
	}
	if obj.Foo != "bar" || obj.Bar != "foo" {
		t.Fatalf("got foo=%q bar=%q", obj.Foo, obj.Bar)
	}
}

func TestBBFormBinderToleratesNonMultipart(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?foo=fromquery&bar=qb", nil)
	var obj bbFooBar
	if err := binding.Form.Bind(req, &obj); err != nil {
		t.Fatalf("GET Form.Bind: %v", err)
	}
	if obj.Foo != "fromquery" {
		t.Fatalf("foo=%q want fromquery", obj.Foo)
	}
}

func TestBBFormPostBodyOnly(t *testing.T) {
	req := bbPostFormRequest("/?foo=fromquery&bar=fromquery", "foo=bar&bar=body")
	var obj bbFooBar
	if err := binding.FormPost.Bind(req, &obj); err != nil {
		t.Fatalf("FormPost.Bind: %v", err)
	}
	if obj.Foo != "bar" || obj.Bar != "body" {
		t.Fatalf("got foo=%q bar=%q want body values", obj.Foo, obj.Bar)
	}
}

func TestBBFormPostIgnoresQuery(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 8))
	for i := 0; i < bbCases; i++ {
		qFoo := fmt.Sprintf("q%d", rng.Intn(1000))
		bFoo := fmt.Sprintf("b%d", rng.Intn(1000))
		path := fmt.Sprintf("/?foo=%s&bar=ignored", url.QueryEscape(qFoo))
		body := fmt.Sprintf("foo=%s&bar=mapped", url.QueryEscape(bFoo))
		req := bbPostFormRequest(path, body)
		var obj bbFooBar
		if err := binding.FormPost.Bind(req, &obj); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if obj.Foo != bFoo {
			t.Fatalf("case %d: foo=%q want body %q", i, obj.Foo, bFoo)
		}
		if obj.Bar != "mapped" {
			t.Fatalf("case %d: bar=%q want mapped", i, obj.Bar)
		}
	}
}

func TestBBQueryBinderURLOnly(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/?foo=bar&bar=foo", bytes.NewBufferString("foo=ignored"))
	var obj bbFooBar
	if err := binding.Query.Bind(req, &obj); err != nil {
		t.Fatalf("Query.Bind: %v", err)
	}
	if obj.Foo != "bar" || obj.Bar != "foo" {
		t.Fatalf("got foo=%q bar=%q", obj.Foo, obj.Bar)
	}
}

func TestBBQueryConversionError(t *testing.T) {
	type boolStruct struct {
		Flag bool `form:"flag" query:"flag"`
	}
	req := httptest.NewRequest(http.MethodGet, "/?flag=notabool", nil)
	var obj boolStruct
	if err := binding.Query.Bind(req, &obj); err == nil {
		t.Fatal("expected conversion error for invalid bool query value")
	}
}

func TestBBHeaderBinderCanonical(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Limit", "1000")
	type headerStruct struct {
		Limit int `header:"limit"`
	}
	var obj headerStruct
	if err := binding.Header.Bind(req, &obj); err != nil {
		t.Fatalf("Header.Bind: %v", err)
	}
	if obj.Limit != 1000 {
		t.Fatalf("limit=%d want 1000", obj.Limit)
	}
}

func TestBBHeaderBinderCaseInsensitive(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 9))
	for i := 0; i < bbCases; i++ {
		val := rng.Intn(10000) + 1
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		key := "X-Custom-Header"
		if i%2 == 0 {
			key = strings.ToLower(key)
		}
		req.Header.Set(key, fmt.Sprintf("%d", val))
		type h struct {
			V int `header:"x-custom-header"`
		}
		var obj h
		if err := binding.Header.Bind(req, &obj); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if obj.V != val {
			t.Fatalf("case %d: v=%d want %d", i, obj.V, val)
		}
	}
}

func TestBBUriBinderParams(t *testing.T) {
	m := map[string][]string{"foo": {"thinkerou"}, "bar": {"baz"}}
	var obj bbFooBar
	if err := binding.Uri.BindUri(m, &obj); err != nil {
		t.Fatalf("Uri.BindUri: %v", err)
	}
	if obj.Foo != "thinkerou" || obj.Bar != "baz" {
		t.Fatalf("got foo=%q bar=%q", obj.Foo, obj.Bar)
	}
}

func TestBBUriNestedStruct(t *testing.T) {
	m := map[string][]string{
		"name": {"mike"},
		"age":  {"25"},
	}
	var tag bbUriNested
	if err := binding.Uri.BindUri(m, &tag); err != nil {
		t.Fatalf("Uri.BindUri: %v", err)
	}
	if tag.Name != "mike" || tag.Inner.Age != 25 {
		t.Fatalf("got name=%q age=%d", tag.Name, tag.Inner.Age)
	}
}

func TestBBBinderNamesLowercase(t *testing.T) {
	names := map[string]string{
		"form":                binding.Form.Name(),
		"form-urlencoded":     binding.FormPost.Name(),
		"query":               binding.Query.Name(),
		"header":              binding.Header.Name(),
		"uri":                 binding.Uri.Name(),
		"multipart/form-data": binding.FormMultipart.Name(),
	}
	for expect, got := range names {
		if got != expect {
			t.Fatalf("Name()=%q want %q", got, expect)
		}
		if got != strings.ToLower(got) {
			t.Fatalf("Name()=%q is not lowercase", got)
		}
	}
}

func TestBBValidationAfterMapping(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?bar=onlybar", nil)
	var obj bbFooBar
	if err := binding.Query.Bind(req, &obj); err == nil {
		t.Fatal("Query.Bind should fail validation when required foo missing")
	}
}

func TestBBValidationAfterMappingRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 10))
	for i := 0; i < bbCases; i++ {
		includeFoo := rng.Intn(2) == 0
		q := url.Values{}
		if includeFoo {
			q.Set("foo", "present")
		}
		q.Set("bar", "b")
		req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
		var obj bbFooBar
		err := binding.Query.Bind(req, &obj)
		if includeFoo {
			if err != nil {
				t.Fatalf("case %d: expected success: %v", i, err)
			}
		} else if err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}
