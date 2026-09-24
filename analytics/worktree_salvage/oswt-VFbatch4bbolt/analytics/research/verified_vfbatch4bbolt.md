# Verified hidden suites — VFbatch4 bbolt (`experiments/pipeline/authored_batch4/bbolt`)

Hidden suites assert only what each unit's `DETAILS.md` commits to, honoring the
per-row `Inferable:` annotation (`yes`/`doc` asserted exactly; `partially` asserted
for the derivable part; `no` asserted at shape level — never a literal a solver
could not derive). Tests live in `tests/hidden/**_bb_test.go` as `TestDetailNN`
numbered to the DETAILS rows; `tests/test.sh` carries SHA-256 checksums and runs
each package's suite inside the built image.

Verification (per unit, in `ladder-base:bbolt` via `_tools/verify_hidden.sh`):

1. excised tree + hidden suite → FAIL (rc=1, reward=0)
2. gold tree + hidden suite → PASS (rc=0, reward=1)
3. cheat tree + hidden suite → FAIL (rc=1, reward=0)
4. A12 — `gold.patch` does not touch any test file

**Result: all 37 testable units pass all four checks.** `cmdlist` ships no
`DETAILS.md` — there is nothing to grade, so no suite was authored; reported
here rather than guessed.

## Repairs made during verification (pre-existing suites)

- `dbclose`, `dblock`, `dblog` — `tests/test.sh` checksums were stale relative to
  edited hidden tests; regenerated with `_tools/gen_testsh.sh`.
- `dbmap` — hidden test assigned a `[1<<20]byte` overlay to `db.data`, whose type
  is `*[common.MaxMapSize]byte`; fixed via `unsafe.Pointer` conversion.
- `dbwrap` — test set `MaxBatchSize`/`MaxBatchDelay` inside `bolt.Options`; those
  are `DB` fields in this revision — set on the opened handle instead.
- `dblog` — `cheat.patch` context line drifted (`raft/blob/logger.go` vs
  `raft/blob/main/logger.go`); patched the context line only.

## Notes on cheat runs that end in panics

For `bucketspill`, `txlife`, and `txpage` the cheat implementation crashes the
test binary (SIGBUS/panic) rather than producing a named test failure. The
harness records the non-zero exit, which satisfies cheat → FAIL.

## Reported anomalies

- `cmdbenchopt` row 3: DETAILS lists `seq-del` inside `Validate`'s accepted
  write-mode set, but the reference `Validate` rejects it (`runWrites`' dispatch
  accepts it — DETAILS conflated the two sets). The suite asserts acceptance of
  `seq`/`rnd`/`seq-nest`/`rnd-nest` and rejection of an unlisted mode; `seq-del`
  is asserted in neither direction, since either choice would fail one side of
  the contradiction.
- `cmdbenchread` row 4: the `numReads == iterations` mismatch cannot be
  triggered through solver-visible API — asserted positively only (a consistent
  run returns nil); the mismatch error text is arbitrary anyway.
- `cmdsurgfree` row 6: the "source mode bits passed to `bolt.Open`" part is
  unobservable — the output file already exists (created by the copy step), so
  `Open`'s mode argument is never applied. Asserted: output exists with a
  persisted freelist. Row 7 (failed `db.Close` propagation): no reliable way to
  force a close failure through solver-visible API — not asserted.
- `bucketspill` row 12 vs `bucketser` row 10: the two units commit to opposite
  literals for `cloneBytes(nil)` (nil vs empty). bucketspill's suite asserts
  copy-independence only and refuses the nil-return literal.
- `cmdlist`: no `DETAILS.md`; unverifiable.

## Summary

| unit | tests | doc | partially | no | yes | excised | gold | cheat | A12 |
|---|---|---|---|---|---|---|---|---|---|
| btestfill | 7 | 1 | 4 | 2 | 0 | FAIL | PASS | FAIL | PASS |
| btestlife | 9 | 4 | 5 | 0 | 0 | FAIL | PASS | FAIL | PASS |
| bucketser | 11 | 2 | 6 | 3 | 0 | FAIL | PASS | FAIL | PASS |
| bucketspill | 12 | 1 | 7 | 4 | 0 | FAIL | PASS | FAIL | PASS |
| bucketwalk | 12 | 0 | 8 | 4 | 0 | FAIL | PASS | FAIL | PASS |
| cmdbenchopt | 7 | 3 | 3 | 1 | 0 | FAIL | PASS | FAIL | PASS |
| cmdbenchread | 6 | 1 | 4 | 1 | 0 | FAIL | PASS | FAIL | PASS |
| cmdbenchstat | 7 | 3 | 4 | 0 | 0 | FAIL | PASS | FAIL | PASS |
| cmdbenchwrite | 11 | 1 | 6 | 4 | 0 | FAIL | PASS | FAIL | PASS |
| cmdcheck | 7 | 0 | 6 | 1 | 0 | FAIL | PASS | FAIL | PASS |
| cmdcompact | 8 | 0 | 6 | 2 | 0 | FAIL | PASS | FAIL | PASS |
| cmdinspect | 6 | 0 | 5 | 1 | 0 | FAIL | PASS | FAIL | PASS |
| cmdlist | — | — | — | — | — | **unverifiable — no DETAILS.md** | | | |
| cmdpageitem | 11 | 0 | 8 | 2 | 1 | FAIL | PASS | FAIL | PASS |
| cmdsurgery | 12 | 0 | 10 | 2 | 0 | FAIL | PASS | FAIL | PASS |
| cmdsurgfree | 9 | 0 | 7 | 2 | 0 | FAIL | PASS | FAIL | PASS |
| cmdver | 5 | 0 | 3 | 2 | 0 | FAIL | PASS | FAIL | PASS |
| cursorwalk | 13 | 0 | 8 | 5 | 0 | FAIL | PASS | FAIL | PASS |
| dbclose | 7 | 0 | 3 | 4 | 0 | FAIL | PASS | FAIL | PASS |
| dbfile | 7 | 1 | 4 | 2 | 0 | FAIL | PASS | FAIL | PASS |
| dbfmt | 7 | 2 | 2 | 3 | 0 | FAIL | PASS | FAIL | PASS |
| dbgrow | 12 | 0 | 7 | 5 | 0 | FAIL | PASS | FAIL | PASS |
| dbinfo | 6 | 2 | 4 | 0 | 0 | FAIL | PASS | FAIL | PASS |
| dblock | 5 | 0 | 2 | 3 | 0 | FAIL | PASS | FAIL | PASS |
| dblog | 10 | 0 | 5 | 5 | 0 | FAIL | PASS | FAIL | PASS |
| dbmap | 13 | 3 | 6 | 4 | 0 | FAIL | PASS | FAIL | PASS |
| dbstats | 7 | 0 | 5 | 2 | 0 | FAIL | PASS | FAIL | PASS |
| dbsync | 5 | 0 | 2 | 3 | 0 | FAIL | PASS | FAIL | PASS |
| dbwrap | 13 | 4 | 7 | 2 | 0 | FAIL | PASS | FAIL | PASS |
| nodemut | 11 | 0 | 6 | 5 | 0 | FAIL | PASS | FAIL | PASS |
| nodespill | 8 | 0 | 3 | 5 | 0 | FAIL | PASS | FAIL | PASS |
| nodewire | 12 | 2 | 5 | 4 | 1 | FAIL | PASS | FAIL | PASS |
| txbucket | 8 | 4 | 3 | 1 | 0 | FAIL | PASS | FAIL | PASS |
| txlife | 12 | 1 | 7 | 4 | 0 | FAIL | PASS | FAIL | PASS |
| txpage | 9 | 1 | 4 | 4 | 0 | FAIL | PASS | FAIL | PASS |
| txstats | 10 | 2 | 4 | 4 | 0 | FAIL | PASS | FAIL | PASS |
| txwrite | 12 | 0 | 7 | 5 | 0 | FAIL | PASS | FAIL | PASS |
| unsafe3 | 9 | 0 | 5 | 4 | 0 | FAIL | PASS | FAIL | PASS |

