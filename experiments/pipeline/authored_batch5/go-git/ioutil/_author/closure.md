# Closure — ioutil

Package: `utils/ioutil`. Files: `common.go`, `context.go`, `sync.go`.

Removed (28 functions stubbed): `NonEmptyReader`, `readCloser.Close`,
`NewReadCloser`, `readCloserCloser.Close`, `NewReadCloserWithCloser`,
`writeCloser.Close`, `NewWriteCloser`, `writeNopCloser.Close`,
`WriteNopCloser`, `readerAtAsReader.Read`, `NewReaderUsingReaderAt`,
`CheckClose`, `NewContextWriteCloser`, `NewContextReadCloser`,
`NewReaderOnError`, `NewReadCloserOnError`, `readerOnError.Read`,
`NewWriterOnError`, `NewWriteCloserOnError`, `writerOnError.Write`,
`CloserFunc.Close`; `NewContextWriter`, `ctxWriter.Write`,
`NewContextReader`, `ReadFinished`, `ctxReader.Read`,
`NewContextReaderWithCloser`; `CopyBufferPool`.

Kept: all types (`ctxReader`, `ctxWriter`, `ioret`, the wrapper
structs), `ErrEmptyReader`, `Reader`/`Writer`/`ReadPeeker`/`CloserFunc`
declarations, every contract comment. `utils/sync` pools are consumed
intact.

Tests deleted: `common_test.go`, `context_test.go` (2 — the package's
only tests).
