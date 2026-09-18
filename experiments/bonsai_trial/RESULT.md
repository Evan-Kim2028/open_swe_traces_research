# RESULT — Ternary-Bonsai-2-27B (PTQ1_0) on a Kaggle 2xT4

**Verdict: yes, it runs on sm_75 — and it is usable, but prefill-bound.** The PrismML
prebuilt CUDA 12.4 binary runs unchanged on a Tesla T4: the release ships no native sm_75
cubins (only sm_86/sm_89/sm_120) but *does* embed sm_75 PTX, so the driver JIT-compiles the
kernels on first load. No source build was needed. Measured on 2026-09-17, kernel
`evandekim/bonsai-t4-trial`, one push, 7.04 min wall, exit `status: "ok"`.

## 1. Does it run on sm_75, and how?

| | |
|---|---|
| Runtime | PrismML fork prebuilt `prism-b10685-7dffb15`, `llama-prism-…-bin-linux-cuda-12.4-x64.tar.gz` (248 MB) |
| Why it works on a T4 | offline `cuobjdump` check: cubins `sm_86/sm_89`, PTX `sm_50/61/70/75/80/90`; the sm_75 PTX contains all 17 `PTQ1_0`/`PQ2_0` kernels and 52 Walsh–Hadamard kernels the format needs |
| Source build | **not attempted** — the prebuilt passed the on-device smoke bench, so the `-DCMAKE_CUDA_ARCHITECTURES=75` fallback stayed unused |
| Host | 2× Tesla T4 (15,360 MiB each, 70 W cap), driver 580.159.04, system CUDA 13.0, nvcc 12.8 already at `/usr/local/cuda/bin/nvcc` (no pip/deb fallback needed), 4 vCPU Xeon @2.0 GHz, 31 GB RAM |
| Note | `/kaggle/temp` does not exist on this image; the script fell back to `/tmp` (20 GB loop device on `/kaggle/working` was untouched) |

The only cost of the JIT path: the **first** process to load a CUDA context pays ~110 s of
model load (PTX JIT), every later process ~6 s (driver JIT cache). Guarded by the smoke run
at the top of the protocol — no per-step surprise.

## 2. Times

| step | measured |
|---|---|
| runtime tarball download + extract (248 MB) | 7.9 s |
| PTQ1_0 model download (5.54 GiB) | 41.7 s (136 MiB/s) |
| PQ2_0 model download (6.71 GiB, bonus) | 33.8 s (204 MiB/s) |
| first CUDA model load (PTX JIT) | ~110 s |
| later model loads | ~6 s |
| **whole kernel session** | **422 s (7.04 min)** |

## 3. Throughput (llama-bench `-ngl 99 -p 512 -n 128 -r 3`, batch 2048/512, KV f16, no `-fa`)

| config | pp512 tok/s | tg128 tok/s |
|---|---:|---:|
| **PTQ1_0, 1×T4** | **197.3** ±0.9 | **14.78** ±0.29 |
| **PTQ1_0, 2×T4 (`-sm layer`)** | **197.7** ±0.4 | **17.76** ±0.05 |
| PTQ1_0, 1×T4, `-fa on` | 190.0 | 14.55 |
| PQ2_0, 1×T4 (bonus) | 298.0 ±3.7 | 15.20 ±0.23 |

Readings:

- **Layer-splitting across both T4s helps decode by +20%** (14.78 → 17.76) and is neutral for
  prefill. The model fits in one card (5.9 GB + KV), so the win is compute spread, not capacity.
- **Flash attention is a small loss on Turing** (`-fa on`: −4% pp, −2% tg), so the vendor's
  `-fa on` quickstart is worth dropping on this GPU.
- **PQ2_0 beats PTQ1_0 on a T4 on both axes** — +51% prefill, +3% decode — the opposite of the
  model card's Ada/A100 split, where PTQ1_0 wins decode. For prefill-dominated agent loops on
  Turing, PQ2_0 is the pack to use; PTQ1_0's advantage is footprint (5.5 vs 6.7 GiB).
- Sanity vs the vendor's own table: the closest listed part is an L4 (72 W) — T4 lands at
  46% / 42% of its PTQ1_0 decode / prefill, consistent with Turing-vs-Ada.

## 4. Completion quality (300-token SWE prompt, temp 0, seed 0, `--jinja -st`, 2×T4)

Two runs: 256 tokens (the protocol) and 768 tokens (to get past the thinking budget).

| run | wall | prompt | generation |
|---|---:|---:|---:|
| main, 256 tok | 35.4 s | 140.2 tok/s (~298 tok) | 17.6 tok/s |
| extended, 768 tok | 52.3 s | 135.6 tok/s | 17.3 tok/s |

**Coherent and agentic — yes.** It opens a `[Start thinking]` block, reasons about how to
investigate, and emits a concrete, correct-looking command list, e.g.:

