# Details — packhandle

1. Construction refuses a missing Pack.Open or Pack.Size, and a zero
   pack hash — each with its own error var. Inferable: doc — the New
   comment names both preconditions.
2. The pack size is fetched once and cached in an atomic — failures are
   NOT cached, so a transient Size error retries next call. Inferable:
   doc — the packSize comment spells out the cache and the zero
   sentinel.
3. Every cursor holds one reference on the shared pack file — the FD
   outlives its cursor until Close releases it, and the grace timer can
   only fire with zero live cursors. Inferable: doc — the lifecycle
   contract is in the type comment.
4. A closed handle answers fs.ErrClosed from open paths, Meta and
   Index — the closed flag is checked before any FD work. Inferable:
   doc — documented on each method.
5. Meta parses and VALIDATES: magic PACK, version 2 or 3, and the footer
   must equal the pinned pack hash — a well-formed pack with a wrong
   footer still errors. Inferable: partially — the PackMeta comment
   covers validation, the footer-pin is in New's comment.
6. Meta caches its first success; parse failures retry. Inferable:
   doc — "first successful call is cached" is kept.
7. Close is idempotent via a once-wrapper, sets closed BEFORE releasing
   FDs, and joins the pack and index errors. Inferable: partially —
   the doClose comment explains the ordering, errors.Join is visible.
8. CloseIdleDescriptors releases FDs without closing the handle and
   keeps the meta/index caches — active readers finish normally.
   Inferable: doc — the method comment describes it exactly.
9. Cursor Read converts a short trailing ReadAt EOF into a clean read —
   data returned with err=nil, EOF only when offset reaches size.
   Inferable: no — the EOF-normalisation is internal.
10. Seek rejects unknown whence and negative absolute positions with
    distinct error vars; SeekEnd resolves against the cached size.
    Inferable: partially — both error vars are kept.
11. Cursor Close releases its reference exactly once — a second Close
    is a no-op. Inferable: partially.
12. Index builds a LazyIndex over the idx/rev sources and refuses when
    either source is absent — with ErrSourceUnconfigured, not a nil
    pointer crash. Inferable: doc — the Sources comment says so.
13. With a pool, the grace timer is inert — the pool's LRU owns FD
    lifetime; without one, the 1s grace governs. Inferable: doc — the
    NewWithPool comment covers the split.
