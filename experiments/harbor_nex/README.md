# harbor_nex — Nex-N2.5-Pro (free) as a Harbor difficulty prober

## Question

Is OpenRouter `nex-agi/nex-n2.5-pro:free` stable enough to probe synthetic
codegraph-injected Go bugs in Harbor (pass/fail per bug, retries, wall time)?

## Config

| item | value |
|---|---|
| Model | `openrouter/nex-agi/nex-n2.5-pro:free` |
| Agent | `terminus-2` (`reasoning_effort=max`) |
| Tasks | Harbor dirs built by `scripts/harbor_tasks.py` from `experiments/codegraph_bugs/` patches |
| Smoke | 1 task, `--n-concurrent 1`, `--n-attempts 1`, `--max-retries 5` |
| Full | valid+verifier-ok bugs, `--n-concurrent 2`, `--n-attempts 2`, `--max-retries 5` |
| Timeouts | `task.toml` agent 10800 s (3 h), verifier 1800 s; job `--timeout-multiplier 1.0` |
| Auth | `OPENROUTER_API_KEY` from `/home/evan/Documents/eval_tasks/.env` (exported, never committed) |

## Reproduce

```bash
# unit test
uv run pytest tests/test_harbor_tasks.py

# build one task (after discovering f2p tests)
uv run python scripts/harbor_tasks.py discover-f2p \
  experiments/codegraph_bugs/repos/<repo> <base_sha> \
  experiments/codegraph_bugs/bugs/<repo>/<Symbol>.patch
uv run python scripts/harbor_tasks.py build \
  experiments/codegraph_bugs/repos/<repo> <base_sha> \
  experiments/codegraph_bugs/bugs/<repo>/<Symbol>.patch \
  experiments/harbor_nex/tasks/<name> --f2p TestFoo

# smoke
set -a && . /home/evan/Documents/eval_tasks/.env && set +a
harbor run \
  --path experiments/harbor_nex/tasks/<name> \
  --agent terminus-2 \
  --model openrouter/nex-agi/nex-n2.5-pro:free \
  --ak reasoning_effort=max \
  --max-retries 5 \
  --n-concurrent 1 \
  --n-attempts 1 \
  --timeout-multiplier 1.0 \
  --jobs-dir experiments/harbor_nex/jobs \
  --yes
```

## Outputs

| Path | Contents |
|---|---|
| `tasks/` | Harbor task directories (src snapshot gitignored) |
| `jobs/` | Harbor job results |
| `verifier/` | alt/cheat patches and verifier table inputs |
| `RESULT.md` | stability, per-bug outcomes, Nex vs codegraph prior |

## What was measured

See `RESULT.md`.
