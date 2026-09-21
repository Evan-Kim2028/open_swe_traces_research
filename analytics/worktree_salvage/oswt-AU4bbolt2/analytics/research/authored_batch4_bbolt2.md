# authored_batch4/bbolt — 20 new units (batch 4, second bbolt wave)

Date: 2026-09-22. Worktree: `oswt-AU4bbolt2` (branch `au4bbolt2`). Pristine
source: `outputs/scratch/bbolt_gold` (`example.internal/boltstore`, go1.26.8).
Authoring tools: `scripts/ops/author_excise.py`, `make_cheat.sh`,
`verify_unit.sh`, `scripts/ops/cheat_validity.py`,
`scripts/check_unit_overlap_bbolt.py`.

Rationale: bbolt is the cheapest cohort in the bank (52% yield at 3.2 trials
per certified unit); this batch targets its remaining free storage/codec and
CLI surface.

## Search-cost protocol

Attempts = distinct candidate closures evaluated before the accepted one was
confirmed disjoint (vs. dose_response bank + prior authored closures). The
free-symbol inventory was computed once up front by mechanically subtracting
every banked/excised symbol from the full function list, so most units were
accepted on the first candidate.

## Units in authoring order

| # | unit | closure (file / symbols) | surface | attempts | overlap rejections | Inferable (yes/doc/part/no) | cheat ratio |
|---|------|--------------------------|---------|----------|--------------------|-----------------------------|-------------|
| 1 | txpage | tx.go: Tx.allocate, Tx.page, Tx.forEachPage, Tx.forEachPageInternal | storage/page alloc | 1 | Tx.write/commit→txwrite; DB.allocate→dbmap | 0/1/4/4 | 0.29 |
| 2 | nodespill | node.go: node.spill | storage/codec | 1 | node.put/del/read/write→nodemut/nodewire; node.dump rejected (dead code in comment block, not a real function) | 0/0/3/5 | 0.21 |
| 3 | txbucket | tx.go: Tx.Stats, Tx.Inspect, Tx.Bucket, Tx.CreateBucket, Tx.CreateBucketIfNotExists, Tx.DeleteBucket, Tx.MoveBucket; cursor.go: Cursor.Bucket | API delegation | 1 | Cursor.seek/nsearch→cursorwalk; Tx.init/ID/rollback→txlife | 0/4/3/1 | 0.50 |
| 4 | dbstats | db.go: Stats.Sub, DB.Stats | stats/codec | 1 | TxStats.Sub→txstats | 0/0/5/2 | 0.39 |
| 5 | dbfmt | db.go: Info.String, Stats.String, Options.String | stringers | 1 | — | 0/2/2/3 | 0.33 |
| 6 | dbinfo | db.go: DB.Path, DB.Logger, DB.Info, DB.IsReadOnly, newFreelist | getters/factory | 1 | DB.loadFreelist→dbmap | 0/2/4/0 | 0.33 |
| 7 | dbclose | db.go: DB.close, DB.removeTx | lifecycle | 1 | DB.mmap/munmap→dbmap | 0/0/3/4 | 0.28 |
| 8 | dblock | db.go: DB.mlock, DB.munlock, DB.mrelock; mlock_unix.go: mlock, munlock | OS plumbing | 1 | mrelock kept intact in dbgrow | 0/0/2/3 | 0.20 |
| 9 | dbsync | db.go: DB.Sync; bolt_linux.go + boltsync_unix.go: fdatasync×2 | OS plumbing | 1 | — | 0/0/2/3 | 0.20 |
| 10 | dbfile | bolt_unix.go: flock, funlock, mmap, munmap | OS plumbing | 1 | DB.mmap/DB.munmap methods→dbmap (package-level wrappers disjoint) | 0/1/4/2 | 0.33 |
| 11 | cmdcheck | command_check.go: checkOptions.AddFlags, newCheckCommand, checkFunc | CLI glue | 1 | checkSourceDBPath→cmdutils; tx.Check→txcheck-L0 | 0/0/6/1 | 0.50 |
| 12 | cmdcompact | command_compact.go: compactOptions.AddFlags, newCompactCommand, compactFunc | CLI glue | 1 | — | 0/0/6/2 | 0.38 |
| 13 | cmdinspect | command_inspect.go: newInspectCommand, inspectFunc | CLI glue | 1 | tx.Inspect stays in txbucket | 0/0/5/1 | 0.55 |
| 14 | cmdver | command_version.go + command_root.go + main.go: versionOptions.AddFlags, newVersionCommand, versionFunc, NewRootCommand, main | CLI wiring | 1 | other subcommand constructors banked (cmdlist etc.) | 0/0/3/2 | 0.33 |
| 15 | cmdbenchopt | command_bench.go: benchOptions.AddFlags, Validate, SetOptionValues, newBenchCommand | CLI option validation | 1 | — | 0/3/3/2 | 0.55 |
| 16 | cmdbenchstat | command_bench.go: benchResults×6, printGoBenchResult, checkProgress | stats/codec | 1 | — | 0/3/4/2 | 0.36 |
| 17 | cmdbenchwrite | command_bench.go: benchFunc, runWrites×9, startProfiling, stopProfiling | orchestration | 1 | — | 0/1/6/8 | 0.51 |
| 18 | cmdbenchread | command_bench.go: runReads×5 | orchestration | 1 | — | 0/1/4/1 | 0.34 |
| 19 | btestlife | btesting.go: MustCreateDB×3, OpenDBWithOption, PostTestCleanup, Close, MustClose, MustDeleteFile, SetOptions, MustReopen, Path, strictModeEnabledDefault, ForceDisableStrictMode | test-harness lifecycle | 1 | — | 0/4/5/1 | 0.56 |
| 20 | btestfill | btesting.go: DB.Fill, DB.MustCheck, DB.CopyTempFile, DB.PrintStats, truncDuration | test-harness fill/check | 1 | — | 0/1/4/4 | 0.52 |

