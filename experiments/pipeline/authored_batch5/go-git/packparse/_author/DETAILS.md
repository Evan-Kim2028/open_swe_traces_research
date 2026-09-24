# Details — packparse

1. A Parser is single-shot: a second `Parse` call returns the consumed error
   without doing any work — even if the first call failed. Inferable: doc —
   the type comment states it.
2. Deltas seen during the scan are QUEUED, not resolved in place: resolution
   starts only after the whole pack has been scanned. Inferable: partially —
   single-pass callers would naturally interleave; deferral is the detail.
3. Resolution walks the delta DAG depth-first from each non-delta base, and
   at every parent advances REF-delta AND OFS-delta children together —
   never a global REF pass followed by a global OFS pass. Inferable: doc —
   the resolveDeltas doc explains why splitting passes is wrong.
4. A REF-delta whose base hash is not in the pack gets a placeholder parent
   flagged as an external reference — resolution proceeds as a thin pack
   rather than failing. Inferable: doc — named in the resolveDeltas comment.
5. An OFS-delta whose recorded base offset matches no in-pack object is
   rejected as malformed — there is no external fallback for offsets.
   Inferable: doc.
6. Delta chain depth is capped at the kept `maxDeltaChainDepth` bound, and
   per-header depth is cached on the entry so the check is linear — a chain
   crossing an already-measured parent reuses its count. Inferable: doc —
   both comments are kept.
7. Low-memory mode engages only when the storage implements the
   `LowMemoryCapable` probe AND the underlying source is seekable — either
   failing disables it silently. Inferable: partially — the interface is
   visible, the seeker coupling is not.
8. In low-memory mode a resolved delta's content buffer is returned to the
   pool, and the release walks UP the parent chain freeing each still-held
   parent buffer. Inferable: no.
9. A source that hits end-of-input having produced zero objects reports the
   empty-packfile sentinel, not the raw EOF. Inferable: partially — the
   sentinel var is visible elsewhere in the package.
10. The grow hint for staging buffers is clamped to the kept 1 GiB bound —
    a malformed declared size cannot drive a huge preallocation. Inferable:
    doc — the const comment states it.
11. When a delta already has a recorded object ID, its declared type/size/hash
    are left alone after patching — they are only filled in when the hash was
    zero. Inferable: no.
12. An external-ref parent's real type and size are REWRITTEN from the object
    fetched out of storage before its content is used as a patch base.
    Inferable: partially.
13. The cache's up-front reservation is capped by the kept
    `maxObjectsPrealloc` bound even when the pack header advertises more —
    growth beyond the hint is organic. Inferable: doc.
14. Only delta payloads are written to storage by the parser itself —
    non-delta objects are already stored by the scanning layer it drives.
    Inferable: no.
15. Observer callbacks fire in section order — header count, per-object
    header, per-object content, footer hash — and the first observer error
    aborts the parse. Inferable: partially — the Observer interface is
    visible, the propagation is the detail.