`cmdlist` ships no `DETAILS.md` — there are no commitments to grade, so
no hidden suite was authored for it. Reported rather than guessed.

## Per-unit detail

### btestfill

- assertions: 7 hidden tests (`TestDetail01..07`)
- inferable breakdown: doc=1, no=2, partially=4
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 6: `PrintStats` emits a fixed two-line `[db]` layout of page/cursor/node and rebalance/spill/write counters.
    → the call writes exactly two non-empty lines to stdout; the literal layout is not asserted.
  - row 7: `truncDuration` strips fractional seconds from a `Duration.String()` via regex, keeping the integer part plus unit.
    → the fractional component is removed and the unit suffix kept; the regex itself is not asserted.
- refused / weakened commitments:
  - row 3: Detail 3: MustCheck collects ALL errors from tx.Check() but stops after 10. Inferable: partially — asserted: on a healthy database MustCheck returns normally (drains an empty channel without exiting). The cap of 10 is an arbitrary constant (Inferable: no) — not asserted.
  - row 4: Detail 4: on check failure MustCheck copies the DB to a temp path, prints errors, and exits the process. Inferable: partially — asserted in a subprocess: a corrupted DB causes a non-zero exit. The exact exit code, print layout, and temp filename are Inferable: no — not asserted.
  - row 5: Detail 5: CopyTempFile writes to a file inside t.TempDir() via tx.CopyFile. Inferable: partially — asserted: after the call, a new file appears under the test's temp tree that opens as a valid DB holding the same data. The filename is arbitrary — not asserted.
  - row 6: Detail 6: PrintStats emits a fixed two-line layout of page/cursor/node and rebalance/spill/write counters. Inferable: no — asserted as shape only: the call writes exactly two non-empty lines to stdout; the literal layout is not asserted.
  - row 7: Detail 7: truncDuration strips fractional seconds from Duration.String, keeping the integer part plus unit. Inferable: no — asserted as shape only via the documented transform: the fractional component is removed and the unit suffix kept; the regex itself is not asserted.

### btestlife

- assertions: 9 hidden tests (`TestDetail01..09`)
- inferable breakdown: doc=4, partially=5
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- refused / weakened commitments:
  - row 6: Detail 6: Close is idempotent (no-op when db.DB == nil), logs, and nils the handle. Inferable: partially — asserted: two Closes succeed and the handle is nil; the -stats flag path is not asserted.

### bucketser

- assertions: 11 hidden tests (`TestDetail01..11`)
- inferable breakdown: doc=2, no=3, partially=6
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: `openBucket` clones the value first when its address is misaligned for the `InBucket+Page` overlay — alignment is tested with an `unsafe.Alignof` struct mask, not a fixed constant.
    → a misaligned input still decodes correctly, and the resulting header/page do not alias the misaligned input (a copy intervened).
  - row 3: In a writable transaction with an aligned value, only the `InBucket` HEADER is copied to the heap — the fake page still aliases the original value bytes; in a read-only transaction the header aliases the value directly.
    → writable -> header NOT aliased, page aliased; read-only -> header aliased.
  - row 9: `maxInlineBucketSize` is `pageSize / 4` — a quarter of a page, not half, not a fixed byte count.
    → it is positive, smaller than a page, divides the page evenly, and is deterministic across calls. The literal fraction is not pinned.

### bucketspill

