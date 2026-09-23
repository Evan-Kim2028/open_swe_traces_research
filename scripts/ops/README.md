# scripts/ops

Commands that run the experiment. The library code is in `src/openswe_traces/`; see
[`docs/ARCHITECTURE.md`](../../docs/ARCHITECTURE.md) for the map and
[`docs/OPERATIONS.md`](../../docs/OPERATIONS.md) for how to run things safely.

| Kind | Files |
|---|---|
| Daemons and drivers | `supervisor.sh`, `monitor_loop.sh`, `sweep_seq.sh`, `orchestrate.py`, `devin_queue.sh`, `cohort_topup.sh`, `post_sweep.sh`, `vps_ladder_driver.sh`, `devin_relaunch_after_backoff.sh`, `devin_throttle_watch.sh`, `agent_watchdog.sh` |
| Checks you run by hand | `devin_watchdog.sh`, `watch.sh`, `status_digest.sh`, `devin_progress.sh`, `status.py`, `trial_ledger.py`, `pipeline_health.py` |
| Agent launchers | `cursor_job.sh`, `devin_solve.sh`, `grok_job.sh`, `grok_author.sh`, `grok_key.sh`, `grok_ladder.py` (grok is out of limits; do not launch) |
| Maintenance | `reap_wedged.sh`, `kill_stale_harbor.sh`, `docker_cleanup.sh`, `docker_cleanup_loop.sh`, `prune_worktrees.sh`, `rebuild_bases.sh`, `regen_env_src.sh` |
| Small scripts without a module | `devin_tokens.py`, `harbor_web_audit.py`, `seed_rerun.py` |
| Shims | Every other `*.py` whose first line says "Moved to openswe_traces…". It runs the package module's `cli()` and keeps `import <name>` working. |

Never edit a `.sh` file while a daemon is running it. Bash reads scripts by byte offset,
so an edit in place breaks the running copy, and a replaced file is never seen by it.
