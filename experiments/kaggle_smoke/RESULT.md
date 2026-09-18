# RESULT — evandekim/openswe-smoke-qlora (2026-09-17)

**Status: CANCELED at the 90-minute kernel cap — the memory rework is validated; no third push.**

The reworked `train_smoke.py` (supervised-position-only loss + message-aligned window sampling)
ran **40 of 60 steps** on a T4 with **peak PyTorch allocation 10.47 GiB of 14.56 GiB, flat from
step 1** — no OOM, loss 1.131 → 0.622. `KernelWorkerStatus.CANCEL_ACKNOWLEDGED` at
2026-09-17T21:55:03Z: the run's own `-t 5400` cap fired because 60 steps × 8 micro-batches ×
15.85 s ≈ 7,600 s of training + ~250 s setup ≈ **2.2 h**. The killed process never reached its
metrics block, so no `smoke_metrics.json` came back; every number below is decoded from the
kernel log.

Run provenance: kernel version 3, run start 20:24:11Z (same script revision as v2, which it
superseded; both versions now read CANCEL_ACKNOWLEDGED). No further push was made.

## Measured (T4, steady state steps 5–40)

| item | value |
|---|---|
| GPU | Tesla T4, sm_75, 14.56 GiB |
| tok_per_s | ~319 (317.2–321.5 across steps 5–40) |
| sup_tok_per_s | ~147 (143.1–152.8) |
| sec_per_trace | 15.85 s (5,071 s ÷ 320 traces at step 40; steady-state delta 15.81) |
| peak_mem_gb | 10.47 (`max_memory_allocated`) |
| avg_len | 5,046 tokens (windowed; MAX_LEN = 6,144) |
| n_truncated / n_examples | 400 / 400 |
| avg window start frac | 0.51 |
| first vs last loss | 1.1313 (step 1) → 0.6224 (step 40) |

Config: `Qwen/Qwen2.5-Coder-1.5B-Instruct`, NF4 QLoRA r16/α32, `MAX_LEN=6144`, `GRAD_ACCUM=8`,
LR 2e-4, seed 0, `MAX_SUP_ROWS=3000` (supervised density 0.46 → ~2.3k rows/window, so the cap
binds only on the densest windows). Trainable params 18,464,768 / 1,562,179,072 (1.18%).
Supervised frac 0.46 vs 0.17 on the old prefix-truncated encode.

Log evidence (decoded, progress frames collapsed):

```
encoded 400 ex in 92s | truncated 400 | avg len 5046 | avg window start frac 0.51 | supervised frac 0.46
trainable params: 18,464,768 || all params: 1,562,179,072 || trainable%: 1.1820
{'step': 1,  'loss': 1.1313, 'tok_per_s': 326.0, 'sup_tok_per_s': 156.9, 'elapsed_s': 138,  'mem_gb': 10.3}
{'step': 5,  'loss': 0.9685, 'tok_per_s': 320.0, 'sup_tok_per_s': 145.5, 'elapsed_s': 643,  'mem_gb': 10.47}
{'step': 15, 'loss': 0.831,  'tok_per_s': 321.0, 'sup_tok_per_s': 152.6, 'elapsed_s': 1817, 'mem_gb': 10.47}
{'step': 25, 'loss': 0.8001, 'tok_per_s': 317.3, 'sup_tok_per_s': 143.1, 'elapsed_s': 3163, 'mem_gb': 10.47}
{'step': 35, 'loss': 0.6225, 'tok_per_s': 317.6, 'sup_tok_per_s': 146.5, 'elapsed_s': 4393, 'mem_gb': 10.47}
{'step': 40, 'loss': 0.6224, 'tok_per_s': 317.2, 'sup_tok_per_s': 145.0, 'elapsed_s': 5071, 'mem_gb': 10.47}
```

## Budget (1 epoch; `hours = N × sec_per_trace / 3600`, sec_per_trace = 15.85 s)

