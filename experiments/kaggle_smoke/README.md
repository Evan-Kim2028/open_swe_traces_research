# kaggle_smoke — T4 QLoRA throughput smoke

## Question

How many trainable tokens/sec (and how much GPU memory) does QLoRA SFT of a 1.5B coder get on a
free Kaggle T4, so the 5k / 10k / 20k / 40k corpus scaling can be budgeted against the weekly
GPU quota? This is a throughput/memory run, not a quality run.

## Config

| item | value |
|---|---|
| Kernel | `evandekim/openswe-smoke-qlora` (`kernel-metadata.json`) |
| Code | `train_smoke.py` (standalone; runs on Kaggle, imports nothing from the repo) |
| Dataset | `evandekim/openswe-smoke-1k` — built from `data/sample_1000.jsonl` |
| Sample | `mini-swe-agent` × `qwen36_27b` × `swe-rebench-v2`, `--tool-max-chars 1500` |
| Model | `Qwen/Qwen2.5-Coder-1.5B-Instruct`, 4-bit NF4, LoRA r=16 α=32 |
| Sequence | `MAX_LEN=6144`, message-aligned window sampling (`N_TRACES=400`) |
| Training | `MAX_STEPS=120`, `GRAD_ACCUM=8`, `LR=2e-4`, assistant-only loss |

All knobs are env-overridable (`MODEL`, `MAX_LEN`, `N_TRACES`, `MAX_STEPS`, `GRAD_ACCUM`, `LR`,
`SEED`, `OUT_DIR`, `PARITY_CHECK`).

## Reproduce

```bash
# 1. Rebuild the sample (writes data/sample_1000.jsonl), then upload/replace the Kaggle dataset
uv run openswe-sample --n 1000

# 2. Local CPU dry run + loss parity check (no GPU needed, tiny model)
cd <repo root>
MODEL=Qwen/Qwen2.5-0.5B-Instruct MAX_LEN=512 N_TRACES=4 MAX_STEPS=2 \
  uv run --with transformers --with peft --with accelerate \
  python experiments/kaggle_smoke/train_smoke.py
PARITY_CHECK=1 ...   # additionally asserts supervised-position loss == full loss within 1e-3

# 3. GPU run: push, poll to terminal status, pull outputs into out/
cd experiments/kaggle_smoke && ./run_kernel.sh
```

Kaggle GPU quota is 30 h/week — do not re-push after a failure without a diagnosis first.

## Outputs

| Path | Contents |
|---|---|
| `out/openswe-smoke-qlora.log` | Raw kernel log (JSON stream) pulled after the run |
| `out/openswe-smoke-qlora.rendered.log` | Human-readable render of the same log |
| `out/smoke_metrics.json` | `tok_per_s`, `sup_tok_per_s`, `sec_per_trace`, peak memory, budget table (on success) |
| `data/sample_1000.jsonl` | The 1k-trace SFT sample uploaded as the dataset |
| `RESULT.md` | Findings for the latest push |
| `poll_status.log` | Kernel status poll history |

## What was measured

First push (2026-09-17) **failed**: `torch.OutOfMemoryError` on the first backward — the
`[1, 6144, 151936]` fp32 lm_head transient (~3.5 GiB per buffer) did not fit a 14.56 GiB T4
alongside the checkpointed forward pass. Pre-training numbers only: 400 traces loaded, 398
encoded, avg 5933 tokens, supervised fraction 0.17, trainable params 1.18%. Full log and
root-cause analysis in `RESULT.md`.

`train_smoke.py` has since been reworked to remove that transient (supervised-positions-only
loss over the head, window sampling, `expandable_segments`, non-reentrant checkpointing); the
next push measures the same throughput/budget numbers.
