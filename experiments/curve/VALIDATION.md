# VALIDATION — 2×T4 paths (Unsloth fp16 LoRA + nccl DDP), 2026-09-17

**Verdict: FAILED — kernel errored ~2 min in, before the first training step. Do not re-push.**
The parts of the stack that could not be tested locally did partially prove out (2 T4s visible, DDP
spawn ran, nccl init succeeded, manifest halves encoded on both ranks), but the model build never
ran: the Kaggle image's Unsloth stack is import-broken against the image's `transformers`
(`merge_with_config_defaults` missing), its pip repair path is blocked by a broken `dill` dist-info,
and the failed Unsloth import poisons the plain-transformers fallback in the same process.
No second push was made (kernel stays at version 1). GPU cost of the run: ~2 minutes.

## Run provenance

| item | value |
|---|---|
| Dataset | `evandekim/openswe-curve-manifests` — created this session, status `ready` before push (60 MB, 5 jsonl + summary) |
| Kernel | `evandekim/openswe-curve` **version 1**, pushed 2026-09-17 ~22:53Z, `-t 1800` (30 min cap) |
| Machine | `NvidiaTeslaT4` shape with `enable_gpu="false"` in metadata — **confirmed to deliver 2 GPUs** (`n_gpus: 2`) |
| Baked config | `{"model": "Qwen/Qwen2.5-Coder-1.5B-Instruct", "manifest": "random", "n_gpus": 2, "world_size": 2, "max_len": 6144, "max_steps": 20, "grad_accum": 8, "max_sup_rows": 2048, "lr": 0.0002, "seed": 0, "eval_every": 10, "stop_after_seconds": 1500.0}` (printed at t=31.9 s) |
| Status | `RUNNING` (22:53:30Z, 22:54:32Z) → `KernelWorkerStatus.ERROR` (22:55:16Z rank-1 error record; observed 22:55:33Z) |
| Artifacts | `out/validate/openswe-curve.log` (raw log), `out/validate/metrics_error_rank1.json`, `out/validate/unsloth_compiled_cache/moe_utils.py` |

The CONFIG block in `train_curve.py` worked as intended: the run used the baked
validation defaults and the env-override path was verified locally before the push
(`BAKED random 20 10 6144 1500.0` / `OVERRIDE top_within_task 3 2 512 99.0`).

## What the run did establish (verified, on the T4 box)

| item | value | evidence |
|---|---|---|
| device_count seen | **2** (`n_gpus: 2`, `world_size: 2`) | config print, t=31.9 s |
| DDP spawn | 2 worker processes started | both ranks print below; `mp.spawn` re-raises at t=114.6 s |
| nccl init | **succeeded** — both ranks continued past `dist.init_process_group` with no "retrying with gloo" line | both `[rank N] encoded ...` lines (encode runs after init) |
| manifest half-split | 23 / 23 traces per rank (46-trace `random.jsonl`) | t=49.4 s / t=49.9 s |
| dataset mount | resolved via recursive glob (new `…/datasets/` path segment) | `…/kaggle/input/datasets/evandekim/openswe-curve-manifests/random.jsonl` |
| encode/data path | 6 s for 23 traces per rank; avg len 4915 / 4519; supervised frac 0.23 / 0.33 | rank prints, t=49 s |
| push/poll/pull pipeline | worked end to end | this directory |

## The requested measurements

| item | value |
|---|---|
| backend (unsloth or transformers) | **neither** — Unsloth import fails in this image; the transformers+peft fallback died for rank 1 at model-class resolution, and rank 0 was SIGTERM'd by the spawn parent before any record was written (a `Loading weights` bar from one of the two processes appears at t=92 s, so the ranks diverged) |
| DDP ran with 2 ranks? | spawn + nccl init + half-split yes; **no training step**, so gradient sync was never exercised |
| tok/s per rank / total | n/a — 0 of 20 steps completed |
| sup_tok/s per rank / total | n/a — 0 of 20 steps completed |
| peak mem | n/a — no step ran; no metrics snapshot was ever written |
| loss trajectory | n/a |
| eval CE values | n/a (eval at step 10 never reached) |

## Root cause (evidence chain)

1. **Unsloth is present in the image but import-broken.** `unsloth` + `unsloth_zoo` live in
   `/usr/local/lib/python3.12/dist-packages/`. The script's first `import unsloth` fails, its
   self-repair `pip install -q unsloth` also fails, and the retry still fails — the script logs
   `unsloth install failed (CalledProcessError(1, ['/usr/bin/python3', '-m', 'pip', 'install', '-q', 'unsloth']))`
   (t=69.3 s) and `unsloth import still fails after install` (t=110.0 s, printed after Unsloth's
   banner `🦥 Unsloth: Will patch your computer to enable 2x faster free finetuning.`).

2. **pip can't repair it: the image's `dill` install is corrupt.**
   ```
   ERROR: Could not install packages due to an OSError: [Errno 2] No such file or directory:
   '/usr/local/lib/python3.12/dist-packages/dill-0.4.1.dist-info/'
   ```
   pip tries to touch dill as part of resolving Unsloth's deps and aborts the whole transaction.

