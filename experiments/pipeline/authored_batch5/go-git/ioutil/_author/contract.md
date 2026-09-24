# Contract — ioutil

Hidden suite: `tests/hidden/utils/ioutil/ioutil_bb_test.go` (package `ioutil`, in-package).
One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | `NonEmptyReader` returns `ErrEmptyReader` on empty input, propagates a first-byte read error, and on non-empty input returns a `ReadPeeker` whose `Peek(1)` shows the first byte while a full `Read` still yields it. |
| TestDetail02 | 2 | doc | `NewContextWriter`'s `Write`, when the context is cancelled mid-write, does not return until the in-flight underlying write completes, then returns `ctx.Err()`. |
| TestDetail03 | 3 | no | SHAPE: a panic inside the underlying writer is converted to a non-nil error return; no panic escapes `Write` (message text not pinned). |
| TestDetail04 | 4 | no | SHAPE: after `Write`/`Read` returns `ctx.Err()`, bytes a still-blocked underlying read later produces never appear in the caller's buffer (buffer ownership internals not pinned). |
| TestDetail05 | 5 | partially | `NewContextReaderWithCloser` closes the attached closer by the time the cancelled `Read` returns `ctx.Err()`. |
| TestDetail06 | 6 | doc | `ReadFinished` truth table: false iff `ctx.Err() != nil` AND err is a cancellation error (`context.Canceled`/`DeadlineExceeded`); live context or non-cancellation error ⇒ true, including a nil error under a dead context. |
| TestDetail07 | 7 | doc | `CheckClose` assigns the `Close` error only when `*err` is nil; a prior error is preserved; a nil close error writes nothing. |
| TestDetail08 | 8 | doc | `NewReaderOnError`/`NewReadCloserOnError`/`NewWriterOnError`/`NewWriteCloserOnError` invoke notify exactly once on a non-EOF error and never on `io.EOF`. |
| TestDetail09 | 9 | partially | `NewReaderUsingReaderAt(r, 2)` reads sequentially "234","567","89" — internal offset advances per `Read` and honors the start offset; EOF terminates. |
| TestDetail10 | 10 | partially | `NewReadCloser`/`NewWriteCloser` run the attached closer and surface its error; `NewReadCloserWithCloser` runs the func strictly after the underlying `Close`, both run even on failure, and the first error is returned. |
| TestDetail11 | 11 | no | SHAPE: `CopyBufferPool` performs a complete correct copy (n = len, bytes equal, empty source ⇒ 0/nil); pool mechanics are unobservable and unpinned. |
| TestDetail12 | 12 | doc | `WriteNopCloser` returns a `WriteCloser` whose `Close` is a no-op returning nil (twice), while `Write` still reaches the underlying writer. |

Refusals/softening: line 3 asserted as error-not-panic shape, not the quoted
message; line 4 as observable buffer-safety shape (post-cancel writes never land
in caller memory), not pool internals; line 11 as copy correctness only — buffer
provenance is unobservable.
