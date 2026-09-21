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
| `escalate.py` | which rung a unit that failed L0 **and** L2 should try next, and staging it | 48 units were written off as "non-flipping". Nine had been escalated by hand and **all nine flipped higher up** (11 passes, 2 fails). They were the hardest tasks in the bank, not broken ones. Probes L5, then bisects for the lowest flipping rung: ~2.6 trials/unit against 4 for a linear climb. |
| `composer_budget.py --set` | granting an allowance and re-anchoring the baseline in one atomic step | Every allowance was a hand-edit of the JSON, and the first one overshot by 23% because raising the cap and re-anchoring were separate actions. |
## Rules learned the hard way

- **Never edit a running bash script.** Bash reads by byte offset; an edit mid-run kills it
  with a syntax error after the gate has already passed. Python is safe (loaded at import).
  Sweeps run from a content-addressed frozen copy for this reason.
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

- **A ladder rung means two different things and the guard has to tell them apart.** On a
  certified unit it is affordance-study data, sampled, and can never certify. On a unit
  that failed L0 *and* L2 it is an escalation - the only route by which that unit ever
  certifies. One rule rejecting both with "certify first" is what kept 46 of the hardest
  units in the bank classified as waste.
- **The rung at which a unit flips is its difficulty, not its failure.** `certificates()`
  binds at the lowest passing rung >= 2 and records `escalated`.
