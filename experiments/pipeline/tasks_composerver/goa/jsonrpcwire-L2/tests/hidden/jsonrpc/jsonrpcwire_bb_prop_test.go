// Black-box property suite for jsonrpcwire (JSON-RPC envelope helpers).
// Exported API only: MakeSuccessResponse, MakeErrorResponse, DecodeServiceErrorData,
// MakeNotification, Response.MarshalJSON, IDToString, SinglePositionalParam,
// RawRequest.UnmarshalJSON, RawResponse.UnmarshalJSON, RawResponse.Validate.
// Seed 20260919; >=10k cases.
//
// Coverage table (contract sentence -> property):
//   "success envelope version 2.0 with result always encoded including null"
//       -> TestJsonrpcwireSuccessMarshalProperty / TestJsonrpcwireSuccessMarshalRandom
//   "error envelope omits result; empty messages filled from standard codes"
//       -> TestJsonrpcwireErrorMarshalProperty / TestJsonrpcwireDefaultErrorMessagesRandom
//   "notification has version, method, params, no id"
//       -> TestJsonrpcwireNotificationProperty
//   "id converts to text only for string or JSON number"
//       -> TestJsonrpcwireIDToStringTable / TestJsonrpcwireIDToStringRandom
//   "positional params are sole array element"
//       -> TestJsonrpcwireSinglePositionalParamTable / TestJsonrpcwireSinglePositionalRandom
//   "designed service error data needs name and body"
//       -> TestJsonrpcwireDecodeServiceErrorDataTable / TestJsonrpcwireDecodeServiceErrorRandom
//   "RawRequest records id/method presence, numeric ids, structured params"
//       -> TestJsonrpcwireRawRequestAdversarial / TestJsonrpcwireRawRequestRandom
//   "RawResponse.Validate enforces envelope, null-id exceptions, id match"
//       -> TestJsonrpcwireRawResponseValidateTable / TestJsonrpcwireValidateRandom
package jsonrpc_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"example.internal/apikit/v3/jsonrpc"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

func jsonrpcOracleIDToString(id any) (string, error) {
	switch v := id.(type) {
	case string:
		return v, nil
	case json.Number:
		return v.String(), nil
	default:
		return "", fmt.Errorf("JSON-RPC id has unexpected type %T", id)
	}
}

func jsonrpcOracleDecodeService(data json.RawMessage) (string, json.RawMessage, bool) {
	var value struct {
		Name *string         `json:"name"`
		Body json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(data, &value); err != nil || value.Name == nil || len(value.Body) == 0 {
		return "", nil, false
	}
	return *value.Name, value.Body, true
}

func jsonrpcOracleDefaultErrorMessage(code jsonrpc.Code) string {
	switch code {
	case jsonrpc.ParseError:
		return "Parse error"
	case jsonrpc.InvalidRequest:
		return "Invalid request"
	case jsonrpc.MethodNotFound:
		return "Method not found"
	case jsonrpc.InvalidParams:
		return "Invalid params"
	case jsonrpc.InternalError:
		return "Internal error"
	default:
		return "Unknown error"
	}
}

func TestJsonrpcwireSuccessMarshalProperty(t *testing.T) {
	for _, id := range []any{"call-1", "", json.Number("42"), nil} {
		resp := jsonrpc.MakeSuccessResponse(id, nil)
		if resp.JSONRPC != "2.0" {
			t.Fatalf("jsonrpc version: %q", resp.JSONRPC)
		}
		b, err := resp.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON: %v", err)
		}
		if !strings.Contains(string(b), `"result"`) {
			t.Fatalf("success must include result: %s", b)
		}
		if strings.Contains(string(b), `"error"`) {
			t.Fatalf("success must omit error: %s", b)
		}
	}
}

func TestJsonrpcwireSuccessMarshalRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		var id any
		switch rng.Intn(4) {
		case 0:
			id = fmt.Sprintf("id-%d", i)
		case 1:
			id = ""
		case 2:
			id = json.Number(fmt.Sprintf("%d", rng.Int63n(1_000_000)))
		case 3:
			id = nil
		}
		result := map[string]int{"n": i}
		if rng.Intn(2) == 0 {
			result = nil
		}
		resp := jsonrpc.MakeSuccessResponse(id, result)
		b, err := resp.MarshalJSON()
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if !strings.Contains(string(b), `"result"`) {
			t.Fatalf("case %d: missing result in %s", i, b)
		}
	}
}

func TestJsonrpcwireErrorMarshalProperty(t *testing.T) {
	resp := jsonrpc.MakeErrorResponse("x", jsonrpc.InternalError, "boom", nil)
	b, err := resp.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if strings.Contains(string(b), `"result"`) {
		t.Fatalf("error response must omit result: %s", b)
	}
	if !strings.Contains(string(b), `"error"`) {
		t.Fatalf("error response must include error: %s", b)
	}
}

func TestJsonrpcwireDefaultErrorMessagesRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	codes := []jsonrpc.Code{
		jsonrpc.ParseError, jsonrpc.InvalidRequest, jsonrpc.MethodNotFound,
		jsonrpc.InvalidParams, jsonrpc.InternalError, jsonrpc.Code(-32000),
	}
	for i := 0; i < bbCases; i++ {
		code := codes[rng.Intn(len(codes))]
		want := jsonrpcOracleDefaultErrorMessage(code)
		resp := jsonrpc.MakeErrorResponse("id", code, "", nil)
		if resp.Error == nil || resp.Error.Message != want {
			t.Fatalf("case %d code %d: got %q want %q", i, code, resp.Error.Message, want)
		}
	}
}

func TestJsonrpcwireNotificationProperty(t *testing.T) {
	req := jsonrpc.MakeNotification("notify", map[string]int{"a": 1})
	if req.JSONRPC != "2.0" || req.Method != "notify" {
		t.Fatalf("notification fields: %+v", req)
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), `"id"`) {
		t.Fatalf("notification must not include id: %s", b)
	}
}

func TestJsonrpcwireIDToStringTable(t *testing.T) {
	cases := []struct {
		id      any
		want    string
		wantErr bool
	}{
		{id: "call-1", want: "call-1"},
		{id: "", want: ""},
		{id: json.Number("9007199254740993123456789"), want: "9007199254740993123456789"},
		{id: float64(1), wantErr: true},
		{id: nil, wantErr: true},
		{id: map[string]any{}, wantErr: true},
	}
	for _, tc := range cases {
		got, err := jsonrpc.IDToString(tc.id)
		want, oracleErr := jsonrpcOracleIDToString(tc.id)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("id %v: expected error", tc.id)
			}
			continue
		}
		if oracleErr != nil || err != nil || got != want || got != tc.want {
			t.Fatalf("id %v: got %q err %v want %q", tc.id, got, err, want)
		}
	}
}

func TestJsonrpcwireIDToStringRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		var id any
		switch rng.Intn(5) {
		case 0:
			id = fmt.Sprintf("s%d", rng.Intn(1000))
		case 1:
			id = json.Number(fmt.Sprintf("%d", rng.Int63()))
		case 2:
			id = float64(rng.Intn(10))
		case 3:
			id = nil
		case 4:
			id = []int{i}
		}
		got, err := jsonrpc.IDToString(id)
		want, oracleErr := jsonrpcOracleIDToString(id)
		if (err != nil) != (oracleErr != nil) {
			t.Fatalf("case %d: err mismatch %v vs %v", i, err, oracleErr)
		}
		if err == nil && got != want {
			t.Fatalf("case %d: got %q want %q", i, got, want)
		}
	}
}

