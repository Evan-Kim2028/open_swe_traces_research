# Details — mergebase

1. Both commits are sorted committer-date DESCENDING first — the newer
   history is indexed, the older one walked — and this ordering is an
   optimisation, not semantics. Inferable: doc — the strategy comment
   spells it out.
2. If the older commit is reachable from the newer's history, the answer
   is exactly the older commit — a single-element result, no further
   filtering. Inferable: partially — the sentinel's doc comment describes
   the case, the direct return is the detail.
3. Reachability is detected by walking the newer commit's whole ancestor
   set and erroring the moment the older one appears — the walk aborts,
   it does not finish collecting. Inferable: partially.
4. Merge-base candidates are the older commit's ancestors that also sit
   in the newer's ancestor index — found via a filtered walk where the
   index gates BOTH the visit and the descent. Inferable: partially —
   the same filter as both args is the detail.
5. Independents drops any candidate reachable from another candidate —
   each round walks one candidate's history and evicts matches, stopping
   early when a single candidate remains. Inferable: partially — the
   function doc states the goal, the eviction loop is the detail.
6. Candidates are processed newest-first and duplicates are removed by
   hash BEFORE any walking — identical input hashes can never evict each
   other. Inferable: partially.
7. The per-round walk is bounded by a seen-set limiter — ancestors of an
   already-eliminated lineage are never re-traversed. Inferable: no —
   the limiter construction is internal.
8. IsAncestor is a preorder walk that stops at the first hash match —
   equality IS ancestry (a commit is its own ancestor). Inferable:
   partially — `merge --is-ancestor` semantics are in the doc comment.
9. Helper removal/dedup preserve input order and operate by hash
   equality — not pointer identity, not date. Inferable: no.
10. An untraversable history (missing objects) propagates the walker
    error rather than returning a partial answer. Inferable: partially.
