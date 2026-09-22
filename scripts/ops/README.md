# ops modules

Shared code. Import these rather than re-deriving; each one exists because the same
logic was written several times, the copies drifted, and the drift cost trials.

| module | owns | the bug that created it |
|---|---|---|
| `agents.py` | every headless connector: keys, model slugs, harbor kwargs, **required egress hosts**, per-agent quirks | Devin's host requirement lived in `sweep_seq.sh` while the allowlist lived in `affordance.py`. Nothing joined them: 98 of 102 trials died in an hour on `Unknown model`, which is what a blocked host looks like from outside. |
| `agent_session.py` | the only launcher for a headless session | `grok_job.sh` and `cursor_job.sh` each re-derived the PTY dance, key file and model slug. Refuses grok in a git worktree, where it hangs with 148 bytes and no deliverable. |
| `slots.py` | who occupies a solver slot (**sessions + in-container trials**) | Counted five ways in five files. dq2 saw sessions only and started more on top of 3 trials: 7 against a cap of 4, while every file reported "under 4". |
| `roots.py` | unit discovery across the main checkout and all `oswt-*` worktrees | Globbed in 11 files with 11 answers. The census took the max across roots and said "20/20 verified" while the stager read main only and saw nothing — 74 finished units invisible for hours. |
| `trial_ledger.py` | trials, rewards, classification | Reward is nested at `verifier_result.rewards.reward`; a trial is a DIRECTORY. Reading it wrong inflated trials 869→1632. |
| `trial_guard.py` | may this unit be trialled at this rung | Settled verdicts outrank contract staleness, or a contract rewrite resurrects decided units — 8 certified units re-ran 10-12x each. |
| `reclaim_disk.py` | deleting staged `environment/src` for units whose rung already has a verdict | 11,115 of these trees held 142GB — more than half of everything on disk — and every one is derived from `ladder-base` + a patch. Four guards (settled / not live / sweep quiet / never `repos/*/src`) and a per-unit manifest, because the cheap version deletes the ~90-deep queue waiting behind the Devin cap. |
| `restore_env_src.py` | rebuilding what `reclaim_disk.py` removed | `regen_env_src.sh` alone is wrong: 18 of 20 batch-3 kops excisions delete in-tree `*_test.go` that gold never restores, so gold-reverse leaves them present. The manifest's `excision_deleted` list closes the gap. Round-trip verified bit-identical on kops/certdesc-L0. |
| `worktree_gc.py` | retiring spent `oswt-*` worktrees | 103 worktrees held 95GB, all idle. 88 of them also carried 445 untracked one-off scripts and per-batch notes that existed nowhere else — so salvage runs first, and `git worktree remove` keeps every branch and commit. |
| `docker_gc.py` | Docker images, containers and build cache | `docker system prune -a` would take the 9 `ladder-base` images and the `ladder-base-gocache` volume, which `docker system df` calls "99% reclaimable" only because nothing mounts it at rest. It is the warm Go build cache. |
| `escalate.py` | which rung a unit that failed L0 **and** L2 should try next, and staging it | 48 units were written off as "non-flipping". Nine had been escalated by hand and **all nine flipped higher up** (11 passes, 2 fails). They were the hardest tasks in the dataset, not broken ones. Probes L5, then bisects for the lowest flipping rung: ~2.6 trials/unit against 4 for a linear climb. |
| `composer_budget.py --set` | granting an allowance and re-anchoring the baseline in one atomic step | Every allowance was a hand-edit of the JSON, and the first one overshot by 23% because raising the cap and re-anchoring were separate actions. |
## Rules learned the hard way

- **Never edit a running bash script.** Bash reads by byte offset; an edit mid-run kills it
  with a syntax error after the gate has already passed. Python is safe (loaded at import).
  Sweeps run from a content-addressed frozen copy for this reason.
- **Copy-and-rename protects the running process, which is exactly why it defers your
  change.** Writing a new file and `mv`-ing it over the original is the safe way to edit a
  live script — the running bash keeps its original inode and never sees a torn file. It
  also never sees the fix. `supervisor.sh` ran for 18 hours on a deleted inode while five
  corrections sat on disk doing nothing, including the one that made the dashboard and
  `outputs/rolling.jsonl` report 140 certificates instead of 182. Child processes it
  invokes (`uv run python ...`, `bash reap_wedged.sh`) always pick up current file content,
  so the symptom is partial: the new script runs, with the old arguments. Check with
  `readlink /proc/$(pgrep -f 'supervisor[.]sh')/fd/255` — a `(deleted)` suffix means every
  edit since launch is inert. A long-lived daemon needs a RESTART, not just a careful write.
  Restart it mid-`sleep`, kill by explicit PID, and write the new pid to
  `outputs/supervisor/pid`.
