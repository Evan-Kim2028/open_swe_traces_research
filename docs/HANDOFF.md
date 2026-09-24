# Handoff — affordance ladder experiment, 2026-09-23 (updated 09:50)

## What this project is

A pipeline that generates synthetic SWE tasks with *controlled* difficulty, each one
simultaneously verifiable (hidden tests) and solvable (built from a real commit), in order to
locate where frontier coding agents actually fail.

Difficulty is controlled by the **affordance ladder** — the same underlying task described with
progressively more information:

| rung (code) | what the solver is given | rung (write-up) |
|---|---|---|
| `L0` | bug report only | L1 |
| `L1` | gapped contract (retired — see `docs/tech_debt_level_numbering.md`) | not shown |
| `L2` | full prose contract | L2 |
| `L3` | hidden test names | L3 |
| `L4` | signatures / stubs | L4 |
| `L5` | one restored test | L5 |
| `L6` | all tests | L6 |

A **certificate** is the lowest rung at which a solver flips from fail to pass. Certificates are
**per solver** — cross-model certificates are retired; `certificates()` derives from
`certificates_by_solver()` and `kind` is always `"single"`. `rung_established` says whether the
rung *below* has a recorded failure, which distinguishes "flips BY rung k" from "NEEDS rung k".

Solvers: **composer** (cursor composer-2.5, paid), **devin** (swe-2-max, ACU-limited),
**grok** (4.6/4.7 via grok-build auth — **out of limits, do not launch more**).

## State as of 2026-09-23 09:00

- 436 units, 242 certificates, **40 units with independent curves from two solvers** (was 2 yesterday)
- 1,799 trials (1,686 valid, 113 errored), 4.14B tokens, $704 in-repo cost
- certificates by solver: composer 207, devin 73, grok 2
- rung distribution: composer L2 157 / L3 8 / L4 2 / L5 39 / L6 1 · devin L2 61 / L3 3 / L4 2 / L5 4 / L6 3 · grok L6 2
- **Exhausted units** (no pass at any rung): `exprhash` (composer + grok), `helm-dlmanager` (devin)
- `httpencoding` is the only three-model ladder: composer exhausted, grok L6, devin L6

Devin's phase B closed the run's biggest gap — it had zero certificates above L2 yesterday morning
and now populates every rung L3 through L6.

## Stage 1 compute is finished (2026-09-23 09:46)

`devin_watchdog.sh` reports `cells remaining: 0` and `devin work COMPLETE`. No trial is running.

The last two cells were `advrefs` L4 and L5 in `sweep_devin_holes`. L5 passed at 13:02 UTC and L4
failed at 13:08, both without an exception. Devin's `advrefs` certificate is therefore **L5 with
`rung_established: True`**: 0✗ 2✗✗✗ 3✗ 4✗ 5✓ 6✓. Against Composer's L2 that is a three-rung gap,
not the four first reported, and it is a measurement, not an upper bound.

Final ledger: 436 units, 415 graded, 242 certified, 172 solved from the bug report, 1,801 trials
(1,688 with a verdict). Composer and Devin both hold certificates on 39 units: the same rung on
23, Devin lower on 11, Composer lower on 5. `uv run python -m openswe_traces.reports.paper_numbers`
prints every figure the write-up uses.

## Repository layout changed (branch `repo-tidy`)

Library code moved from `scripts/ops/*.py` into `openswe_traces.{ladder,ops,reports,analysis,
authoring}`, with a shim at every old path. See `docs/ARCHITECTURE.md`. The move was checked
by `scripts/dev/ladder_golden.py` (byte-identical certificates and 10,900 guard decisions) and
by old-vs-new runs of every read-only command. Merge it after the daemons below are stopped or
restarted, since it changes `pyproject.toml` and `uv run` re-syncs on the next call.

## What is left

