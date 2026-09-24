# Contract (L2) — grpcend

A gRPC endpoint expression prepares, validates and finalizes the transport
view of one method: its request message, streaming request, metadata, response
and error mappings.

Preparation defaults the request, streaming-request and metadata attributes
and installs an empty validation object on each; when no response is defined a
default response with status code 0 is installed. Error policy resolves by
name in two passes: every method-level error is looked up first among the
service's mapped gRPC errors, then the API's, and the resolved mapping is
duplicated into the endpoint's error list; then every service-level error not
already covered by a method error is resolved the same way — service mapping
first, API mapping as fallback.

The legacy stream-compat flag reads one shared metadata key
(`grpc:stream:compat`), consulting endpoint, then method, then service, then
API metadata — first hit wins — and is on only when that value equals the one
supported value. The flag is also validated: a value other than the supported
one is an error; when the flag is set at endpoint or method level the method
must declare both a non-empty one-shot payload and a streaming payload, and
every payload attribute not explicitly mapped into metadata must be
metadata-encodable. The same flag inherited from service or API level skips
that requirement check.

Validation rejects a method that declares both a result and a streaming
result, naming the method. Union-typed fields anywhere inside the payload,
streaming payload, result, streaming result and method error types may not
declare array or map branches — each union and each attribute is checked once,
naming the offending union.

Message and metadata are checked against the payload. A non-object payload
maps to a request message of exactly one field whose type is identical to the
payload's; an object payload requires every declared message attribute to
resolve to a payload attribute by name, naming the unresolved one. Every
matched message field needs an `rpc:tag` field number; tag numbers must be
unique and a duplicate reports both attributes by name; union-typed fields are
skipped. Every metadata attribute must likewise resolve to a payload attribute
and be metadata-encodable — a primitive or an array of primitives. When both
message and metadata are defined the payload must be an object and no name may
appear in both; when neither is defined, every non-security payload field must
carry `rpc:tag`. Security attributes are recognized by their per-kind
credential tags, the same set method security validation uses.

Finalization moves credential-tagged payload fields into request metadata —
basic-auth fields keep their own names, the other schemes map to the shared
authorization name — splits the remaining object fields into the request
message, propagates requiredness into the request and metadata validations,
merges the payload fields' metadata onto the message attributes, propagates a
declared protobuf type name to the request, and finalizes the response and
error mappings.

Mapped errors are checked too: a custom object error type must carry `rpc:tag`
on its fields while the built-in error type needs none; and an error mapping
inherited from the service or API whose error type's attribute shape differs
from the method's own error type reports the difference on the mapped
response.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | prepare defaults request, streaming-request and metadata attributes with empty validations; default response has status code 0 |
| `TestDetail02` | error policy: method errors resolve service-mapping first then API; uncovered service errors likewise; mappings are duplicated into the endpoint list |
| `TestDetail03` | stream-compat meta is read endpoint → method → service → API, first hit wins |
| `TestDetail04` | a method declaring both result and streaming result is rejected, naming the method |
| `TestDetail05` | the compat meta accepts only its one supported value; endpoint/method level requires payload plus streaming payload and metadata-encodable attributes; service/API level skips the check |
| `TestDetail06` | union branches inside payload, streaming payload, result, streaming result and method errors may not be arrays or maps |
| `TestDetail07` | non-object payload maps to a single-field message of identical type; object payload requires every message attribute to resolve |
| `TestDetail08` | every matched field needs `rpc:tag`; duplicate tag numbers name both attributes; union fields skipped |
| `TestDetail09` | every metadata attribute must resolve and be metadata-encodable — primitive or array of primitives |
| `TestDetail10` | message plus metadata requires an object payload with disjoint names; neither defined requires `rpc:tag` on all non-security payload fields |
| `TestDetail11` | security attributes are found by per-kind credential tags |
| `TestDetail12` | finalize moves credential fields to metadata — basic-auth keeps its name, others map to the authorization name — splits the rest into the request, propagates requiredness |
| `TestDetail13` | custom object error types need `rpc:tag`; the built-in error type does not |
| `TestDetail14` | an inherited error mapping whose error shape differs from the method's own reports the difference on the mapped response |