- **A host an agent needs but a task does not allow is invisible.** It presents as a bad
  model slug. `agents.egress_hosts()` is the union; `pipeline_health` asserts staged tasks
  match it.
- **Killing a harbor run does nothing** — its `sweep_seq` parent relaunches it in seconds.
  Kill the chain (`sh` wrapper → `sweep_seq` → harbor) and drop the cohort lock.
- **Never kill a Devin session**; trim trials instead. A killed session restarts from scratch.

- **Disk is mostly cache, and the cache is regenerable — but only if you record how.**
  `environment/src` rebuilds from `ladder-base` + `gold.patch` reversed, *plus* the list of
  paths the excision deleted. Two of those three live on disk already; the third only
  exists if something wrote it down before deleting the tree.
- **`docker system df`'s "reclaimable" column is not advice.** It called a 32.7GB warm Go
  build cache 99% reclaimable because no container had it mounted at that instant.
- **`find` here is bfs, not GNU findutils.** `-newermt '-45 minutes'` is a hard error in
  bfs; with stderr swallowed it yields an empty result, which reads as "nothing written
  recently". That inverted two guards: `supervisor.sh`'s "cohort still being written" check
  silently never fired, and `kill_stale_harbor.sh` would have killed every harbor job it
  inspected, including ones writing a file a second. Use `-mmin -N`, or `-newermt @<epoch>`.

## added 2026-09-21/22 — multi-model, second screening, runaway reaping

