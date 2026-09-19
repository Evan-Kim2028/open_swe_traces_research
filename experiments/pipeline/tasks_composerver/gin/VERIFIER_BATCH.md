# VERIFIER_BATCH.md — composerver gin

Composer verifier batch for gin/httprouter feature-excision units.
Seed `20260919`. Hidden black-box property suites via `affordance.py`.
Dockerfile `FROM ladder-base:gin`. L0/L2 proved; L5/L6 packaged only.

Build / reprove:

```
uv run python scripts/build_composerver_gin_batch.py --max-parallel 2
```

## Summary

| unit | properties | coverage % | gold 1st | suite fixes | wall min | verdict |
|---|---:|---:|---|---:|---:|---|
| `formmapping` | 11 | 84.6 | yes | 0 | 0.1 | PASS |
| `bodydecoders` | 9 | 90.0 | yes | 0 | 0.1 | PASS |
| `jsonrenders` | 9 | 100.0 | yes | 0 | 0.1 | PASS |
| `validator` | 13 | 100.0 | yes | 1 | 0.1 | PASS |
| `defaultengine` | 9 | 100.0 | yes | 0 | 0.2 | PASS |
| `multipartfiles` | 10 | 100.0 | yes | 1 | 0.1 | PASS |
| `requestbinders` | 13 | 100.0 | yes | 1 | 0.1 | PASS |
| `htmlrender` | 10 | 100.0 | yes | 0 | 0.1 | PASS |
| `streamrenders` | 8 | 100.0 | yes | 0 | 0.1 | PASS |
| `bindingdispatch` | 9 | 100.0 | yes | 0 | 0.1 | PASS |

**Batch totals:** 10/10 PASS

**Harness notes:** `ladder-base:gin` has no `patch` binary; proofs use `git apply -p1`. Three binding suites needed inline `bbSeed`/`bbCases` (had referenced missing shared const file).

---

## `formmapping`

- **Properties:** 11
- **Contract coverage:** 84.6%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/gin/formmapping-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/formmapping-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/formmapping-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/formmapping-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| every scalar kind (ints, uints, floats, bool, string) is set from a string value | `TestFMContractTableProperty` |
| absent key or empty first value falls back to the `default=` option | `TestFMCollectionFormatProperty` |
| `-` tags and unexported fields are skipped | `TestFMTimeDurationProperty` |
| slices resize to values; arrays must match length exactly | `TestFMStructMapProperty` |
| csv/ssv/tsv/pipes separators split values; unknown format errors | `TestFMPtrCircularProperty` |
| semicolon defaults split per collection format | `TestFMCustomUnmarshalProperty` |
| time_format/utc/location/unix variants and duration parsing | `TestFMBindersProperty` |
| struct and map fields decode from JSON text | `TestFMMapTargetsProperty` |
| pointers allocated only when set; circular types don't recurse forever | `TestFMAdversarialProperty` |
| UnmarshalParam and parser=encoding.TextUnmarshaler custom types win over built-ins | `TestFMScalarsRandomProperty` |
| tag name selects which key namespace is read | `TestFMUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `bodydecoders`

- **Properties:** 9
- **Contract coverage:** 90.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/gin/bodydecoders-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/bodydecoders-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/bodydecoders-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/bodydecoders-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| JSON decodes objects and top-level slices; nil request errors | `TestBDContractTableProperty` |
| UseNumber flag changes number decoding | `TestBDBindBodyProperty` |
| DisallowUnknownFields flag rejects unknown keys | `TestBDJSONUseNumberProperty` |
| BindBody works from byte slices without a request | `TestBDJSONDisallowUnknownProperty` |
| decode goes through the pluggable codec API | `TestBDFormatDecodeValidateProperty` |
| XML decodes then validates | `TestBDProtoBufProperty` |
| YAML and TOML decode then validate | `TestBDBSONProperty` |
| protobuf requires proto.Message and skips validation | `TestBDPlainAdversarialProperty` |
| BSON round-trips then validates | `TestBDJSONUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `jsonrenders`

