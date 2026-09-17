#!/usr/bin/env python3
"""Kaggle T4 smoke run: QLoRA SFT of a 1.5B coder on Open-SWE-Traces samples.

Measures trainable tokens/sec and peak memory on a T4 so the 5k/10k/20k/40k
scaling curve can be budgeted against the weekly GPU quota. Not a quality run.

Memory design (the first push OOM'd allocating the lm_head transient):
- Supervised-position-only loss: run the PEFT base transformer, gather last
  hidden states at shifted labels != -100, apply lm_head to those rows only,
  cross-entropy in fp32. The [1, T, 151936] logits transient (3.5 GiB fp32 at
  T = 6,144) becomes at most [MAX_SUP_ROWS, 151936]: measured supervision
  density is ~0.46 per window, so windows denser than the cap (3,000 rows,
  ~1.8 GiB per fp32 buffer) are stride-subsampled to bound the CE transient.
- Window sampling instead of prefix truncation: a random message-aligned window
  <= MAX_LEN that starts at a user/assistant message, contains >= 1 assistant
  turn, and always keeps the leading system message(s).
- expandable_segments allocator, non-reentrant gradient checkpointing, and the
  torch.amp GradScaler form.

Local CPU dry run / loss parity check (fp32, no bitsandbytes):
  cd <repo root>
  MODEL=Qwen/Qwen2.5-0.5B-Instruct MAX_LEN=512 N_TRACES=4 MAX_STEPS=2 \\
    uv run --with transformers --with peft --with accelerate \\
    python experiments/kaggle_smoke/train_smoke.py
  PARITY_CHECK=1 additionally asserts the gathered loss matches the standard
  model(labels=...) loss within 1e-3 (small MAX_LEN only: it runs the full head).
"""
import os

os.environ["PYTORCH_ALLOC_CONF"] = "expandable_segments:True"
os.environ.setdefault("PYTORCH_CUDA_ALLOC_CONF", "expandable_segments:True")

import glob
import json
import random
import subprocess
import sys
import time
from pathlib import Path

try:
    import accelerate  # noqa: F401
    import peft  # noqa: F401
    import transformers  # noqa: F401
except ImportError:
    subprocess.check_call([sys.executable, "-m", "pip", "install", "-q",
                           "transformers>=4.45", "peft>=0.13", "bitsandbytes>=0.44", "accelerate"])

import torch
import torch.nn.functional as F
from peft import LoraConfig, get_peft_model, prepare_model_for_kbit_training
from transformers import AutoModelForCausalLM, AutoTokenizer, BitsAndBytesConfig

# ---------------- CONFIG ----------------
MODEL = os.environ.get("MODEL", "Qwen/Qwen2.5-Coder-1.5B-Instruct")
MAX_LEN = int(os.environ.get("MAX_LEN", "6144"))
N_TRACES = int(os.environ.get("N_TRACES", "400"))
MAX_STEPS = int(os.environ.get("MAX_STEPS", "60"))
GRAD_ACCUM = int(os.environ.get("GRAD_ACCUM", "8"))
MAX_SUP_ROWS = int(os.environ.get("MAX_SUP_ROWS", "3000"))
LR = float(os.environ.get("LR", "2e-4"))
SEED = int(os.environ.get("SEED", "0"))
PARITY_CHECK = os.environ.get("PARITY_CHECK") == "1"
ENCODE_ONLY = os.environ.get("ENCODE_ONLY") == "1"
OUT = os.environ.get("OUT_DIR") or ("/kaggle/working" if os.path.isdir("/kaggle/working")
                                    else str(Path(__file__).resolve().parent / "out"))
# ----------------------------------------

t0 = time.time()
cuda = torch.cuda.is_available()
if cuda:
    cap = torch.cuda.get_device_capability()
    assert cap >= (7, 0), f"need sm70+, got {cap}"
    props = torch.cuda.get_device_properties(0)
    print(f"GPU: {torch.cuda.get_device_name(0)} sm_{cap[0]}{cap[1]} "
          f"{props.total_memory / 2**30:.1f} GiB", flush=True)
    try:
        import bitsandbytes  # noqa: F401
    except ImportError:
        subprocess.check_call([sys.executable, "-m", "pip", "install", "-q", "bitsandbytes>=0.44"])
