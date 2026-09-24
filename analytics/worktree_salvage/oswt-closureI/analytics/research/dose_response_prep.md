# Dose-response prep (closure-I)

Prep for the overnight run testing whether excised-set internal-edge
structure controls solver difficulty at fixed instruction level.

## Arm 1 — fabricated grid

Generator: `src/openswe_traces/synth/fabricate.py` via
`scripts/dose_response_fab.py`. N=24 functions/unit, local A1/A3 proof on
(gold passes, cheat fails, module build/test/vet/gofmt clean).

A cell is *reached* when the closest achievable ratio in the |S| band is
within `|log2(achieved/requested)| <= 0.5`. Unreachable cells are recorded
in `grid.csv` with the closest achievable ratio and empty stats.

Two generator bugs found and fixed during this run:

1. Consumer functions were emitted when *any* callee was present
   (`intersection`), but their bodies call *all* callees — producing
   uncompilable modules (12 proof failures). Gates moved to `issubset` in
   `store_fns`/`sched_fns`/`store_edges`/`sched_edges`/`_candidate_configs`.
2. Test-local `refSched` helper methods (`add`, `total`, `gap`, `merge`)
   tripped the B4 black-box heuristic (lowercase `.name(` calls).
   Capitalized in the generator and patched in existing hidden files.

### Grid coverage

| domain | band | ratio req | reached | achieved | \|S\| | internal |
|---|---|---|---|---|---|---|
| sched | large | 0.1 | unreachable | 0.2222 | – | – |
| sched | large | 0.2 | 4/4 | 0.2222 | 14 | 4 |
| sched | large | 0.4 | 4/4 | 0.4 | 15 | 6 |
| sched | large | 0.8 | 4/4 | 0.8 | 17 | 8 |
| sched | large | 1.5 | 4/4 | 1.5 | 16 | 9 |
| sched | large | 3.0 | 4/4 | 3.0 | 17 | 12 |
| sched | small | 0.1 | 4/4 | 0.1 | 9 | 1 |
| sched | small | 0.2 | 4/4 | 0.2 | 10 | 2 |
| sched | small | 0.4 | 4/4 | 0.4 | 10 | 4 |
| sched | small | 0.8 | 4/4 | 0.8571 | 10 | 6 |
| sched | small | 1.5 | unreachable | 1.0 | – | – |
| sched | small | 3.0 | unreachable | 1.0 | – | – |
| store | large | 0.1 | unreachable | 0.3684 | – | – |
| store | large | 0.2 | unreachable | 0.3684 | – | – |
| store | large | 0.4 | 4/4 | 0.3889 | 14 | 7 |
| store | large | 0.8 | 4/4 | 0.8 | 14 | 8 |
| store | large | 1.5 | 4/4 | 1.5 | 16 | 12 |
| store | large | 3.0 | 4/4 | 3.0 | 16 | 12 |
| store | small | 0.1 | unreachable | 0.1905 | – | – |
| store | small | 0.2 | 4/4 | 0.2 | 10 | 4 |
| store | small | 0.4 | 4/4 | 0.4 | 10 | 4 |
| store | small | 0.8 | 4/4 | 0.8333 | 8 | 5 |
| store | small | 1.5 | 4/4 | 1.4 | 9 | 7 |
| store | small | 3.0 | 4/4 | 3.0 | 10 | 9 |

72 units reached (18 cells × 4 seeds), 24 rows unreachable (6 cells), 0
proof failures. Internal-edge coverage: small band 1–9, large band 4–12.
Module size 193–240 LOC.

## Packaging

144 Harbor task dirs under `experiments/dose_response/tasks/` (`<unit>-L0`,
`<unit>-L2`), built through `build_affordance_levels` (A-2 → L0, A0 → L2):

- `[agent] network_mode=allowlist`, `allowed_hosts=["openrouter.ai",
  "*.openrouter.ai"]` (new `solver_hosts("openrouter")` in
  `pipeline/safety.py`).
- `[verifier] network_mode=no-network`; `tests/test.sh` checksum-guards
  each hidden file and re-verifies after install.
- Hidden tests only under `tests/hidden/`; `environment/src` is the
  excised tree (golang:1.23 base, no `.git`).
- All 144 dirs pass `assert_harbor_safe(solver="openrouter")`.

## Arm 2 — authored excisions

12 units picked from `closure_vs_flip.md` spanning internal_edges 0–53
(`experiments/dose_response/arm2.csv`): memfs(0), ignorerules(2),
flagbuilder(2), depresolver(3), provenance(4), repindex(5), addonparse(7),
kindsorter(9), chartloader(10), coalesce(14), formmapping(45),
strvalsparser(53). Ten of twelve have lines_removed in 150–300; the two
anchors (memfs 84, formmapping 430, strvalsparser 455) were kept to
cover the edges=0 and edges≥45 ends.