- assertions: 12 hidden tests (`TestDetail01..12`)
- inferable breakdown: doc=1, no=4, partially=7
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 3: `spill` SKIPS a child whose `rootNode` is nil entirely — unmaterialized buckets produce no parent update at all.
    → a fetched-but-untouched child leaves the commit clean and its content unchanged.
  - row 4: Updating the parent locates the child's key with `c.seek` and PANICS if the returned key differs from the bucket name or lacks `BucketLeafFlag` — silent misplacement is a crash, not a repair.
    → deleting the child's key from the parent's node before spill makes spill panic; the panic text is not pinned.
  - row 7: After `rootNode.spill()` the bucket's `rootNode` is reset to `rootNode.root()` and its pgid must be below `meta.Pgid()` — a root at or above the high-water mark panics.
    → after spill the rootNode is its own root and lands below the tx's high-water mark.
  - row 12: `cloneBytes` returns nil for nil input and a fresh copy otherwise — callers can't mutate the source through the result.
    → equal length and content, independent backing store. NOTE: the nil-input return (nil vs empty-non-nil) is not asserted — the sibling unit bucketser commits to the opposite literal for the same function, so the nil-return contract is refused rather than guessed.
- refused / weakened commitments:
  - row 12: Detail 12: cloneBytes returns an exact copy — callers can't mutate the source through the result. Inferable: no — asserted as shape: equal length and content, independent backing store. NOTE: the nil-input return (nil vs empty-non-nil) is not asserted — the sibling unit bucketser commits to the opposite literal for the same function, so the nil-return contract is refused rather than guessed.

### bucketwalk

- assertions: 12 hidden tests (`TestDetail01..12`)
- inferable breakdown: no=4, partially=8
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: `forEachPageNode` on an inline bucket yields `(b.page, nil, 0)` — the materialized `rootNode` is NOT visited even when it exists.
    → exactly one hit, the inline page, a nil node, depth 0, even after materialization.
  - row 7: `node` asserts the `nodes` map is non-nil — on a read-only bucket (where `newBucket` left it nil) materialization panics instead of degrading.
    → the call panics; the assertion text is not pinned.
  - row 8: `node` wires parentage asymmetrically: `parent == nil` installs the node as `rootNode`, otherwise it appends to `parent.children` — a node never appears in both places at once.
    → the two wiring targets are disjoint.
  - row 10: `node` bumps `tx.stats.NodeCount` on every MATERIALIZATION — cache hits do not count.
    → the first node() call for a page increments once, a repeat call does not.
- refused / weakened commitments:
  - row 9: Detail 9: node reads from the inline b.page when present and only then calls n.read — inline materialization never touches the transaction's page table. Inferable: partially — asserted: node(0) on an inline bucket materializes a leaf whose inodes carry the stored key. REFUSED: the parenthetical "verifying pgId == 0" is not asserted — no panic/error contract for nonzero ids is committed.

### cmdbenchopt

- assertions: 7 hidden tests (`TestDetail01..07`)
- inferable breakdown: doc=3, no=1, partially=3
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 7: Flag default for `count` is 1000; `key-size` 8; `value-size` 32.
    → the flags bound to the iterations/keySize/valueSize fields carry positive non-zero defaults. Spellings and exact values are not asserted.
- refused / weakened commitments:
  - row 1: Detail 1: AddFlags binds a pflag for every exported knob of the benchmark. Inferable: partially — asserted as shape: at least fifteen flags are registered (one per enumerated knob); spellings and defaults are not asserted here.
  - row 3: Detail 3: Validate rejects a write-mode outside the accepted set. Inferable: partially — asserted: seq / rnd / seq-nest / rnd-nest pass and an unlisted mode fails. seq-del is deliberately not asserted in either direction: DETAILS lists it in the accepted set but the behaviour disagrees, so asserting it would fail a faithful solver.
  - row 5: Detail 5: SetOptionValues generates a temporary database path when path is empty. Inferable: partially — asserted: path becomes non-empty; the prefix pattern is not asserted.
  - row 7: Detail 7: flag defaults — count 1000, key-size 8, value-size 32. Inferable: no — asserted as shape only: the flags bound to the iterations/keySize/valueSize fields carry positive non-zero defaults. Spellings and exact values are not asserted.

### cmdbenchread

- assertions: 6 hidden tests (`TestDetail01..06`)
- inferable breakdown: doc=1, no=1, partially=4
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 4: Sequential-mode runs verify `numReads == iterations` and error on mismatch ("iter mismatch" message).
    → a consistent run returns nil. The mismatch path cannot be triggered through solver-visible API and the message text is not asserted.
- refused / weakened commitments:
  - row 4: Detail 4: sequential-mode runs verify numReads == iterations and error on mismatch. Inferable: no — asserted positively only: a consistent run returns nil. The mismatch path cannot be triggered through solver-visible API and the message text is not asserted.

### cmdbenchstat

- assertions: 7 hidden tests (`TestDetail01..07`)
- inferable breakdown: doc=3, partially=4
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- refused / weakened commitments:
  - row 5: Detail 5: printGoBenchResult emits a testing.BenchmarkResult-shaped line with the bench name column padded to width maxLen. Inferable: partially — asserted: a single line containing the name and an ns/op figure, with the numeric columns starting at or beyond maxLen. The exact column layout is not asserted.
  - row 6: Detail 6: checkProgress emits a progress line every second until the finish channel closes, computing rate from the delta. Inferable: partially — asserted: after ~1s of running with completed ops, at least one line naming the count is written; closing finish stops it. The message text is not asserted.

### cmdbenchwrite

- assertions: 11 hidden tests (`TestDetail01..11`)
- inferable breakdown: doc=1, no=4, partially=6
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 3: `benchFunc` writes "starting write benchmark." / "starting read benchmark." to stderr.
    → at least two non-empty lines reach stderr (cmd.Err or os.Stderr); the exact strings are not asserted.
  - row 6: `runWritesWithSource` collects keys only when `readMode == "rnd"`.
    → rnd collects one key per iteration; a non-rnd mode collects none.
  - row 7: `runWritesDeletesWithSource` deletes `ceil(batchSize*deleteFraction)` of the previously inserted keys at the top of each iteration.
    → a delete-run leaves strictly fewer keys than an equivalent pure-write run with the same key source, without emptying the bucket. The ceil(batchSize*deleteFraction) count is not asserted.
  - row 11: Per-iteration "Starting write iteration N" progress goes to stderr.
    → a multi-iteration write run emits at least one non-empty stderr line. The message text is not asserted.