## Rejected candidates (overlap)

Rejected before excision, each recorded here per the no-duplicate rule:

- `Bucket.pageNode`/`Bucket.stats`/`Bucket.forEachPage` (bucket.go) — collide
  with bucketser/bucketwalk/bucket-L0.
- `Cursor.seek`/`nsearch`/`keyValue`/`node` (cursor.go) — collide with
  cursorwalk/cursor-L0. `Cursor.Bucket` was free and folded into txbucket.
- `node.put`/`del`/`read`/`write`/`split` (node.go) — nodemut/nodewire/node-L0.
  `node.spill` was the only free node symbol (→ nodespill).
- `Tx.write`/`writeMeta`/`commitFreelist`/`Commit` (tx.go) — txwrite/tx-L0.
- `Tx.init`/`ID`/`close`/`rollback` (tx.go) — txlife.
- `DB.mmap`/`munmap`/`invalidate`/`pageInBuffer`/`freepages` (db.go) — dbmap.
- `DB.grow`/`growSize`/`allocate` (db.go) — dbgrow.
- freelist package — fully banked (flarray/flhashmap/flshared).
- `command_utils.go`, `command_surgery*.go`, `command_page*.go`,
  `command_dump.go` — cmdutils/cmdsurgery/cmdsurgfree/cmdpageitem/cmddump-L0.
- Non-Linux `bolt_*.go` platform files — cannot build/verify on linux/amd64.

Non-overlap rejection: `node.dump` — planned, then found to be dead code
inside a `/* */` comment block; replaced by splitting the bench runner into
cmdbenchwrite + cmdbenchread.

## Validation

- `check_unit_overlap_bbolt.py` (after fixing its `+++ /dev/null` hunk
  attribution — deleted test-file symbols were being credited to the
  preceding file, producing false OVERLAPs): **CLEAN for all 20 new units**
  vs sweep_bbolt + sweep_bbolt_L2 (45 existing) and pairwise. The single
  remaining OVERLAP is `bucketser` vs `bucketspill` sharing `cloneBytes` —
  a pre-existing collision inside the earlier batch, not this wave.
- Manual grep of all `authored*/*/*/_author/closure.md` for each new unit's
  file+symbol: 9 co-occurrences, all "Kept:" prose mentions (sibling surface
  intact); zero double-excisions.
- `verify_unit.sh`: 3/3 (excised, gold, cheat) build-OK for all 20.
- `cheat_validity.py`: all 20 `plausible-cheat` (ratios 0.20–0.56, threshold
  0.60). Six first-draft cheats exceeded the threshold and were shrunk by
  leaving off-path stubs as panics (txbucket 0.93→0.50, btestlife 0.68→0.56,
  cmdbenchwrite 0.60→0.51, cmdinspect 0.67→0.55, cmdbenchstat 0.61→0.36,
  dbstats 0.61→0.39).

## Search cost

Every accepted unit was accepted on attempt 1 — the up-front free-symbol
inventory made overlap checking a lookup, not a search. One candidate
(nodedump) was rejected after selection for being dead code, at a cost of
one inventory re-check. **Search cost per accepted unit was flat (1.0) across
all 20; no diminishing-returns inflection was reached.** The binding
constraint was surface exhaustion, not search cost: after this batch the
remaining unbanked bbolt surface is ~5 thin symbols (unused helpers,
non-Linux platform files, dmflakey/utils harness) — the next wave would
climb sharply.