func TestJsonrpcwireSinglePositionalParamTable(t *testing.T) {
	cases := []struct {
		params  string
		want    string
		wantErr bool
	}{
		{params: `["value"]`, want: `"value"`},
		{params: `[{"name":"value"}]`, want: `{"name":"value"}`},
		{params: `[null]`, want: `null`},
		{params: `{"name":"value"}`, wantErr: true},
		{params: `[]`, wantErr: true},
		{params: `["one","two"]`, wantErr: true},
	}
	for _, tc := range cases {
		val, err := jsonrpc.SinglePositionalParam(json.RawMessage(tc.params))
		if tc.wantErr {
			if err == nil {
				t.Fatalf("params %s: expected error", tc.params)
			}
			continue
		}
		if err != nil {
			t.Fatalf("params %s: %v", tc.params, err)
		}
		if string(val) != tc.want {
			t.Fatalf("params %s: got %s want %s", tc.params, val, tc.want)
		}
	}
}

func TestJsonrpcwireSinglePositionalRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		n := rng.Intn(4)
		var raw string
		wantErr := n != 1
		switch n {
		case 0:
			raw = `[]`
		case 1:
			raw = fmt.Sprintf(`[%d]`, i)
		case 2:
			raw = `["a","b"]`
		case 3:
			raw = `{"k":1}`
		}
		_, err := jsonrpc.SinglePositionalParam(json.RawMessage(raw))
		if wantErr && err == nil {
			t.Fatalf("case %d params %s: expected error", i, raw)
		}
		if !wantErr && err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

func TestJsonrpcwireDecodeServiceErrorDataTable(t *testing.T) {
	for _, tc := range []struct {
		data     string
		wantName string
		wantBody string
		wantOK   bool
	}{
		{data: `{"name":"busy","body":{"retryAfter":3}}`, wantName: "busy", wantBody: `{"retryAfter":3}`, wantOK: true},
		{data: `{"name":"quiet","body":null}`, wantName: "quiet", wantBody: `null`, wantOK: true},
		{data: `{"name":"busy","body":{}}`, wantName: "busy", wantBody: `{}`, wantOK: true},
		{data: `{"body":{}}`, wantOK: false},
		{data: `{"name":null,"body":{}}`, wantOK: false},
		{data: `{"name":1,"body":{}}`, wantOK: false},
		{data: `{"name":"busy"}`, wantOK: false},
		{data: `[]`, wantOK: false},
		{data: `{`, wantOK: false},
	} {
		name, body, ok := jsonrpc.DecodeServiceErrorData(json.RawMessage(tc.data))
		oname, obody, ook := jsonrpcOracleDecodeService(json.RawMessage(tc.data))
		if ok != tc.wantOK || ok != ook {
			t.Fatalf("data %s: ok %v want %v oracle %v", tc.data, ok, tc.wantOK, ook)
		}
		if !ok {
			continue
		}
		if name != tc.wantName || name != oname {
			t.Fatalf("name %q want %q", name, tc.wantName)
		}
		if string(body) != tc.wantBody || string(body) != string(obody) {
			t.Fatalf("body %q want %q", body, tc.wantBody)
		}
	}
}

func TestJsonrpcwireDecodeServiceErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases; i++ {
		var data string
		switch rng.Intn(6) {
		case 0:
			data = fmt.Sprintf(`{"name":"n%d","body":{"v":%d}}`, i, i)
		case 1:
			data = `{"name":"x","body":null}`
		case 2:
			data = `{"body":{}}`
		case 3:
			data = `{"name":null,"body":{}}`
		case 4:
			data = `[]`
		case 5:
			data = `{`
		}
		_, _, ok := jsonrpc.DecodeServiceErrorData(json.RawMessage(data))
		_, _, ook := jsonrpcOracleDecodeService(json.RawMessage(data))
		if ok != ook {
			t.Fatalf("case %d data %s: ok %v oracle %v", i, data, ok, ook)
		}
	}
}

