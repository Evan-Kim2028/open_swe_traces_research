# VERIFIER_BATCH.md — composerver goa

Verifier batch for goa (identity-obfuscated apikit) feature-excision units.
Seed `20260919`. Hidden black-box property suites via `affordance.py`.
Dockerfile `FROM ladder-base:goa`. L0/L2 proved; L5/L6 packaged only.

Build / reprove:

```
uv run python scripts/build_composerver_goa_batch.py --max-parallel 2
```

## Summary

| unit | properties | coverage % | gold 1st | suite fixes | wall min | verdict |
|---|---:|---:|---|---:|---:|---|
| `errloc` | 8 | 100.0 | yes | 0 | 0.2 | PASS |
| `evalctx` | 10 | 100.0 | yes | 0 | 0.1 | PASS |
| `evalrun` | 17 | 100.0 | yes | 0 | 0.2 | PASS |
| `grpcerr` | 10 | 100.0 | yes | 0 | 0.1 | PASS |
| `grpctrace` | 13 | 100.0 | yes | 0 | 0.1 | PASS |
| `grpcxray` | 13 | 83.3 | yes | 0 | 0.1 | PASS |
| `httpxray` | 13 | 100.0 | yes | 0 | 0.1 | PASS |
| `jsonrpcwire` | 15 | 100.0 | yes | 0 | 0.1 | PASS |
| `pkgvalidation` | 7 | 100.0 | yes | 0 | 0.1 | PASS |
| `xrayseg` | 11 | 100.0 | yes | 0 | 0.1 | PASS |

**Batch totals:** 10/10 PASS

---

## `errloc`

- **Properties:** 8
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.2

- **L0:** `experiments/pipeline/tasks_composerver/goa/errloc-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/errloc-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/errloc-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/errloc-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| an error recorded from a source function is tagged with that function's file and the call line | `TestErrlocErrorFormatProperty` |
| a validation failure is tagged with the source function's declaration line | `TestErrlocErrorFormatRandom` |
| a validation message includes `[file:line]` when a source function exists | `TestErrlocMultiErrorProperty` |
| the top-level placeholder prints without a location prefix | `TestErrlocMultiErrorRandom` |
| a nil source function prints the expression name and message only | `TestErrlocReportErrorIntegrationProperty` |
| `@version` path segments are stripped before package matching | `TestErrlocReportErrorRandom` |
| `@version` path segments are stripped before package matching | `TestErrlocValidationWithSourceProperty` |
| `@version` path segments are stripped before package matching | `TestErrlocValidationNoSourceProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `evalctx`

- **Properties:** 10
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/goa/evalctx-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/evalctx-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/evalctx-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/evalctx-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| a registered root is ordered and executed; recorded errors are visible after the run | `TestEvalctxRegisterDuplicateProperty` |
| validation errors recorded on the context are returned by the run | `TestEvalctxRegisterDuplicateRandom` |
| validation errors recorded on the context are returned by the run | `TestEvalctxRegisterPackagesProperty` |
| validation errors recorded on the context are returned by the run | `TestEvalctxRootsOrderProperty` |
| validation errors recorded on the context are returned by the run | `TestEvalctxRootsOrderRandom` |
| validation errors recorded on the context are returned by the run | `TestEvalctxRootsCycleProperty` |
| validation errors recorded on the context are returned by the run | `TestEvalctxStackCurrentProperty` |
| validation errors recorded on the context are returned by the run | `TestEvalctxStackCurrentRandom` |
| validation errors recorded on the context are returned by the run | `TestEvalctxRecordErrorProperty` |
| validation errors recorded on the context are returned by the run | `TestEvalctxRecordRandom` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `evalrun`

- **Properties:** 17
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.2

