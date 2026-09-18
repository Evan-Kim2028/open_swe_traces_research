# curve — budget-matched SFT arms on the 1.5B coder

## Question

At a fixed supervised-token budget and one recipe (fp16 LoRA, no 4-bit quantization),
does trace selection move held-out cross-entropy vs steps — uniform (`random`) vs the
best/worst labeled sibling of each task (`top_within_task` / `bottom_within_task`) —
and does masking low-value assistant steps (`random_masked`: repeated identical tool
calls, error observations, edits outside the gold patch) help or hurt? This is the
selection arm of the model-size curve; `MODEL` is the only knob that changes for the
3B/7B points, so the curve stays on one recipe.

## Arms

| file (`MANIFEST=`) | selection |
|---|---|
| `random.jsonl` | uniform over the traces of sampled tasks (every trace of the task is in the pool) |
| `top_within_task.jsonl` | per (instance_id, harness, teacher): the labeled sibling with the highest `p_resolved`; ties by fewer assistant turns, then trajectory id |
| `bottom_within_task.jsonl` | same, lowest `p_resolved` |
| `random_masked.jsonl` | the `random` traces, plus a per-assistant-message `mask` boolean (see Notes); `train_curve.py` sets labels to `-100` for masked messages |
| `eval_heldout.jsonl` | 300 traces, one per instance, from instances in no manifest, 100 per bucket — the arm-comparable metric |
| `eval_heldout_small.jsonl` | seeded 20-per-bucket subset of the 300 (60 traces) — the frequent in-run evals |

Sampling: **tasks** (instances) are drawn without replacement with weights `mid` 2×,
`hard` 1×, `easy` 1× (Efraimidis--Spirakis keys in `_task_weights`), and a drawn task
contributes its traces in rule order. The within-task arms keep one representative per
`(harness, teacher)` pair, so every teacher that ran a task is represented whenever the
task is drawn. Every arm has the same supervised-char budget and the same language mix
(budget allocated per language by the corpus language share; `ts`/`js` normalized as in
`difficulty.py`), and one window per trace per pass (see the run plan).

## Run plan (Saturday quota reset)

Four arms, four Kaggle script kernels, one kernel id per concurrent run, each on the
baked real-run config (`train_curve.py` `CONFIG`, env overrides nothing since a push
receives no env):

| item | value |
|---|---|
| Kernel | `evandekim/openswe-curve-{random, top-within-task, bottom-within-task, random-masked}` (`experiments/curve/arms/<arm>/kernel-metadata.json`; Kaggle sanitizes `_`→`-` in slugs, so the ids are hyphenated while `MANIFEST` keeps the arm names) |
| Push | `experiments/curve/launch_arms.sh` — copies `train_curve.py` into `arms/<arm>/`, seds the `CONFIG` `MANIFEST` default, pushes with `-t 43200`, retries on HTTP 500, then polls every 5 min and pulls outputs into `arms/<arm>/out/`. `DRY_RUN=1` prints the commands without touching Kaggle |
| Session | `STOP_AFTER_SECONDS=39600` (11 h) with `MAX_STEPS=100000`, so the time guard always wins and the adapter + metrics land before Kaggle's 12 h cap; `-t 43200` keeps the platform cap from firing first |
| Steps | `GRAD_ACCUM=8`, 2×T4 DDP → 8 windows per rank per step (16 global); `windows_seen = steps_done × grad_accum × world_size` is logged in every metrics record |
| Eval | every 100 steps **and** at `windows_seen` 1000/2000/3000 (all on `eval_heldout_small`, 60 traces, per-trace-seeded windows), then a final eval on the full `eval_heldout` (300) **before** the adapter is saved; `metrics.json` is rewritten every 5 steps |
| Metrics | `out/metrics.json` per arm: `steps_done`, `windows_seen`, `tok_per_s`, `sup_tok_per_s`, `loss`, `mem_gb`, `eval[]` (`step`, `windows_seen`, `eval_ce`, `event` ∈ step/windows/final_full) |