| traces | hours @ 1 epoch | fits a 12 h session? | fits the 30 h weekly quota? |
|---:|---:|:--:|:--:|
| 5,000 | 22.0 | no | yes (73% of the quota) |
| 10,000 | 44.0 | no | no |
| 20,000 | 88.1 | no | no |
| 40,000 | 176.1 | no | no |

A 12 h session fits ~2,700 traces; the 30 h/week quota buys ~6,800 traces/week. Training-only
time at `MAX_LEN=6144` (add ~250 s of encode + model load per session).

## Why it was canceled (root cause)

Not a code, data, or memory defect — a run-length budget error:

- 60 steps × 8 micro-batches × 15.85 s = 7,608 s of training + ~250 s setup = **~2.2 h**
  against a 5,400 s kernel cap. The process was healthy the whole time (mem flat, loss down,
  no traceback).
- Real cost per 6,144-token window is NF4 dequant + checkpoint recompute on sm_75: ~126.5 s
  per step (a pre-run estimate of ~5 s/micro-batch from T4 fp16 FLOPs was ~3× optimistic).
- Memory is solved: the gathered-head loss held peak allocation at 10.47 GiB with ~4 GiB
  headroom, vs the full `[1, 6144, 151936]` fp32 path (~8.7 GiB of head transients per
  micro-batch) that OOM'd in the first push.

## What a third push (if ever) should change

- Keep `MAX_STEPS=60` but raise the cap: `-t 9000`+ (12 h session cap allows it) — the full run
  needs ~2.2 h.
- Or keep `-t 5400` and set `MAX_STEPS=30` (240 × 15.85 s + 250 s ≈ 68 min). Throughput is
  already stable at step 1, so 30 steps is enough for a measurement.
- Lower `MAX_LEN` (e.g. 4,096) to scale the dominant cost almost linearly.

## Verification before the push (local, CPU)

- Loss parity vs the standard `model(labels=...)` path: max |diff| **4.77e-07** (tol 1e-3),
  Qwen2.5-0.5B-Instruct, MAX_LEN 512, 4 windows → `out/loss_parity.json`.
- Stride-cap branch exercised with `MAX_SUP_ROWS=64` (loss 3.34 → 2.00 over 2 steps).
- Full dry run (0.5B, `MAX_LEN=512`, `N_TRACES=4`, `MAX_STEPS=2`, fp32 CPU) exits 0 and writes
  metrics + adapter → `out/smoke_metrics.json`, `out/adapter/` (re-run 21:56Z against the exact
  pushed revision, after `train_smoke.py` gained `MAX_SUP_ROWS`; CPU numbers, not T4-comparable).
- Encode-only at `MAX_LEN=6144` over all 400 traces: avg len 5,046, supervised frac 0.46,
  per-window supervised rows p50 2,426 / p90 3,765 / max 4,760 (basis for the 3,000-row cap).

## Artifacts

- `out/openswe-smoke-qlora.log` — raw pulled log of the canceled run (JSON stream, 102,592 B).
- `out/openswe-smoke-qlora.rendered.log` — decoded non-progress lines of that log.
- `out/openswe-smoke-qlora.v1.rendered.log` — render of the first push's OOM log (kept).
- `out/v2/openswe-smoke-qlora.log` — copy pulled via the `/2` ref (byte-identical; Kaggle
  serves the latest version's log for both refs).
- `poll_status.log` — status poll history for the run (RUNNING → CANCEL_ACKNOWLEDGED at
  21:55:03Z).

## First push (2026-09-17T20:03:53Z, kernel v1) — for the record

Died ~2 min in with `torch.OutOfMemoryError: Tried to allocate 3.45 GiB ... 3.42 GiB free,
11.14 GiB in use` inside the first `backward()`. Root cause: `MAX_LEN=6144` × vocab 151,936 →
the fp32 `lm_head` CE transient alone is 3.48 GiB, plus a matching backward buffer and fp16
logits (~8.7 GiB of head transients per micro-batch), with 398/398 examples fully truncated
(`avg len 5,933`, prefix-dominated supervised frac 0.17).