- **Properties:** 9
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/gin/jsonrenders-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/jsonrenders-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/jsonrenders-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/jsonrenders-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| plain marshal+write with the right content type; marshal errors propagate | `TestJSONWriteJSONContentTypeProperty` |
| 4-space-indented output | `TestJSONMarshalErrorProperty` |
| prefix emitted only for top-level arrays | `TestIndentedJSONIndentProperty` |
| callback escaping, '(' json ');' wrapping, empty-callback passthrough, javascript content type | `TestSecureJSONPrefixProperty` |
| non-ASCII runes become \uXXXX escapes; charset-less content type | `TestJsonpJSONCallbackProperty` |
| encoder path with HTML escaping off and trailing newline | `TestJsonpJSONEmptyCallbackProperty` |
| encoder path with HTML escaping off and trailing newline | `TestAsciiJSONEscapeProperty` |
| encoder path with HTML escaping off and trailing newline | `TestPureJSONNoHTMLEscapeProperty` |
| encoder path with HTML escaping off and trailing newline | `TestJSONPreservesPresetContentTypeProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `validator`

- **Properties:** 13
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 1 (added per-file `bbSeed`/`bbCases`; had referenced missing `bb_const_test.go`)
- **L2:** `experiments/pipeline/tasks_composerver/gin/validator-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/validator-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/validator-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| non-struct kinds validate to nil without panic | `TestBBValidateNil` |
| pointers to non-struct values are re-validated by kind, not validated as structs | `TestBBValidateNilRandom` |
| struct values actually reach the engine | `TestBBValidatePrimitives` |
| Engine() returns the live engine for registering custom checks | `TestBBValidatePrimitivesRandom` |
| per-element errors aggregate with [i]: prefixes; all-pass returns nil | `TestBBValidatePointerToNonStruct` |
| per-element errors aggregate with [i]: prefixes; all-pass returns nil | `TestBBValidateStructRules` |
| per-element errors aggregate with [i]: prefixes; all-pass returns nil | `TestBBValidateStructRandom` |
| per-element errors aggregate with [i]: prefixes; all-pass returns nil | `TestBBValidateSlicePass` |
| per-element errors aggregate with [i]: prefixes; all-pass returns nil | `TestBBValidateSliceRandomPass` |
| per-element errors aggregate with [i]: prefixes; all-pass returns nil | `TestBBSliceValidationErrorFormat` |
| per-element errors aggregate with [i]: prefixes; all-pass returns nil | `TestBBValidateSliceFail` |
| per-element errors aggregate with [i]: prefixes; all-pass returns nil | `TestBBValidatorEngineCustom` |
| per-element errors aggregate with [i]: prefixes; all-pass returns nil | `TestBBValidatorEngineLazyInit` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `defaultengine`

- **Properties:** 9
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.2

- **L0:** `experiments/pipeline/tasks_composerver/gin/defaultengine-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/defaultengine-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/defaultengine-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/defaultengine-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| each method wrapper registers a working route on the shared engine | `TestDEContractTableProperty` |
| group and middleware wrappers affect the shared engine | `TestDEHTTPMethodsProperty` |
| fallback handlers apply to unmatched requests | `TestDESharedRoutesProperty` |
| routes registered through wrappers are listed | `TestDEGroupMiddlewareProperty` |
| template and static-file wrappers configure the same engine | `TestDENoRouteNoMethodProperty` |
| template and static-file wrappers configure the same engine | `TestDEStaticProperty` |
| template and static-file wrappers configure the same engine | `TestDEHTMLGlobProperty` |
| template and static-file wrappers configure the same engine | `TestDEAdversarialLazyInitProperty` |
| template and static-file wrappers configure the same engine | `TestDEUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `multipartfiles`

- **Properties:** 10
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/gin/multipartfiles-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/multipartfiles-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/multipartfiles-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/multipartfiles-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| single file lands on *multipart.FileHeader fields and file-name fields | `TestBBMultipartPointerFirstFile` |
| multiple files fill slice/array fields element-wise | `TestBBMultipartPointerRandom` |
| wrong field kind under a file key returns the exported error | `TestBBMultipartValueFirstFile` |
| wrong field kind under a file key returns the exported error | `TestBBMultipartSliceCount` |
| wrong field kind under a file key returns the exported error | `TestBBMultipartSliceRandom` |
| wrong field kind under a file key returns the exported error | `TestBBMultipartArrayExactLen` |
| wrong field kind under a file key returns the exported error | `TestBBMultipartArrayLenInvalid` |
| wrong field kind under a file key returns the exported error | `TestBBMultipartWrongTypeError` |
| wrong field kind under a file key returns the exported error | `TestBBMultipartWrongTypeRandom` |
| wrong field kind under a file key returns the exported error | `TestBBMultipartFormValueFallback` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `requestbinders`

