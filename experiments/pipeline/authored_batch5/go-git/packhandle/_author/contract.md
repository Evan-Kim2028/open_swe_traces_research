# Contract — packhandle

`internal/packhandle.PackHandle` — refcounted, lazily-opened access to
one pack triple (.pack/.idx/.rev): construction preconditions, size and
meta caching, cursor lifecycle and the idle-grace timer, index
construction, and pooled-vs-unpooled descriptor ownership. Every
commitment below is covered by a hidden test; every hidden test maps to
a commitment.

## Commitments

1. **Construction preconditions.** `New` returns
   `ErrPackSourceRequired` when `Sources.Pack.Open` or `.Size` is nil,
   and `ErrInvalidPackHash` for a zero pack hash. Covered by
   `TestDetail01`.
2. **Size caching.** The pack size is consulted once per handle —
   two cursors share one `Size` call — and a failed `Size` is not
   cached, so the next operation retries it. Covered by
   `TestDetail02`.
3. **Cursor pinning.** While a cursor is live the shared pack FD
   survives past the documented ~1s grace window; once the last cursor
   releases, the timer closes it. Covered by `TestDetail03`.
4. **Closed-handle surface.** After `Close`, `OpenPackReader`,
   `OpenRandomReader`, `Meta` and `Index` all answer `fs.ErrClosed`.
   Covered by `TestDetail04`.
5. **Meta validation.** `Meta` parses the 12-byte header (magic
   `PACK`, version 2 or 3) and verifies the footer equals the pinned
   pack hash — corrupt magic, an unsupported version, or a footer
   mismatch all error. Covered by `TestDetail05`.
6. **Meta caching.** A second `Meta` performs no new pack reads; a
   failed parse is not cached and retries on the next call. Covered
   by `TestDetail06`.
7. **Idempotent Close.** `Close` surfaces a failing underlying close,
   marks the handle closed regardless, and never re-runs the close
   body — the underlying file sees exactly one close. Covered by
   `TestDetail07`.
8. **Idle-descriptor release.** `CloseIdleDescriptors` closes the pack
   FD without closing the handle: new cursors reopen it on demand, and
   the cached `Meta`/`Index` results survive. Covered by
   `TestDetail08`.
9. **EOF normalisation (shape).** A `Read` spanning the pack tail
   returns the short data with nil error; `io.EOF` appears only at the
   exact end offset. Covered by `TestDetail09`.
10. **Seek validation.** Unknown whence answers
    `ErrInvalidSeekWhence`; a negative absolute position —
    however arrived at — answers `ErrNegativeSeekPosition`;
    `SeekEnd` resolves against the cached size. Covered by
    `TestDetail10`.
11. **Cursor Close.** A second `Close` on a cursor is a harmless no-op;
    reads on a closed cursor answer `fs.ErrClosed`. Covered by
    `TestDetail11`.
12. **Index sources.** `Index` returns `ErrSourceUnconfigured` when
    the idx or rev source is absent, and a usable index when both are
    configured. Covered by `TestDetail12`.
13. **Pool ownership.** With a pool the grace timer is inert — the FD
    stays open past the grace window until the pool or `Close` owns
    its lifetime. Covered by `TestDetail13`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | doc |
| TestDetail04 | 4 | doc |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | doc |
| TestDetail07 | 7 | partially — memoized result accepted either way |
| TestDetail08 | 8 | doc |
| TestDetail09 | 9 | no — observable EOF shape |
| TestDetail10 | 10 | partially |
| TestDetail11 | 11 | partially |
| TestDetail12 | 12 | doc |
| TestDetail13 | 13 | doc |