- **L0:** `experiments/pipeline/tasks_composerver/goa/evalrun-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/evalrun-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/evalrun-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/evalrun-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| a wrongly typed extra argument records one error naming the expected shape | `TestEvalrunInvalidArgError` |
| a missing argument records one error naming the design function | `TestEvalrunInvalidArgErrorRandom` |
| extra arguments record one error naming the design function | `TestEvalrunTooFewArgError` |
| a design function used in the wrong surrounding type records an incompatible-context error | `TestEvalrunTooFewArgErrorRandom` |
| executing stored source records an error that evaluation then returns | `TestEvalrunTooManyArgError` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunTooManyArgErrorRandom` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunIncompatibleDSL` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunIncompatibleDSLRandom` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunRunDSLReportError` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunRunDSLReportErrorRandom` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunRunDSLValidationError` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunRunDSLValidationRandom` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunCurrentStack` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunCurrentStackRandom` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunExecuteNilSource` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunExecuteSuccessRandom` |
| validation failures after execute are returned and later phases do not hide them | `TestEvalrunUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `grpcerr`

- **Properties:** 10
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/goa/grpcerr-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/grpcerr-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/grpcerr-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/grpcerr-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| a single error has no history; a merged error copies each cause | `TestGrpcerrEncodeStatusTable` |
| validation names map to InvalidArgument; timeout/fault/temporary map to deadline/internal/unavailable | `TestGrpcerrEncodeStatusRandom` |
| encoded validation errors remain small enough for gRPC headers | `TestGrpcerrErrorResponseHistoryProperty` |
| Unavailable is temporary; other codes are not | `TestGrpcerrHistoryRandom` |
| an unknown detail type yields a nil decoded message rather than a panic | `TestGrpcerrDecodeDetailProperty` |
| canceled/deadline contexts wrap matching transport codes | `TestGrpcerrDecodeRandom` |
| a one-element join is treated like a wrapper | `TestGrpcerrTransportErrorProperty` |
| two causes stay separate; no context wrap | `TestGrpcerrContextErrorProperty` |
| an active context never wraps | `TestGrpcerrContextErrorRandom` |
| a missing transport error is not wrapped | `TestGrpcerrInvalidTypeProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `grpctrace`

- **Properties:** 13
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/goa/grpctrace-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/grpctrace-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/grpctrace-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/grpctrace-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| new vs continued vs discarded vs zero-rate traces on unary RPCs | `TestGrpctraceUnaryContinueTrace` |
| the same rules on streaming RPCs, with the wrapped stream context | `TestGrpctraceUnaryNewTrace` |
| outgoing metadata carries trace id and parent span when the context is traced | `TestGrpctraceUnaryDiscardRandom` |
| the same outgoing metadata on a stream dial | `TestGrpctraceZeroRateProperty` |
| the same outgoing metadata on a stream dial | `TestGrpctraceZeroRateRandom` |
| the same outgoing metadata on a stream dial | `TestGrpctraceStreamServerContext` |
| the same outgoing metadata on a stream dial | `TestGrpctraceStreamContextRandom` |
| the same outgoing metadata on a stream dial | `TestGrpctraceUnaryClientMetadata` |
| the same outgoing metadata on a stream dial | `TestGrpctraceClientMetadataRandom` |
| the same outgoing metadata on a stream dial | `TestGrpctraceClientNoTrace` |
| the same outgoing metadata on a stream dial | `TestGrpctraceClientNoTraceRandom` |
| the same outgoing metadata on a stream dial | `TestGrpctraceStreamContinueTraceRandom` |
| the same outgoing metadata on a stream dial | `TestGrpctraceUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `grpcxray`

- **Properties:** 13
- **Contract coverage:** 83.3%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/goa/grpcxray-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/grpcxray-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/grpcxray-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/grpcxray-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| a bad collector address fails interceptor construction | `TestGrpcxrayNewUnaryServerBadDaemon` |
| a bad collector address fails interceptor construction | `TestGrpcxrayNewStreamServerBadDaemon` |
| unary server opens a segment, records the RPC, and records errors | `TestGrpcxrayUnaryServerNoTrace` |
| streaming server wraps the stream and records success or failure | `TestGrpcxrayStreamServerNoTraceRandom` |
| unary client is a no-op without a context segment and records a remote subsegment with one | `TestGrpcxrayUnaryServerSegment` |
| stream client treats EOF as success and records other errors once | `TestGrpcxrayUnaryServerRandom` |
| stream client treats EOF as success and records other errors once | `TestGrpcxrayStreamServerWrap` |
| stream client treats EOF as success and records other errors once | `TestGrpcxrayStreamServerRandom` |
| stream client treats EOF as success and records other errors once | `TestGrpcxrayUnaryClientNoSegment` |
| stream client treats EOF as success and records other errors once | `TestGrpcxrayUnaryClientSubsegment` |
| stream client treats EOF as success and records other errors once | `TestGrpcxrayStreamClientEOF` |
| stream client treats EOF as success and records other errors once | `TestGrpcxrayStreamClientErrorRandom` |
| stream client treats EOF as success and records other errors once | `TestGrpcxrayUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `httpxray`

