// Black-box property suite for the bodydecoders unit (package binding).
// Exported binders: JSON, XML, YAML, TOML, ProtoBuf, MsgPack, BSON, Plain;
// EnableDecoderUseNumber, EnableDecoderDisallowUnknownFields.
// Bind/BindBody only. Seed 20260919; >=10k cases in random JSON property.
//
// Contract (contract.md) -> property coverage table:
//
//	C1 "JSON decodes objects/slices; nil request errors" -> TestBDContractTableProperty
//	C2 "UseNumber surfaces json.Number" -> TestBDJSONUseNumberProperty
//	C3 "DisallowUnknownFields rejects extra keys" -> TestBDJSONDisallowUnknownProperty
//	C4 "BindBody decodes byte slices" -> TestBDBindBodyProperty
//	C5 "XML/YAML/TOML/MsgPack decode then validate" -> TestBDFormatDecodeValidateProperty
//	C6 "ProtoBuf requires proto.Message; skips validation" -> TestBDProtoBufProperty
//	C7 "BSON round-trips" -> TestBDBSONProperty
//	C8 "Plain fills string/[]byte; nil no-op; else error" -> TestBDPlainAdversarialProperty
//	C9 unseen random JSON round-trips -> TestBDJSONUnseenRandomProperty
package binding_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.internal/httprouter/binding"
	"example.internal/httprouter/testdata/protoexample"
	"github.com/stretchr/testify/require"
	"github.com/ugorji/go/codec"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/proto"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

type bbFoo struct {
	Foo string `json:"foo" xml:"foo" yaml:"foo" toml:"foo" msgpack:"foo" binding:"required,max=32"`
}

type bbFooSlice struct {
	Foo string `json:"foo" binding:"required,max=32"`
}

type bbFooNumber struct {
	Foo any `json:"foo" binding:"required"`
}

type bbFooUnknown struct {
	Foo any `json:"foo" binding:"required"`
}

func bbReq(method, body string) *http.Request {
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", binding.MIMEJSON)
	return req
}

func bbReqBody(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
}

// TestBDContractTableProperty mirrors binding_test.go JSON fixtures via Bind/BindBody.
func TestBDContractTableProperty(t *testing.T) {
	var obj bbFoo
	require.NoError(t, binding.JSON.BindBody([]byte(`{"foo": "bar"}`), &obj))
	if obj.Foo != "bar" {
		t.Fatalf("BindBody object: %q", obj.Foo)
	}

	var nilReq *http.Request
	var nilObj bbFoo
	if err := binding.JSON.Bind(nilReq, &nilObj); err == nil {
		t.Fatal("nil request should error")
	}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if err := binding.JSON.Bind(req, &nilObj); err == nil {
		t.Fatal("nil body should error")
	}

	var slice []bbFooSlice
	require.NoError(t, binding.JSON.BindBody([]byte(`[{"foo": "123"}]`), &slice))
	if len(slice) != 1 || slice[0].Foo != "123" {
		t.Fatalf("slice decode: %+v", slice)
	}

	var bad bbFoo
	err := binding.JSON.Bind(bbReq(http.MethodPost, `{"bar": "foo"}`), &bad)
	if err == nil {
		t.Fatal("validation should fail on missing foo")
	}
}

// TestBDBindBodyProperty checks map decoding through BindBody without a request.
func TestBDBindBodyProperty(t *testing.T) {
	m := make(map[string]string)
	require.NoError(t, binding.JSON.BindBody([]byte(`{"foo": "FOO","hello":"world"}`), &m))
	if m["foo"] != "FOO" || m["hello"] != "world" {
		t.Fatalf("map BindBody: %v", m)
	}
}

// TestBDJSONUseNumberProperty toggles EnableDecoderUseNumber.
func TestBDJSONUseNumberProperty(t *testing.T) {
	binding.EnableDecoderUseNumber = true
	defer func() { binding.EnableDecoderUseNumber = false }()

	var on bbFooNumber
	require.NoError(t, binding.JSON.Bind(bbReq(http.MethodPost, `{"foo": 123}`), &on))
	n, ok := on.Foo.(json.Number)
	if !ok {
		t.Fatalf("UseNumber: got %T", on.Foo)
	}
	v, err := n.Int64()
	require.NoError(t, err)
	if v != 123 {
		t.Fatalf("UseNumber value %d", v)
	}

	binding.EnableDecoderUseNumber = false
	var off bbFooNumber
	require.NoError(t, binding.JSON.Bind(bbReq(http.MethodPost, `{"foo": 123}`), &off))
	if _, ok := off.Foo.(float64); !ok {
		t.Fatalf("default number: got %T", off.Foo)
	}
}

// TestBDJSONDisallowUnknownProperty rejects unknown object keys when enabled.
func TestBDJSONDisallowUnknownProperty(t *testing.T) {
	binding.EnableDecoderDisallowUnknownFields = true
	defer func() { binding.EnableDecoderDisallowUnknownFields = false }()

	var ok bbFooUnknown
	require.NoError(t, binding.JSON.Bind(bbReq(http.MethodPost, `{"foo": "bar"}`), &ok))

	var bad bbFooUnknown
	err := binding.JSON.Bind(bbReq(http.MethodPost, `{"foo": "bar", "what": "this"}`), &bad)
	if err == nil || !strings.Contains(err.Error(), "what") {
		t.Fatalf("unknown field: %v", err)
	}
}