- refused / weakened commitments:
  - row 2: Detail 2: benchFunc applies pageSize and initialMmapSize to DefaultOptions before Open and sets db.NoSync. Inferable: partially — asserted: the produced database uses the requested page size. The mmap-size and NoSync applications are not externally observable and are not asserted.
  - row 3: Detail 3: benchFunc writes a write-phase and a read-phase banner to stderr. Inferable: no — asserted as shape: at least two non-empty lines reach stderr (cmd.Err or os.Stderr); the exact strings are not asserted.
  - row 5: Detail 5: write runners emit fixed-size keys in batches inside a single db.Update, calling addCompletedOps(1) per insert. Inferable: partially — asserted: completedOps equals iterations and every written key is keySize bytes; endianness and the Update boundary are not asserted.
  - row 7: Detail 7: runWritesDeletesWithSource deletes part of the previously inserted keys at the top of each iteration. Inferable: no — asserted as shape: a delete-run leaves strictly fewer keys than an equivalent pure-write run with the same key source, without emptying the bucket. The ceil(batchSize*deleteFraction) count is not asserted.
  - row 8: Detail 8: runWritesNestedWithSource creates one sub-bucket per batch iteration, named from the same key source. Inferable: partially — asserted: iterations/batchSize sub-buckets appear under the bench bucket. Naming derivation is not asserted.
  - row 9: Detail 9: startProfiling creates each requested profile file; stopProfiling stops CPU and writes heap+block profiles. Inferable: partially — asserted: all three files exist after start, and the heap/block dumps are non-empty after stop. Rate constants are not asserted.
  - row 10: Detail 10: result output prints "# Write"/"# Read" lines, or gobench-format lines when gobench-output is set. Inferable: partially — asserted: the committed "# Write"/"# Read" tokens appear on stdout in normal mode and "ns/op" appears in gobench mode. Column layout is not asserted.
  - row 11: Detail 11: per-iteration write progress goes to stderr. Inferable: no — asserted as shape: a multi-iteration write run emits at least one non-empty stderr line. The message text is not asserted.

### cmdcheck

- assertions: 7 hidden tests (`TestDetail01..07`)
- inferable breakdown: no=1, partially=6
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 3: The database is opened `ReadOnly: true` AND `PreLoadFreelist: true` — a check never takes a write lock and never mutates the file.
    → the file checksum is identical before and after a successful check.
- refused / weakened commitments:
  - row 6: Detail 6: count>0 prints an errors summary and returns guts_cli.ErrCorrupt; zero errors prints OK and returns nil. Inferable: partially — asserted: corrupt → errors.Is ErrCorrupt, healthy → nil with an OK line; the summary layout is not asserted.
  - row 7: Detail 7: AddFlags registers a uint64 flag defaulting to 0 bound to o.fromPageID — a malformed value is a flag error. Inferable: partially — asserted via discovery: the uint64 flag exists with a zero default, parses into the field, and rejects a non-numeric value. The flag spelling is not asserted.

### cmdcompact

- assertions: 8 hidden tests (`TestDetail01..08`)
- inferable breakdown: no=2, partially=6
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 4: `Run` opens source `ReadOnly: true` with mode `0400` and destination with `fi.Mode()` — the destination inherits the SOURCE file's permission bits — plus `NoSync: o.dstNoSync`.
    → the source file is byte-identical afterwards and the destination mode equals the source mode.
  - row 7: The command's `Args` is `cobra.MinimumNArgs(1)` — extra positional args are tolerated, not rejected.
    → zero args fail, one and three pass the arg-count check.
- refused / weakened commitments:
  - row 2: Detail 2: AddFlags registers the output flag (required), an int64 tx-max-size flag with a positive default, and a boolean no-sync flag. Inferable: partially — asserted by discovery: a string flag bound to dstPath carries the required annotation; an int64 flag bound to txMaxSize has a positive default; a bool flag binds dstNoSync. Flag spellings are not asserted.

### cmdinspect

- assertions: 6 hidden tests (`TestDetail01..06`)
- inferable breakdown: no=1, partially=5
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 3: The DB is opened `ReadOnly: true` — inspect never writes or locks exclusively.
    → the file is byte-identical after a successful inspect.
- refused / weakened commitments:
  - row 5: Detail 5: output is MarshalIndent'd JSON printed to os.Stdout with a trailing newline. Inferable: partially — asserted: multi-line indented JSON ending in a newline; the indent width itself is not asserted.

### cmdlist

No `DETAILS.md` — nothing is committed, so no hidden suite can be
written fairly. **Unverifiable; reported, not tested.**

### cmdpageitem

- assertions: 11 hidden tests (`TestDetail01..11`)
- inferable breakdown: no=2, partially=8, yes=1
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 8: `itemID` is silently truncated to `uint16` on its way to the element reader — a huge `--` arg wraps rather than erroring.
    → an itemID that is the real index plus 65536 resolves to the same element without error.
  - row 9: `pageItemFunc` never validates `pageID` against the database page count — out-of-range pages fail inside `guts_cli.ReadPage`, not here.
    → a huge page id returns a non-nil error.
- refused / weakened commitments:
  - row 1: Detail 1: --key-only and --value-only are mutually exclusive — an error before any file access. Inferable: partially — asserted: with both set and a MISSING path, the error is not the source-path error (the flag check ran first). The error text is not asserted.
  - row 5: Detail 5: pageItemLeafPageElement bounds-checks index >= p.Count() — the error names the bound and the index. Inferable: partially — asserted: an out-of-range index errors mentioning both numbers; the format is not asserted.
  - row 6: Detail 6: a non-leaf page is rejected and the error reports the actual page type. Inferable: partially — asserted: a meta page produces an error naming "meta"; the format is not asserted.
  - row 10: Detail 10: --format defaults to "auto" and passes straight to writelnBytes — no validation. Inferable: partially — asserted: a hex format renders hex output and an unsupported format errors. The flag binding is discovered (a string flag whose default is "auto" bound to o.format); the spelling is not asserted.