```
git rev-parse HEAD
python -m pytest tests/test_duration.py -x -q
python - <<'PY'
from utils.timeparse import parse_duration, _UNIT_RE
for s in ["1h30m", "PT2H", "90s", "5m", "1H", "PT2H", "PT1H30M", "P1DT2H"]:
    print(repr(s), parse_duration(s), bool(_UNIT_RE.match(s)))
PY
cat utils/timeparse.py
grep -rn "parse_duration" .
grep -n "timeout" middleware/retry.py
```

Both runs stopped at the token cap, mid-reasoning — this is a thinking model (`xhigh` effort
by default) and 256 tokens is entirely consumed by the think block, so a rollout harness needs
a thinking budget (or `medium` effort) before it sees an action. No loops, no gibberish, no
template leakage. Raw output: `out/bonsai_completion_{main,extended}_jinja_single_turn.txt`.

## 5. GPU quota spent

**7.04 min wall on the T4x2 shape ≈ 0.23 GPU-h** if Kaggle bills 2×T4x2 at two GPU-hours per
wall hour — **0.8% of the 30 h weekly quota**. One push, no re-run.

## 6. Rollout verdict (SWE-agent rollout = 70 turns, 20k generated tokens, 1.5M prefill tokens)

Prefill dominates: with no cache credit, the prefill term is 84–178 min against ~19–23 min of
generation.

| config | prefill | generation | rollout | rollouts / 12 h |
|---|---:|---:|---:|---:|
| PTQ1_0, 1×T4 | 127 min | 22.6 min | **149 min** | **4.8** |
| PTQ1_0, 2×T4 | 126 min | 18.8 min | **145 min** | **5.0** |
| **PQ2_0, 1×T4** | 84 min | 21.9 min | **106 min** | **6.8** |
| PTQ1_0, 2×T4, using the completion's *measured* 140 pp tok/s | 178 min | 18.9 min | 197 min | 3.7 |

KV-cache-reuse sensitivity (PTQ1_0, 2×T4) — real agent loops re-use the prefix, so the
no-reuse row is the pessimistic bound:

| prefill served from cache | rollout | rollouts / 12 h |
|---|---:|---:|
| 0% | 145 min | 5.0 |
| 50% | 82 min | 8.8 |
| 80% | 44 min | 16.3 |

**Reading:** on a free 2×T4 session this model supports roughly **5–7 rollouts per 12 h**
without prefill caching, and up to roughly **9–16** once a rollout harness reuses KV prefixes.
It is the prefill phase, not decoding, that decides throughput — so use **PQ2_0**, keep the
prompt prefix stable across turns, and consider `medium` reasoning effort (the default
`xhigh` spends the whole budget thinking). Fine for evaluation-style runs and small
experiments; not a throughput platform for large SFT-trace generation, where the 30 h/week
quota (~5 rollouts/h of wall time) is the binding limit, not the model.

## 7. What failed, root causes, and whether the fixes are plausible

Nothing failed that mattered — status `ok`, no step skipped. Three small instrument findings:

1. **`llama-cli -no-cnv` is rejected** by this build (`error: invalid argument: -no-cnv`), so the
   script's fallback completion mode exited rc=1. The `--jinja -st` mode is the one that ran and
   worked, so no result was lost. Fix: trivially avoidable (use `--no-conversation`/jinja path only).
2. **Perf block lands on stdout, not stderr, in llama-cli**: my metrics parser looked for the
   `common_perf_print` block on stderr, so the recorded `completion` block captured the failed
   retry instead; the real numbers survive in the raw `.txt` artifacts and were folded back in as
   `completion_corrected` in `out/bonsai_metrics.json`. Fix: parse the inline
   `[ Prompt: X t/s | Generation: Y t/s ]` console line (or both streams).
3. **`llama-bench --version` is not supported** in this build (rc=1, prints usage) — cosmetic
   version probe only.

## 8. Caveats — what this does *not* claim

- The T4 runs **PTX-JIT kernel images**; a native `sm_75` source build was never tested and could
  be modestly faster (the vendor compiles no sm_75 cubins). The build path is wired up in
  `bonsai_trial.py` and would need one more push to measure — not done, by design (one push).
- pp512 is a 512-token yardstick: prefill throughput on much longer agent contexts will be
  somewhat lower (the completion measured 140 t/s on a ~300-token prompt vs 197 t/s at pp512).
- The completion runs used `-c 16384`, temp 0, seed 0 and a synthetic SWE issue — enough to judge
  coherence and speed, not enough to judge task success.
- Single push, single measurement of each configuration (`-r 3` for the main benches, `-r 2` for
  PQ2_0, `-r 1` for the FA variant). Spreads were tight (≤2% on the headline numbers).
