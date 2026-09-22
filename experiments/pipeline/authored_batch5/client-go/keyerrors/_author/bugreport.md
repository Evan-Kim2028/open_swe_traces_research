# Bug report

The `error` package's KeyError mapping and error-type surfaces panic:
`ExtractKeyErr`, the `Error()` methods, the `IsErr*`/`IsErrorUndetermined`
predicates and `NewErrWriteConflictWithArgs` are stubbed.

Expected: `ExtractKeyErr(&kvrpcpb.KeyError{Conflict: wc})` returns an
`ErrWriteConflict` (`IsErrWriteConflict` true, even wrapped); a KeyError
with only `Retryable` set returns `ErrRetryable` carrying that string; a
KeyError with nothing set returns an `"unexpected KeyError: ..."` error.
`(&ErrTxnTooLarge{Size: 3}).Error()` = `"txn too large, size: 3."`.
`NewErrWriteConflictWithArgs(1, 2, 3, []byte("k"), reason)` builds an
`ErrWriteConflict` whose proto fields match the args.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