### cmdsurgery

- assertions: 12 hidden tests (`TestDetail01..12`)
- inferable breakdown: no=2, partially=10
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 5: `surgeryCopyPageFunc` reads the SOURCE meta afterwards and warns when `IsFreelistPersisted()` — the advisory mentions abandoning the freelist.
    → iff the source meta reports a persisted freelist, a warning naming the freelist is printed.
  - row 12: Success messages go to `os.Stdout` (e.g. the page-copied line), while wrapped errors carry a `[<cmd>] copy file failed` prefix.
    → a successful surgery emits at least one stdout line and a failing one returns a non-nil error with nothing on stdout.

### cmdsurgfree

- assertions: 9 hidden tests (`TestDetail01..09`)
- inferable breakdown: no=2, partially=7
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 3: The abandon message warns that the next startup may be slower because the freelist must be rescanned — the caveat is part of the contract.
    → a successful abandon prints a non-empty advisory naming the freelist.
  - row 8: Both funcs wrap lower-level errors with a `[freelist abandon]`/ `[freelist rebuild]`-prefixed message — the operation tag survives in the error chain.
    → a copy failure surfaces a non-nil error naming the operation.
- refused / weakened commitments:
  - row 6: Detail 6: the rebuild opens the copied db and closes it to reconstruct the freelist. Inferable: partially — asserted: on a no-persisted-freelist source the output exists and reports a persisted freelist. The source mode bits passed to bolt.Open are not observable — the copy already exists, so Open's mode argument is unused — and not asserted.
  - row 7: Detail 7: a failed db.Close() propagates as a wrapped error. Inferable: partially — NOT asserted: no reliable way to force a close failure through solver-visible API.

### cmdver

- assertions: 5 hidden tests (`TestDetail01..05`)
- inferable breakdown: no=2, partially=3
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: `newVersionCommand` prints `versionOutput()` via `fmt.Print` inside `Run` (not `RunE`) — the command cannot return an error.
    → the command is runnable, has no error path, and executing it puts the banner on process stdout.
  - row 3: `NewRootCommand` sets `Version: version.Version` AND `SetVersionTemplate(versionOutput())` — cobra's `--version` output is the banner, not the default template.
    → root.Version tracks the version var and a --version execution emits the banner text.
- refused / weakened commitments:
  - row 4: Detail 4: NewRootCommand registers all fifteen subcommands. Inferable: partially — asserted: the fifteen named subcommands are all wired (set membership; order is not asserted).
  - row 5: Detail 5: main on a non-nil Execute error prints `Error: <err>` to stderr (gated on SilenceErrors) and exits 1; a successful run exits 0. Inferable: partially — asserted in subprocesses: a successful invocation exits 0 with the banner on stdout; a failing invocation exits non-zero with an `Error:` line on stderr. The SilenceErrors gate itself is not asserted.

### cursorwalk

- assertions: 13 hidden tests (`TestDetail01..13`)
- inferable breakdown: no=5, partially=8
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 1: `search` panics when the page it lands on is neither branch nor leaf — an unknown page type is fatal, not skipped.
    → descending onto a flagless page panics; the text is not pinned.
  - row 9: `next` that cannot advance leaves the cursor ON the last element and returns a nil triple — it does not invalidate the position.
    → after exhaustion keyValue still returns the last element.
  - row 10: `prev` at the very first element resets to `first` and returns nil — the position is left at the head so a following `next` still works, rather than staying put or erroring.
    → Prev at head returns nil, and a subsequent Next returns the second element (position preserved).
  - row 12: `node` returns the top-of-stack node when it is already a leaf; otherwise it re-walks from the stack root via `childAt`, asserting branch on the way down and leaf at the end.
    → the returned node is a leaf holding the positioned key.
  - row 13: `elemRef.isLeaf`/`count` trust the NODE when one is set — the page field may hold a stale or unrelated page and is ignored.
    → a ref with both fields set answers from the node, contradicting the page.

### dbclose

- assertions: 7 hidden tests (`TestDetail01..07`)
- inferable breakdown: no=4, partially=3
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: `close` clears `db.freelist` to nil and nils `db.ops.writeAt` BEFORE touching the mmap or the file — teardown order is state first, then unmap, then file.
    → after close, both fields are nil (order itself not observable).
  - row 4: `close` aggregates errors from munmap, funlock, AND file.Close into a slice and returns only the FIRST — later cleanup still runs even if an earlier step failed.
    → a failing munmap still leaves file closed and path cleared, and the returned error is the munmap one.
  - row 5: `close` always clears `db.path` to "" and `db.file` to nil before returning — even on the error path.
    → (see test comment)
  - row 6: `removeTx` releases the tx's `mmaplock` read lock FIRST, then takes `metalock` to drop the txid from the freelist's readonly set — the two locks are never held together.
    → a removeTx on a live read tx completes without deadlock and drops the txid from the freelist's readonly set (observable via a second call being safe and stats consistency). No deferred db.Close: an excised removeTx panics while the read tx still holds mmaplock.RLock, and a blocking Close would hang the leg.

### dbfile

- assertions: 7 hidden tests (`TestDetail01..07`)
- inferable breakdown: doc=1, no=2, partially=4
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 5: `mmap` calls `unix.Madvise(b, MADV_RANDOM)` after mapping and ignores exactly one error: `ENOSYS`. Any other madvise failure is returned as an error.
    → mmap of a valid file succeeds (the tolerated outcome) and returns no madvise-derived error on this kernel.
  - row 7: `munmap` on a nil `dataref` returns nil without a syscall; otherwise it `Munmap`s `dataref` and clears `dataref`, `data`, AND `datasz` to zero values even when Munmap fails.
    → (see test comment)

