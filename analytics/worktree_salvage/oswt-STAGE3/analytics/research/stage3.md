# Stage 3 — staging verified batch-3 units into Harbor task directories

Date: 2026-02-20 (run on `stage3` branch, worktree `oswt-STAGE3`)

## What was built

`scripts/ops/stage_units.py <src_authored_dir> <dest_stage_dir> --rungs 0,2`
(logic in `src/openswe_traces/synth/stage_units.py`) converts authored
`_author/` units into Harbor task directories:

    <unit>-L0/  and  <unit>-L2/
      affordance.json  instruction.md  validation.json  task.toml
      patches/{gold,cheat}.patch  tests/{test.sh,measure_gold.sh,hidden/*_test.go}
      environment/Dockerfile  environment/src/...

Emission is idempotent: a complete directory is left untouched on re-run; an
incomplete one is rebuilt.

## Staged output

| repo | source authored dir | L0 dest | L2 dest | units |
|---|---|---|---|---|
| kops | `oswt-VF3kops/…/authored_batch3/kops` | `experiments/dose_response/sweep_kops3/` | `experiments/dose_response/sweep_kops3_L2/` | 20 |
| go-git | `oswt-VF3gogit/…/authored_batch3/go-git` | `experiments/dose_response/sweep_gogit3/` | `experiments/dose_response/sweep_gogit3_L2/` | 20 |

80 task directories total. Base images `ladder-base:kops`, `ladder-base:go-git`
confirmed present before emission.

## Load-bearing decisions and deviations

- **`environment/src` = pristine `/app` + forward `excision.patch`, not
  reverse-applied gold.** 18/20 kops and most go-git excision patches *delete*
  in-tree test files (`+++ /dev/null` hunks) that `gold.patch` does not restore
  (e.g. `certdesc` deletes `issue_test.go`, `certificate_test.go`; `inforefs`
  deletes 23 files). Reverse-applying gold would leave the deleted tests in the
  tree — a leak, and not the tree VF verified. Forward excision reproduces
  exactly the tree the hidden suite fails on.
- **`affordance.json.level` = directory rung** (0 for `-L0`, 2 for `-L2`). The
  authored internal levels (-2/0) and the historical `-L{k+2}` convention were
  deliberately NOT copied; directory name is authoritative.
- **L0 `instruction.md` = `bugreport.md` + no-web clause; L2 = `contract.md` +
  no-web clause.** Contracts never appear in L0.
- **`task.toml` allowlist unchanged** from authored (cursor hosts only).
- **Hidden suite bytes copied verbatim** from the verified `_author/tests/`.
- **`git apply --whitespace=nowarn` used for in-container patch application.**
  `ladder-base:go-git` has no `patch` binary; first go-git verify failed all 20
  units on `gold_restore` for this reason. `git apply` exists in all base
  images and is what VF's `verify_hidden.sh` uses.
- **`test.sh` checksum guards refreshed for 13 go-git units.** VF edited the
  hidden files after generating `test.sh`, leaving embedded sha256 digests
  that match neither the shipped hidden file nor its post-gold copy. The
  staging tool recomputes the digest over the shipped bytes and patches the
  `sha256sum -c` lines; hidden test content itself is untouched. Affected:
  `cgenc filechange indexenc inforefs modconfig negside objfile pathutil
  reflog reportstatus revfile unidiff updreq` (13; the other 7 matched).
  Kops: 0 refreshes needed.

## Docker verification (every staged dir)

Per unit, in Docker against `environment/src`: `buggy_fails` = excised+hidden
suite FAILs; `gold_restore` = gold.patch → suite PASSes (reward 1);
`cheat_rejected` = cheat.patch → suite still FAILs; `blackbox_hygiene` = gold
patch touches no `*test*` path (A12). `-L0` dirs were verified directly;
`-L2` dirs share byte-identical `environment/` + `tests/` (verified via
`diff -r`) so the proven results carry over.

**Result: 80/80 dirs — all four checks pass (40 direct, 40 by byte-identical
inheritance).**

### kops (sweep_kops3, sweep_kops3_L2)

| unit | n_props | buggy_fails | gold_restore | cheat_rejected | blackbox_hygiene |
|---|---|---|---|---|---|
| certdesc | 6 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| distros | 8 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| fidownload | 6 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| fieldmap | 6 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| fieldpath | 11 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| filemodes | 4 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| fivalues | 8 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| iamsubj | 7 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| igrole | 8 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| jsonstream | 10 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| kopscodecs | 9 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| reflectfmt | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| sshfinger | 5 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| strvals | 13 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| tablesfmt | 7 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| tfhcl2 | 14 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| tfliterals | 13 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| tfwriter | 10 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| vfspaths | 8 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| zonespec | 10 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |

### go-git (sweep_gogit3, sweep_gogit3_L2)

| unit | n_props | buggy_fails | gold_restore | cheat_rejected | blackbox_hygiene |
|---|---|---|---|---|---|
| binio | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| capability | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| cgenc | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| filechange | 10 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| indexenc | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| inforefs | 11 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| lsrefs | 13 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| modconfig | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| negside | 10 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| objfile | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| pathutil | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| reflog | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| reportstatus | 10 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| reqframe | 13 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| revfile | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| sigblock | 10 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| srvresp | 10 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| tagparse | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| unidiff | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |
| updreq | 12 | FAIL ✓ | PASS ✓ | FAIL ✓ | clean ✓ |

## Free gates (reported, nothing dropped)

### task_lint.py

- `sweep_kops3` (L0): **0 of 20 blocked.** All clean.
- `sweep_kops3_L2`: **0 of 20 blocked.** Every unit carries INFO findings —
  `COVERAGE` (contract commitment tables are thinner than hidden assertion
  counts, e.g. certdesc "6 contract commitments for 22 hidden assertions"),
  several `UNGROUNDED` (example literals absent from the hidden suite),
  `B7` (file/line names in an L2 contract), and one `A3-SURFACE` on
  `fidownload-L2`.
- `sweep_gogit3` (L0): **0 of 20 blocked.** All clean.
- `sweep_gogit3_L2`: **1 of 20 blocked — `binio-L2`.**
  - `[BLOCK] SCRUB: unresolved symbol-scrub placeholder "the call" in the
    prose` — the contract's TestDetail05 row reads "…or the call is still
    running at timeout". This is descriptive prose, not an actual placeholder,
    but the gate's regex cannot tell; `binio-L2` will be dropped at sweep time
    unless the contract wording is changed. Also INFO `COVERAGE` 14/51 (27%),
    INFO `UNGROUNDED` (5 literals), WARN `B7`.
  - The other 19 gogit L2 units: WARN/INFO only — same pattern as kops
    (`COVERAGE` ~27–38%, `UNGROUNDED`, `B7` — most contracts cite
    `tests/hidden/…/*_bb_test.go` paths and `line N` references).

Nothing was dropped or edited in response to gate findings.

### trial_guard.py

- All 40 `-L0` dirs (both sweeps): `RUN — verdict still open`.
- All 40 `-L2` dirs (both sweeps): `SKIP — L2 before L0 — run L0 first, it is
  the cheaper verdict`. Expected: the trial ledger has no L0 verdicts yet; L2
  unlocks only after its L0 posts a verdict.

## Units not staged (34 of 74 authored batch-3)

| repo | authored units | reason |
|---|---|---|
| goa | 13 | no verified hidden suite — no `tests/hidden`/`test.sh` exists in any worktree |
| go-github | 21 | same — VF job has not landed for this repo |

These remain in `…/authored_batch3/{goa,go-github}/` awaiting their VF jobs.

## Reproduce

    uv run python scripts/ops/stage_units.py \
        /home/evan/Documents/oswt-VF3kops/experiments/pipeline/authored_batch3/kops \
        experiments/dose_response/sweep_kops3 --rungs 0,2 --jobs 4
    uv run python scripts/ops/stage_units.py \
        /home/evan/Documents/oswt-VF3gogit/experiments/pipeline/authored_batch3/go-git \
        experiments/dose_response/sweep_gogit3 --rungs 0,2 --jobs 4
    uv run python scripts/ops/task_lint.py --batch experiments/dose_response/sweep_kops3_L2 --quiet
    uv run python scripts/ops/trial_guard.py certdesc-L0

Verify images are tagged `stage3-<unit>-l<rung>`. Nothing committed.