- **Properties:** 13
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/goa/httpxray-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/httpxray-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/httpxray-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/httpxray-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| a bad collector address fails middleware construction | `TestHttpxrayNewBadDaemon` |
| requests with trace metadata open a segment; without metadata they do not | `TestHttpxrayNewBadDaemonRandom` |
| a Doer without a context segment is a pass-through; with one it opens a remote subsegment | `TestHttpxrayMiddlewareNoTrace` |
| a RoundTripper records a remote call when a segment is present | `TestHttpxrayMiddlewareNoTraceRandom` |
| a RoundTripper without a segment does not open one | `TestHttpxrayMiddlewareWithTrace` |
| wrapping the default transport still round-trips | `TestHttpxrayWriteHeaderRandom` |
| the recorded request URL, method, and client address match the incoming request | `TestHttpxrayWrapDoerNoSegment` |
| 429/4xx/5xx set throttle, fault, or error on the segment | `TestHttpxrayWrapDoerWithSegmentRandom` |
| concurrent writes to the wrapping writer do not race | `TestHttpxrayWrapTransport` |
| concurrent writes to the wrapping writer do not race | `TestHttpxrayWrapTransportRandom` |
| concurrent writes to the wrapping writer do not race | `TestHttpxrayRecordRequestProperty` |
| concurrent writes to the wrapping writer do not race | `TestHttpxrayRecordResponseRandom` |
| concurrent writes to the wrapping writer do not race | `TestHttpxrayUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `jsonrpcwire`

- **Properties:** 15
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/goa/jsonrpcwire-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/jsonrpcwire-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/jsonrpcwire-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/jsonrpcwire-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| an empty-string id remains an id in the encoded request | `TestJsonrpcwireSuccessMarshalProperty` |
| a large numeric id round-trips as a JSON number, not float64 | `TestJsonrpcwireSuccessMarshalRandom` |
| only string and JSON-number ids convert; other types error | `TestJsonrpcwireErrorMarshalProperty` |
| params must be object or array when present | `TestJsonrpcwireDefaultErrorMessagesRandom` |
| missing vs empty vs null method are distinguishable | `TestJsonrpcwireNotificationProperty` |
| exactly one array element is required; objects and other lengths fail | `TestJsonrpcwireIDToStringTable` |
| a designed error needs a name and a body; otherwise ok is false | `TestJsonrpcwireIDToStringRandom` |
| envelope shape, null-id exceptions, and id matching | `TestJsonrpcwireSinglePositionalParamTable` |
| envelope shape, null-id exceptions, and id matching | `TestJsonrpcwireSinglePositionalRandom` |
| envelope shape, null-id exceptions, and id matching | `TestJsonrpcwireDecodeServiceErrorDataTable` |
| envelope shape, null-id exceptions, and id matching | `TestJsonrpcwireDecodeServiceErrorRandom` |
| envelope shape, null-id exceptions, and id matching | `TestJsonrpcwireRawRequestAdversarial` |
| envelope shape, null-id exceptions, and id matching | `TestJsonrpcwireRawRequestRandom` |
| envelope shape, null-id exceptions, and id matching | `TestJsonrpcwireRawResponseValidateTable` |
| envelope shape, null-id exceptions, and id matching | `TestJsonrpcwireValidateRandom` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `pkgvalidation`

- **Properties:** 7
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/goa/pkgvalidation-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/pkgvalidation-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/pkgvalidation-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/pkgvalidation-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| valid and invalid examples for every supported format, including UUID renderings and IPv4/IPv6 cross-rejection | `TestPkgValidationFormatTableProperty` |
| a matching value passes; a non-matching value is an invalid-pattern error | `TestPkgValidationUUIDRenderingsProperty` |
| a matching value passes; a non-matching value is an invalid-pattern error | `TestPkgValidationFormatRandomProperty` |
| a matching value passes; a non-matching value is an invalid-pattern error | `TestPkgValidationIPCrossRejectRandomProperty` |
| a matching value passes; a non-matching value is an invalid-pattern error | `TestPkgValidationPatternProperty` |
| a matching value passes; a non-matching value is an invalid-pattern error | `TestPkgValidationPatternRandomProperty` |
| a matching value passes; a non-matching value is an invalid-pattern error | `TestPkgValidationUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `xrayseg`

- **Properties:** 11
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/goa/xrayseg-L0`
- **L2:** `experiments/pipeline/tasks_composerver/goa/xrayseg-L2`
- **L5:** `experiments/pipeline/tasks_composerver/goa/xrayseg-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/goa/xrayseg-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| a child has a 16-hex id, copied name/parent/trace, and is in progress with a start time no earlier than the parent | `TestXraysegNewSegmentProperty` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegStartTimeRandom` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegNewSubsegmentProperty` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegSubsegmentRandom` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegRecordErrorProperty` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegRecordErrorRandom` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegCaptureProperty` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegSubmitInProgressProperty` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegSubmitRandom` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegUDPHeaderProperty` |
| the second in-progress submit is ignored; close sends a finished document that replaces the pending one | `TestXraysegAnnotationProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