3. **The concrete incompatibility: `merge_with_config_defaults` does not exist in the image's
   `transformers.utils.generic`, while Unsloth Zoo's fused-loss import hook executes model code
   that imports it.** Final frames of the crash:
   ```
   File ".../unsloth_zoo/fused_losses/forward_install.py", line 372, in exec_module
       self._inner.exec_module(module)
   File ".../transformers/models/qwen2/modeling_qwen2.py", line 30, in <module>
       from ...utils.generic import maybe_autocast, merge_with_config_defaults
   ImportError: cannot import name 'merge_with_config_defaults' from 'transformers.utils.generic'
   ```
   Checked locally: stock transformers **5.17.0** has exactly this line 30 in `modeling_qwen2.py`
   **and** defines the symbol in `utils.generic`. The image has the modern modeling file but an
   older `utils.generic` (or Unsloth Zoo synthesizes code that assumes a newer transformers than
   installed) — either way the image's transformers/Unsloth pairing is internally inconsistent, an
   environment defect that hits whichever training config we pick.

4. **The failed Unsloth import poisons the fallback in the same process.** The plain
   `AutoModelForCausalLM.from_pretrained` call (script line 279) is intercepted by the already
   installed Unsloth Zoo import hook (`forward_install.exec_module`) and dies with the same
   ImportError. So the `USE_UNSLOTH=0`-style fallback was **not** actually tested in a clean
   process — sys.modules/import hooks were polluted by the earlier `import unsloth` attempt.

5. Minor, logged: `Skipping import of cpp extensions due to incompatible torch version. Please
   upgrade to torch >= 2.11.0 (found 2.10.0+cu128)` — Unsloth's cpp extensions skip on the image's
   torch (not fatal by itself).

`metrics_error_rank1.json`: `{"status": "error", "error": "ImportError(\"cannot import name
'merge_with_config_defaults' from 'transformers.utils.generic' (...)\")", "rank": 1,
"updated_at": "2026-09-17T22:55:16Z"}`. Rank 0's error record was never captured — the spawn
parent SIGTERM'd process 0 (t=114.2 s) once process 1 raised, and the only `Loading weights`
progress (t=92 s) shows one rank had gotten further than the other. The Unsloth import hook's
effect is per-process, so the two workers did not fail at the same instruction.

## Last 60 log lines (lines 102–161 of `out/validate/openswe-curve.log`, decoded)

Raw log has 160 entries; below are the last 60, decoded verbatim (`t` = seconds since container
start; container started ≈22:53:25Z — crash at t≈110.4 s matches the 22:55:16Z rank-1 record).

```
t=114.62 | stderr |               ^^^^^^^^^^^^^^
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/torch/multiprocessing/spawn.py", line 211, in join
t=114.62 | stderr |     raise ProcessRaisedException(msg, error_index, failed_process.pid)
t=114.62 | stderr | torch.multiprocessing.spawn.ProcessRaisedException: 
t=114.62 | stderr | 
t=114.62 | stderr | -- Process 1 terminated with the following error:
t=114.62 | stderr | Traceback (most recent call last):
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/torch/multiprocessing/spawn.py", line 87, in _wrap
t=114.62 | stderr |     fn(i, *args)
t=114.62 | stderr |   File "/kaggle/src/script.py", line 567, in worker_entry
t=114.62 | stderr |     train(rank, world_size)
t=114.62 | stderr |   File "/kaggle/src/script.py", line 398, in train
t=114.62 | stderr |     model, backend_name = build_model(device)
t=114.62 | stderr |                           ^^^^^^^^^^^^^^^^^^^
t=114.62 | stderr |   File "/kaggle/src/script.py", line 279, in build_model
t=114.62 | stderr |     base = AutoModelForCausalLM.from_pretrained(MODEL, torch_dtype=dtype)
t=114.62 | stderr |            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/transformers/models/auto/auto_factory.py", line 369, in from_pretrained
t=114.62 | stderr |     model_class = get_class_from_dynamic_module(
t=114.62 | stderr |                   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/transformers/models/auto/auto_factory.py", line 178, in _get_model_class
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/transformers/models/auto/auto_factory.py", line 570, in __getitem__
t=114.62 | stderr |     self._config_mapping = config_mapping
t=114.62 | stderr |                ^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/transformers/models/auto/auto_factory.py", line 584, in _load_attr_from_module
t=114.62 | stderr |     model_type = self._reverse_config_mapping[key.__name__]
t=114.62 | stderr |            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/transformers/models/auto/auto_factory.py", line 496, in getattribute_from_module
t=114.62 | stderr |     result = []
t=114.62 | stderr |        ^^^^^^^^^
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/transformers/utils/import_utils.py", line 2098, in __getattr__
t=114.62 | stderr |     callable = backend.is_satisfied
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/transformers/utils/import_utils.py", line 2288, in _get_module
t=114.62 | stderr |     logger.debug(f"Could not create tokenizer alias: {e}")
t=114.62 | stderr |     ^^^^^^^
t=114.62 | stderr |   File "/usr/local/lib/python3.12/dist-packages/transformers/utils/import_utils.py", line 2286, in _get_module
t=114.62 | stderr |     break
t=114.62 | stderr |   File "/usr/lib/python3.12/importlib/__init__.py", line 90, in import_module
t=114.62 | stderr |     return _bootstrap._gcd_import(name[level:], package, level)
t=114.63 | stderr |            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=114.63 | stderr |   File "<frozen importlib._bootstrap>", line 1387, in _gcd_import
t=114.63 | stderr |   File "<frozen importlib._bootstrap>", line 1360, in _find_and_load
t=114.63 | stderr |   File "<frozen importlib._bootstrap>", line 1331, in _find_and_load_unlocked
t=114.63 | stderr |   File "<frozen importlib._bootstrap>", line 935, in _load_unlocked
t=114.63 | stderr |   File "/usr/local/lib/python3.12/dist-packages/unsloth_zoo/fused_losses/forward_install.py", line 372, in exec_module
t=114.63 | stderr |     self._inner.exec_module(module)
t=114.63 | stderr |   File "<frozen importlib._bootstrap_external>", line 999, in exec_module
t=114.63 | stderr |   File "<frozen importlib._bootstrap>", line 488, in _call_with_frames_removed
t=114.63 | stderr |   File "/usr/local/lib/python3.12/dist-packages/transformers/models/qwen2/modeling_qwen2.py", line 30, in <module>
t=114.63 | stderr |     from ...utils.generic import maybe_autocast, merge_with_config_defaults
t=114.63 | stderr | ImportError: cannot import name 'merge_with_config_defaults' from 'transformers.utils.generic' (/usr/local/lib/python3.12/dist-packages/transformers/utils/generic.py)
t=114.63 | stderr | 
t=115.10 | stderr | /usr/lib/python3.12/multiprocessing/resource_tracker.py:279: UserWarning: resource_tracker: There appear to be 1 leaked semaphore objects to clean up at shutdown
t=115.10 | stderr |   warnings.warn('resource_tracker: There appear to be %d '
t=117.85 | stderr | /usr/local/lib/python3.12/dist-packages/mistune.py:435: SyntaxWarning: invalid escape sequence '\|'
t=117.85 | stderr |   cells[i][c] = re.sub('\\\\\|', '|', cell)
t=118.09 | stderr | /usr/local/lib/python3.12/dist-packages/nbconvert/filters/filter_links.py:36: SyntaxWarning: invalid escape sequence '\_'
t=118.09 | stderr |   text = re.sub(r'_', '\_', text) # Escape underscores in display text
t=118.76 | stderr | [NbConvertApp] Converting notebook __script__.ipynb to html
t=119.68 | stderr | [NbConvertApp] Writing 389086 bytes to __results__.html
```

