# Details — keyerrors

1. `ExtractKeyErr` checks KeyError fields in a FIXED priority: Conflict ->
   `ErrWriteConflict` (wrapped with stack), Retryable -> `ErrRetryable`,
   AssertionFailed -> `ErrAssertionFailed` (unwrapped), Abort ->
   `errors.Errorf`, CommitTsTooLarge -> `Errorf`, TxnNotFound -> `Errorf`,
   else `"unexpected KeyError"`. Order matters when several fields are set.
   Inferable: no.
2. A `mockRetryableErrorResp` failpoint can rewrite the KeyError before
   dispatch (clears Conflict, sets Retryable). Inferable: partially —
   failpoint wiring is visible, its priority is not.
3. `IsErrKeyExist`/`IsErrWriteConflict`/`IsErrorUndetermined` use
   `errors.As`/`errors.Is` so WRAPPED errors match. Inferable: yes.
4. `Error()` strings embed the wrapped proto (`write conflict { %s }`,
   `assertion failed { %s }`) or fixed layouts (`txn too large, size:
   %v.`). Inferable: no — only the shape is committed.
5. `NewErrWriteConflictWithArgs` packs StartTs/ConflictTs/Key/
   ConflictCommitTs/Reason into a `kvrpcpb.WriteConflict` then wraps it.
   Inferable: yes.
6. `ErrRetryable.Error` returns the raw `Retryable` string;
   `ErrPDServerTimeout.Error` returns its stored msg;
   `NewErrPDServerTimeout` returns `error` (not the concrete type).
   Inferable: partially.