1. **Devin's cost** is now computed from exact per-request counts in its session databases
   (`devin_usage --exact`): 1.69B tokens over 295 trial sessions and 2.95B over 213 host
   (authoring) sessions, about $654 at the SWE-2 promotional rate. The paper uses this. An ACU
   export would confirm the dollars; it is no longer a blocker.
2. **The per-level cost table** the paper used could not be reproduced and has been replaced with
   figures from all Composer and Grok runs ($0.78 a run above L2, $0.38 at L2).
3. **Euler diagram geometry** in the paper: the numbers were updated, the block proportions are stale.
4. **`ladder_purity` is nondeterministic on ties**: the owner and stray columns for `cronparse`,
   `httpencoding` and `reflectfmt` change with `PYTHONHASHSEED`. Pre-existing; not fixed in the move.
5. **Loose objects**: 29k, and a gc.log warning on every commit. Garbage collection is deferred.
6. **Ladder renumbering** (code L0–L6 → write-up L1–L6), now unblocked since Stage 1 runs have
   finished. Plan is in `docs/tech_debt_level_numbering.md`.
7. **Daemons** `monitor_loop.sh` and `supervisor.sh` are still up with nothing left to schedule.

## The paper

`/home/evan/Documents/evan_writings/src/writings/difficulty-is-an-information-gap.md` — Eleventy,
live on GitHub Pages. **Edit that file directly**; it is the source of truth, not a local copy.
Working tree clean and level with `origin/main` as of this handoff (HEAD `d58f8dc`).
Vale / cspell / markdownlint run on commit and will block: no sentence-initial "So", no "extremely",
periods inside quotation marks.

A side note lives at `analytics/research/NOTE_mid_ladder_dip.md` — the mid-ladder dip observation.
The user's call: *"not sure if i will add it into the final paper or not... its a really small
datapoint it might just be noise."* The experiment that would settle it was stood down: suite kind
is confounded with repository (kops 18/0 property, go-git plumbing 0/24 example, only 3
repo-matched pairs exist).

## Operational rules — violating these has cost real work

- **DO NOT KILL RUNNING DEVIN SESSIONS** unless genuinely duplicated, or stalled 30+ minutes with no
  files written. Containers that look idle are often still working.
- **Devin cap is 4 concurrent.** Enforced at one choke point: the `flock`-guarded admission gate at
  the harbor launch line in `scripts/ops/sweep_seq.sh`. Do not add a fifth caller-side check —
  patching callers instead of the choke point is exactly what caused repeated cap violations.
- **Never match a process by a pattern that appears in your own command line.** `pkill -f` on a
  cohort name killed my own shell once.
- Never delete `ladder-base:*` docker images or `experiments/pipeline/repos/*/src`.
- **No grok launches** — out of limits. `grok_ladder.py` requires `GROK_ENABLED=1` for this reason.
  Grok stays on its original tasks only; do not expand it to new ones.
- **No re-measurements.** `RUNG_TRIAL_CAP = 1` in `trial_guard.py`; 41% of trials were once
  re-measurements of cells already decided.
- Composer budget must not be raised without asking.
- Do not use OpenRouter. Grok must come from grok-build auth.
- GitHub Actions billing failures are chronic and not real breakage — ignore them and merge anyway.
- A **failed probe is not a zero**. Both `devin_watchdog.sh` and `slots.py` have been bitten by
  treating an unreadable state as "nothing running" and then starting things on top of healthy work.

## Monitoring

- `scripts/ops/monitor_loop.sh 300` — pid 198806, 19h up
- `scripts/ops/supervisor.sh 300` — pid 1275389. **Editing `supervisor.sh` does nothing to a
  running supervisor**; bash holds the loop. `watch.sh` detects this by comparing process start time
  to file mtime.
- `scripts/ops/devin_watchdog.sh` — run by hand; checks, then puts things back on course
- `scripts/ops/status_digest.sh`, `scripts/ops/devin_progress.sh <cohort>` — read-only reports

The user's standing instruction: *"please keep checking in every 45 minutes and making sure that
progress is happening and reporting and if not then getting back on course so that devin trials all
finish."*
