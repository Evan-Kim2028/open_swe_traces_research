# Exported API — ioutil

Package `utils/ioutil` — reader/writer plumbing helpers across
`common.go`, `context.go`, `sync.go`: `NonEmptyReader`, the
`New{Read,Write}Closer{,WithCloser,OnError}` combiners, `WriteNopCloser`,
`NewReaderUsingReaderAt`, `CheckClose`, the `OnError` notifiers,
`CloserFunc`, the context-aware `NewContext{Reader,Writer,ReadCloser,
WriteCloser,ReaderWithCloser}`, `ReadFinished`, `CopyBufferPool`.

Kept visible: all wrapper struct types, `ErrEmptyReader`, `Writer`/
`Reader`/`ReadPeeker` interfaces, the long contract comments on
cancellation semantics and buffer pooling.

Callers: transport, packfile, object readers everywhere. In-tree tests
removed: 2 (`common_test.go`, `context_test.go`).