## Budget projection

**Cannot be computed at a measured rate — no training step completed, so fp16/DDP throughput is
unmeasured.** For reference, the formula this table will use once a rate exists:

```
windows_per_12h = tok_per_s_total × 43200 / avg_window_tokens
hours(arm)      = windows × avg_window_tokens / tok_per_s_total / 3600
```

Reference-only numbers at the **smoke baseline** (317 tok/s single GPU, 4-bit, avg window
5,046 tokens — *not* this run's rate, do not budget from these for fp16/DDP):

| windows | hours at 317 tok/s | fits a 12 h session? |
|---:|---:|:--:|
| 2,000 | 8.8 | yes |
| 4,000 | 17.7 | no |
| 8,000 | 35.4 | no |
| 16,000 | 70.8 | no |

`windows_per_12h` at that baseline: ~2,700 windows.

## Comparison vs the smoke baseline (317 tok/s single GPU 4-bit)

Not comparable this run: the smoke run measured a training step; this run died before the model
was built. The fp16-vs-NF4 speed/memory question and the 2×T4 scaling factor remain **unmeasured**.
Everything here is a setup/import-layer failure, not a throughput result.

## What a future attempt needs (diagnosis first — no re-push was made)

1. **Probe Unsloth in a subprocess, never in-process.** `import unsloth` here poisons the process
   (its meta-path hook survives the failed import) and kills the fallback too. A one-line
   `subprocess.run([sys.executable, "-c", "import unsloth"])` probe would return False cleanly and
   let the transformers fallback run in an unpoisoned process. (Script change — not made.)
2. **Fix the image's Unsloth/transformers pair, or pin one.** The image's `transformers.utils.generic`
   lacks `merge_with_config_defaults`; the executed qwen2 modeling code requires it. Options: install
   a matching `transformers` in the kernel before importing (`pip install -U transformers`), or use an
   image/runtime where Unsloth matches. Verify with a **CPU kernel preflight** (no GPU quota):
   `pip install -q unsloth transformers && python -c "import unsloth"`.
3. **Work around the corrupt `dill` dist-info** if pip must repair packages
   (`pip install --ignore-installed dill` / `--force-reinstall`), since the plain `pip install -q unsloth`
   aborts with the `dill-0.4.1.dist-info` OSError.
4. Only after 1–3 pass in a CPU preflight is a GPU push worth the quota. The DDP plumbing
   (spawn, nccl, half-split) already validated here and is not the risk.

## Artifacts

| path | contents |
|---|---|
| `out/validate/openswe-curve.log` | raw Kaggle log stream, 160 entries |
| `out/validate/metrics_error_rank1.json` | rank-1 error record (`ImportError(...)`, 22:55:16Z) |
| `out/validate/unsloth_compiled_cache/moe_utils.py` | evidence Unsloth Zoo's compile/import machinery ran before failing |

## v2 — plain transformers+peft fp16, 2×T4, pushed 2026-09-17 ~23:07Z (Unsloth off by default)

**Verdict: FAILED — again before the first training step, but one layer deeper: the plain path
now builds the model and dies inside PEFT's LoRA injection. No third push.**
Unsloth was never imported (default `USE_UNSLOTH=0` held; zero `unsloth` strings in the whole
log), the fp16 checkpoint loaded fine through the image's own transformers, and both ranks died
identically on the image's stale `torchao` 0.10.0: PEFT's LoRA dispatcher calls
`is_torchao_available()` unconditionally and *raises* (rather than returning False) for
unsupported versions, so **every** `get_peft_model` fails in this image regardless of recipe.
GPU cost of the run: ~2 minutes of session time.

### Run provenance

| item | value |
|---|---|
| Code | `train_curve.py` with the new `USE_UNSLOTH=0` default: no Unsloth code path reached, no `unsloth`/`unsloth_zoo` import anywhere, `sys.modules` guard did not trip |
| Kernel | `evandekim/openswe-curve` **version 2**, pushed 2026-09-17 ~23:07Z in one attempt, `-t 1800` (30 min cap) |
| Machine | `NvidiaTeslaT4` shape with `enable_gpu="false"` — again delivered **2 GPUs** (`n_gpus: 2`, `world_size: 2`) |
| Baked config | `{"model": "Qwen/Qwen2.5-Coder-1.5B-Instruct", "manifest": "random", "n_gpus": 2, "world_size": 2, "max_len": 6144, "max_steps": 20, "grad_accum": 8, "max_sup_rows": 2048, "lr": 0.0002, "seed": 0, "eval_every": 10, "stop_after_seconds": 1500.0}` (printed at t=32.6 s) |
| Status | `RUNNING` (23:07:14Z) → both rank error records written 23:08:41Z (t=91.6 s) → `KernelWorkerStatus.ERROR` observed 23:09:16Z; container start ≈23:07:09Z |
| Artifacts | `out/validate2/openswe-curve.log` (136 entries; decoded copy `openswe-curve-decoded.txt`), `metrics_error.json`, `metrics_error_rank1.json` |

### What the run did establish (verified, on the T4 box)

| item | value | evidence |
|---|---|---|
| device_count seen | **2** (`n_gpus: 2`, `world_size: 2`) | config print, t=32.6 s |
| DDP spawn | 2 worker processes started | rank prints below |
| nccl init | **succeeded** — no "retrying with gloo" line, zero `gloo` strings in the log | both ranks continued past `dist.init_process_group` |
| manifest half-split | 23 / 23 traces per rank (46-trace `random.jsonl`) | t=49.1 s / t=49.8 s |
| encode/data path | 5–6 s for 23 traces per rank; avg len 4915 / 4519; supervised frac 0.23 / 0.33 | rank prints, t=49 s |
| **plain fp16 model load** | **succeeded** — 338/338 tensors materialized for Qwen2.5-Coder-1.5B-Instruct through the image's transformers (fp16, SDPA, no bitsandbytes); both ranks reached `get_peft_model`, which is only reachable after a successful load | `Loading weights` progress t≈80–85 s; crash frames at `script.py:297` |
| image transformers | current-generation (emits the 5.x-style `` `torch_dtype` is deprecated! `` warning, once per rank) and imports qwen2 modeling cleanly in an unpoisoned process — the v1 `merge_with_config_defaults` error was specific to Unsloth Zoo's import hook, not a transformers defect | t=56.6 s / t=57.1 s warnings; successful load |
| Unsloth | never touched — 0 mentions of `unsloth` anywhere in the decoded log | decoded-log grep |
| training steps | 0 of 20 — the crash is inside `build_model`, before `metrics.json` exists | error records |

### The requested measurements

| item | value |
|---|---|
| backend (unsloth or transformers) | **plain transformers+peft** (the traceback is inside the plain path's `get_peft_model`; Unsloth was never imported) — still no on-disk `backend` record because no `metrics.json` was ever written |
| DDP ran with 2 ranks? | spawn + nccl init + half-split yes, and **both ranks reached model build**; no training step, so gradient sync was never exercised |
| tok/s per rank / total | n/a — 0 of 20 steps completed |
| sup_tok/s per rank / total | n/a — 0 of 20 steps completed |
| peak mem | n/a — no step ran, and `model.to(device)` was never reached (the successful load is CPU-side weight materialization) |
| loss trajectory | n/a |
| eval CE values | n/a (eval at step 10 never reached) |

### Root cause (evidence chain)

1. **Both ranks crash inside PEFT's LoRA module dispatcher, not in model loading.** Rank traceback
   (t=91.6 s, verbatim, `script.py` is the pushed `train_curve.py`):

   ```
   Traceback (most recent call last):
     File "/kaggle/src/script.py", line 581, in worker_entry
       train(rank, world_size)
     File "/kaggle/src/script.py", line 412, in train
       model, backend_name = build_model(device)
                             ^^^^^^^^^^^^^^^^^^^
     File "/kaggle/src/script.py", line 297, in build_model
       model = get_peft_model(
               ^^^^^^^^^^^^^^^
     File "/usr/local/lib/python3.12/dist-packages/peft/mapping_func.py", line 122, in get_peft_model
       return MODEL_TYPE_TO_PEFT_MODEL_MAPPING[peft_config.task_type](
              ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
     File "/usr/local/lib/python3.12/dist-packages/peft/peft_model.py", line 1955, in __init__
       super().__init__(model, peft_config, adapter_name, **kwargs)
     File "/usr/local/lib/python3.12/dist-packages/peft/peft_model.py", line 129, in __init__
       self.base_model = cls(model, {adapter_name: peft_config}, adapter_name)
                         ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
     File "/usr/local/lib/python3.12/dist-packages/peft/tuners/tuners_utils.py", line 315, in __init__
       self.inject_adapter(self.model, adapter_name, low_cpu_mem_usage=low_cpu_mem_usage, state_dict=state_dict)
     File "/usr/local/lib/python3.12/dist-packages/peft/tuners/tuners_utils.py", line 913, in inject_adapter
       self._create_and_replace(
     File "/usr/local/lib/python3.12/dist-packages/peft/tuners/lora/model.py", line 269, in _create_and_replace
       new_module = self._create_new_module(lora_config, adapter_name, target, device_map=device_map, **kwargs)
                    ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
     File "/usr/local/lib/python3.12/dist-packages/peft/tuners/lora/model.py", line 418, in _create_new_module
       new_module = dispatcher(target, adapter_name, config=lora_config, **kwargs)
                    ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
     File "/usr/local/lib/python3.12/dist-packages/peft/tuners/lora/torchao.py", line 142, in dispatch_torchao
       if not is_torchao_available():
              ^^^^^^^^^^^^^^^^^^^^^^
     File "/usr/local/lib/python3.12/dist-packages/peft/import_utils.py", line 143, in is_torchao_available
       raise ImportError(
   ImportError: Found an incompatible version of torchao. Found version 0.10.0, but only versions above 0.16.0 are supported
   ```

2. **The image's `torchao` (0.10.0) is too old for the image's PEFT, and this PEFT version raises
   instead of reporting unavailable.** `dispatch_torchao` runs as part of the dispatcher chain for
   plain `nn.Linear` targets (we never ask for torchao quantization), and `is_torchao_available()`
   raises `ImportError` on version mismatch rather than returning `False`. Net effect: **any**
   `get_peft_model` call fails in this image, whatever the training recipe. Nothing in our config
   touches torchao or bitsandbytes.

3. **The v1 failure class is gone.** Zero `unsloth` strings in 136 log entries; the guard never
   tripped because nothing imported it; the model loaded through the plain path with the image's
   own transformers. The v1 hypothesis that the image's transformers/Unsloth pairing was
   internally inconsistent is now refined: the missing `merge_with_config_defaults` import came
   from Unsloth Zoo's re-execution hook, not from the installed transformers, which loads and
   imports qwen2 fine in a clean process.

4. **Both ranks failed at the same instruction this time** (v1's ranks diverged), and rank 0's
   error record was captured (`metrics_error.json`; v1 only had rank 1's).

5. Cosmetic on the error path only: `[rank0] Warning: destroy_process_group() was not called
   before program exit` (t=92.1 s) — rank exit via exception skips the barrier; and spawn parent
   `Terminating process 30 via signal SIGTERM` (t=94.7 s) after process 1 raised.

### Last 60 log lines (entries 77–136 of `out/validate2/openswe-curve.log`, decoded)

`t` = seconds since container start; both ranks' exit tracebacks (t=91.6 s) sit immediately
*before* this window, quoted in the root-cause section above.

```
t=   92.12 | stderr | [rank0]:[W917 23:08:42.741166388 ProcessGroupNCCL.cpp:1553] Warning: WARNING: destroy_process_group() was not called before program exit, which can leak resources. For more info, please see https://pytorch.org/docs/stable/distributed.html#shutdown (function operator())
t=   94.68 | stderr | W0917 23:08:44.750000 7 torch/multiprocessing/spawn.py:165] Terminating process 30 via signal SIGTERM
t=   94.82 | stderr | Traceback (most recent call last):
t=   94.82 | stderr |   File "/kaggle/src/script.py", line 642, in <module>
t=   94.82 | stderr |     main()
t=   94.82 | stderr |   File "/kaggle/src/script.py", line 636, in main
t=   94.82 | stderr |     mp.spawn(worker_entry, args=(world,), nprocs=world, join=True)
t=   94.82 | stderr |   File "/usr/local/lib/python3.12/dist-packages/torch/multiprocessing/spawn.py", line 340, in spawn
t=   94.82 | stderr |     return start_processes(fn, args, nprocs, join, daemon, start_method="spawn")
t=   94.82 | stderr |            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=   94.82 | stderr |   File "/usr/local/lib/python3.12/dist-packages/torch/multiprocessing/spawn.py", line 296, in start_processes
t=   94.83 | stderr |     while not context.join():
t=   94.83 | stderr |               ^^^^^^^^^^^^^^
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/torch/multiprocessing/spawn.py", line 211, in join
t=   94.83 | stderr |     raise ProcessRaisedException(msg, error_index, failed_process.pid)
t=   94.83 | stderr | torch.multiprocessing.spawn.ProcessRaisedException:
t=   94.83 | stderr | 
t=   94.83 | stderr | -- Process 1 terminated with the following error:
t=   94.83 | stderr | Traceback (most recent call last):
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/torch/multiprocessing/spawn.py", line 87, in _wrap
t=   94.83 | stderr |     fn(i, *args)
t=   94.83 | stderr |   File "/kaggle/src/script.py", line 581, in worker_entry
t=   94.83 | stderr |     train(rank, world_size)
t=   94.83 | stderr |   File "/kaggle/src/script.py", line 412, in train
t=   94.83 | stderr |     model, backend_name = build_model(device)
t=   94.83 | stderr |                           ^^^^^^^^^^^^^^^^^^^
t=   94.83 | stderr |   File "/kaggle/src/script.py", line 297, in build_model
t=   94.83 | stderr |     model = get_peft_model(
t=   94.83 | stderr |             ^^^^^^^^^^^^^^^
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/peft/mapping_func.py", line 122, in get_peft_model
t=   94.83 | stderr |     return MODEL_TYPE_TO_PEFT_MODEL_MAPPING[peft_config.task_type](
t=   94.83 | stderr |            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/peft/peft_model.py", line 1955, in __init__
t=   94.83 | stderr |     super().__init__(model, peft_config, adapter_name, **kwargs)
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/peft/peft_model.py", line 129, in __init__
t=   94.83 | stderr |     self.base_model = cls(model, {adapter_name: peft_config}, adapter_name)
t=   94.83 | stderr |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/peft/tuners/tuners_utils.py", line 315, in __init__
t=   94.83 | stderr |     self.inject_adapter(self.model, adapter_name, low_cpu_mem_usage=low_cpu_mem_usage, state_dict=state_dict)
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/peft/tuners/tuners_utils.py", line 913, in inject_adapter
t=   94.83 | stderr |     self._create_and_replace(
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/peft/tuners/lora/model.py", line 269, in _create_and_replace
t=   94.83 | stderr |     new_module = self._create_new_module(lora_config, adapter_name, target, device_map=device_map, **kwargs)
t=   94.83 | stderr |                  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/peft/tuners/lora/model.py", line 418, in _create_new_module
t=   94.83 | stderr |     new_module = dispatcher(target, adapter_name, config=lora_config, **kwargs)
t=   94.83 | stderr |                  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/peft/tuners/lora/torchao.py", line 142, in dispatch_torchao
t=   94.83 | stderr |     if not is_torchao_available():
t=   94.83 | stderr |            ^^^^^^^^^^^^^^^^^^^^^^
t=   94.83 | stderr |   File "/usr/local/lib/python3.12/dist-packages/peft/import_utils.py", line 143, in is_torchao_available
t=   94.83 | stderr |     raise ImportError(
t=   94.83 | stderr | ImportError: Found an incompatible version of torchao. Found version 0.10.0, but only versions above 0.16.0 are supported
t=   94.83 | stderr | 
t=   97.97 | stderr | /usr/local/lib/python3.12/dist-packages/mistune.py:435: SyntaxWarning: invalid escape sequence '\|'
t=   97.97 | stderr |   cells[i][c] = re.sub('\\\\\|', '|', cell)
t=   98.21 | stderr | /usr/local/lib/python3.12/dist-packages/nbconvert/filters/filter_links.py:36: SyntaxWarning: invalid escape sequence '\_'
t=   98.21 | stderr |   text = re.sub(r'_', '\_', text) # Escape underscores in display text
t=   98.82 | stderr | [NbConvertApp] Converting notebook __script__.ipynb to html
t=  100.17 | stderr | [NbConvertApp] Writing 390061 bytes to __results__.html
```

### Budget projection

**Cannot be computed at a measured rate — still no training step completed, so fp16/DDP
throughput is unmeasured.** Formula this table will use once a rate exists:

```
windows_per_12h = tok_per_s_total × 43200 / avg_window_tokens
hours(arm)      = windows × avg_window_tokens / tok_per_s_total / 3600
```

Reference-only numbers at the **smoke baseline** (317 tok/s single GPU, 4-bit, avg window
5,046 tokens — *not* this run's rate, do not budget from these for fp16/DDP):

| windows | hours at 317 tok/s | fits a 12 h session? |
|---:|---:|:--:|
| 2,000 | 8.8 | yes |
| 4,000 | 17.7 | no |
| 8,000 | 35.4 | no |
| 16,000 | 70.8 | no |

`windows_per_12h` at that baseline: ~2,700 windows.

### Comparison vs the smoke baseline (317 tok/s single GPU 4-bit)

Not comparable on throughput: 0 of 20 steps ran, so the fp16-vs-NF4 speed/memory question and the
2×T4 scaling factor remain **unmeasured**. New signal from v2: the unquantized fp16 1.5B
checkpoint **loads cleanly** through the image's transformers (338/338 tensors, no quantizer),
and the pipeline now fails only at LoRA adapter injection — one library layer short of the
training loop.

### What a future attempt needs (diagnosis first — no third push was made)

1. **Neutralize the image's `torchao` before PEFT is imported.** This recipe never uses torchao
   (plain fp16 LoRA on `nn.Linear` targets), so either `pip uninstall -y torchao` in-kernel, or
   make the availability probe return `False` (patch
   `peft.tuners.lora.torchao.is_torchao_available` / `peft.import_utils.is_torchao_available`
   before `get_peft_model`). Prefer uninstall over `pip install` — the image's corrupt
   `dill-0.4.1.dist-info` aborts install transactions; an uninstall of an unrelated package does
   not touch it.
2. **CPU preflight that reproduces the image's PEFT+torchao pair (no GPU quota):** build a tiny
   model, run `get_peft_model` with this `LoraConfig`, then a 2-step CPU train. The preflight
   should assert `is_torchao_available()` no longer raises and that injection succeeds.
3. Only after 1–2 pass is a GPU push worth the quota. Everything after `build_model` (the DDP
   step loop, nccl gradient sync, metrics writes, stop guard) is still unexercised.

### Artifacts (v2)

| path | contents |
|---|---|
| `out/validate2/openswe-curve.log` | raw Kaggle log stream, 136 entries |
| `out/validate2/openswe-curve-decoded.txt` | decoded log (`t=… | stream | line`), used for the quotes above |
| `out/validate2/metrics_error.json` | rank-0 error record (`ImportError('Found an incompatible version of torchao…')`, 23:08:41Z) |
| `out/validate2/metrics_error_rank1.json` | rank-1 error record (same, 23:08:41Z) |

## v3 — plain transformers+peft fp16, 2×T4, Kaggle startup pip step, pushed 2026-09-17 ~23:26Z

**Verdict: PASSED the layers that failed in v1/v2 — both ranks built the model, ran 14 real
DDP steps each, and saved the adapter; the run ended by the 25-min stop guard, not by an
error. Throughput is finally measured: 685.3 tok/s combined (fp16, 2×T4, no quantization)
= 2.16× the smoke baseline (317 tok/s, 1×T4 NF4). GPU cost: ~28 min of session time.**

The only code change vs v2 is the Kaggle-only startup step, run before any transformers/peft
import (`USE_UNSLOTH=0` unchanged): on `/kaggle/working`, `pip install -q --upgrade peft>=0.13
accelerate` then `pip uninstall -y torchao` (failure ignored). Locally the step is skipped;
the CPU dry run (`MODEL=Qwen/Qwen2.5-0.5B-Instruct MAX_LEN=512 MAX_STEPS=2 EVAL_EVERY=1`)
passed before the push (2 steps, eval CE 3.0207 → 2.7035, adapter saved).

### Run provenance

| item | value |
|---|---|
| Code | `train_curve.py` + the Kaggle-only startup step; nothing else changed |
| Kernel | `evandekim/openswe-curve` **version 3**, pushed 2026-09-17 ~23:26Z in one attempt (no HTTP 500), `-t 1800` (30 min cap) |
| Machine | `NvidiaTeslaT4` shape with `enable_gpu="false"` — again **2 GPUs** (`n_gpus: 2`, `world_size: 2`) |
| Baked config | same as v1/v2 (printed t=34.1 s) |
| Status | `RUNNING` 23:27:09Z (first poll) → `COMPLETE` observed 23:55:39Z; container t≈1,680 s at the last log line (~28 min), exited before the 30-min cap (`stop_reason: stop_after_seconds`) |
| Artifacts | `out/validate3/openswe-curve.log` (37 entries), `openswe-curve-decoded.txt` (1,411 lines), `metrics.json`, `metrics_rank1.json`, `adapter/` (73.9 MB safetensors) |

### The startup fix, as it appears in the log

- t=4.1 s: the `-q` upgrade of `peft>=0.13`/`accelerate` (its two wheel downloads — nothing
  else in the run installs anything).
- t=8.35–8.71 s: `Found existing installation: torchao 0.10.0` →
  `Uninstalling torchao-0.10.0` → `Successfully uninstalled torchao-0.10.0`.
- t=38.65 / 38.76 s: the two spawned workers re-run the block at import time; torchao is
  already gone → `WARNING: Skipping torchao as it is not installed.` (no-op).
- `get_peft_model` — v2's exact crash point — then succeeded on both ranks, and the adapter
  was written (v2 died there with `ImportError: Found an incompatible version of torchao`).

### What the run did establish (verified, on the T4 box)

| item | value | evidence |
|---|---|---|
| device_count seen | **2** (`n_gpus: 2`, `world_size: 2`) | config print, t=34.1 s |
| DDP spawn | 2 worker processes | both ranks print; nccl init with **no gloo fallback** (zero `gloo` strings in the log) |
| manifest half-split | 23 / 23 traces per rank (46-trace `random.jsonl`) | t=55.5 s / t=56.2 s |
| encode/data path | 5–6 s per rank; avg len 4915 / 4519; supervised frac 0.23 / 0.33 | rank prints |
| model build (the v2 failure point) | **passes** — plain fp16 load + LoRA injection on both ranks | step records exist |
| training | **14 steps per rank** (`steps_done: 14` in both records) | `metrics.json` + `metrics_rank1.json` |
| gradient sync | exercised — DDP all_reduce on the last micro-batch per step; eval CE identical on both ranks | eval records below |
| adapter | saved (`/kaggle/working/adapter`, t=1672.1 s) and pulled | `adapter_model.safetensors`, 73.9 MB |
| Unsloth | never touched — 0 `unsloth` strings in the decoded log | decoded-log grep |
| errors | none — no traceback, no SIGTERM; clean `stop_reason: stop_after_seconds` | log |

### The requested measurements

| item | value |
|---|---|
| backend | **plain transformers+peft** (`backend: "transformers"` in both metrics records), fp16 LoRA r=16, DDP over nccl |
| DDP ran with 2 ranks? | **yes, with training steps completed** — 14 steps on each rank; per-rank losses differ (different manifest halves) while eval CE is identical on both ranks (all-reduce worked) |
| tok/s per rank | **358.0** (rank 0) / **327.3** (rank 1) at step 10 (logged over 1,084 s / 10 steps); end-of-run snapshots (×world_size, incl. eval time): 698.9 / 640.0 |
| tok/s total | **685.3** (sum of the two ranks at step 10; ≈669 averaged over the full run including evals) |
| sup_tok/s per rank | **80.8** (rank 0) / **104.7** (rank 1) at step 10; end-of-run snapshots: 160.7 / 212.5 |
| sup_tok/s total | **185.5** at step 10 |
| peak mem | **11.5 GiB** on both ranks (step 1: 11.27 / 11.35) — ~79% of the T4's 14.56 GiB |
| loss trajectory | rank 0: 2.0439 → 1.1101 → 1.2569 (steps 1 / 5 / 10); rank 1: 2.4994 → 1.5628 → 1.2981. The step-10 > step-5 uptick on rank 0 is 8-microbatch window variance, not divergence — eval CE falls monotonically |
| eval CE at step 10 | **0.8995** (global; identical value on both ranks) |
| eval CE at step 20 | **not reached** — the 1,500 s stop guard fired entering step 15; final eval at step 14: **0.8633** |
| steps completed | **14 / 20** (`stop_reason: stop_after_seconds`, elapsed 1,581 s) |

Note on the missed step 20: at 108.4 s/step (1,084 s / 10 steps) a 20-step run needs ~36 min
of training; the 1,500 s guard stops it at step 14 and the 1,800 s platform cap alone would
have killed it near step 17. `MAX_STEPS=20` + `STOP_AFTER_SECONDS=1500` cannot both hold at
this rate — the guard behaved correctly (adapter + final eval still landed), it just means
step 20 was never measured on this push.

### Budget projection (measured rate)

```
windows_per_12h = tok_per_s_total × 43200 / avg_window_tokens
hours(arm)      = windows × avg_window_tokens / tok_per_s_total / 3600
```

Measured inputs: `tok_per_s_total = 685.3` (step-10 window; the elapsed time it divides by
already includes the step-10 eval, so it is conservative for pure training throughput),
`avg_window_tokens = 4,717` (mean of 4,915 / 4,519 across the two ranks).

| windows | hours at 685 tok/s | fits a 12 h session? |
|---:|---:|:--:|
| 2,000 | **3.8** | yes |
| 4,000 | **7.6** | yes |
| 8,000 | **15.3** | no |
| 16,000 | **30.6** | no |

`windows_per_12h` at the measured rate: **~6,276 windows** (~523 windows/hour).

### Comparison vs the smoke baseline (317 tok/s single GPU 4-bit)

| metric | smoke (v0) | v3 (this run) | ratio |
|---|---:|---:|---:|
| tok/s | 317 (1×T4, NF4) | **685.3** (2×T4, fp16) | **2.16×** aggregate |
| per-GPU tok/s | 317 | 342.7 | 1.08× |
| windows / 12 h | ~2,700 (at 5,046-token windows) | **~6,276** (at 4,717-token windows) | 2.32× |
| peak mem | 10.47 GiB | 11.5 GiB | +1.03 GiB |

The aggregate gain is almost entirely the second GPU; per GPU the fp16 path (no NF4 dequant)
is ~8% faster than the NF4 smoke path, well within the differences between the runs
(different manifests, `MAX_SUP_ROWS` 2,048 vs 3,000, shorter windows).

### Residual gaps

1. **Step-20 eval missing** (stop guard; see above) — the scheduling pair needs revisiting
   before a long-form run, not the code.
2. The budget table uses the logged training-window rate; the two evals added ~1 min per
   10 steps on this eval slice, which the logged rates already absorb.
3. Only the 46-trace / 2M-char demo manifest ran; the real arms are the 20M-char manifests
   (README), so this rate is the input to their window counts, not a measurement of them.
4. DDP gradient-sync correctness is evidenced by all-reduced identical eval CE; no
   cross-rank loss-equality check was run (rank losses legitimately differ).

### Artifacts (v3)

| path | contents |
|---|---|
| `out/validate3/openswe-curve.log` | raw Kaggle log stream, 37 entries |
| `out/validate3/openswe-curve-decoded.txt` | decoded log (`t=… \| stream \| line`) |
| `out/validate3/metrics.json` | rank-0 record: `backend: "transformers"`, `ddp: true`, `steps_done: 14`, step log + 2 evals |
| `out/validate3/metrics_rank1.json` | rank-1 record (same structure, rank-1 rates) |
| `out/validate3/adapter/` | the trained LoRA adapter (`adapter_model.safetensors`, 73.9 MB) |
