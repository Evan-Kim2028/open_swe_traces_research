# authored_batch2 — goa L0-hard units (15)

Authored per `authoring_hard_l0_units.md`. Each unit excises one closure,
deletes/trims the in-tree tests that pin its commitments, and ships a
complete L2 contract plus a cheat patch that passes the worked examples
while breaking edge-case properties. Overlap check vs batch-1 authored
units: **CLEAN (0 overlaps)** — `uv run python scripts/check_unit_overlap.py --extra /home/evan/Documents/oswt-AUgoa`.

## Batch-1 closures avoided

`eval/error.go` (errloc), `eval/context.go` (evalctx), `eval/eval.go`
(evalrun), `grpc/error.go` (grpcerr), `grpc/middleware/trace.go`
(grpctrace), `grpc/middleware/xray/middleware.go` (grpcxray),
`http/middleware/xray/segment.go` (httpxray), `jsonrpc/types.go`
(jsonrpcwire), `pkg/validation.go` (pkgvalidation),
`middleware/xray/segment.go` (xrayseg). None of the batch-2 closures
touch those files.

## Units

| unit | closure | file | lines | syms | details | predicted_flip | commitments | compiles |
|---|---|---|---|---|---|---|---|---|
| dupexpr | deep type copier, origin identity | `expr/dup.go` | 168 | 10 | 11 | L2 | shared-instance-per-origin, Empty/primitive passthrough, union TypeKey/ValueKey preservation, authored-attribute pointer, DSL registry append gate, cycle safety | yes |
| exprhash | structural type hashing | `expr/hasher.go` | 134 | 8 | 10 | L2 | prefix grammar, ignoreNames-unless-ignoreFields, member sort, union comparator reads sorted copy, tag folding, `seen` recursion | yes |
| httpclienterr | client error ctors + retryability | `http/client.go` | 260 | 14 | 13 | L2 | error kind strings, Unwrap, temporary-vs-permanent classification (conn reset/timeout retryable, DNS not), debug doer dump | yes |
| httpencoding | content negotiation | `http/encoding.go` | 366 | 18 | 13 | L2 | accept parse+retry normalization, `+json`/suffix matching, default content-type, charset param, gob/xml dispatch, request decoder selection | yes |
| httperrresp | error → HTTP response | `http/error.go` | 102 | 3 | 8 | L2 | errors.As unwrapping (wrapped service errors), status→timeout/temporary mapping, XML envelope, content negotiation of error body | yes |
| httpmux | chi-backed mux | `http/mux.go` | 227 | 9 | 10 | L2 | `{*name}` rewrite remembering only first wildcard, r.Pattern before middleware, queue-then-apply middlewares, conditional 404, nil-vars semantics, unescape passthrough | yes |
| importalias | per-package import alias plan | `codegen/import_aliases.go` | 235 | 12 | 12 | L2 | fixed>generated>metadata priorities, declare-vs-freeze error split, explicit-beats-inferred spelling, shared type/import namespace, explicit flag → Import().Name | yes |
| mappedattr | object-key→element mapping | `expr/mapped_attribute.go` | 202 | 13 | 12 | L2 | `att:elem` convention, mapped>identity>panic lookups, validation fallback to user type, requiredness preserved, Delete un-requires, Merge(nil) no-op | yes |
| namescope | unique-name reservation engine | `codegen/scope.go` | 501 | 17 | 12 | L2 | count+1 counters, suffix-then-counter, peek vs reserve, hash idempotency, origin-keyed user types, unbound-union panic after freeze, bind-requires-reservation | yes |
| reqidgen | request-ID options + generation | `middleware/requestid.go` | 104 | 8 | 8 | L2 | UseRequestIDOption sets header even when false, header option implies use, truncation only on reuse with positive limit, 8-char RawURLEncoding IDs | yes |
| retrypolicy | retry-backoff + error typing | `pkg/retry.go` | 70 | 3 | 7 | L2 | Retryable() interface check, Temporary() check, IsConnectionReset, duration arithmetic, stop channel | yes |
| sampler | fixed + adaptive sampling | `middleware/sampler.go` | 101 | 4 | 9 | L2 | boundary panics (0/100 valid), fresh adaptive samples all until first window, exact counter==size adjust, rate formula + clamp [1,UB], rate never 0 | yes |
| skipwriter | response-writer pipe | `pkg/skip_response_writer.go` | 66 | 7 | 7 | L2 | lazy pipe init on first write, Close initializes un-read pipe, WriteThrough/Written flag, read-then-write interplay | yes |
| svcerror | service error type + conversions | `pkg/error.go` | 303 | 26 | 12 | L2 | Name/Timeout/Temporary/Permanent fields, error-name constants, MergeError nil handling, Fault() semantic, MissingField errors, custom http code | yes |
| traceopts | trace options + helpers | `middleware/trace.go` | 181 | 14 | 10 | L2 | defaults (100%, size 1000), panic-at-construction validation, adaptive iff rate>0, WithSpan conditional parent key, logger pair on non-empty traceID | yes |

Validation: `go build ./...` and `go test -count=1 -run '^$'` on all
affected test packages pass in every unit's excised tree (15/15).

## Notes for the verifier session

- `namescope` deleted `codegen/generated_types_test.go` (it asserted
  unique-name conventions); shared helpers were moved to the added file
  `codegen/generation_test_helpers_test.go`.
- `reqidgen` trimmed `grpc/middleware/requestid_test.go` to keep only
  `testServerStream` (needed by `trace_test.go`); the request-ID tests
  were removed, not deleted wholesale.
- `mappedattr` deleted no files; `TestHTTPAuthorizationMapping` was
  trimmed from `expr/http_endpoint_test.go` and requiredness assertions
  from `expr/http_body_types_test.go`.
- `exprhash` trimmed hash-equality assertions from `codegen/scope_test.go`
  and `codegen/generated_types_test.go` (Union/UserType `Hash()` route
  through the excised `Hash`).
- `dupexpr` trimmed five pinning functions across
  `expr/user_type_test.go`, `expr/error_contract_test.go`, and
  `expr/types_test.go`.
- `importalias` trimmed three pinning functions across
  `generated_import_plan_test.go`, `go_type_plan_test.go`, and
  `go_transform_test.go`.
- `namescope` also trimmed `TestGeneratedImportPlanLinkExactAliases`.
- `gold.patch` in every unit touches no `*_test.go` (A12).