- **Properties:** 13
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/gin/requestbinders-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/requestbinders-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/requestbinders-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/requestbinders-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| urlencoded+multipart binder maps parsed form incl. query values | `TestBBFormBinderQueryAndBody` |
| post-form binder reads body values only | `TestBBFormBinderToleratesNonMultipart` |
| query binder maps URL values and propagates conversion errors | `TestBBFormPostBodyOnly` |
| header binder maps canonicalized header keys via the header tag | `TestBBFormPostIgnoresQuery` |
| uri binder maps the params map via the uri tag, nested structs included | `TestBBQueryBinderURLOnly` |
| multipart binder binds files and values from the same request | `TestBBQueryConversionError` |
| multipart binder binds files and values from the same request | `TestBBHeaderBinderCanonical` |
| multipart binder binds files and values from the same request | `TestBBHeaderBinderCaseInsensitive` |
| multipart binder binds files and values from the same request | `TestBBUriBinderParams` |
| multipart binder binds files and values from the same request | `TestBBUriNestedStruct` |
| multipart binder binds files and values from the same request | `TestBBBinderNamesLowercase` |
| multipart binder binds files and values from the same request | `TestBBValidationAfterMapping` |
| multipart binder binds files and values from the same request | `TestBBValidationAfterMappingRandom` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `htmlrender`

- **Properties:** 10
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/gin/htmlrender-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/htmlrender-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/htmlrender-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/htmlrender-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| production instance executes the named template | `TestHTMLProductionNamedTemplateProperty` |
| nil template returns the not-configured error | `TestHTMLProductionEmptyNameProperty` |
| empty Name executes the template root | `TestHTMLNotConfiguredProperty` |
| debug renderer re-parses from files, glob, or FS+patterns per render | `TestHTMLDebugFilesProperty` |
| debug renderer with no source configured panics | `TestHTMLDebugGlobProperty` |
| parse/execute failures propagate | `TestHTMLDebugFSProperty` |
| parse/execute failures propagate | `TestHTMLDebugPanicProperty` |
| parse/execute failures propagate | `TestHTMLContentTypeProperty` |
| parse/execute failures propagate | `TestHTMLDelimsProperty` |
| parse/execute failures propagate | `TestHTMLExecuteErrorProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `streamrenders`

- **Properties:** 8
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/gin/streamrenders-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/streamrenders-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/streamrenders-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/streamrenders-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| stream copy with Content-Length only when >= 0 | `TestReaderContentLengthProperty` |
| bytes written with Content-Length only when non-empty | `TestReaderHeaderMergeProperty` |
| status-code validation panic outside 3xx/201 and normal redirect otherwise | `TestDataContentLengthProperty` |
| format-with-args vs literal format string | `TestRedirectStatusProperty` |
| write failures propagate | `TestStringFormatProperty` |
| write failures propagate | `TestStringLiteralPercentProperty` |
| write failures propagate | `TestStreamPreservesPresetContentTypeProperty` |
| write failures propagate | `TestStreamWriteErrorProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `bindingdispatch`

- **Properties:** 9
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/gin/bindingdispatch-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/bindingdispatch-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/bindingdispatch-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/bindingdispatch-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| every MIME row of the dispatch table returns its binder; GET overrides content type | `TestBBDefaultGetAlwaysForm` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultGetOverridesMIME` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultMIMETable` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultDualMIMEAliases` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultFormFallback` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultUnknownMIME` |
| with no validator installed the validate hook is a no-op | `TestBBValidationDisabledNilValidator` |
| with no validator installed the validate hook is a no-op | `TestBBValidationDisabledRandomBodies` |
| with no validator installed the validate hook is a no-op | `TestBBValidationEnabledRequiresField` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

