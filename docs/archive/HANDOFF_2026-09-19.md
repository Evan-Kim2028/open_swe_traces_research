# Handoff (2026-09-19 16:25 UTC)

Everything is stopped on the laptop. Resume from this repo on any machine.

## What is here
- Code: `src/openswe_traces/` (pipeline, ladder policy, hack audit), `scripts/`, `tests/` (green).
- Research notes: `analytics/research/` (start with `verifier_rules.md`, `task_space_framework.md`).
- 60 authored units in `experiments/pipeline/authored/` (excision patches included); Composer-verified
  task packages in `experiments/pipeline/tasks_composerver/` and the solve queue in `experiments/pipeline/tasks/`
  (hidden suites, gold/cheat patches, task.toml, validation.json; no `environment/src`).
- Every trial row and audit verdict: `experiments/pipeline/state.db`; dashboard `experiments/pipeline/results.md`.
- Agent briefs and ops scripts: `docs/briefs/` (`shutdown_all.sh`, `start_solve_watch.sh`, ...).

## Resume
```
uv sync && uv run pytest -q
# needs docker, harbor (uv tool install harbor), cursor-agent; CURSOR_API_KEY in ~/Documents/eval_tasks/.env
# OPENROUTER_API_KEY (throwaway, 1000 free req/day) in ./.env (gitignored)
uv run python scripts/materialize_tasks.py        # clones pinned commits, builds ladder-base:<repo>, rebuilds environment/src
uv run python scripts/pipeline.py solve-watch --interval 180 --host laptop
```
Config: `experiments/pipeline/config.yaml` (Composer 2.5 solver, 500M token cap, 4 parallel units, Devin cap 2).

## Known gaps
- `materialize_tasks.py` reproduces helm/gin/goa/kops trees exactly. client-go trees carry an
  `excised-import-keep` var block the packager added after the excision patch; regenerate client-go
  excision patches from the L2 environments (`diff -ruN base env/src`) before relying on it there.
- Devin backend was degraded today (55-70 s for a one-word reply, 502s on connect); solver switched to Composer.
- Most authored units pass at L0. Next step: an authoring difficulty gate (one Composer L0 probe per
  unit, reject if it passes; multi-file excisions with in-tree tests removed). Brief drafted in chat.

## Results so far
Composer: errloc, assetsremap, clustervalid pass L0; formmapping and coalesce flip L2→L5; evalrun passes L2.
Devin: connarray, memdbstaging pass L0; doactionbatches fails L2 twice. All passes audit-clean under a fresh seed.
