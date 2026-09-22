# Difficulty — packlookup

predicted_flip: L3
details: 12

Missed edges: resolver-owned descriptor semantics; once-init with
PackHash vs seek-end id; post-lock closed recheck; clean-end-of-scan
→ ErrObjectNotFound; FSObject laziness for non-deltas only; delta type
→ base type substitution; REFDelta-by-hash vs OFSDelta-by-offset base
resolution; missing-base error; FD probe + reopen on os.ErrClosed only;
ReadAt concurrency contract; delta-header type filtering in iteration;
ownership-aware idempotent Close.

Hardness driver: a straightforward "inflate at offset" reader works on
undeltified packs and non-filesystem storage — the lazy FSObject path,
delta base resolution and FD lifecycle only fail under realistic packs.