Window arithmetic at the measured v3 rate (685 tok/s total, 4.7k-token windows, ~113
s/step with evals): an 11 h session runs ~350 steps ≈ **2,800 windows per rank** (~5,600
`windows_seen`), i.e. ~1.8 passes over a rank's half of a 3k-trace manifest; the window
milestones land around step 63 / 125 / 188. `random` covers whole tasks (243 tasks in
the build below), so its passes are over fewer, larger tasks than the within-task arms
(~850–930 tasks).

Quota notes: a 2×T4 kernel consumes accelerator quota at ~2× wall-clock, and concurrent
GPU sessions are capped per account (extra kernels queue — the poll loop handles that).
Check the account's weekly balance before pushing all four; `ARMS=(...)` at the top of
`launch_arms.sh` can be trimmed to push a subset.

## Manifest builder

`build_manifests.py` is a thin CLI over `src/openswe_traces/sft/manifests.py`:

- candidates = `outputs/proxy_features.parquet` (assistant content chars, harness,
  language) + `outputs/trace_scores.parquet` (`p_resolved`) + `outputs/task_difficulty.parquet`
  (`difficulty_bucket`), joined on trajectory/instance ids; restricted to
  `difficulty_bucket` in hard/mid/easy.
- **supervised chars** = assistant content chars + per assistant tool call (name +
  argument chars + 60 structural allowance): a monotone proxy for supervised tokens
  (~chars / 4). Tool observations are excluded — they are loss-masked at train time.
- one streaming pass over `traces_data` computes the tool-call chars (never loads the
  corpus); the result is cached to `outputs/manifests_candidates.parquet` — rerun with
  `--refresh-cache` to rescan.
- messages are written in the `sft.sample` format (tool observations truncated to 1,500
  chars, tool-call arguments parsed to dicts) — the format `train_curve.py` consumes.

Real build (2026-09-18, `--budget-chars 165000000`, seed 0, ~10 min with the candidates
cache warm; 1.7 GB on disk):

| arm | traces | tasks | supervised chars (of budget) | buckets easy/hard/mid |
|---|---:|---:|---:|---|
| random | 3,143 | 243 | 165.21M (1.001) | 1,106 / 556 / 1,481 |
| top_within_task | 3,366 | 934 | 165.31M (1.002) | 1,351 / 634 / 1,381 |
| bottom_within_task | 3,034 | 843 | 165.46M (1.003) | 1,213 / 578 / 1,243 |
| random_masked | 3,143 | 243 | 165.21M (1.001) | 1,106 / 556 / 1,481 (75,827 masked assistant msgs) |
| eval_heldout | 300 | 300 | — | 100 / 100 / 100 |
| eval_heldout_small | 60 | 60 | — | 20 / 20 / 20 |

Budget sizing: 165M ≈ 3,000 traces at the corpus per-trace means (random 53.0k chars,
top 48.9k, bottom 54.3k — the floor "every arm ≥ 3,000 traces" is what binds, so bottom
lands at 3,034 while the shorter-trace arms overshoot to 3.1–3.4k). Per-arm language
counts and teacher counts are in `data/manifests_summary.json`; all arms are within 0.3%
of the budget and mid is the largest bucket in every arm.

## Config (`train_curve.py`, standalone — imports nothing from the repo)

| item | value |
|---|---|
| Kernel | `evandekim/openswe-curve` (`kernel-metadata.json`; `enable_gpu="false"` + `machine_shape="NvidiaTeslaT4"` — that combination is required, `enable_gpu="true"` + `machine_shape` returns HTTP 400); the real run uses the per-arm ids under `arms/` |
| Dataset | `evandekim/openswe-curve-manifests` — the files under `data/` |
| Model | `Qwen/Qwen2.5-Coder-1.5B-Instruct`, **no bitsandbytes**: plain `transformers`+`peft` fp16 LoRA (SDPA attention, gradient checkpointing with `use_reentrant=False`); Unsloth is opt-in only (`USE_UNSLOTH=1`) and **defaults to off** — the 2026-09-17 image's Unsloth stack is import-broken and poisons the plain path in-process (see `VALIDATION.md`); the path that ran is recorded as `backend` in `metrics.json` |
| Adapter | LoRA r=16, α=32, dropout 0.05, all attn/MLP projections, gradient checkpointing |
| Sequence | `MAX_LEN=6144`, smoke's message-aligned random window sampling |
| Loss | CE over shifted supervised positions only, fp32, `MAX_SUP_ROWS=2048` stride cap per micro-batch |
| Training | `GRAD_ACCUM=8`, `LR=2e-4`, AdamW, clip 1.0, fp16 autocast + `torch.amp.GradScaler`; `MAX_STEPS=100000` (the stop guard wins) |
| Multi-GPU | `torch.cuda.device_count()==2` → 2 spawned workers, contiguous halves of the manifest, DDP `nccl` (`gloo` fallback), gradients synced on the last micro-batch of each step (`no_sync` for the rest) |
| Guard rails | `metrics.json` rewritten every 5 steps (with `windows_seen`); in-run evals on `eval_heldout_small` every 100 steps + at windows 1000/2000/3000; final full-300 eval before saving; `STOP_AFTER_SECONDS=39600` (11 h) saves the adapter and exits before Kaggle's 12 h cap |