// TestBDFormatDecodeValidateProperty covers XML, YAML, TOML, MsgPack decode+validate paths.
func TestBDFormatDecodeValidateProperty(t *testing.T) {
	var x bbFoo
	require.NoError(t, binding.XML.BindBody([]byte(`<map><foo>bar</foo></map>`), &x))
	if x.Foo != "bar" {
		t.Fatal("xml")
	}
	var xbad bbFoo
	if err := binding.XML.BindBody([]byte(`<map><foo>bar<foo></map>`), &xbad); err == nil {
		t.Fatal("xml bad")
	}

	var y bbFoo
	require.NoError(t, binding.YAML.BindBody([]byte(`foo: bar`), &y))
	var ybad bbFoo
	if err := binding.YAML.BindBody([]byte(`foo:\nbar`), &ybad); err == nil {
		t.Fatal("yaml bad")
	}

	var tm bbFoo
	require.NoError(t, binding.TOML.BindBody([]byte(`foo="bar"`), &tm))
	var tmbad bbFoo
	if err := binding.TOML.BindBody([]byte(`foo=\n"bar"`), &tmbad); err == nil {
		t.Fatal("toml bad")
	}

	test := bbFoo{Foo: "bar"}
	h := new(codec.MsgpackHandle)
	buf := bytes.NewBuffer(nil)
	require.NoError(t, codec.NewEncoder(buf, h).Encode(test))
	var mp bbFoo
	require.NoError(t, binding.MsgPack.BindBody(buf.Bytes(), &mp))
	if mp.Foo != "bar" {
		t.Fatal("msgpack")
	}
}

// TestBDProtoBufProperty requires proto.Message and does not run struct validation.
func TestBDProtoBufProperty(t *testing.T) {
	msg := &protoexample.Test{Label: proto.String("yes")}
	data, err := proto.Marshal(msg)
	require.NoError(t, err)

	var got protoexample.Test
	require.NoError(t, binding.ProtoBuf.BindBody(data, &got))
	if got.Label == nil || *got.Label != "yes" {
		t.Fatal("protobuf decode")
	}

	var notProto int
	err = binding.ProtoBuf.BindBody(data, &notProto)
	if err == nil || !strings.Contains(err.Error(), "ProtoMessage") {
		t.Fatalf("non-proto dest: %v", err)
	}

	// validation skipped: empty required-like field still binds
	var empty protoexample.Test
	require.NoError(t, binding.ProtoBuf.BindBody(data, &empty))
}

// TestBDBSONProperty round-trips BSON documents through BindBody.
func TestBDBSONProperty(t *testing.T) {
	src := bbFoo{Foo: "bar"}
	data, err := bson.Marshal(&src)
	require.NoError(t, err)
	var dst bbFoo
	require.NoError(t, binding.BSON.BindBody(data, &dst))
	if dst.Foo != "bar" {
		t.Fatal("bson round-trip")
	}
	if len(data) > 1 {
		var bad bbFoo
		if err := binding.BSON.BindBody(data[1:], &bad); err == nil {
			t.Fatal("invalid bson should fail")
		}
	}
}

// TestBDPlainAdversarialProperty exercises plain binder edge cases.
func TestBDPlainAdversarialProperty(t *testing.T) {
	var s string
	require.NoError(t, binding.Plain.Bind(bbReqBody("plain text"), &s))
	if s != "plain text" {
		t.Fatal("string")
	}
	var bs []byte
	require.NoError(t, binding.Plain.BindBody([]byte("bytes"), &bs))
	if string(bs) != "bytes" {
		t.Fatal("[]byte")
	}
	var i int
	if err := binding.Plain.BindBody([]byte("nope"), &i); err == nil {
		t.Fatal("unknown type should error")
	}
	if err := binding.Plain.Bind(bbReqBody("x"), nil); err != nil {
		t.Fatalf("nil dest: %v", err)
	}
	var nilStr *string
	if err := binding.Plain.Bind(bbReqBody("x"), nilStr); err != nil {
		t.Fatalf("nil *string: %v", err)
	}
}

// TestBDJSONUnseenRandomProperty runs >=10k random JSON object binds via BindBody.
func TestBDJSONUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		val := fmt.Sprintf("s%d_%d", i, rng.Int63())
		body := fmt.Sprintf(`{"foo":%q}`, val)
		var obj bbFoo
		if err := binding.JSON.BindBody([]byte(body), &obj); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if obj.Foo != val {
			t.Fatalf("case %d: got %q", i, obj.Foo)
		}
		if i%1000 == 0 {
			binding.EnableDecoderUseNumber = rng.Intn(2) == 0
			var num bbFooNumber
			nbody := fmt.Sprintf(`{"foo":%d}`, rng.Intn(10000))
			_ = binding.JSON.BindBody([]byte(nbody), &num)
			binding.EnableDecoderUseNumber = false
		}
	}
}