func TestJsonrpcwireRawRequestAdversarial(t *testing.T) {
	const bigNum = `{"jsonrpc":"2.0","id":9007199254740993123456789,"method":"lookup"}`
	var decoded jsonrpc.RawRequest
	if err := json.Unmarshal([]byte(bigNum), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !decoded.HasID {
		t.Fatal("expected HasID")
	}
	if decoded.ID != json.Number("9007199254740993123456789") {
		t.Fatalf("numeric id: %v", decoded.ID)
	}
	encoded, err := json.Marshal(jsonrpc.MakeSuccessResponse(decoded.ID, nil))
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if !strings.Contains(string(encoded), "9007199254740993123456789") {
		t.Fatalf("round-trip id missing: %s", encoded)
	}
}

func TestJsonrpcwireRawRequestRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	for i := 0; i < bbCases; i++ {
		var req string
		var wantInvalid bool
		switch rng.Intn(8) {
		case 0:
			req = `{"jsonrpc":"2.0","method":"m","params":{}}`
		case 1:
			req = `{"jsonrpc":"2.0","method":"m","params":[]}`
		case 2:
			req = `{"jsonrpc":"2.0","method":"m","params":"x"}`
			wantInvalid = true
		case 3:
			req = `{"jsonrpc":"2.0","method":null}`
			wantInvalid = true
		case 4:
			req = `{"jsonrpc":"2.0"}`
		case 5:
			req = `{"jsonrpc":"2.0","method":"","id":""}`
		case 6:
			req = `{"jsonrpc":"2.0","id":null,"method":"m"}`
		case 7:
			req = `{"jsonrpc":"2.0","id":{},"method":"m"}`
			wantInvalid = true
		}
		var decoded jsonrpc.RawRequest
		if err := json.Unmarshal([]byte(req), &decoded); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if decoded.Invalid != wantInvalid {
			t.Fatalf("case %d req %s: invalid %v want %v", i, req, decoded.Invalid, wantInvalid)
		}
	}
}

func TestJsonrpcwireRawResponseValidateTable(t *testing.T) {
	cases := []struct {
		response   string
		expectedID string
		wantErr    string
	}{
		{response: `{"jsonrpc":"2.0","id":"call-1","result":null}`, expectedID: "call-1"},
		{response: `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}}`, expectedID: "call-1"},
		{response: `{"jsonrpc":"2.0","id":null,"error":{"code":-32601,"message":"x"}}`, expectedID: "call-1", wantErr: "response id is null"},
		{response: `{"jsonrpc":"2.0","id":1,"result":null}`, expectedID: "1", wantErr: "response id is a number"},
		{response: `{"jsonrpc":"2.0","id":"call-2","result":null}`, expectedID: "call-1", wantErr: `response id "call-2" does not match request id "call-1"`},
	}
	for _, tc := range cases {
		var response jsonrpc.RawResponse
		if err := json.Unmarshal([]byte(tc.response), &response); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		err := response.Validate(tc.expectedID)
		if tc.wantErr == "" {
			if err != nil {
				t.Fatalf("response %s: %v", tc.response, err)
			}
			continue
		}
		if err == nil || err.Error() != tc.wantErr {
			t.Fatalf("response %s: got %v want %q", tc.response, err, tc.wantErr)
		}
	}
}

func TestJsonrpcwireValidateRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 6))
	for i := 0; i < bbCases; i++ {
		id := fmt.Sprintf("c%d", i)
		var resp string
		var wantErr bool
		switch rng.Intn(5) {
		case 0:
			resp = fmt.Sprintf(`{"jsonrpc":"2.0","id":"%s","result":null}`, id)
		case 1:
			resp = fmt.Sprintf(`{"jsonrpc":"2.0","id":"%s","error":{"code":-32602,"message":"bad"}}`, id)
		case 2:
			resp = `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse"}}`
			wantErr = false
		case 3:
			resp = `{"jsonrpc":"2.0","id":null,"result":null}`
			wantErr = true
		case 4:
			resp = fmt.Sprintf(`{"jsonrpc":"2.0","id":"%s","result":null,"error":{"code":1,"message":"x"}}`, id)
			wantErr = true
		}
		var raw jsonrpc.RawResponse
		if err := json.Unmarshal([]byte(resp), &raw); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		err := raw.Validate(id)
		if wantErr && err == nil {
			t.Fatalf("case %d: expected validation error for %s", i, resp)
		}
		if !wantErr && err != nil && !strings.Contains(resp, "-32700") {
			t.Fatalf("case %d: unexpected %v for %s", i, err, resp)
		}
	}
}
