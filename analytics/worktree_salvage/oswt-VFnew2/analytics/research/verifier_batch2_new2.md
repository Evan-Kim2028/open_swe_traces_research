# Verifier job: hidden black-box suites for bbolt batch-2 (worktree oswt-VFnew2)

Date: 2026-09-20. Role: VERIFIER AUTHOR — authored from `api.md` + `DETAILS.md` +
black-box probing only; `bugreport.md`, `contract.md`, `gold.patch` never read.

Result: **25/25 units packaged, 50/50 task dirs pass the consolidated gate,
205 `TestDetailNN_*` property tests — exactly one per numbered `DETAILS.md`
line. Zero exclusions; overlap check clean (0 overlaps).**

## Environment

- Base image `ladder-base:bbolt` (Go 1.26 toolchain, builds the module).
- Staging: `scripts/vf_bbolt_stage.py` materializes excised/gold/cheat trees
  per unit under `experiments/pipeline/work/vf_bbolt_{excised,gold,cheat}`.
- Dev loop: `scripts/vf_bbolt_dev.sh <unit> [gold|excised|cheat] [seed]` —
  mounts the staged tree + `work/vf_bbolt_hidden/<unit>` into the image,
  sets `HIDDEN_SEED` (default 20260919, audit 20260920, gate alt 7301989).
- Packaging: `scripts/vf_bbolt_package.py` →
  `experiments/pipeline/tasks_batch2/bbolt/<unit>-L{0,2}` (50 dirs).
- Gate: closure-O `scripts/gate_tasks.py` (exec_rules + static_rules);
  matrix at `outputs/gate_matrix_bbolt.json`, log `outputs/VFbbolt.log`.

## Per-unit results

| unit | details | tests | L0 gate | L2 gate | L0 s | L2 s |
|---|---|---|---|---|---|---|
| bucket | 15 | 15 | pass | pass | 44 | 45 |
| cmddump | 7 | 7 | pass | pass | 56 | 40 |
| cmdget | 6 | 6 | pass | pass | 48 | 43 |
| cmdpage | 8 | 8 | pass | pass | 56 | 39 |
| cmdpages | 6 | 6 | pass | pass | 37 | 31 |
| cmdsurgerymeta | 9 | 9 | pass | pass | 34 | 44 |
| cmdutils | 9 | 9 | pass | pass | 43 | 52 |
| compact | 6 | 6 | pass | pass | 64 | 94 |
| cursor | 10 | 10 | pass | pass | 73 | 32 |
| db | 10 | 10 | pass | pass | 35 | 28 |
| flarray | 8 | 8 | pass | pass | 37 | 42 |
| flhashmap | 8 | 8 | pass | pass | 39 | 34 |
| flshared | 12 | 12 | pass | pass | 48 | 47 |
| gutscli | 7 | 7 | pass | pass | 36 | 38 |
| inbucket | 5 | 5 | pass | pass | 35 | 34 |
| inode | 7 | 7 | pass | pass | 26 | 24 |
| loadutil | 5 | 5 | pass | pass | 33 | 32 |
| meta | 8 | 8 | pass | pass | 23 | 28 |
| node | 8 | 8 | pass | pass | 30 | 26 |
| page | 12 | 12 | pass | pass | 24 | 23 |
| surgeon | 8 | 8 | pass | pass | 31 | 44 |
| tx | 11 | 11 | pass | pass | 55 | 48 |
| txcheck | 8 | 8 | pass | pass | 63 | 79 |
| verifyenv | 6 | 6 | pass | pass | 77 | 42 |
| xray | 6 | 6 | pass | pass | 50 | 50 |
| **total** | **205** | **205** | **25/25** | **25/25** | — | — |

Seconds = wall-clock of the last full (non-cached) gate evaluation per dir.

## Gate verdicts (uniform across all 50 dirs)

- **A1** gold restores pass ×2 in-image (`REWARD=1, REWARD=1`).
- **A3** cheat patch fails (`REWARD=0`, panic/assertion or behavior mismatch).
- **A5** stable across repeats and all three seeds
  (20260919, 20260920, 7301989): bare=fail,fail; gold=pass,pass; cheat=fail.
- **A8** bare excised tree fails ×2 through the suite — `panic: excised: <sym>`
  recovered by the test harness, never setup/build/import errors.
- **A10** gold passes with `docker --network=none`.
- **A12** no test-file hunks in gold/alt/cheat patches.
- **B1–B8** all pass: checksum-guarded instructions, `go test` reproducer,
  symptom keywords, no excised-symbol/file/line leaks, seeded-property
  markers present.
