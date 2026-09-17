# RESULT — evandekim/openswe-smoke-qlora

**Status: ERROR** (`KernelWorkerStatus.ERROR` first seen 2026-09-17T20:03:53Z; last RUNNING poll 20:02:52Z — it died ~2 min after starting).
The kernel was **not re-pushed** (GPU quota), and **no `smoke_metrics.json` was produced** — the script OOM'd on the first training step, before any step completed.

- Log pulled to `experiments/kaggle_smoke/out/openswe-smoke-qlora.log` (JSON stream log: 30 entries).
- Human-readable render: `experiments/kaggle_smoke/out/openswe-smoke-qlora.rendered.log`.
- Poll history: `experiments/kaggle_smoke/poll_status.log`.

## What the run managed to report (pre-training only)

| item | value |
|---|---|
| GPU | Tesla T4, sm_75, 14.56 GiB usable |
| traces loaded | 400 |
| examples encoded | 398 (2 dropped — no assistant tokens survived) |
| truncated | **398 / 398** (every example hit MAX_LEN) |
| avg len | 5933 tokens (MAX_LEN = 6144) |
| supervised frac | 0.17 |
| trainable params | 18,464,768 / 1,562,179,072 (1.18%) |

No tok/s, no sec/trace, no loss values, no budget table — training never completed a single step (`MAX_STEPS=120`, `GRAD_ACCUM=8`; the trainer died inside the first step's backward).

## Last 60 lines of the kernel log

The whole log is only 30 JSON entries and renders to 31 lines, so the complete log follows (88 KB of weight-loading progress-bar frames collapsed to their final frame).

```
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 43.1/43.1 MB 46.2 MB/s eta 0:00:00
GPU: Tesla T4 (7, 5)
loaded 400 traces from /kaggle/input/datasets/evandekim/openswe-smoke-1k/sample_1000.jsonl
Warning: You are sending unauthenticated requests to the HF Hub. Please set a HF_TOKEN to enable higher rate limits and faster downloads.
encoded 398 ex in 9s | truncated 398 | avg len 5933 | supervised frac 0.17
`torch_dtype` is deprecated! Use `dtype` instead!
[... 88 KB of weight-loading progress output collapsed (690 frames); final frame:] Loading weights: 100%|██████████| 338/338 [00:01<00:00, 169.33it/s, Materializing param=model.norm.weight]
trainable params: 18,464,768 || all params: 1,562,179,072 || trainable%: 1.1820
/kaggle/src/script.py:83: FutureWarning: `torch.cuda.amp.GradScaler(args...)` is deprecated. Please use `torch.amp.GradScaler('cuda', args...)` instead.
  scaler = torch.cuda.amp.GradScaler()
`use_cache=True` is incompatible with gradient checkpointing. Setting `use_cache=False`.
/usr/local/lib/python3.12/dist-packages/torch/_dynamo/eval_frame.py:1181: UserWarning: torch.utils.checkpoint: the use_reentrant parameter should be passed explicitly. Starting in PyTorch 2.9, calling checkpoint without use_reentrant will raise an exception. use_reentrant=False is recommended, but if you need to preserve the current default behavior, you can pass use_reentrant=True. Refer to docs for more details on the differences between the two variants.
  return fn(*args, **kwargs)
Traceback (most recent call last):
  File "/kaggle/src/script.py", line 98, in <module>
    scaler.scale(loss).backward()
  File "/usr/local/lib/python3.12/dist-packages/torch/_tensor.py", line 630, in backward
    torch.autograd.backward(
  File "/usr/local/lib/python3.12/dist-packages/torch/autograd/__init__.py", line 364, in backward
    _engine_run_backward(
  File "/usr/local/lib/python3.12/dist-packages/torch/autograd/graph.py", line 865, in _engine_run_backward
    return Variable._execution_engine.run_backward(  # Calls into the C++ engine to run the backward pass
           ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
torch.OutOfMemoryError: CUDA out of memory. Tried to allocate 3.45 GiB. GPU 0 has a total capacity of 14.56 GiB of which 3.42 GiB is free. Including non-PyTorch memory, this process has 11.14 GiB memory in use. Of the allocated memory 10.02 GiB is allocated by PyTorch, and 1006.76 MiB is reserved by PyTorch but unallocated. If reserved but unallocated memory is large try setting PYTORCH_ALLOC_CONF=expandable_segments:True to avoid fragmentation.  See documentation for Memory Management  (https://pytorch.org/docs/stable/notes/cuda.html#environment-variables)
/usr/local/lib/python3.12/dist-packages/mistune.py:435: SyntaxWarning: invalid escape sequence '\|'
  cells[i][c] = re.sub('\\\\\|', '|', cell)
/usr/local/lib/python3.12/dist-packages/nbconvert/filters/filter_links.py:36: SyntaxWarning: invalid escape sequence '\_'
  text = re.sub(r'_', '\_', text) # Escape underscores in display text
[NbConvertApp] Converting notebook __script__.ipynb to html
[NbConvertApp] Writing 305610 bytes to __results__.html
```

(The last four lines are Kaggle's post-mortem notebook→HTML conversion, not the program.)

## Root-cause diagnosis

**The LM head at 6144 tokens is an ~8–9 GiB transient — the model itself is tiny.** This is a token-count × vocab-size memory problem, not a QLoRA/weight problem:

- `MAX_LEN=6144` × Qwen2.5 vocab `151,936` → logits tensor `[1, 6144, 151936]` = 933.5M elements.
- HF's causal-LM loss upcasts logits to float32 for cross-entropy, and backward materializes a same-shaped grad buffer. The failing request, **3.45 GiB, matches one float32 logits buffer almost exactly: 6144 × 151,936 × 4 B = 3.48 GiB** (within ~1%).
- Full head transient per micro-batch ≈ 1.74 GiB (fp16 logits) + 3.48 GiB (fp32, saved for backward) + 3.48 GiB (backward grad) ≈ **8.7 GiB**, on a 14.56 GiB T4 also holding NF4 weights (~0.85 GiB), LoRA/optimizer state, and checkpointed activations. It missed free memory by 30 MB.

Contributing factors, in order of impact:

1. **Every sequence is maximal.** `truncated 398 / 398`, `avg len 5933` against `MAX_LEN 6144` — there is no small-sequence relief; each of the 8 grad-accum micro-batches pays the full head cost.
2. **Assistant-only masking (supervised frac 0.17) saves zero memory here.** Masking via `labels=-100` only happens *inside* the loss — all 6144×152k logits are still computed, upcast, and backpropped. ~6× of that compute and memory is spent on tokens that never contribute loss.
3. **Gradient checkpointing does not help.** It is enabled (the `use_cache` warning confirms it), but it wraps transformer blocks only — the final norm + `lm_head` run un-checkpointed, so the dominant term is untouched.
4. `1006.76 MiB reserved but unallocated` — fragmentation, and `PYTORCH_ALLOC_CONF=expandable_segments:True` was not set. At a 30 MB miss this is the kind of slack that decides pass/fail, though the underlying overshoot is ~2× regardless.
5. Time of death (~126 s, ~60 s into training) with no `step 1` loss line printed means it OOM'd mid-first-step (during one of the 8 accumulation backwards), consistent with the allocator fragmenting as the logits buffer is freed and re-allocated per micro-batch.

## Fix menu for the next push (not done here — quota)

- **Cut `MAX_LEN` to 2048–3072** for the T4 smoke: head cost scales linearly, so 2048 → fp32 logits ≈ 1.24 GB (vs 3.48 GB), 3072 → ≈ 1.87 GB. This is the single lever that makes it fit.
- Or **compute the loss on supervised positions only** (e.g., gather hidden states at assistant-token positions before `lm_head`) — same MAX_LEN, ~6× less head memory *and* compute given supervised frac 0.17.
- Set `PYTORCH_ALLOC_CONF=expandable_segments:True`; move `GradScaler` to `torch.amp.GradScaler('cuda')` and pass `use_reentrant=False` to silence the two upcoming-hard-error deprecations.
- Optional: truncate sequences to end inside the supervised region (the message-boundary truncation currently keeps mostly masked tool-observation tokens), and note `kernel-metadata.json` currently says `"enable_gpu": "false"` (the T4 was allocated anyway via `machine_shape`; still worth fixing before a re-push).
