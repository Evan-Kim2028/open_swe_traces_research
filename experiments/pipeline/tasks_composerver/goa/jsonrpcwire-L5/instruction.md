# Contract (L2) — jsonrpcwire

A success envelope is version 2.0, carries the given id and result, and when encoded always includes a result member even if the result is null; an error envelope omits result. Empty default messages are filled from the standard codes (parse, invalid request, method not found, invalid params, internal, else "Unknown error"). Designed application errors in the error data object have a name string and a body; anything else is not that object. A notification has version, method, and params and no id. An id converts to text only when it is a string or a JSON number; other Go types are rejected. A positional-params value is the sole element of a JSON array. Decoding a request records whether id and method keys were present (including empty string and null), preserves numeric ids as JSON numbers, and marks the request invalid when params exist but are not an object or array, or when id/method/jsonrpc have the wrong JSON type. Decoding a response records presence of result, error, and id, including nulls. Validation requires version 2.0, exactly one of result or error, and an id; a null id is allowed only for parse and invalid-request errors; a numeric id is rejected; the string id must match the expected request id.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRequestKeepsEmptyStringID` | an empty-string id remains an id in the encoded request |
| `TestRawRequestPreservesNumericID` | a large numeric id round-trips as a JSON number, not float64 |
| `TestIDToString` | only string and JSON-number ids convert; other types error |
| `TestRawRequestRequiresStructuredParams` | params must be object or array when present |
| `TestRawRequestRecordsMethodPresence` | missing vs empty vs null method are distinguishable |
| `TestSinglePositionalParam` | exactly one array element is required; objects and other lengths fail |
| `TestDecodeServiceErrorData` | a designed error needs a name and a body; otherwise ok is false |
| `TestRawResponseValidation` | envelope shape, null-id exceptions, and id matching |

Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./jsonrpc/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestJsonrpcwireSuccessMarshalProperty`, `TestJsonrpcwireSuccessMarshalRandom`, `TestJsonrpcwireErrorMarshalProperty`, `TestJsonrpcwireDefaultErrorMessagesRandom`, `TestJsonrpcwireNotificationProperty`, `TestJsonrpcwireIDToStringTable`, `TestJsonrpcwireIDToStringRandom`, `TestJsonrpcwireSinglePositionalParamTable`, `TestJsonrpcwireSinglePositionalRandom`, `TestJsonrpcwireDecodeServiceErrorDataTable`, `TestJsonrpcwireDecodeServiceErrorRandom`, `TestJsonrpcwireRawRequestAdversarial`, `TestJsonrpcwireRawRequestRandom`, `TestJsonrpcwireRawResponseValidateTable`, `TestJsonrpcwireValidateRandom`: TestJsonrpcwireSuccessMarshalProperty

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
