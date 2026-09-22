# Details — ioutil

1. NonEmptyReader peeks at one byte — an empty reader yields
   ErrEmptyReader, a non-empty reader comes back as a ReadPeeker with
   the peeked byte still unread. Inferable: doc.
2. The context writer runs each write on a goroutine over a pooled
   buffer — on cancellation it WAITS for the in-flight write before
   returning ctx.Err(), so the underlying writer is quiescent when Write
   returns. Inferable: doc — the comment spells out the guarantee.
3. A panic in the underlying writer/read surfaces as a normal error —
   "underlying writer resulted in panic" — not a crash. Inferable: no.
4. The context reader reads through a pooled window and copies out —
   caller memory is never touched by a still-blocked goroutine, and the
   pool slice is released only after the main goroutine finishes
   copying. Inferable: no — buffer ownership is internal.
5. On ctx.Done the context reader closes its closer (when one was
   attached) before returning — a cancellation tears down the source.
   Inferable: partially.
6. ReadFinished answers true only when the read was NOT ended by this
   context's cancellation — a foreign DeadlineExceeded or a finished
   read under a now-dead context still means finished, and both ctx.Err()
   != nil AND a cancellation error are required to say no. Inferable:
   doc — the comment walks the whole truth table.
7. CheckClose writes the Close error only into a nil *err — an existing
   error wins. Inferable: doc.
8. The OnError wrappers invoke notify on any non-EOF error — EOF is
   silent. Inferable: doc.
9. NewReaderUsingReaderAt gives a stateful sequential reader over a
   ReaderAt — each Read advances an internal offset. Inferable:
   partially.
10. The Closer combiners chain both closes and surface the first error;
    the WithCloser variant runs the function after the underlying
    close. Inferable: partially.
11. CopyBufferPool draws its buffer from a pool — it does not allocate a
    fresh 32KiB per call. Inferable: no.
12. WriteNopCloser and writeNopCloser.Close are genuine no-ops — Close
    succeeds without touching the writer. Inferable: doc.
