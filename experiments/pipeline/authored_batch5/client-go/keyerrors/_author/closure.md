# Closure — keyerrors

Package: `error`. File: `error/error.go` (336 lines, partial).

Removed (all bodies stubbed): every `Error()` method (12 types),
`IsErrKeyExist`, `IsErrWriteConflict`, `IsErrorUndetermined`,
`NewErrWriteConflictWithArgs`, `NewErrPDServerTimeout`, `ExtractKeyErr`.

Kept: all error type definitions, sentinel vars (`ErrNotExist`,
`ErrResultUndetermined`, ...), `IsErrNotFound` (owned elsewhere),
`Log`, `MismatchClusterID`.

Tests edited: none — the package has no dedicated test file; failures
surface through callers.
