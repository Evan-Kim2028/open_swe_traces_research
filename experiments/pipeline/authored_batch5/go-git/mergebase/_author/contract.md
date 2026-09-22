# Contract — mergebase

`Commit.MergeBase`, `Commit.IsAncestor`, and `Independents` implement
`git merge-base` / `--is-ancestor` / `--independent` semantics over the
commit graph. Every commitment below is covered by a hidden test; every
hidden test maps to a commitment.

## Commitments

1. **Newest-first indexing.** Commits are sorted committer-date descending
   — the newer history is indexed, the older walked. The ordering is an
   optimisation; `MergeBase` results are identical in either argument
   order. Covered by `TestDetail01`.
2. **Ancestor short-circuit.** When the older commit is reachable from the
   newer's history the answer is exactly the older commit — a
   single-element result. Covered by `TestDetail02`.
3. **Reachability abort.** The ancestor index reports reachability via the
   `errIsReachable` sentinel — the walk aborts on discovery rather than
   finishing. Covered by `TestDetail03`.
4. **Candidate intersection.** Merge-base candidates are the older
   commit's ancestors also present in the newer's ancestor index; the
   index gates both the visit and the descent. Covered by `TestDetail04`.
5. **Independents eviction.** A candidate reachable from another candidate
   is dropped; divergent tips are all kept. Covered by `TestDetail05`.
6. **Hash dedup first.** Duplicates are removed by hash before any
   walking — identical input hashes never evict each other. Covered by
   `TestDetail06`.
7. **Bounded eviction walks (shape).** Per-round walks are seen-set
   limited; asserted shape is that a shared-history DAG returns exactly
   the mutually-unreachable tips. Covered by `TestDetail07`.
8. **IsAncestor.** Preorder walk stopping at first hash match; equality is
   ancestry — a commit is its own ancestor. Covered by `TestDetail08`.
9. **Order-preserving helpers (shape).** `remove`/`removeDuplicated`
   preserve input order and match by hash equality, not pointer identity.
   Covered by `TestDetail09`.
10. **Traversal errors.** An untraversable indexed history propagates the
    walker error rather than returning a partial answer. Covered by
    `TestDetail10`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | partially |
| TestDetail07 | 7 | no — shape only |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | no — shape only |
| TestDetail10 | 10 | partially |