else:
    print("no CUDA: fp32 CPU dry run (4-bit quant skipped)", flush=True)
dev = "cuda" if cuda else "cpu"

print(json.dumps({"model": MODEL, "max_len": MAX_LEN, "n_traces": N_TRACES, "max_steps": MAX_STEPS,
                  "grad_accum": GRAD_ACCUM, "max_sup_rows": MAX_SUP_ROWS, "lr": LR, "seed": SEED,
                  "out": OUT, "parity_check": PARITY_CHECK, "encode_only": ENCODE_ONLY}), flush=True)

if os.path.isdir("/kaggle/input"):
    data_path = min(glob.glob("/kaggle/input/**/sample_*.jsonl", recursive=True))
else:
    data_path = str(Path(__file__).resolve().parent / "data" / "sample_1000.jsonl")
with open(data_path) as f:
    rows = [json.loads(line) for line in f][:N_TRACES]
print(f"loaded {len(rows)} traces from {data_path}", flush=True)

tok = AutoTokenizer.from_pretrained(MODEL)
tok.pad_token = tok.pad_token or tok.eos_token
rng = random.Random(SEED)


def render(msgs, tools):
    return tok.apply_chat_template(msgs, tools=tools, tokenize=False, add_generation_prompt=False)


def msg_spans(msgs, tools):
    """Per-message token ids from delta-rendering the chat template."""
    spans, prev_text = [], ""
    for i in range(len(msgs)):
        text = render(msgs[: i + 1], tools)
        delta = text[len(prev_text):]
        prev_text = text
        spans.append(tok(delta, add_special_tokens=False)["input_ids"])
    return spans


def window_bounds(spans, msgs):
    """Message indices (start, end) of a <= MAX_LEN window. Full trace when it
    fits; otherwise a random start at a user/assistant message with >= 1
    assistant turn inside the window. Leading system messages are always kept."""
    n = len(msgs)
    if sum(len(s) for s in spans) <= MAX_LEN:
        return 0, n
    lead = 0
    while lead < n and msgs[lead]["role"] == "system":
        lead += 1
    starts = [s for s in range(lead, n) if msgs[s]["role"] in ("assistant", "user")]
    prefix = sum(len(spans[i]) for i in range(lead))

    def extent(s):
        used, e = prefix, s
        while e < n and used + len(spans[e]) <= MAX_LEN:
            used += len(spans[e])
            e += 1
        return e

    def ok(s, e):
        return e > s and any(msgs[j]["role"] == "assistant" for j in range(s, e))

    if not starts:
        return 0, n
    for _ in range(50):
        s = rng.choice(starts)
        e = extent(s)
        if ok(s, e):
            return s, e
    for s in reversed(starts):
        e = extent(s)
        if ok(s, e):
            return s, e
    return 0, n


def encode(row):
    """Token ids + assistant-only labels for one window of a trace."""
    msgs, tools = row["messages"], row["tools"]
    spans = msg_spans(msgs, tools)
    total = sum(len(s) for s in spans)
    s, e = window_bounds(spans, msgs)
    lead = 0
    while lead < len(msgs) and msgs[lead]["role"] == "system":
        lead += 1
    keep = list(range(min(lead, s))) + list(range(s, e))
    ids, labels = [], []
    for i in keep:
        d = spans[i]
        ids += d
        labels += d if msgs[i]["role"] == "assistant" else [-100] * len(d)
    ids, labels = ids[:MAX_LEN], labels[:MAX_LEN]
    if not any(l != -100 for l in labels):
        return None
    return {"input_ids": ids, "labels": labels,
            "truncated": len(ids) < total,
            "start_frac": sum(len(spans[j]) for j in range(s)) / max(total, 1)}


enc_t = time.time()
examples = [e for e in (encode(r) for r in rows) if e]
n_trunc = sum(e["truncated"] for e in examples)
avg_start = sum(e["start_frac"] for e in examples) / max(len(examples), 1)
tot_tok = sum(len(e["input_ids"]) for e in examples)
sup_tok = sum(sum(l != -100 for l in e["labels"]) for e in examples)
print(f"encoded {len(examples)} ex in {time.time() - enc_t:.0f}s | truncated {n_trunc} | "
      f"avg len {tot_tok / len(examples):.0f} | avg window start frac {avg_start:.2f} | "
      f"supervised frac {sup_tok / tot_tok:.2f}", flush=True)

