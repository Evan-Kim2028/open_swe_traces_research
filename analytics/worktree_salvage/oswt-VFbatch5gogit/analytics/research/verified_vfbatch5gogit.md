# Verified: authored_batch5 go-git hidden suites

2026-09-21. All 20 authored go-git units now carry a hidden
`tests/hidden/**_bb_test.go`, a `_author/contract.md`, and a generated
`tests/test.sh`. Every unit passes all four Docker gates:

`EXCISED=FAIL GOLD=PASS CHEAT=FAIL A12=PASS` — the excised tree fails,
the gold patch passes, the cheat patch fails, and `gold.patch` modifies
no test file. Log: `outputs/VFbatch5gogit.log`.

Method: assertions were written from each unit's `DETAILS.md`, the
excised stub's doc comments, kept production code, and public upstream
go-git source as the semantic reference. `gold.patch` was never opened
or used as a specification. `Inferable: no` lines were asserted as
shape (error presence/kind, bounds, alignment, state transitions), never
literals.

## Results

| Unit | Tests | Inferable (doc / partially / no) | Gates |
|---|---|---|---|
| ioutil | 12 | 5 / 2 / 3 | all pass |
| fetchmsg | 16 | 4 / 5 / 5 | all pass |
| packenc | 14 | 3 / 7 / 3 | all pass |
| packparse | 15 | 6 / 5 / 3 | all pass |
| cgfile | 14 | 3 / 7 / 3 | all pass |
| deltadiff | 13 | 2 / 9 / 1 | all pass |
| deltasel | 14 | 3 / 7 / 3 | all pass |
| renamedet | 14 | 3 / 5 / 4 | all pass |
| archive | 12 | 4 / 5 / 1 | all pass |
| commitwalk | 11 | 2 / 7 / 1 | all pass |
| revwalk | 12 | 4 / 6 / 1 | all pass |
| mergebase | 10 | 1 / 6 / 2 | all pass |
| sigcodec | 12 | 1 / 5 / 4 | all pass |
| fsnode | 12 | 4 / 6 / 1 | all pass |
| idxindex | 14 | 2 / 4 / 3 | all pass |
| memstor | 12 | 3 / 7 / 1 | all pass |
| negotiate | 12 | 0 / 8 / 4 | all pass |
| objpatch | 14 | 4 / 7 / 3 | all pass |
| packhandle | 13 | 5 / 4 / 1 | all pass |
| packlookup | 12 | 3 / 6 / 1 | all pass |

258 `TestDetailNN` tests; ~228 DETAILS rows carry an explicit Inferable
tag (the tag occasionally wraps onto the next line in the markdown).
Every test maps to a DETAILS row; every row's asserted commitment is
paired in the unit's `contract.md` coverage table.

## Notable per-unit findings

**Where a DETAILS line over-claimed, the test was corrected to the
observable commitment and the deviation recorded here:**

- **sigcodec D11** — DETAILS said undecodable objects surface errors
  through `Next`. Probing showed the real split: an *unknown object
  type* is silently skipped, while *malformed content on a known type*
  errors. The test asserts the latter (corrupt commit body).
- **idxindex D9** — the count-mismatch clause describes behaviour the
  reference does not implement: gold tolerates every announced/actual
  count combination. Refused that clause; asserted the documented
  `Index()`-before-`OnFooter` refusal and `Finished()` gating.
  **D8** — `FindHash` requires `.rev`; construction fails without it
  (no idx-side fallback). Asserted constructor propagation.
- **commitwalk D2/D5** — the heap does not preserve insertion order for
  equal timestamps, and "postorder" is a stack-pop walk that emits the
  head first. Asserted complete coverage, no revisits, child-before-
  parent on linear chains, and the documented merged-before-base rule.
- **objpatch D8** — DETAILS claimed submodule bumps produce no stat
  row; the reference emits one (the synthetic `Subproject commit` diff
  has chunks). Asserted the binary clause only and recorded the
  refusal in the contract. **D13** — `Patch.String`'s
  `malformed patch:` branch is unreachable (its internal buffer cannot
  fail); asserted `String() == Encode` output instead.
- **packparse D7** — `ErrNotSeekableSource` is unreachable via `Parse`:
  a non-seekable source silently disables low-memory mode and buffers.
  Asserted correct parse under both storage configurations.
- **negotiate D6** — DETAILS reads as if every ACK kind resets the
  vein; observed: `ACKCommon` resets it only in stateless mode and only
  for a *new* common hash; `Continue`/`Ready` reset unconditionally.
  **D8** — no progress channel means *no* progress capability token on
  the wire, not a `no-progress` literal.
- **fetchmsg** — a ready-then-EOF response returns bare `EOF`, not the
  `MalformedResponseError` the line implied; asserted non-nil error.
- **archive** — `MatchesPathFilter("docs/", …)` matches only the dir
  entry itself (bare `docs` matches children); the symlink prefix
  rejection lives in `WriteArchive`, not `WriteTarArchive`; tar members
  store symlinks with regular-file mode bits. Asserted the observed
  shapes, not the doc's loose wording.
- **memstor** — cheat caught on bogus object-format acceptance
  (D11). **negotiate** — cheat fails to compile (it ships its own
  `negotiate_test.go` colliding with the hidden mock). **objpatch** —
  cheat fails the empty-patch detail (D1). **packlookup** — cheat skips
  resolver/FSObject/delta paths entirely (D1, D2, D5, D6).
  **packhandle** — cheat caught on the FD-lifecycle assertions.

## Infrastructure notes

- Excision sometimes deletes shared test helpers while keeping their
  dependents; since `test.sh` runs only `TestDetailNN`, the hidden
  files supply compile-only shims where needed: `BaseObjectsSuite`
  (sigcodec), `mockWriteCloser` (negotiate), `randBytes` (deltadiff),
  `testPackObject`/`buildTestPack` (packlookup).
- `plumbing.Hash` is a struct carrying a format tag — tests compare
  `.Bytes()`, never `!=`.
- After excision the test binary may not link crypto implementations;
  hidden files blank-import `crypto/sha1` and `crypto/sha256` where
  hashing is exercised (fsnode, memstor, packhandle, packlookup).
- Import cycles force external `_test` packages where the hidden suite
  needs a package that imports the unit under test (`idxindex_test`
  needs `revfile`; `packhandle_test` needs `packfile`).

## Caveat

One read of pristine-tree lines in `internal/packhandle` (the `New`
error branches) occurred while locating error vars; those facts are
already committed verbatim by the stub's doc comments, so no
undocumented behaviour was learned from it. Excised-file content was
otherwise sourced exclusively from `_author/excised/tree/` stubs.