| module | owns | the bug that created it |
|---|---|---|
| `solver_match.py` | may THIS solver trial this unit — the no-cross-certification rule, and the agent→solver name map | Cross-solver certificates reached 31 of 212 because L0 screening and L2 certification were routed by whoever had a free slot. Separately, `AGENT_TO_SOLVER` had no `grok-build` entry, so a grok sweep asked the guard "may **composer** screen this unit", was correctly told no, and ran 1 of 3 units. An unmapped agent now warns instead of silently defaulting. |
| `second_screen.py` | re-screening a unit at L0 with the solver that never saw it, and reporting how often difficulty is model-specific rather than a property of the task | Composer ran 447 L0 trials to Devin's 51, so "these tasks are hard for frontier agents" rested on one model's opinion per unit. Built to CONDEMN units and relabelled when condemnation went per-solver: a second screener's pass now withdraws nothing, it starts that solver's own curve. `--report` refuses to extrapolate a pooled rate when families disagree (go-github 8/10 vs 3/32 elsewhere, p=5e-5; the pooled 26% predicts ~59 affected units against ~21), and flags that the rate is one-way — Composer is the L0 failer in 41 of 42 pairs, so the reverse direction is unmeasured. |
| `ladder_purity.py` | is each unit's escalation ladder the work of one model, and how many units have two independent curves | `solver_match` only asks whether a solver failed *below* the rung, not whether one model did every rung. Three units carried a stray rung from the other solver. Now a coverage report: `defval` needs L5 for composer and L2 for devin — the pooled ledger recorded L2, understating it three rungs. |
| `reap_runaway.py` | killing a runaway agent COMMAND without killing the trial | `reap_wedged` requires 0% cpu, no logs and no network, so it is structurally blind to a process burning every cycle. Composer issued `grep -r pattern /` and two trials sat on one tool call for over an hour each. 23 of 1347 trials did this and cost 5.2x the median — ~96M tokens. Root-anchored searches get a 120s floor, everything else 1800s, and SIGTERM hits the command so the agent reads a failed call and carries on. |
| `quarantine_trial.py` | removing a verdict when the INFRASTRUCTURE failed, not the agent | `httpresp` had its certificate settled by a trial that spent 1h37m of a ~100 minute budget blocked on a hung grep. A spurious failure at rung k pushes the unit up the ladder and overstates the one quantity this dataset measures. Moves the dir out of the ledger's glob with a reason file; `--restore` puts it back. |
| `budget_forecast.py` | does the remaining budget cover the work still OWED | A rate-based runway alarm fired at "1.8h left" while the finite escalation queue was draining fastest — loudest exactly when the work was closest to done. Subtraction, not extrapolation: rungs owed, priced from observed history, on median **and** mean because the tail is heavy (L3's mean is 2.3x its median). |
| `watch.sh` | one monitor pass; prints only on change or alarm | Sixty lines of inline bash re-pasted into a new Monitor every 30 minutes. `slots.py --supervisor-pid` did not exist, so `tr -dc 0-9` turned the human summary into a plausible PID and both supervisor checks were dead while looking healthy. Includes the stale-supervisor check: process start time vs `supervisor.sh` mtime. |
| `status.py` | HEALTH / QUEUE / LADDER in one place — earned vs by-jump rungs and what each unit needs next | "Which units still owe a rung" was re-derived by hand every time it was asked, and answered wrongly twice. |
| `preaudit.py` | contract audits off the critical path, single-instance locked | Auditing 12 contracts takes minutes and `orchestrate` runs every tick; inline it starved sweep launches. |
| `grok_build_patch.py` | making harbor's `grok-build` authenticate from this machine's `grok login` | The agent accepts only `XAI_API_KEY`, a real xAI key. This box has a grok.com session token, which `api.x.ai` takes as a raw bearer but the CLI rejects as that variable — verified both ways. 21 of the first 26 grok trials errored on it and were filed as capacity. Injects `$GROK_AUTH_JSON` to `~/.grok/auth.json` inside the container. **Edits site-packages: `uv tool upgrade harbor` silently reverts it**, so `sweep_seq` refuses to launch a grok sweep when `--check` fails. |
| `devin_usage.py`, `devin_tokens.py` | Devin's own accounting | Harbor leaves the top-level token fields `None` for Devin and fills only `agent_result.model_usage`, so all 161 devin trials scored zero and every token figure in this project was composer-only. Devin had used 545M. |
| `repair_b6.py` | adding a reproduce command to an L0 bug report | Ten units BLOCKed on B6 and orchestrate relaunched their cohort every tick, dropping all ten. Strips the `-run '^(TestDetail01|...)$'` filter so hidden test names never leak into an L0. |
| `cohorts.py` | `barren()`, shared by orchestrate and the monitor | Defined inside `orchestrate`, which runs at import time, so nothing else could import it without launching sweeps. |

### hard-won, 2026-09-21/22

- **Condemnation is PER SOLVER, and this reverses an earlier rule.** "One solver passing L0
  disqualifies the unit for everyone" was coherent while the dataset made a single claim.
  Once the ladder went per-solver a certificate means *this* solver could not fix it from
  the bug report, and another model solving it says nothing about that. Changed in
  `certificates`, `certificates_by_solver`, `summary().too_easy` and `trial_guard`
  **together** — for one night the guard condemned while `certificates` did not, and units
  sat in `certified` and `too_easy` at once.
- **Sizing a cohort total as `n x per-trial-worst-case x margin` is not conservative, it is
  impossible.** It asserts every trial lands in its own tail simultaneously: 436M for a
  cohort whose mean trial is 4M, refusing work the budget covered twice over. Bound the
  SUM (`n*mean + 1.645*sd*sqrt(n)`). Also: with 7 samples `int(7*0.9)` indexes the
  maximum, so "the 90th percentile" was the single most expensive trial ever seen.
- **A metric that does not match the mechanism it polices measures nothing.** `over-cap`
  and `re-measured a decided rung` were keyed `(base, rung)` while the guard's cap had gone
  per-solver, so every legitimate second curve scored as waste — 28% until keyed per
  solver, then 8%.
- **"Certified did not move" is not a stall.** Escalation establishes the rung *below* a
  certificate and a second screen *removes* units; both are progress and neither raises the
  count. Scoring only the rise reported a clean hour as `FAIL STALL` and printed a
  condemnation as `dataset +-1`.
- **An alarm that cannot go quiet is worse than no alarm.** The gate line carries a drifting
  sample count, so comparing the whole string re-fired on an unchanged state every five
  minutes; SHORT budget is persistent and re-fired the same way. Compare the verdict, or
  the state transition — never the whole line.
- **Absence of a log line means "not yet" as often as "never".** `reap_wedged` samples for
  120s and `reap_runaway` runs after it; checking inside that window twice led to
  announcing a fix as inert when it was simply pending.

- **A ladder rung means two different things and the guard has to tell them apart.** On a
  certified unit it is affordance-study data, sampled, and can never certify. On a unit
  that failed L0 *and* L2 it is an escalation - the only route by which that unit ever
  certifies. One rule rejecting both with "certify first" is what kept 46 of the hardest
  units in the dataset classified as waste.
- **The rung at which a unit flips is its difficulty, not its failure.** `certificates()`
  binds at the lowest passing rung >= 2 and records `escalated`.
