# Contract (L2) — keyerrors

`ExtractKeyErr` checks `KeyError` fields in a fixed priority order —
conflict first, then retryable, assertion failure, abort, commit-ts too
large, txn not found — so when several fields are set the earliest wins,
and an unset error produces a generic unexpected-key-error value. A
failpoint can rewrite the `KeyError` before dispatch, changing which
branch produces the returned error. `IsErrKeyExist`, `IsErrWriteConflict`,
and `IsErrorUndetermined` match through wrapped errors. The error strings
embed the wrapped proto or a fixed layout — the shape is committed, not
the exact text. `NewErrWriteConflictWithArgs` packs its arguments into a
`WriteConflict` proto and wraps it. `ErrRetryable.Error` returns the
stored retryable string; `ErrPDServerTimeout.Error` returns its stored
message; `NewErrPDServerTimeout` is typed as `error`, not the concrete
type.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | `ExtractKeyErr` dispatches on `KeyError` fields in a fixed priority order |
| `TestDetail02` | the retryable-error failpoint rewrites the `KeyError` before dispatch |
| `TestDetail03` | the `Is*` predicates match through wrapped errors |
| `TestDetail04` | error strings embed the wrapped proto or a fixed layout (shape only) |
| `TestDetail05` | `NewErrWriteConflictWithArgs` packs its args into a `WriteConflict` proto and wraps it |
| `TestDetail06` | `ErrRetryable.Error`/`ErrPDServerTimeout.Error` return their stored strings; the constructor is `error`-typed |
