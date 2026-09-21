# authored_au5bbolt/bbolt — 2 new units, then the wall (batch 5, third bbolt wave)

Date: 2026-09-21. Worktree: `oswt-AU5bbolt` (branch `au5bbolt`). Pristine
source: `outputs/scratch/bbolt_gold` (`example.internal/boltstore`, go1.26.8,
copied from the AU4 worktree). Authoring tools: `scripts/ops/author_excise.py`,
`make_cheat.sh`, `verify_unit.sh`, `scripts/ops/cheat_validity.py`,
`scripts/check_unit_overlap.py` (new: auto-detects author-layout and
task-layout bank dirs), `scripts/free_symbols_bbolt.py` (new: mechanical
banked-vs-free inventory).

## The headline: the bank is full

`free_symbols_bbolt.py` lists every function in the linux/amd64 build set of
the pristine tree (`go list ./...`, 560 funcs across 49 files) and subtracts
every symbol carrying an `excised:` marker or a patched func decl in the
existing bank — dose_response `sweep_bbolt` (10), `sweep_bbolt_L2` (21),
`sweep_loop` bbolt units, and `authored_batch4/bbolt` (38) — 98 existing
closures covering 538 symbols.

Nominally free: 22. Actually authorable: 7, in two closures.

| remainder | verdict |
|---|---|
| `node.dump` (node.go:499) | rejected — dead code inside a `/* */` block, not a live symbol; excising it is a no-op patch |
| `tests/utils: init` | rejected — a panic stub in `init` poisons every test binary that imports the package; mechanically unexcisable |
| `tests/utils: RequiresRoot` | **accepted → testroot** |
| dmflakey `WithIntervalFeatOpt`, `WithSyncFSFeatOpt`, `validateFSType`, `flakey.DevicePath`, `flakey.Filesystem`, `createEmptyFSImage` | **accepted → dmfopts** |
| dmflakey `InitFlakey`, `AllowWrites`, `DropWrites`, `ErrorWrites`, `Teardown`, `createEmptyFSImage`-tail, `newFlakeyDevice`, `reloadFlakeyDevice`, `deleteFlakeyDevice`, `getBlkSize64`, `getBlkSize`, `attachToLoopDevice`, `detachLoopDevice`, `getFreeLoopDevice` (14) | rejected — unmeasurable: in a container without root/dmsetup/loop devices, gold's only observable behavior is `return error`, which a bare stub also produces. The interesting logic (dm table strings, EBUSY retry, ENXIO tolerance) has no injection point — Go package funcs can't be stubbed by tests. A unit whose entire contract is "fails without devices" certifies the image, not the code — worse than TOO-EASY, it is unfalsifiable as implemented-vs-stubbed. |
| non-linux `bolt_*.go` platform files | rejected (as batch4) — cannot build/verify on linux/amd64 |
| `*_test.go` helpers | rejected — outside excision granularity; the checker drops test files from overlap tracking by convention, and a panic stub in a shared helper fails every unrelated test |
| `version/version.go`, `errors/`, `internal/common/types.go`, `bolt_amd64.go`, `doc.go` | no functions — types/consts/vars only |

The closest call was `getBlkSize64`: `BLKGETSIZE64` on a regular file fails
with ENOTTY on any kernel, so "returns error on non-block-device" is
genuinely deterministic. Rejected anyway — a `return errors.New(...)` cheat
satisfies it identically; nothing distinguishes implementation from stub.

## Units in authoring order

| # | unit | closure (file / symbols) | surface | attempts | overlap rejections | Inferable (yes/doc/part/no) | cheat ratio |
|---|------|--------------------------|---------|----------|--------------------|-----------------------------|-------------|
| 1 | dmfopts | tests/dmflakey/dmflakey.go: WithIntervalFeatOpt, WithSyncFSFeatOpt, validateFSType, flakey.DevicePath, flakey.Filesystem, createEmptyFSImage | pure options + predicate + validation ordering | 1 | none (file untouched by all 98 banked units) | 0/4/4/1 | 0.36 |
| 2 | testroot | tests/utils/helpers.go: RequiresRoot | exit-path predicate | 1 | none | 0/1/3/1 | 0.36 |

Attempts = distinct candidate closures evaluated before the accepted one was
confirmed disjoint. Both accepted on attempt 1 because the free-symbol
inventory was computed before any excision — same trick as batch4.

## Overlap check

`uv run python scripts/check_unit_overlap.py experiments/pipeline/authored_au5bbolt/bbolt <sweep_bbolt> <sweep_bbolt_L2> <sweep_loop> <AU4 authored_batch4/bbolt>`:
**CLEAN**, 2 new vs 98 existing, zero shared files (not even
symbol-disjoint ones — the new units are the first to touch `tests/`).
Manual grep of all `authored*/*/*/_author/closure.md` for
`dmflakey|RequiresRoot|tests/utils`: hits only inside this batch.

## Validation

- `verify_unit.sh`: excised / gold / cheat all `go build` clean; gold
  restores byte-exactly — 3/3 for both units.
- `cheat_validity.py` (via tests/-layout symlink shim): both
  `plausible-cheat`, ratios 0.36 and 0.36.
- testroot deletes 3 test files (`dmflakey_test.go`,
  `robustness/{main,powerfailure}_test.go`) — their `TestMain` calls the
  excised `RequiresRoot` unconditionally; the deletion loses nothing since
  those suites exit 0 without `-test.root` anyway.

## Search cost

Flat 1.0 for the two accepted units, then a cliff: there is no attempt-3
candidate, because there is no unclaimed symbol left that a hidden suite
can measure. The inflection batch4 predicted ("the next wave would climb
sharply") is here — the marginal cost of unit 3 is not higher, it is
infinite. **Stopping at 2 of 20 requested.** The remaining free symbols
aren't sparse-but-real, they're structurally unauthorable: dead code, a
package init, and a device harness whose contract is owned by the kernel
and the container image, not by the code.

If more bbolt units are wanted, the honest move is not finer excision but
a different lever: port the pipeline's reachability story to a repo with
remaining predicate surface, or re-open bbolt at a different version where
the bank doesn't already own the tree.