- **coverage** every `TestDetailNN_*` wired into `tests/test.sh` (`go test -v`)
  and declared exactly once in `validation.json` + L2 instruction table.
- **nesting** L2 instruction strictly contains L0's information.

## Authoring method

All assertions go through the documented API surface (`api.md`), driven with
`rand.New(rand.NewSource(HIDDEN_SEED))` inputs so a hardcoded-worked-example
solution cannot pass (A3 enforced empirically: cheat fails ≥1 detail per unit,
usually most). Real DB fixtures are built through `bbolt.Open`/`bolt.Open`
itself; page-level expectations are read back via the exported
`common.LoadPage`/`LoadPageMeta` aliases rather than assumed internals.

Notable semantics pinned by black-box probing (not source reading):

- **flarray/flshared**: `Rollback` restores pending pages that carry an alloc
  record to `allocs` and drops unallocated pending entirely; self-alloc free +
  `Rollback` panics; release-extent rule is `(r_i, r_{i+1}]` on both `tid` and
  `alloctx`. `FreelistPageIds` strips the slot-0 extended-format marker.
- **bucket/cursor**: `ForEach` hands raw bucket-entry values (not nil);
  after `Delete` the cursor sits *on* the successor so `Next` skips it;
  `Stats` only sees committed children; `KeyN` counts bucket entries too.
- **tx/txcheck**: `Page` on negative ids returns `(nil, nil)`; out-of-range
  positive ids panic; `WithPageId` changes the traversal root (skipped
  corruption surfaces as unreachable-not-freed instead of its type error);
  invalid page type aborts the scan after one error.
- **cmd units**: cobra-layer sentinels (`ErrPageIDRequired`,
  `ErrInvalidPageArgs`, `ErrPathRequired`) live on `newXCommand()`, not the
  `xFunc` bodies; `ErrBucketRequired` is unreachable (empty name →
  `ErrBucketNotFound`); `CmdKvStringer` renders `%q`-safe printable strings
  raw and hex for binary.
- **cmdsurgerymeta**: `updateMetaField` normalizes
  magic/version/flags/checksum unconditionally and is idempotent —
  "Nothing changed!" = updating an already-normalized meta with no new
  fields; meta-1 updates read page size through the meta-0 chain and warn.
- **node**: FillPercent is observable through `BucketStats.LeafPageN`
  (~2× leaf pages at half-fill), the only honest black-box split signal.

## Defects found and fixed during the gate

1. `cmddump` A5 flake: 16-periodic filler produced a second collapsed run in
   hexdump output; replaced with per-line-distinct data.
2. `txcheck` seed flake: detail 6 assumed a fixed "clean" alternate start —
   map iteration made it flaky; replaced with a seeded pick over the set of
   alternate leaves that demonstrably skip the corrupted page.
3. B7 leaks: instruction prose named excised symbols. Sanitizer rewords
   camelCase to lowercase words (`printPage`→"print page") and maps
   lowercase-only symbols to non-matching words (`init`→"initialization",
   `release`→"releasing", `walk`→"walking", …).
4. coverage: (a) `validation.json` rows must put the test name in
   `property` (it shadows `test`); (b) contract `|` rows name upstream tests
   that don't ship — `_fix_coverage_table` rewrites them positionally to the
   hidden `TestDetailNN_*` names.
5. B4: the six `cmd/bbolt/command` suites call documented-unexported
   internals by design (`api.md` surface is lowercase) →
   `blackbox_hygiene` annotation in `validation.json`.
6. B5: `cmdpage`/`txcheck` lacked gate-recognized seed markers; both now
   funnel through `rand.NewSource(HIDDEN_SEED)`.
7. B6: appended a "Reproduce with: `tests/test.sh` … Expected/got" block so
   the instruction names `go test` + symptom keywords.
8. nesting: L2 instruction = contract + full bugreport (info monotonicity).

## Exclusions

None. All 25 authored units shipped; overlap check reported 0.

## Reproduce

```bash
# rebuild task dirs from hidden suites + author artifacts
uv run python scripts/vf_bbolt_package.py
# full gate (bare x2 / gold x2 / cheat x1 / alt seed / netdeny per dir)
cd /home/evan/Documents/oswt-closureO
uv run python scripts/gate_tasks.py \
  /home/evan/Documents/oswt-VFnew2/experiments/pipeline/tasks_batch2/bbolt \
  --out /home/evan/Documents/oswt-VFnew2/outputs/gate_matrix_bbolt.json
```

Executed-run verdicts are content-key-cached in each dir's `validation.json`;
a rerun with unchanged hidden tests re-evaluates statics only.