**Materialisation repair.** The shipped `tasks_composerver` dirs for
helm/kops contained the *unexcised* base tree: authored patches/hidden
tests use the identity-pass branding (`example.internal/chartkit/v4`,
`example.internal/clustkit`) while the materialised src uses
`example.internal/helm`, `example.internal/kops`, so patch application
had silently failed. `prepare_arm2` (in `dose_response.py`) copies each
task dir to `experiments/dose_response/tasks_arm2/<repo>/<unit>-L{0,2}`,
applies the word-boundary rebrand from `trees.json` to `tests/` +
`instruction.md`, reverse-applies the rebranded `tests/gold.patch` onto
`environment/src` (excised → full ⇒ reverse gives full → excised), and
retargets the sha256 checksums inside `tests/test.sh`. All 24 dirs now
show excision stubs (4–23 per unit), hidden-test imports match the src
module, and `assert_harbor_safe(solver="cursor")` passes; the runner
rewrites the allowlist to OpenRouter hosts at launch.

## Runner

`scripts/dose_response_run.py` → `openswe_traces.synth.dose_response`.
`harbor run --path <task> --agent mini-swe-agent --model openrouter/<m>`
per (task, model); resume-safe via `trials.parquet` counts; 429/backoff
on the whole job; `--limit` caps attempts per invocation; `--dry-run`
prints the plan. Cursor anchor supported (`--agent cursor-cli --model
cursor/composer-2.5`) but not run.

Models (`runner_models.md`): `deepseek/deepseek-v4-flash-0731:free` and
`nvidia/nemotron-3-super-120b-a12b:free` — both free-tier
(`pricing.prompt=="0"`) with `tools` in `supported_parameters`.

## Validation trial

(one trial, `store-s11-r040-L2`, mini-swe-agent,
`openrouter/deepseek/deepseek-v4-flash-0731:free`, `--limit 1`)

**OUTCOME: reward 0.0, recorded.** The trial exercised the full loop:

- Job `store-s11-r040-L2-openrouter_deepseek_deepseek-v4-flash-0731_free-seq1`
  → trial `store-s11-r040-L2__p93PxQP`, 18:36 → 20:00 UTC (~84 min:
  agent hit the 3600 s timeout, then the verifier ran).
- Trajectory saved: `agent/mini-swe-agent.trajectory.json` (~470 KB) +
  `mini-swe-agent.txt` transcript; tokens 793,215 in / 184,859 out.
- Verifier ran under no-network: checksum-guarded hidden test installed
  and executed; `TestBucketOfProperty` panicked on `excised: NewStore`
  (the model spent its budget on base64 file round-trips and never wrote
  a working `store.go`), so `reward.txt` = 0 — a legitimate FAIL, not an
  infra error.
- Row written to `experiments/dose_response/trials.parquet`
  (unit, arm=1, level=2, model, attempt=1, reward=0.0, wall_s=5052,
  tokens, job_dir).

Observation for the full run: `deepseek-v4-flash-0731:free` under
mini-swe-agent frequently emits responses with no tool call
(reprompted by the harness) and prefers `base64 -d <<EOF` writes —
expect low pass rates and near-timeout wall times from this model.
`nemotron-3-super-120b-a12b:free` is the second model precisely so the
dose signal isn't confounded by one weak model.

## Full-run commands

Task-list files pin the exact dirs (`--task-list`, one path per line):
`tasklist_arm1_small.txt` (36 units × 2 levels = 72 dirs),
`tasklist_arm2.txt` (12 × 2 = 24), `tasklist_first_run.txt` (both = 96
dirs = 48 units), `tasklist_arm1_large.txt` (36 × 2 = 72).

```bash
# FIRST RUN — small band + arm-2: 48 units × 2 levels × 5 attempts
#             × 2 models = 960 requests ≈ one free-tier day
uv run python scripts/dose_response_run.py \
  --task-list experiments/dose_response/tasklist_first_run.txt \
  --agent mini-swe-agent \
  --model openrouter/deepseek/deepseek-v4-flash-0731:free \
  --model openrouter/nvidia/nemotron-3-super-120b-a12b:free \
  --attempts 5 --n-concurrent 4 \
  --log outputs/closure_I.log

# SECOND RUN — large band: 36 units × 2 × 5 × 2 = 720 requests
uv run python scripts/dose_response_run.py \
  --task-list experiments/dose_response/tasklist_arm1_large.txt \
  --agent mini-swe-agent \
  --model openrouter/deepseek/deepseek-v4-flash-0731:free \
  --model openrouter/nvidia/nemotron-3-super-120b-a12b:free \
  --attempts 5 --n-concurrent 4 \
  --log outputs/closure_I.log
```

Both are resume-safe: re-running skips (unit, level, model) pairs that
already have 5 rows in `trials.parquet`. `--limit N` caps attempts per
invocation; `--dry-run` prints the plan.