Env overrides: `MODEL`, `MANIFEST`, `MAX_LEN`, `MAX_STEPS`, `GRAD_ACCUM`, `MAX_SUP_ROWS`,
`LR`, `SEED`, `EVAL_EVERY`, `EVAL_N`, `EVAL_WINDOWS`, `METRICS_EVERY`, `STOP_AFTER_SECONDS`,
`OUT_DIR`, `USE_UNSLOTH` (0 default; 1 opts into Unsloth), `DDP_BACKEND` (nccl/gloo),
`WORLD_SIZE` (force a rank count), `NO_DDP`, `MANIFEST_PATH`, `EVAL_PATH` (periodic;
falls back to the full file), `FINAL_EVAL_PATH`.

## Reproduce

```bash
# 0. Manifests (real budget: ~3k traces per arm; deterministic given seed 0)
cd <repo root>
uv run python experiments/curve/build_manifests.py --budget-chars 165000000

# 1. Version the manifest dataset and wait for "ready" (a dataset upload, no GPU quota)
uv run kaggle datasets version -p experiments/curve/data -m "curve manifests 3k windows" --dir-mode skip
uv run kaggle datasets status evandekim/openswe-curve-manifests

# 2. Local CPU dry run — exercises the transformers fallback by construction
#    (no CUDA -> Unsloth is skipped); on this machine prefix with `env -u NODE_ENV`
MODEL=Qwen/Qwen2.5-Coder-0.5B-Instruct MAX_LEN=512 MAX_STEPS=3 EVAL_EVERY=2 \
  uv run --with transformers --with peft --with accelerate \
  python experiments/curve/train_curve.py

# 3. Check the launcher, then push/poll/pull all four arms (TIMEOUT is the kernel
#    kill-switch in seconds; 43200 = 12 h cap)
DRY_RUN=1 ./experiments/curve/launch_arms.sh
nohup ./experiments/curve/launch_arms.sh > experiments/curve/launch_arms.log 2>&1 &
```

One arm per kernel: Kaggle script kernels take no custom env vars, so `MANIFEST` is baked
at push time — `launch_arms.sh` seds the `CONFIG` default in each per-arm copy (the env
var still works for local runs). GPU quota is 30 h/week: do not re-push after a failure
without a diagnosis first, and remember the `STOP_AFTER_SECONDS` guard already saves a
partial curve.

## Saturday checklist

- [ ] `uv run kaggle datasets status evandekim/openswe-curve-manifests` → `ready` and
      the version is the 3k build (arm files ~415 MB, evals 41 MB + 8.3 MB)
- [ ] `git status` clean and `MANIFEST`/eval files under `experiments/curve/data/` are
      the fresh build (`manifests_summary.json` says `budget_chars: 165000000`)
- [ ] Check the Kaggle account's weekly GPU balance and concurrent-session cap before
      pushing all four (2×T4 burns quota at ~2× wall-clock; extra kernels queue)
- [ ] `DRY_RUN=1 ./experiments/curve/launch_arms.sh` prints the four pushes + poll block
- [ ] `nohup ./experiments/curve/launch_arms.sh > launch_arms.log 2>&1 &` — then watch
      `kernels status` in the log; a push that fails non-500 can be retried by rerunning
      the script (it re-copies and re-pushes)