### dbfmt

- assertions: 7 hidden tests (`TestDetail01..07`)
- inferable breakdown: doc=2, no=3, partially=2
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 3: `Options.String` on a nil `*Options` returns `"{}"` — it does not panic.
    → returns a short brace-wrapped string.
  - row 5: Function-typed fields (`OpenFile`, `Logger`) print as pointers, not by calling them.
    → a nil OpenFile and nil Logger print without panic, and a non-nil OpenFile renders an address-like token.
  - row 7: `Options.String` renders `FreelistType` through the field's own `String()` method (it is a stringer), not as a raw integer.
    → the output contains the String() form of the value, whatever that is, and not a bare number when the String form is non-numeric.

### dbgrow

- assertions: 12 hidden tests (`TestDetail01..12`)
- inferable breakdown: no=5, partially=7
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 4: `allocate` only remaps when `minsz >= datasz` — where `minsz` is `(pgid + count + 1) * pageSize`, one spare page past the allocation.
    → just below the threshold no remap is attempted (allocate succeeds with no file), at the threshold a remap is attempted (fileless db fails).
  - row 5: `allocate` enforces `MaxSize` before remapping: with `MaxSize > 0` the projected size goes through `mmapSize`, and on non-Windows is inflated by `growSize` — exceeding `MaxSize` fails with `ErrMaxSizeReached`.
    → the sentinel error is returned.
  - row 8: On non-Windows, `grow` calls `file.Truncate` before `file.Sync`; with `Mlock` it re-locks the range afterwards.
    → a writable grow actually extends the file to the growSize-computed size.
  - row 9: `growSize` returns `mmapSize` unchanged while `mmapSize <= AllocSize`, otherwise `growSize + AllocSize` — growth jumps in AllocSize chunks only once the mapping outgrows the threshold.
    → (see test comment)
  - row 12: `freepages` panics inside a deferred rollback when `Rollback` itself fails — teardown failure is a panic, not a returned error.
    → freepages on a healthy db returns without panicking (the no-failure shape).

### dbinfo

