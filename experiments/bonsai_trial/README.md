# bonsai_trial — Ternary-Bonsai-2-27B (PTQ1_0) on a free Kaggle 2xT4

## Question

Can PrismML's **Ternary Bonsai 2 27B** in the custom `PTQ1_0` ternary GGUF format run on a
Kaggle T4 (sm_75), and how fast is it — token generation, prefill, single vs dual GPU — for
the purpose of budgeting SWE-agent rollouts against the 30 h/week GPU quota? Findings:
`RESULT.md`.

## Config

| item | value |
|---|---|
| Kernel | `evandekim/bonsai-t4-trial` (`kernel-metadata.json`; `machine_shape=NvidiaTeslaT4` with `enable_gpu="false"` — that combination is required, `enable_gpu="true"` + `machine_shape` returns HTTP 400) |
| Code | `bonsai_trial.py` (standalone; runs on Kaggle, imports nothing from the repo) |
| Runtime | PrismML llama.cpp fork release [`prism-b10685-7dffb15`](https://github.com/PrismML-Eng/llama.cpp/releases/tag/prism-b10685-7dffb15), asset `llama-prism-b10685-7dffb15-bin-linux-cuda-12.4-x64.tar.gz` (248 MB prebuilt) |
| Model | `prism-ml/Ternary-Bonsai-2-27B-gguf` :: `Ternary-Bonsai-2-27B-PTQ1_0.gguf` (5.93 GB) |
| Bench | `llama-bench -ngl 99 -p 512 -n 128 -r 3 -o json`, 1xT4 then 2xT4 with `-sm layer` |
| Completion | `llama-cli -c 16384 -ngl 99 --temp 0 --seed 0 -n 256 --jinja -st`, SWE-style issue prompt (~300 tokens) |
| Cap | pushed with `-t 2700`; the script self-limits to `DEADLINE_S=2400` so `bonsai_metrics.json` lands before the kill-switch |

### Why the prebuilt is expected to work on a T4

The release is compiled with llama.cpp's default arch set — checked offline with `cuobjdump`:

```
cubins (native SASS): sm_86, sm_89, (sm_120 in the 12.8 asset)
PTX (driver JIT):     sm_50, sm_61, sm_70, sm_75, sm_80, sm_90
```

so an sm_75 device runs the embedded **sm_75 PTX through the CUDA driver's JIT**. The sm_75
PTX was checked to contain all 17 `PTQ1_0`/`PQ2_0` kernels and 52 Walsh–Hadamard kernels the
Ternary Bonsai 2 format needs (the 1.76 bpw packing uses a rotated weight basis plus an
activation-side FWHT — `prism.hadamard.*` keys in the GGUF metadata). Neither tarball bundles
`libcudart`/`libcublas`, so the script `ldd`-checks and, if needed, installs
`nvidia-cuda-runtime-cu12` + `nvidia-cublas-cu12` wheels into a local dir on `LD_LIBRARY_PATH`.

If the prebuilt fails on-device, the script falls back to a source build (`git clone -b prism`
+ `cmake -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=75 -DGGML_NATIVE=OFF`, `-j4`) and records
the build time, or the last 80 log lines on failure.

## Reproduce

```bash
cd experiments/bonsai_trial
./run_kernel.sh                                    # push + poll (60 s) + pull output
uv run --project /home/evan/Documents/kaggle_e4b kaggle kernels status evandekim/bonsai-t4-trial
uv run --project /home/evan/Documents/kaggle_e4b kaggle kernels output evandekim/bonsai-t4-trial -p out
```

GPU quota is 30 h/week: push once, read `RESULT.md` before considering a re-push.

## Outputs

| Path | Contents |
|---|---|
| `out/bonsai_metrics.json` | everything: env (nvidia-smi, nvcc discovery), runtime path + timings, model download, all bench results, completion perf + reply, elapsed time, per-command timings (`completion_corrected` holds the real completion numbers, see RESULT.md §7) |
| `out/bonsai_bench_{smoke_prebuilt,single_gpu,dual_gpu,fa_on_single_gpu,pq2_single_gpu}.json` | raw llama-bench JSON per configuration |
| `out/bonsai_completion_{main,extended}_jinja_single_turn.txt` / `.stderr.txt` | raw completion answer and the runtime log |
| `out/bonsai-t4-trial.log` | raw kernel log (JSON stream) pulled after the run |
| `RESULT.md` | verdict for the run |
| `poll_status.log` | kernel status poll history |

## What was measured

One push, 2026-09-17, status `ok`, 7.04 min wall. It **runs on sm_75** with the prebuilt
release (no source build needed — the release has sm_75 PTX, no sm_75 cubins, so the driver
JIT-compiles on first load: ~110 s once, ~6 s afterwards). On the 2×T4 shape, PTQ1_0:
**pp512 197.3 / tg128 14.78 tok/s on one T4, 197.7 / 17.76 on two** (`-sm layer`); PQ2_0
prefills 51% faster (298 tok/s). A 256-token completion answered with a coherent agentic
command list at 17.6 tok/s. Full numbers, caveats and the rollout verdict (≈5 rollouts/12 h
uncached, ≈6.8 with PQ2_0, up to ≈16 with 80% KV reuse) are in `RESULT.md`.