if ENCODE_ONLY:
    q = lambda a, p: a[min(int(p * len(a)), len(a) - 1)]
    lens = sorted(len(e["input_ids"]) for e in examples)
    sups = sorted(sum(l != -100 for l in e["labels"]) for e in examples)
    fracs = sorted(e["start_frac"] for e in examples)
    print(f"encode-only: len min {lens[0]} p50 {q(lens, .5)} p90 {q(lens, .9)} max {lens[-1]} | "
          f"sup min {sups[0]} p50 {q(sups, .5)} p90 {q(sups, .9)} max {sups[-1]} | "
          f"start frac min {fracs[0]:.2f} p50 {q(fracs, .5):.2f} max {fracs[-1]:.2f} | "
          f"dropped {len(rows) - len(examples)}", flush=True)
    sys.exit(0)

if cuda:
    bnb = BitsAndBytesConfig(load_in_4bit=True, bnb_4bit_quant_type="nf4",
                             bnb_4bit_compute_dtype=torch.float16, bnb_4bit_use_double_quant=True)
    base = AutoModelForCausalLM.from_pretrained(MODEL, quantization_config=bnb,
                                                device_map={"": 0}, torch_dtype=torch.float16)
    base = prepare_model_for_kbit_training(base, use_gradient_checkpointing=False)
else:
    base = AutoModelForCausalLM.from_pretrained(MODEL, torch_dtype=torch.float32)