- assertions: 6 hidden tests (`TestDetail01..06`)
- inferable breakdown: doc=2, partially=4
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`

### dblock

- assertions: 5 hidden tests (`TestDetail01..05`)
- inferable breakdown: no=3, partially=2
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 1: Package-level `mlock`/`munlock` clamp the requested `fileSize` to `db.datasz` — you can never lock/unlock more than the actually mapped slice; the syscall sees `db.dataref[:min(fileSize, datasz)]`.
    → requesting a lock far larger than the mapped slice succeeds (unclamped would slice out of range or syscall-fail).
  - row 2: `munlock` on a DB whose `dataref` is nil returns nil WITHOUT calling the syscall — unmapping is idempotent; `mlock` has no such nil guard (it locks a zero-length slice harmlessly).
    → both return nil on an unmapped DB.
  - row 4: `db.mrelock(from, to)` is unlock-then-lock in THAT order — `munlock(from)` first, `mlock(to)` second — and returns the first error encountered, aborting the second step.
    → a failing first step propagates an error, and a fixture where both steps are no-ops returns nil.

### dblog

- assertions: 10 hidden tests (`TestDetail01..10`)
- inferable breakdown: no=5, partially=5
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: `Info`, `Error`, `Warning` route through `l.Output(calldepth, ...)` with a fixed `calldepth` of 2 — the reported caller is the DB code site, not the logger's own frame.
    → with Lshortfile the logged source is this test file, not logger.go.
  - row 4: `Panic`/`Panicf` delegate to the embedded `log.Logger.Panic/Panicf` with NO `header` prefix — they panic with the raw message, unlike every other level.
    → recovered panic value is the raw message.
  - row 5: `header` produces `"<LVL>: <msg>"` — a level tag, colon, space, message.
    → (see test comment)
  - row 6: Level tags are `DEBUG`, `INFO`, `ERROR`, `WARN`, `FATAL` — `Warning` writes under the `WARN` tag, not `WARNING`.
    → each level's tag appears and "WARNING" never does.
  - row 9: The `f` variants format eagerly with `fmt.Sprintf` while plain variants use `fmt.Sprint` — Debug-gating means `Debugf` does NOT evaluate its arguments through Sprintf when disabled... it still receives them, but the Output call is skipped.
    → nothing reaches the writer when disabled, including with format args. REFUSED in part: whether Sprintf itself ran is not observable.

### dbmap

- assertions: 13 hidden tests (`TestDetail01..13`)
- inferable breakdown: doc=3, no=4, partially=6
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 1: `fileSize` fails when the file is smaller than two pages — the error mentions the actual size.
    → the error names the size value.
  - row 3: `mmap` rejects only when `MaxSize > 0`, `size > MaxSize`, `size > fileSize`, AND the platform is Windows — on other platforms an oversized map is allowed to proceed.
    → an oversized request proceeds.
  - row 4: `mmap` dereferences an open read-write transaction's root BEFORE unmapping — stale page pointers are dropped before the old map goes away.
    → remapping while a write tx is open succeeds without stale-pointer faults.
  - row 9: Above 1GB `mmapSize` rounds up to `MaxMmapStep` multiples, then to a page-size multiple, and caps at `MaxMapSize` — an over-cap REQUEST errors ("mmap too large") while an over-cap RESULT is clamped.
    → rounding shape plus the two cap behaviors.

### dbstats

- assertions: 7 hidden tests (`TestDetail01..07`)
- inferable breakdown: no=2, partially=5
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: `Sub(nil)` returns `*s` — a plain copy of the receiver, no error and no zeroing.
    → result equals the receiver, no panic.
  - row 5: `OpenTxN` is neither differenced nor copied — a `Sub` result always carries zero in `OpenTxN`.
    → the differencing path carries zero in OpenTxN. REFUSED in part: the "always" claim over the Sub(nil) path is not derivable (Detail 2 commits Sub(nil) to a plain copy, which may legitimately keep it).

### dbsync

- assertions: 5 hidden tests (`TestDetail01..05`)
- inferable breakdown: no=3, partially=2
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 1: On linux `fdatasync` issues `syscall.Fdatasync` on `db.file.Fd()` — data-only sync; it is NOT `file.Sync()`/`fsync` and does not flush metadata.
    → Sync on an open RW db performs a flush and returns nil.
  - row 3: `DB.Sync` logs through `db.Logger()` BEFORE and AFTER the flush — a `Debugf` on entry and either a success `Debugf` or an `Errorf` on the way out, so a recording logger sees exactly two lines on success.
    → exactly two calls, first a Debugf, second Debugf or Errorf.
  - row 5: `DB.Sync` does NOT check `NoSync` or `IgnoreNoSync` — those are honoured by the write path's caller, not here; calling `Sync` on a `NoSync` DB still performs the syscall.
    → Sync on a NoSync db still produces the two log calls, proving the flush path ran.

### dbwrap

- assertions: 13 hidden tests (`TestDetail01..13`)
- inferable breakdown: doc=4, no=2, partially=7
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 8: `batch.run` detaches itself from `db.batch` BEFORE running — late arrivals form a new batch instead of joining a running one.
    → after a batch completes, db.batch is nil and a subsequent call starts fresh.
  - row 9: On a call failure `batch.run` swaps the failing call to the end of the slice, sends it `trySolo`, and RETRIES the rest of the batch in a fresh Update — one bad call doesn't fail the others.
    → in a 3-call batch with one failing fn, the failing caller gets an error and the other two get nil with their writes committed.

### nodemut

- assertions: 11 hidden tests (`TestDetail01..11`)
- inferable breakdown: no=5, partially=6
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: `put` panics on a zero-length `oldKey` OR `newKey` — empty keys are rejected before any search happens.
    → either empty key is rejected by a panic (the signature has no error return); message content is not pinned.
  - row 4: `put` writes `newKey` into the inode — not `oldKey`: rename-during-update stores the new key at the old key's sorted position.
    → after the call the stored key equals newKey and oldKey is gone; the index at which the renamed inode sits is NOT pinned.
  - row 6: `childIndex` binary-searches the parent's inodes for `child.key` — it returns the first index whose key >= child key, which can be `numChildren()` when the child sorts last.
    → the result is the sorted-search position against the parent's inodes.
  - row 8: `numChildren` counts INODES, not the materialized `children` slice — the two lists can disagree mid-mutation.
    → with the two lists deliberately divergent, the count equals len(inodes).
  - row 10: `dereference` asserts non-empty inode keys AFTER copying — a zero-length key triggers the assert even though the copy already ran.
    → a node carrying an empty inode key panics; the copy-vs-assert ordering is not pinned.

### nodespill

- assertions: 8 hidden tests (`TestDetail01..08`)
- inferable breakdown: no=5, partially=3
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: Dirty children are spilled FIRST, bottom-up, in `sort.Sort(n.children)` order — the loop re-checks `len(n.children)` each iteration because a child spill can materialize siblings. After the loop `n.children` is dropped entirely (set to nil).
    → every child is spilled and paged, and the children list is cleared. The intra-loop ordering is not pinned.
  - row 4: Each piece gets its OLD page freed (`node.free()`) before a contiguous run of `ceil(size/pageSize)` pages is allocated for it — free-before- allocate ordering matters to the freelist.
    → the node's prior page ends up in the freelist pending set and the node ends on a different live page. The free-vs-allocate ordering is not pinned.
  - row 5: An allocated page id at or above `tx.meta.Pgid()` is a hard `panic`, not an error return — the high-water check is an invariant violation.
    → allocation past the high-water mark panics rather than returning an error or writing anyway.
  - row 6: A piece with a non-nil parent inserts itself into the parent's branch via `parent.put(key, firstInodeKey, nil, pgid, 0)` — `key` is the node's existing `node.key` if set, else `inodes[0].Key()`; after the put, `node.key` is overwritten with `inodes[0].Key()` and asserted non-empty.
    → after spill the parent holds an inode keyed by the child's first key pointing at the child's page, and node.key equals that first key.
  - row 8: If the node's own parent has `pgid == 0` after the loop (a brand-new root created by the split), the parent is spilled recursively — its children list is already clear so it cannot respill this node.
    → after a splitting spill of a parentless node, the materialized parent is itself spilled onto a page.

### nodewire

- assertions: 12 hidden tests (`TestDetail01..12`)
- inferable breakdown: doc=2, no=4, partially=5, yes=1
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 4: `minKeys` is 1 for leaf nodes and 2 for branch nodes — asymmetric minima.
    → both minima are positive and the branch minimum is strictly greater than the leaf minimum; the literal values are NOT pinned.
  - row 7: `write` panics (with the page id in the message) when the node has `>= 0xFFFF` inodes rather than truncating the count to `uint16`.
    → it panics; the page id appearing in the message is NOT pinned.
  - row 8: `write` early-returns on a zero-count page — no element bytes are touched.
    → a sentinel-filled tail survives a zero-inode write untouched (header zeroed for the assert).
  - row 12: `nodes.Less` orders nodes by their FIRST inode key (`inodes[0].Key()`) via `bytes.Compare` — an empty-inode node would panic, not sort first.
    → ordering follows inodes[0].Key(); a node with no inodes panics rather than sorting first.

### txbucket

- assertions: 8 hidden tests (`TestDetail01..08`)
- inferable breakdown: doc=4, no=1, partially=3
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 8: None of these methods check `tx.db`/`tx.Writable()` themselves — a read-only or closed tx hits the same code path and the error (if any) comes from the delegate.
    → on a read-only tx the reads succeed and the writes surface an error (not a panic); which error is not pinned.

### txlife

- assertions: 12 hidden tests (`TestDetail01..12`)
- inferable breakdown: doc=1, no=4, partially=7
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 4: `Tx.ID` returns `-1` on a nil receiver or nil meta — a closed transaction has id -1 rather than panicking.
    → the result is negative and the call does not panic; the literal -1 is not pinned.
  - row 8: `rollback` additionally reconstructs the freelist after the txid rollback: `NoSyncReload(freepages())` when the freelist isn't synced, else `Reload` from the freelist page — and skips the whole reload when `db.data` is nil.
    → rollback clears the tx's pending frees and leaves the freelist consistent and usable; the reload-vs-skip choice is not pinned.
  - row 9: `close` on a writable tx: grabs freelist counts, clears `db.rwtx`, releases `db.rwlock`, THEN merges freelist+tx stats under `statlock` — the stats merge happens after the write lock is dropped.
    → after close the write slot is free (a new writable tx begins), and the tx's stats land in db stats. Ordering vs the lock drop is not pinned.
  - row 12: `FreeAlloc` in the merged stats is `(FreeCount + PendingCount) * pageSize` — counted in bytes, not pages.
    → FreeAlloc is 0 when the freelist is empty, and a positive multiple of pageSize when pages are pending.

### txpage

- assertions: 9 hidden tests (`TestDetail01..09`)
- inferable breakdown: doc=1, no=4, partially=4
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: `allocate` records the returned page in `tx.pages` keyed by `p.Id()` — the tx's dirty-page cache — so a later `tx.page(p.Id())` returns that same pointer, not the mmap view.
    → tx.page(p.Id()) returns the identical *Page, not a copy.
  - row 5: `page` runs `p.FastCheck(id)` on BOTH the cache-hit and the mmap path before returning — a mismatched id panics, it is never silently returned.
    → planting a page under a wrong key panics rather than returning it.
  - row 6: `forEachPage` seeds a fixed-capacity stack (cap 10) with `pgidnum` at index 0 and hands `stack[:1]` to the internal walker — the root page is visited at depth 0, not 1.
    → the first callback sees the root page at depth 0 with a single-element stack ending in the root id.
  - row 9: Children are reached by `append(pgidstack, child)` — siblings deeper in the walk may share backing array; the stack slice a `fn` receives is only valid for the duration of that call.
    → every call's stack ends at the visited page and starts at the root.

### txstats

- assertions: 10 hidden tests (`TestDetail01..10`)
- inferable breakdown: doc=2, no=4, partially=4
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 6: The three duration fields use `atomicAddDuration`/`atomicLoadDuration`, which perform int64 atomics through an `unsafe.Pointer` reinterpretation of `*time.Duration` — they do not allocate a separate atomic cell.
    → duration counters store and return exact nanosecond values like the int counters.
  - row 7: `IncXxx` accepts negative deltas — nothing clamps a counter at zero, so a counter can go negative.
    → a negative delta lowers the counter below zero.
  - row 8: `Sub` of `a - b` where `b` postdates `a` yields negative fields — there is no saturation.
    → (see test comment)
  - row 9: `add` takes a `*TxStats` (pointer, mutates receiver) while `Sub` takes `*TxStats` but returns a value — the asymmetry is deliberate.
    → add leaves the argument untouched while changing the receiver; Sub leaves both untouched (asserted in D2) and yields a usable value.

### txwrite

- assertions: 12 hidden tests (`TestDetail01..12`)
- inferable breakdown: no=5, partially=7
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 2: `write` flushes each page in chunks no larger than `common.MaxAllocSize - 1` — a page bigger than the cap takes multiple `writeAt` calls with the offset advancing by the chunk size.
    → no writeAt call exceeds the cap (the >cap path needs a >2GB page on amd64, which is not constructible; per-call bound is the derivable shape).
  - row 4: `write` skips `fdatasync` only when `NoSync` is set AND the `common.IgnoreNoSync` escape flag is false — either alone is not enough.
    → write succeeds on a NoSync db (the sync-skip branch) exactly as on a synced one; the syscall itself is not observable.
  - row 5: After writing, only pages with `Overflow() == 0` are zeroed byte-by-byte and returned to `pagePool` — larger pages are never pooled.
    → a written single-page buffer is handed back to the pool (the next single-page allocation reuses its backing memory).
  - row 6: `write` increments the write stat once per CHUNK, not once per page — a multi-chunk page counts multiple writes.
    → the Write counter delta equals the number of writeAt calls made (chunks == calls at this scale).
  - row 10: `commitFreelist` failure runs `tx.rollback()` and returns the allocation error — the transaction is already closed when the caller sees the error.
    → an allocation failure surfaces an error and leaves the tx closed.

### unsafe3

- assertions: 9 hidden tests (`TestDetail01..09`)
- inferable breakdown: no=4, partially=5
- docker: `excised=rc=1 gold=reward=1 cheat=rc=1 A12=PASS`
- `Inferable: no` rows:
  - row 4: The result is produced by a FULL slice expression `[i:j:j]` — the returned slice's capacity ends at `j`, so `append` on it always reallocates instead of writing into the mapped region.
    → cap == len == j-i, and appending does not write into the source buffer past j.
  - row 6: `i`/`j` outside `[0, MaxAllocSize]` or `i > j` fail through the ordinary runtime bounds/panic path — there is no sentinel error value.
    → i > j panics.
  - row 7: `UnsafeIndex` with `n < 0` or a huge `elemsz` silently wraps the address arithmetic — no guard exists.
    → the arithmetic result is returned with no panic.
  - row 8: `UnsafeByteSlice` with `i == j` returns an empty, non-nil slice aliasing the shifted base address — not `nil`.
    → len 0, non-nil.
