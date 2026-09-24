# Details — objfile

1. The stream is zlib; inside it the header is `<type> SP <size> NUL` and the object body
   follows immediately. Inferable: doc — the bound const's comment names the parts.
2. The 32-byte bound is a SHARED budget across the whole header (type + both delimiters +
   size digits) — a 20-digit size alone can blow it even though the type field is short.
   Inferable: partially.
3. An unparseable type name fails before the size field is ever read; a non-numeric size is
   a generic header failure. Inferable: partially.
4. A NEGATIVE numeric size in the stream is accepted by the reader without complaint — only
   the writer refuses negatives. Inferable: no.
5. Hitting the byte budget mid-field and hitting end-of-stream mid-header are DIFFERENT
   failures (too-long vs invalid header). Inferable: partially.
6. Reading before a successful header call fails with a dedicated error; hashing before it
   returns a zero hash sized to the configured object format — NOT a nil/absent hash.
   Inferable: no.
7. The object hash covers the canonical `<type> <size>\0` header plus the body even though
   only the body bytes flow through the read/write path — the hash is seeded with the
   declared type and size. Inferable: partially.
8. Header write rejects an invalid object type before emitting any byte, refuses a negative
   size, and refuses a header that would exceed the bound. Inferable: doc — the method's
   doc comment enumerates the three.
9. Writing more bytes than the declared size TRUNCATES the input to what fits, commits those
   bytes, and still reports the overflow — a partial write is not rolled back. Inferable: no.
10. A further write after the declared size is exhausted commits zero bytes and reports
    overflow — it is not silently ignored. Inferable: no.
11. The writer's data write does NOT guard against being called before its header — it
    dereferences a nil pipeline and crashes, unlike the reader which returns a dedicated
    error. Inferable: no.
12. Writer close is idempotent and returns the SAME error on every repeat call, cached from
    the first close; neither direction closes the wrapped stream. Inferable: doc.