- [ ] After ~1 h, check one arm's kernel log for `backend: transformers`, `world_size:
      2`, and the startup `pip install peft` + `torchao` uninstall lines (the v2 crash)
- [ ] During the run: `arms/<arm>/out/metrics.json` (pulled on completion) — `windows_seen`
      should climb ~16/step, evals at steps 63/100/125/188/200/…
- [ ] After all four: compare `eval_ce` at the `final_full` event (300 traces) across
      arms; sanity-check tok/s (~600–700 total) and that no run has `status: error`
- [ ] Do not re-push on a failure without a diagnosis; the adapter + metrics of a
      stopped arm are still usable

## Outputs

| Path | Contents |
|---|---|
| `/kaggle/working/metrics.json` | the whole curve so far — `steps_done`, `windows_seen`, `tok_per_s`, `sup_tok_per_s`, `loss`, `elapsed_s`, `mem_gb`, `eval_ce`, `log[]`, `eval[]`, `backend` — rewritten every 5 steps, so a kill still leaves data |
| `/kaggle/working/metrics_rank{0,1}.json` | per-rank records when DDP ran |
| `/kaggle/working/adapter/` | final LoRA adapter (Unsloth or PEFT) |
| `arms/<arm>/out/` | pulled kernel log + metrics per arm |
| `data/*.jsonl` | the manifests + evals (regenerable; gitignored) |

## Notes

- **Memory.** `kaggle_smoke` measured 10.47 GiB peak for the same loss path with NF4
  weights; the unquantized fp16 base adds ~2.3 GiB, so `MAX_SUP_ROWS` defaults to 2,048
  (was 3,000) to keep the fp32 CE transient inside the T4's 14.56 GiB (v3 measured 11.5 GiB).
- **Startup encode.** One window per trace is encoded at startup (per trace ~0.25 s at
  ~70 assistant messages/trace), so a 3k-trace manifest costs ~6–7 min per rank before
  the first step; eval windows are seeded per trace id so `eval_heldout_small` (⊂ the
  full set) keeps the same windows in the in-run and final evals.
- **DDP.** Each worker takes a contiguous half of the manifest; steps are per rank (both
  ranks run `MAX_STEPS`), `metrics.json` reports global tok/s (`×world_size`). The
  spawn/DDP/half-split/all-reduce plumbing was validated locally with `WORLD_SIZE=2` +
  gloo on CPU. `USE_UNSLOTH` defaults to 0 so DDP always runs the plain path; a
  `sys.modules` guard raises at model build if `unsloth`/`unsloth_zoo` got imported anyway.
- **Masks** (`random_masked` only): an assistant message is masked when (a) it repeats an
  identical `(tool name, arguments)` call already made in the trace, or (b) the tool
  messages directly after it contain `Error` / `Traceback` / `FAILED` (case-sensitive),
  or (c) it makes an edit-type call (`EDIT_TOOL_NAMES`, non-`view` `str_replace_editor`,
  or `BASH_EDIT_RE` on the command) whose arguments mention no gold path or basename.
  Gold paths come from `metadata.reference_patch.patch` (same regexes as the temporal
  features); a trace whose patch parsed empty counts all edits as outside-gold.
- **Eval** is `eval_heldout_small` (60 traces) at every in-run checkpoint, so the CE
  curve is comparable across arms; the final eval runs on all 300 and is the offline set.

## Quota accounting (verified 2026-09-17 from Kaggle staff posts)

- 2xT4 charges **1 quota-hour per wall-clock hour** (same as P100). Weekly quota 30 h.
- **2 script kernels run concurrently**; further pushes queue. Queued time does not burn quota.
- Therefore each arm runs **7 h** (`STOP_AFTER_SECONDS=25200`): 4 arms x 7 h = 28 h, ~14 h wall-clock
  with arms 3 and 4 queued behind 1 and 2. Push all four at once; Kaggle serialises them.
- At the v3 rate (~113 s/step, 16 windows/step across 2 ranks) 7 h is ~220 steps, ~3,500 windows seen.
  Milestone evals at 1,000 / 2,000 / 3,000 windows still land inside the run.
