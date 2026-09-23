# Handoff — affordance ladder experiment, 2026-09-23

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

## What is still running

`advrefs` **L4 and L5** in cohort `sweep_devin_holes` (containers `advrefs-l4__pvpwmer`,
`advrefs-l5__t2vrwip`, driver pid 2416792). These are the last two devin cells anywhere.

They matter: devin's `advrefs` certificate is L6 with `rung_established: False` — it jumped L2 → L6
with L3/L4/L5 never measured, so "needs L6" is currently an *upper bound*, not a measurement.
These two cells settle it. No driver targeted them because the climb roster is derived from units
that failed L0 **and** L2, and `advrefs` had since flipped, so it left the roster while its inferred
cells stayed open; they were launched by hand as `sweep_devin_holes`.

Check with:

```bash
cd /home/evan/Documents/open_swe_traces_research && bash scripts/ops/devin_watchdog.sh
```

It prints `cells remaining: N   admissible now: M`. When remaining hits 0 it prints
`devin work COMPLETE`. **That is the finish line for the compute side of the experiment.**

## What is left after that

1. **Devin's cost row in the paper** needs the user's ACU export. It is not derivable from the repo:
   the repo reports 704.7M input / 664.1M cache against the table's 162M / 3,270M. Blocked on the user.
2. **Euler diagram geometry** in the paper — the numbers were updated, the block proportions are stale.
3. **Uncommitted**: `analytics/research/dashboard.html`, `outputs/supervisor/ladder_backfill`,
   untracked `docs/tech_debt_level_numbering.md`. Review and commit.
4. **`git gc`** deferred — 29k loose objects, gc.log warning on every commit.
5. **Ladder renumbering** (code L0–L6 → write-up L1–L6) deliberately deferred until Stage 1 runs
   finish, because rung keys are baked into job directory names and ledger rows. Plan is in
   `docs/tech_debt_level_numbering.md`.

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
