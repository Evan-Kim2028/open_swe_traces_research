# kaggle_smoke — single-T4 QLoRA throughput smoke

## Question

How long does one epoch of QLoRA SFT of a 1.5B coder take on a free Kaggle T4 at
`MAX_LEN=6144` (and does it fit the 14.5 GiB card), so the 5k / 10k / 20k / 40k trace
corpus sizes can be budgeted against the 30 h/week GPU quota? This is a throughput and
memory run, not a quality run. Answer for the second push (2026-09-17) in `RESULT.md`:
**15.85 s/trace, ~318 tok/s, 10.47 GiB peak** — memory fits, but the run was canceled by
its own 90-minute kernel cap at step 40/60 (budget table in `RESULT.md`).

## Config

| item | value |
|---|---|
| Kernel | `evandekim/openswe-smoke-qlora` (`kernel-metadata.json`; `machine_shape=NvidiaTeslaT4` with `enable_gpu="false"` — that combination is required, `enable_gpu="true"` + `machine_shape` returns HTTP 400) |
| Code | `train_smoke.py` (standalone; runs on Kaggle, imports nothing from the repo) |
| Dataset | `evandekim/openswe-smoke-1k` — `sample_1000.jsonl`, built by `build_sample.py` (mini-swe-agent × qwen36_27b × swe-rebench-v2, tool observations capped at 1,500 chars) |
| Model | `Qwen/Qwen2.5-Coder-1.5B-Instruct`, 4-bit NF4 + double quant, LoRA r=16 α=32 on all attn/MLP projections, gradient checkpointing (`use_reentrant=False`) |
| Sequence | `MAX_LEN=6144`, message-aligned random window sampling (`N_TRACES=400`, seed 0) |
| Loss | CE over shifted supervised (assistant) positions only, fp32, stride-capped at 3,000 rows/micro-batch |
| Training | `MAX_STEPS=60`, `GRAD_ACCUM=8` (480 micro-batches), `LR=2e-4`, AdamW, clip 1.0, fp16 autocast + `torch.amp.GradScaler("cuda")` |

All knobs are env-overridable: `MODEL`, `MAX_LEN`, `N_TRACES`, `MAX_STEPS`, `GRAD_ACCUM`,
`MAX_SUP_ROWS`, `LR`, `SEED`, `OUT_DIR`, plus the diagnostic modes `PARITY_CHECK=1` and
`ENCODE_ONLY=1`.

## Reproduce

```bash
# 1. Local CPU dry run (fp32, no bitsandbytes; tiny model, same code path)
cd <repo root>
MODEL=Qwen/Qwen2.5-0.5B-Instruct MAX_LEN=512 N_TRACES=4 MAX_STEPS=2 \
  uv run --with transformers --with peft --with accelerate \
  python experiments/kaggle_smoke/train_smoke.py

# 2. Loss parity: asserts the supervised-position loss == standard model(labels=...) loss < 1e-3
MODEL=Qwen/Qwen2.5-0.5B-Instruct MAX_LEN=512 N_TRACES=4 PARITY_CHECK=1 \
  uv run --with transformers --with peft --with accelerate \
  python experiments/kaggle_smoke/train_smoke.py

# 3. Encoding stats only (no model download beyond the tokenizer; validates window sampling)
MODEL=Qwen/Qwen2.5-Coder-1.5B-Instruct MAX_LEN=6144 N_TRACES=400 ENCODE_ONLY=1 \
  uv run --with transformers --with peft --with accelerate \
  python experiments/kaggle_smoke/train_smoke.py

# 4. GPU run: push once, poll, pull outputs (TIMEOUT is the kernel kill-switch in seconds)
cd experiments/kaggle_smoke
uv run --project /home/evan/Documents/kaggle_e4b kaggle kernels push -p . -t 5400
uv run --project /home/evan/Documents/kaggle_e4b kaggle kernels status evandekim/openswe-smoke-qlora   # repeat every 60 s
uv run --project /home/evan/Documents/kaggle_e4b kaggle kernels output evandekim/openswe-smoke-qlora -p out
# or: TIMEOUT=5400 ./run_kernel.sh   (push + poll + pull in one go)
```

Kaggle GPU quota is 30 h/week: do not re-push after a failure without a diagnosis first, and
size `-t` (kernel timeout) to the measured `sec_per_trace` — the second push was canceled at
the 90-minute cap with ~2.2 h of training still needed. `-t 43200` is Kaggle's 12 h session cap.

## Outputs

| Path | Contents |
|---|---|
| `out/smoke_metrics.json` | `tok_per_s`, `sup_tok_per_s`, `sec_per_trace`, `peak_mem_gb`, budget table — written only if the run reaches the end |
| `out/adapter/` | LoRA adapter from the smoke run (evidence the training path works) |
| `out/openswe-smoke-qlora.log` | Raw kernel log (JSON stream) pulled after the run |
| `out/openswe-smoke-qlora.rendered.log` | Decoded, progress-collapsed render of the latest pull |
| `out/openswe-smoke-qlora.v1.rendered.log` | Render of the first push's OOM log (kept) |
| `data/sample_1000.jsonl` | The 1k-trace sample uploaded as the dataset (local dry runs read it too) |
| `RESULT.md` | Findings for the latest push |
| `poll_status.log` | Kernel status poll history |

Note: `out/smoke_metrics.json` / `out/loss_parity.json` / `out/adapter/` currently hold a
**local CPU dry run** (0.5B, `MAX_LEN=512`) from 2026-09-17T20:16–20:19Z — the canceled GPU
run produced no metrics file. See `RESULT.md` for the GPU numbers read from the kernel log.

## Memory design (why the first push OOM'd, and what changed)

The first push died in the first backward: `MAX_LEN=6144` × vocab 151,936 makes the fp32
`lm_head` CE transient 3.48 GiB (plus fp16 logits and a matching backward buffer ≈ 8.7 GiB
per micro-batch) on a 14.56 GiB T4. `train_smoke.py` now runs the PEFT base transformer,
gathers last hidden states at shifted-label != -100 rows, applies `lm_head` to those rows
only (fp32 CE, stride-capped at `MAX_SUP_ROWS=3000`), and samples a message-aligned window
per trace instead of prefix-truncating — measured peak allocation on the T4: 10.47 GiB.
