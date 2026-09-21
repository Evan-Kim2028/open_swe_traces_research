# Verified hidden suites — authored_xrepo2 (15 cross-repo units)

Date: 2026-09-21. Worktree: `oswt-XREPO2`. Log: `outputs/VFXREPO2.log`.

Method: per unit, black-box `tests/hidden/**/*_bb_test.go` with one `TestDetailNN` per
`DETAILS.md` commitment, checksum-guarded `tests/test.sh`, and `_author/contract.md`
with a bidirectional coverage table. Suites were written from `DETAILS.md` + `api.md`
and excised-tree behavior only (`experiments/xrepo2/work/<pair>/scratch`); `gold.patch`,
`cheat.patch`, and `bugreport.md` were not used as grading sources.

Preflight (final run): `uv run python scripts/ops/stage_units.py
experiments/pipeline/authored_xrepo2 experiments/dose_response/sweep_xrepo2 --rungs 0,2
--jobs 2 --units <unit> --image ladder-base:<pair> --xrepo2-work experiments/xrepo2
--force`. Pristine trees come from `experiments/xrepo2/work/<pair>/scratch` (not
ladder-base `/app`), via `--xrepo2-work`.

## Per-unit summary

| unit | commitments | TestDetail count | coverage rows | preflight (L0) |
|------|------------:|-----------------:|--------------:|----------------|
| chrootbind | 9 | 9 | 9 | PASS (buggy_fails, gold_restore, cheat_rejected) |
| errtrace | 11 | 11 | 11 | PASS |
| fsutil | 9 | 9 | 9 | PASS |
| godifflines | 9 | 9 | 9 | PASS |
| jwtdecode | 8 | 8 | 8 | PASS |
| jwtmigrate | 7 | 7 | 7 | PASS |
| memdborder | 11 | 11 | 11 | FAIL gold_restore (Docker); stage_units validation KeyError fixed |
| memfsfile | 8 | 8 | 8 | PASS |
| memfsstore | 9 | 9 | 8 | PASS |
| nuid | 6 | 6 | 6 | PASS (after suffix increment-band check added) |
| osbound | 8 | 8 | 8 | FAIL cheat_rejected (cheat patch passes consumer suite) |
| securejoin | 8 | 8 | 8 | PASS |
| strkey | 7 | 7 | 7 | PASS |
| uitable | 9 | 9 | 6 | PASS |
| xkeys | 8 | 8 | 8 | PASS |

**13/15** units pass all three harness checks on the final preflight run. **memdborder**
fails `gold_restore` in Docker (suite vs gold mismatch — needs verifier fix, not gold).
**osbound** fails `cheat_rejected`: the
shipped cheat reimplementation still passes the consumer suite (including verbatim
`Readlink` for `./r.txt`); a stronger deps-level or escape-translation assertion is
still needed.

## Assertions not fully pinned (derivable / refused)

- **errtrace**: juju predicates (DETAILS 10) — substring convention asserted only negatively on sentinels; `Cause`/`Unwrap` depth (DETAILS 5) and `AddStack` dedup (DETAILS 6) not isolated in dedicated tests (partially covered via `%+v` shapes).
- **fsutil**: `Walk` (DETAILS 5) — no direct go-git `Walk` call site; glob/ordering commitments exercised via `AddGlob` / `RemoveAll`.
- **jwtmigrate**: activation/authorization activation (DETAILS 5) — no separate server hook; migration outcomes covered indirectly.
- **memfsstore**: `TempFile`/`TempDir` umask (DETAILS 8–9) — not reachable through worktree FS; temp-dir behavior approximated via create-parent paths.
- **uitable**: wrap / multiline / right-align (DETAILS 2–4) — not observable from helm `pkg/cmd` black-box commands; table layout tested on list/history/search paths only.
- **memdborder**: MVCC scan edge cases — asymmetric ranges covered; not every DETAILS line has an isolated micro-test (grouped by consumer scenario).

## L0 solve resistance (belief)

**14 of 15** should resist an L0 solve from the bug report alone: each unit excises
behavior that is arbitrary without call-site context (path rebasing, error `%+v` layout,
line-diff pipeline, JWT wire rules, MVCC ordering, NUID increment band, etc.). Round-1
failure mode was *memorized famous utilities*; round-2 dependencies are obscure forks
(`deps/errors`, `deps/billy`, `deps/jwt`, `deps/nuid`, helm securejoin) with
project-specific rules.

**Weakest L0 risk: osbound** — once the verifier accepts the cheat patch, the consumer
surface may still be too coarse for `BoundOS`/`os.Root` semantics; tightening cheat
rejection is required before trial. **Strongest resistance: errtrace, godifflines,
jwtdecode, memdborder, strkey** — many `Inferable: no` commitments with tight shape
tests and no name-level recall shortcut.
