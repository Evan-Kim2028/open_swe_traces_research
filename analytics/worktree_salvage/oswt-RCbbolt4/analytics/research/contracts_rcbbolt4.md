# Contracts for authored_batch4/bbolt (worktree RCbbolt4)

2026-09-21. Wrote `_author/contract.md` for the 8 verified bbolt units copied from
`oswt-VFbatch4bbolt`. Each unit already shipped `gold.patch`, `bugreport.md`,
`DETAILS.md`, and a hidden `TestDetailNN` suite under `tests/hidden/`; only the
contract was missing, which blocked staging at every rung.

Shape follows `authored_batch3/kops`: one prose paragraph of behavioural
commitments, then a coverage table pairing every hidden test with the sentence
that justifies it. Sources of truth: the hidden suite (asserted behaviour),
`gold.patch` (the mechanism the suite expects), `DETAILS.md` (author's own
inferable/not-inferable annotation), `api.md`/`closure.md` (excised surface).

## Per-unit accounting

| unit | commitments in prose | coverage rows | hidden assertions | unjustified |
|---|---|---|---|---|
| nodemut | 11 | 11 | 11 | 0 |
| nodespill | 8 | 8 | 8 | 0 |
| nodewire | 12 | 12 | 12 | 0 |
| txbucket | 8 | 8 | 8 | 0 |
| txlife | 14 (12 tested + 2 prose-only) | 12 | 12 | 0 |
| txpage | 9 | 9 | 9 | 0 |
| txstats | 10 | 10 | 10 | 0 |
| txwrite | 12 | 12 | 12 | 0 |

Every hidden `TestDetailNN` maps to exactly one contract commitment and every
coverage row names a real hidden test — verified mechanically with the
pipeline's own regexes (`COVERAGE_ROW_RE`, `_SENTENCE_RE`, `_TEST_FUNC_RE`).

## Shape-vs-literal decisions

Commitments whose literal the suite deliberately leaves unpinned are stated as
shape only:

- **nodewire `minKeys`**: suite asserts leaf >= 1 and branch > leaf. Contract
  says "both positive, branch strictly greater" — not the 1/2 literals.
- **txlife `Tx.ID`**: suite asserts a negative sentinel on nil receiver/meta.
  Contract says "a negative sentinel" — not `-1`.
- **Panic messages**: suites assert panics, never text. Contracts say "panics"
  throughout; no message wording is stated.

Boundaries the suite pins exactly are stated exactly, because omitting them
would ambush the solver:

- **nodewire write**: panic at `>= 0xFFFF` inodes (the uint16 count boundary).
- **txwrite write**: chunk cap `common.MaxAllocSize - 1`; page span
  `(Overflow()+1) * pageSize` at `Id() * pageSize`.
- **txwrite commitFreelist**: allocation run is `EstimatedWritePageSize /
  pageSize + 1` pages (suite asserts the page's overflow equals
  `est / pageSize`).
- **txlife close**: merged `FreeAlloc` is `(FreeCount + PendingCount) *
  pageSize` — bytes, not pages.

## Flagged

- **txlife** carries two prose-only commitments with no dedicated assertion:
  `Writable` reports the begin-time flag and `DB` returns the owning database.
  Both are excised stubs the solver must restore; both are trivially derivable
  from the doc comments on the stubs. Stated for completeness of the excised
  surface; no hidden test grades them.
- No assertion in any of the 8 suites lacks a derivable commitment. Nothing was
  papered over.

## Caveat

`excision_recover.py` rebuilds a unit's excision by stripping the tests named
in the coverage table from the upstream tree. These contracts name the hidden
`TestDetailNN` suite (per the batch4 spec), not deleted in-tree tests — for
these units the removed in-tree tests live in files like `node_test.go`
(`api.md` "In-tree tests removed"), whose original function names the table
does not carry. Excision recovery on a batch4 unit would report the
`TestDetailNN` names as `missing_tests`. The committed `_author/excised/`
artifacts make recovery unnecessary today, but if batch4 units are ever fed
through `excision_recover`, that is the interaction to revisit.
