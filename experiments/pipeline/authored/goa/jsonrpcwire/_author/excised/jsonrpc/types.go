// This file defines the JSON-RPC messages shared by generated clients and
// servers. It preserves request IDs exactly so a server can return the value
// it received and a client can reject a response for another request.
package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type (
	// Request represents a JSON-RPC request.
	Request struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
		ID      any    `json:"id,omitempty"`
	}

	// Response represents a JSON-RPC response.
	Response struct {
		JSONRPC string         `json:"jsonrpc"`
		Result  any            `json:"result,omitempty"`
		Error   *ErrorResponse `json:"error,omitempty"`
		ID      any            `json:"id"`
	}

	// ErrorResponse represents a JSON-RPC error response.
	ErrorResponse struct {
		Code    Code   `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	}

	// RawRequest represents a JSON-RPC request with a marshalled params.
	RawRequest struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
		ID      any             `json:"id"`
		// Invalid is true when the JSON value is not shaped like a JSON-RPC
		// request object.
		Invalid bool `json:"-"`
		// HasID is true when the "id" key is present in the incoming JSON, even
		// when its value is null. Generated servers compare it with the method's
		// declared request or notification contract before calling the service.
		HasID bool `json:"-"`
		// HasMethod is true when the "method" key is present, including when its
		// value is an empty string.
		HasMethod bool `json:"-"`
	}

	// RawResponse represents a JSON-RPC response with a marshalled result
	// and error.
	RawResponse struct {
		JSONRPC string            `json:"jsonrpc"`
		Result  json.RawMessage   `json:"result,omitempty"`
		Error   *RawErrorResponse `json:"error,omitempty"`
		ID      any               `json:"id,omitempty"`
		// Invalid is true when the JSON value is not shaped like a JSON-RPC
		// response object.
		Invalid bool `json:"-"`
		// HasResult is true when the response contains the "result" key,
		// including a null result.
		HasResult bool `json:"-"`
		// HasError is true when the response contains the "error" key.
		HasError bool `json:"-"`
		// HasID is true when the response contains the "id" key, including a
		// null ID.
		HasID bool `json:"-"`
	}

	// RawErrorResponse represents a JSON-RPC error response with marshalled
	// data.
	RawErrorResponse struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data,omitempty"`
	}

	// Code is a JSON-RPC error code, see JSON-RPC 2.0 section 5.1
	Code int
)

const (
	ParseError     Code = -32700
	InvalidRequest Code = -32600
	MethodNotFound Code = -32601
	InvalidParams  Code = -32602
	InternalError  Code = -32603
)

// MakeSuccessResponse creates a success response.
func MakeSuccessResponse(id any, result any) *Response { panic("excised: MakeSuccessResponse") }

// MakeErrorResponse creates an error response.
func MakeErrorResponse(id any, code Code, message string, data any) *Response { panic("excised: MakeErrorResponse") }

// DecodeServiceErrorData decodes the data object written for a designed Apikit
// error. It returns ok false and empty name and body values when data is not
// that object, so callers can preserve the original JSON-RPC error.
func DecodeServiceErrorData(data json.RawMessage) (string, json.RawMessage, bool) { panic("excised: DecodeServiceErrorData") }

// MakeNotification creates a notification.
func MakeNotification(method string, params any) *Request { panic("excised: MakeNotification") }

// MarshalJSON writes the result member for every success response, including
// responses whose result is null, and writes only the error for failures.
func (r *Response) MarshalJSON() ([]byte, error) { panic("excised: MarshalJSON") }

// Error returns a string representation of the error.
func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("jsonrpc: code %d: %s", e.Code, e.Message)
}

// Error returns a string representation of the error.
func (e *RawErrorResponse) Error() string {
	return fmt.Sprintf("jsonrpc: code %d: %s", e.Code, e.Message)
}

// IDToString converts a decoded JSON-RPC string or number ID to its exact text.
func IDToString(id any) (string, error) { panic("excised: IDToString") }

// SinglePositionalParam returns the only value in a positional params array.
// It rejects named params and arrays that do not contain exactly one value.
func SinglePositionalParam(params json.RawMessage) (json.RawMessage, error) { panic("excised: SinglePositionalParam") }

// UnmarshalJSON decodes one request and records invalid input and ID presence.
func (r *RawRequest) UnmarshalJSON(data []byte) error { panic("excised: UnmarshalJSON") }

// UnmarshalJSON decodes one response and records the fields whose presence is
// required to distinguish a valid response from a zero Go value.
func (r *RawResponse) UnmarshalJSON(data []byte) error { panic("excised: UnmarshalJSON") }

// Validate checks that the response is a complete JSON-RPC 2.0 envelope for
// expectedID. Parse and invalid-request errors may use a null ID because the
// server could not recover the request ID from malformed input.
func (r *RawResponse) Validate(expectedID string) error { panic("excised: Validate") }

// decodeRawError decodes the required members of one JSON-RPC error object.
// Empty messages and any integer code remain valid values.
func decodeRawError(raw json.RawMessage) (*RawErrorResponse, bool) { panic("excised: decodeRawError") }

// decodeID preserves a JSON-RPC string, number, or null ID without converting
// numbers through float64. The boolean is false for every other JSON value.
func decodeID(raw json.RawMessage) (any, bool) { panic("excised: decodeID") }

func _keepExcisedImports() {
	_ = bytes.TrimSpace
}