base.config.use_cache = False
base.gradient_checkpointing_enable(gradient_checkpointing_kwargs={"use_reentrant": False})
base.enable_input_require_grads()
model = get_peft_model(base, LoraConfig(
    r=16, lora_alpha=32, lora_dropout=0.05, task_type="CAUSAL_LM",
    target_modules=["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]))
model.config.use_cache = False
model.print_trainable_parameters()


def base_transformer_and_head(peft_model):
    """PeftModel -> LoraModel -> Qwen2ForCausalLM: (Qwen2Model, lm_head)."""
    base = peft_model.get_base_model() if hasattr(peft_model, "get_base_model") else peft_model
    if not hasattr(base, "lm_head"):
        base = base.model
    return base.model, base.lm_head


def supervised_loss(transformer, lm_head, input_ids, labels):
    """Causal-LM cross-entropy over shifted supervised positions only. Mirrors
    the standard loss (shift by one, ignore -100, mean over supervised tokens,
    fp32 CE) without ever materializing [B, T, vocab] logits. Windows denser
    than MAX_SUP_ROWS rows are stride-subsampled so the fp32 CE transient stays
    bounded (~3 buffers x 4 B x vocab x MAX_SUP_ROWS, the dominant term)."""
    hidden = transformer(input_ids=input_ids, use_cache=False).last_hidden_state
    shift_labels = labels[:, 1:]
    mask = shift_labels != -100
    sel_hidden = hidden[:, :-1, :][mask]
    sel_labels = shift_labels[mask]
    if sel_labels.numel() > MAX_SUP_ROWS:
        stride = -(-sel_labels.numel() // MAX_SUP_ROWS)
        sel_hidden = sel_hidden[::stride]
        sel_labels = sel_labels[::stride]
    logits = lm_head(sel_hidden)
    return F.cross_entropy(logits.float(), sel_labels, reduction="mean")


transformer, lm_head = base_transformer_and_head(model)
params = [p for p in model.parameters() if p.requires_grad]
opt = torch.optim.AdamW(params, lr=LR, weight_decay=0.0)
scaler = torch.amp.GradScaler("cuda") if cuda else None


def run_parity_check(n=4):
    model.eval()
    rows_out = []
    with torch.no_grad():
        for e in examples[:n]:
            ids = torch.tensor([e["input_ids"]], device=dev)
            lab = torch.tensor([e["labels"]], device=dev)
            full = model(input_ids=ids, labels=lab).loss.float().item()
            sup = supervised_loss(transformer, lm_head, ids, lab).float().item()
            rows_out.append({"tokens": len(e["input_ids"]), "full_loss": full,
                             "supervised_loss": sup, "abs_diff": abs(full - sup)})
            print(f"parity tokens={len(e['input_ids'])} full={full:.6f} "
                  f"supervised={sup:.6f} diff={abs(full - sup):.2e}", flush=True)
    worst = max(r["abs_diff"] for r in rows_out)
    os.makedirs(OUT, exist_ok=True)
    with open(f"{OUT}/loss_parity.json", "w") as f:
        json.dump({"rows": rows_out, "max_abs_diff": worst, "tol": 1e-3, "model": MODEL,
                   "max_len": MAX_LEN}, f, indent=1)
    print(f"loss parity: max |diff| {worst:.2e} (tol 1e-3)", flush=True)
    assert worst < 1e-3, f"loss parity failed: {worst}"


if PARITY_CHECK:
    run_parity_check()
    sys.exit(0)

model.train()
log, step, seen_tok, seen_sup = [], 0, 0, 0
train_t = time.time()
i = 0
while step < MAX_STEPS:
    opt.zero_grad(set_to_none=True)
    acc_loss = 0.0
    for _ in range(GRAD_ACCUM):
        e = examples[i % len(examples)]
        i += 1
        ids = torch.tensor([e["input_ids"]], device=dev)
        lab = torch.tensor([e["labels"]], device=dev)
        if cuda:
            with torch.autocast("cuda", dtype=torch.float16):
                loss = supervised_loss(transformer, lm_head, ids, lab) / GRAD_ACCUM
        else:
            loss = supervised_loss(transformer, lm_head, ids, lab) / GRAD_ACCUM
        if scaler is not None:
            scaler.scale(loss).backward()
        else:
            loss.backward()
        acc_loss += loss.item()
        seen_tok += ids.numel()
        seen_sup += int((lab != -100).sum())
    if scaler is not None:
        scaler.unscale_(opt)
    torch.nn.utils.clip_grad_norm_(params, 1.0)
    if scaler is not None:
        scaler.step(opt)
        scaler.update()
    else:
        opt.step()
    step += 1
    el = time.time() - train_t
    rec = {"step": step, "loss": round(acc_loss, 4), "tok_per_s": round(seen_tok / el, 1),
           "sup_tok_per_s": round(seen_sup / el, 1), "elapsed_s": round(el),
           "mem_gb": round(torch.cuda.max_memory_allocated() / 1e9, 2) if cuda else 0.0}
    log.append(rec)
    if step % 5 == 0 or step == 1:
        print(rec, flush=True)

el = time.time() - train_t
sec_per_trace = el / max(i, 1)
budget = {str(n): {"hours": round(n * sec_per_trace / 3600, 1), "fits_12h": n * sec_per_trace / 3600 <= 12}
          for n in (5000, 10000, 20000, 40000)}
metrics = {
    "model": MODEL, "loss_path": "supervised_positions_only", "max_len": MAX_LEN,
    "max_sup_rows": MAX_SUP_ROWS,
    "n_examples": len(examples), "n_truncated": n_trunc, "avg_len": tot_tok / len(examples),
    "avg_window_start_frac": avg_start, "supervised_frac": sup_tok / tot_tok,
    "steps": step, "grad_accum": GRAD_ACCUM, "traces_seen": i,
    "train_seconds": el, "tok_per_s": seen_tok / el, "sup_tok_per_s": seen_sup / el,
    "sec_per_trace": sec_per_trace,
    "peak_mem_gb": torch.cuda.max_memory_allocated() / 1e9 if cuda else 0.0,
    "first_loss": log[0]["loss"], "last_loss": log[-1]["loss"], "budget": budget,
    "total_seconds": time.time() - t0, "log": log,
}
os.makedirs(OUT, exist_ok=True)
with open(f"{OUT}/smoke_metrics.json", "w") as f:
    json.dump(metrics, f, indent=1)
model.save_pretrained(f"{OUT}/adapter")
print("budget (hours by corpus size, 12h session cap):", flush=True)
for n, b in budget.items():
    print(f"  {n:>6} traces: {b['hours']:8.1f} h  {'fits' if b['fits_12h'] else 'DOES NOT fit'}", flush=True)
print(json.dumps({k: v for k, v in metrics.items() if k != "log"}, indent=1), flush=True)
