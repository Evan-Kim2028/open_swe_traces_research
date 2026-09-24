# Contract — firesources

`Resource` implementations and the streaming comparison helpers. Every
commitment below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **Streaming compare.** `ResourcesMatch` compares the full byte stream —
   equal iff every byte matches; different lengths are `false`, not an error;
   open and read errors propagate. Covered by `TestDetail01` (including a
   multi-chunk >8192-byte case).
2. **Unwrapped not-exist.** `FileResource.Open`, `VFSResource.Open`, and
   `CopyResource` return not-exist errors unwrapped, so `os.IsNotExist` holds
   directly on the returned error. Covered by `TestDetail02`.
3. **`BytesResource.MarshalJSON` (shape).** Marshals to a JSON string; the
   byte-to-string encoding choice is an implementation detail and is not
   pinned. Covered by `TestDetail03`.
4. **`TaskDependentResource`.** `Open` errors while `Resource` is nil and
   delegates once set; `IsReady` is exactly `Resource != nil`. Covered by
   `TestDetail04`.
5. **Singleton dependency.** `GetDependencies` returns exactly the producing
   task. Covered by `TestDetail05`.
6. **Memoized function resource.** `FunctionToResource`'s `fn` runs on first
   `Open` only; its non-nil result is cached and served on later calls. (The
   nil-result re-run edge is an implementation detail, not asserted.)
   Covered by `TestDetail06`.
7. **In-memory resources never fail.** `StringResource.Open` and
   `BytesResource.Open` never return an error and yield their contents.
   Covered by `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | partially — unwrap rule asserted via os.IsNotExist |
| TestDetail03 | 3 | no — shape only (valid JSON string) |
| TestDetail04 | 4 | yes |
| TestDetail05 | 5 | yes |
| TestDetail06 | 6 | partially — memoization asserted; nil edge not pinned |
| TestDetail07 | 7 | yes |
