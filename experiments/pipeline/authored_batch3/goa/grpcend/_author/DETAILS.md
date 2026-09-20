# Commitments — grpcend

1. Prepare defaults request/streaming-request/metadata attributes AND
   installs empty validation objects on each; the default response has
   status code 0. In-tree coverage: none. Inferable: partially.
2. Endpoint error policy resolution: for each method error, look first
   in the service's mapped errors then the API's — duplicated into the
   endpoint list; then for each SERVICE error not yet covered, service
   mapping first, API fallback. In-tree coverage: none. Inferable: no.
3. Legacy stream compat reads a shared meta key: endpoint, then method,
   then service, then API — first hit wins — and must equal the one
   supported value. In-tree coverage: deleted tests only. Inferable:
   no.
4. A method declaring both Result and StreamingResult is rejected for
   gRPC. In-tree coverage: none. Inferable: partially.
5. The stream-compat meta only accepts its one legal value; set at
   endpoint or method level it requires a non-empty payload plus a
   streaming payload, and every payload attribute not explicitly in
   metadata must be metadata-encodable. Service/API level meta skips
   the requirement check. In-tree coverage: deleted tests only.
   Inferable: no.
6. Union-typed fields anywhere inside payload, streaming payload,
   result, streaming result, and method errors may not contain array or
   map branches — deduplicated by union and attribute identity.
   In-tree coverage: none. Inferable: no.
7. Message vs payload: a non-object payload maps to a message of
   EXACTLY one field of identical type; an object payload requires
   every message attribute to resolve. In-tree coverage: deleted tests
   only. Inferable: no.
8. rpc:tag validation: every matched field needs a tag, tag numbers
   must be unique (the duplicate names both attributes), and
   union-typed fields are skipped. In-tree coverage: deleted tests
   only. Inferable: partially.
9. Metadata vs payload: every metadata attribute must resolve and be
   metadata-encodable — a primitive or an array of primitives. In-tree
   coverage: none. Inferable: partially.
10. Message+metadata both defined requires an object payload; names in
    both error; neither defined requires rpc:tags on all non-security
    payload fields. In-tree coverage: deleted tests only. Inferable:
    no.
11. Security attributes are found by per-kind credential tags (same set
    as method validation). In-tree coverage: none. Inferable: no.
12. Finalize moves security-tagged payload fields into metadata
    (basic-auth uses no mapped name, others map to "authorization"),
    splits the remaining object fields into the request, propagates
    requiredness into request/metadata validations, merges field meta,
    propagates a proto type name, and finalizes response and errors.
    In-tree coverage: none. Inferable: no.
13. Custom object error types need rpc:tags; the built-in error type
    does not. In-tree coverage: none. Inferable: partially.
14. An inherited error mapping whose attribute shape differs from the
    method's own error type reports a diff on the mapped response.
    In-tree coverage: none. Inferable: no.
