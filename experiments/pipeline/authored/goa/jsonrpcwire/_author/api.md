# Exported API — jsonrpcwire

```
func MakeSuccessResponse(id any, result any) *Response
func MakeErrorResponse(id any, code Code, message string, data any) *Response
func DecodeServiceErrorData(data json.RawMessage) (string, json.RawMessage, bool)
func MakeNotification(method string, params any) *Request
func (r *Response) MarshalJSON() ([]byte, error)
func IDToString(id any) (string, error)
func SinglePositionalParam(params json.RawMessage) (json.RawMessage, error)
func (r *RawRequest) UnmarshalJSON(data []byte) error
func (r *RawResponse) UnmarshalJSON(data []byte) error
func (r *RawResponse) Validate(expectedID string) error
```

IDs are string, number (json.Number, never float64), or null. Empty string id is still an id. Params, when present, must be object or array. Success marshal always emits result, including null; errors omit result. Validate requires jsonrpc 2.0, exactly one of result/error, an id; null id is allowed only for parse (-32700) and invalid-request (-32600) errors.

## Pre-existing callers

Generated JSON-RPC clients and servers; jsonrpc codegen; package tests.
