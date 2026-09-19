# Contract (L2) — bodydecoders

Every body binder decodes the request body (or a supplied byte slice) into the destination and then runs struct validation. JSON additionally honors two global switches: one makes the decoder surface numbers as Number instead of float64, the other rejects unknown object keys; a nil request or nil body is an 'invalid request' error before decoding. XML, YAML, TOML and MsgPack decode with their format decoder then validate. ProtoBuf reads the whole body first, requires the destination to implement proto.Message (else 'obj is not ProtoMessage'), unmarshals, and deliberately skips validation. The plain binder writes the raw body into a string or []byte destination after chasing pointers (nil pointer or nil destination is a silent no-op) and errors 'type (%T) unknown type' for anything else. Decode errors propagate; validation errors propagate after a successful decode.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestBindingJSON/TestBindingJSONSlice/TestBindingJSONNilBody` | JSON decodes objects and top-level slices; nil request errors |
| `TestBindingJSONUseNumber/UseNumber2` | UseNumber flag changes number decoding |
| `TestBindingJSONDisallowUnknownFields` | DisallowUnknownFields flag rejects unknown keys |
| `TestJSONBindingBindBody(+Map)` | BindBody works from byte slices without a request |
| `TestCustomJsonCodec` | decode goes through the pluggable codec API |
| `TestBindingXML/TestBindingXMLFail` | XML decodes then validates |
| `TestBindingYAML(+Fail)/TOML(+Fail)` | YAML and TOML decode then validate |
| `TestBindingProtoBuf/TestBindingProtoBufFail` | protobuf requires proto.Message and skips validation |
| `TestBindingBSON` | BSON round-trips then validates |
| `TestPlainBinding` | plain binder fills string/[]byte, no-ops on nil, errors otherwise |
