#!/usr/bin/env python3
"""Kaggle curve run: fp16 LoRA SFT of a 1.5B coder on budget-matched manifests.

Arms (env `MANIFEST`): `random`, `top_within_task`, `bottom_within_task`,
`random_masked` — built by `experiments/curve/build_manifests.py`, shipped as the
Kaggle dataset `evandekim/openswe-curve-manifests`. Every arm has the same
supervised-char budget and language mix, so held-out cross-entropy vs steps is
comparable across arms.

Design:
- fp16 LoRA (r=16, all attn/MLP projections) on the unquantized
  Qwen2.5-Coder-1.5B-Instruct: no bitsandbytes, no NF4 dequant. The
  supervised-position-only loss and the message-aligned window sampling are the ones
  measured in `kaggle_smoke` (317 tok/s, 10.47 GiB peak at MAX_LEN=6144 with NF4); the
  fp16 base costs ~2.3 GiB more than NF4, so MAX_SUP_ROWS defaults to 2,048 to keep the
  fp32 CE transient under the T4's 14.56 GiB.
- Plain transformers+peft by default (fp16, SDPA attention, gradient checkpointing with
  use_reentrant=False). Unsloth is opt-in only (`USE_UNSLOTH=1`) and OFF by default:
  the 2026-09-17 T4 validation showed the image's Unsloth stack is import-broken and a
  failed import leaves a meta-path hook that poisons the plain path in the same
  interpreter, so when USE_UNSLOTH=0 neither `unsloth` nor `unsloth_zoo` is ever
  imported — a sys.modules guard at model build raises if either got in anyway. The
  path that ran is recorded in metrics.json.
- Two GPUs when `torch.cuda.device_count() == 2`: two spawned worker processes, one
  contiguous half of the manifest each, DDP (nccl, gloo fallback) gradient sync. Single
  GPU (and CPU) runs use the same training function with world_size=1.
- `metrics.json` is rewritten every METRICS_EVERY (5) steps with tok/s, loss, elapsed,
  mem, steps done, `windows_seen` (= steps × grad_accum × world_size) and eval CE, so a
  Kaggle kill still leaves the curve on disk.
- Eval schedule: every `EVAL_EVERY` steps, plus checkpoint evals at the `EVAL_WINDOWS`
  window counts (1000/2000/3000), all on `eval_heldout_small.jsonl` (60 traces); a final
  eval on the full `eval_heldout.jsonl` (300) runs before the adapter is saved. Eval
  windows are seeded per trace id, and the small file is a subset of the full one, so the
  two evals are nested for shared traces.
- `STOP_AFTER_SECONDS` (default 25,200 = 7 h) saves the adapter and exits cleanly
  before Kaggle's 12 h session cap; pushes use `-t` so the platform cap never hits first.

Env: MODEL, MANIFEST, MAX_LEN, MAX_STEPS, GRAD_ACCUM, MAX_SUP_ROWS, LR, SEED, EVAL_EVERY,
EVAL_N, EVAL_WINDOWS, METRICS_EVERY, STOP_AFTER_SECONDS, OUT_DIR, USE_UNSLOTH (0 default;
1 opts into Unsloth), DDP_BACKEND (nccl/gloo), NO_DDP, MANIFEST_PATH, EVAL_PATH,
FINAL_EVAL_PATH.

The real-run config is baked in `CONFIG` below (env still wins): MANIFEST=random (each
arm's copy is sed'd by `launch_arms.sh`), MAX_STEPS=100000 (the stop guard always wins),
EVAL_EVERY=100, MAX_LEN=6144, STOP_AFTER_SECONDS=25200, EVAL_N=0 (use the eval file as
built: 60 small / 300 full). The `launch_arms.sh` push path is one kernel per arm.

Local CPU dry run (exercises the transformers fallback by construction):
  cd <repo root>
  MODEL=Qwen/Qwen2.5-0.5B-Instruct MAX_LEN=512 MAX_STEPS=3 EVAL_EVERY=2 \
    uv run --with transformers --with peft --with accelerate \
    python experiments/curve/train_curve.py
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
from contextlib import nullcontext
from pathlib import Path

if os.path.isdir("/kaggle/working"):
    subprocess.check_call(
        [sys.executable, "-m", "pip", "install", "-q", "--upgrade", "peft>=0.13", "accelerate"]
    )
    subprocess.run([sys.executable, "-m", "pip", "uninstall", "-y", "torchao"], check=False)

try:
    import accelerate  # noqa: F401
    import peft  # noqa: F401
    import transformers  # noqa: F401
except ImportError:
    subprocess.check_call(
        [
            sys.executable,
            "-m",
            "pip",
            "install",
            "-q",
            "transformers>=4.45",
            "peft>=0.13",
            "accelerate",
        ]
    )

import torch
import torch.distributed as dist
import torch.nn.functional as F
from peft import LoraConfig, get_peft_model
from transformers import AutoModelForCausalLM, AutoTokenizer

# Baked defaults for the real 4-arm run: Kaggle script kernels receive no env vars, and
# per-arm copies of this file are sed'd (MANIFEST) by launch_arms.sh. Every env var still
# takes precedence, so the local CPU dry run is unaffected.
CONFIG = {
    "MANIFEST": "random",
    "MAX_STEPS": "100000",
    "EVAL_EVERY": "100",
    "MAX_LEN": "6144",
    "STOP_AFTER_SECONDS": "25200",
    "EVAL_N": "0",
}


def cfg(name: str, fallback: str) -> str:
    """Env var wins, then the baked CONFIG, then the hardcoded fallback."""
    return os.environ.get(name) or CONFIG.get(name) or fallback


MODEL = cfg("MODEL", "Qwen/Qwen2.5-Coder-1.5B-Instruct")
MANIFEST = cfg("MANIFEST", "random")
MAX_LEN = int(cfg("MAX_LEN", "6144"))
MAX_STEPS = int(cfg("MAX_STEPS", "1000"))
GRAD_ACCUM = int(cfg("GRAD_ACCUM", "8"))
MAX_SUP_ROWS = int(cfg("MAX_SUP_ROWS", "2048"))
LR = float(cfg("LR", "2e-4"))
SEED = int(cfg("SEED", "0"))
EVAL_EVERY = int(cfg("EVAL_EVERY", "100"))
EVAL_N = int(cfg("EVAL_N", "24"))
EVAL_WINDOWS = [
    int(x.strip()) for x in cfg("EVAL_WINDOWS", "1000,2000,3000").split(",") if x.strip()
]
EVAL_FILE = "eval_heldout_small.jsonl"
FINAL_EVAL_FILE = "eval_heldout.jsonl"
METRICS_EVERY = int(cfg("METRICS_EVERY", "5"))
STOP_AFTER_SECONDS = float(cfg("STOP_AFTER_SECONDS", "25200"))
USE_UNSLOTH = os.environ.get("USE_UNSLOTH", "0")
DDP_BACKEND = os.environ.get("DDP_BACKEND", "nccl")
NO_DDP = os.environ.get("NO_DDP") == "1"
WORLD_SIZE = int(os.environ.get("WORLD_SIZE", "0"))
OUT = os.environ.get("OUT_DIR") or (
    "/kaggle/working"
    if os.path.isdir("/kaggle/working")
    else str(Path(__file__).resolve().parent / "out")
)
LORA_TARGETS = ["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]


def find_manifest(name: str) -> Path:
    if os.environ.get("MANIFEST_PATH"):
        return Path(os.environ["MANIFEST_PATH"])
    if os.path.isdir("/kaggle/input"):
        hits = sorted(glob.glob(f"/kaggle/input/**/{name}.jsonl", recursive=True))
        if hits:
            return Path(hits[0])
    local = Path(__file__).resolve().parent / "data" / f"{name}.jsonl"
    if local.exists():
        return local
    raise SystemExit(f"manifest {name!r} not found under /kaggle/input or {local}")


def find_eval(name: str, env: str, fallbacks: tuple[str, ...] = ()) -> Path:
    """Eval jsonl by name: env override, then /kaggle/input, then the local data dir."""
    if os.environ.get(env):
        return Path(os.environ[env])
    for cand in (name, *fallbacks):
        if os.path.isdir("/kaggle/input"):
            hits = sorted(glob.glob(f"/kaggle/input/**/{cand}", recursive=True))
            if hits:
                return Path(hits[0])
        local = Path(__file__).resolve().parent / "data" / cand
        if local.exists():
            return local
    raise SystemExit(f"eval jsonl {name!r} not found under /kaggle/input or local data/")


def load_rows(path: Path) -> list[dict]:
    with open(path) as f:
        return [json.loads(line) for line in f]


def build_encoder(tok, rng: random.Random):
    """Smoke's window sampler, extended with `mask` -> labels -100 for masked steps."""

    def render(msgs, tools):
        return tok.apply_chat_template(
            msgs, tools=tools, tokenize=False, add_generation_prompt=False
        )

    def msg_spans(msgs, tools):
        spans, prev_text = [], ""
        for i in range(len(msgs)):
            text = render(msgs[: i + 1], tools)
            delta = text[len(prev_text) :]
            prev_text = text
            spans.append(tok(delta, add_special_tokens=False)["input_ids"])
        return spans

    def window_bounds(spans, msgs):
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

    def encode(row: dict):
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
            supervised = msgs[i]["role"] == "assistant" and not msgs[i].get("mask")
            labels += d if supervised else [-100] * len(d)
        ids, labels = ids[:MAX_LEN], labels[:MAX_LEN]
        if not any(l != -100 for l in labels):
            return None
        return {"input_ids": ids, "labels": labels, "truncated": len(ids) < total}

    return encode


def base_transformer_and_head(peft_model):
    """PeftModel (or Unsloth-wrapped) -> (transformer, lm_head)."""
    base = peft_model.get_base_model() if hasattr(peft_model, "get_base_model") else peft_model
    if not hasattr(base, "lm_head"):
        base = base.model
    return base.model, base.lm_head


def unsloth_available() -> bool:
    """Unsloth is opt-in only, exactly `USE_UNSLOTH=1`.

    At the default 0 this returns before any import, so neither `unsloth` nor
    `unsloth_zoo` ever enters sys.modules: v1 showed a failed Unsloth import leaves a
    meta-path hook behind that breaks the plain transformers path in the same process.
    """
    if USE_UNSLOTH != "1":
        return False
    try:
        import unsloth  # noqa: F401
    except ImportError:
        print("unsloth import failed; staying on the plain transformers path", flush=True)
        return False
    return True


def raise_if_unsloth_imported() -> None:
    """USE_UNSLOTH=0 must build the model in an interpreter clean of Unsloth imports."""
    imported = sorted(m for m in sys.modules if m.split(".")[0] in ("unsloth", "unsloth_zoo"))
    if imported:
        raise RuntimeError(
            f"unsloth already imported in this process ({', '.join(imported[:3])}…); "
            "USE_UNSLOTH=0 requires a clean interpreter — refusing to build the model"
        )


def build_model(device: str) -> tuple[torch.nn.Module, str]:
    """Plain transformers+peft LoRA (Unsloth only behind USE_UNSLOTH=1)."""
    dtype = torch.float16 if device == "cuda" else torch.float32
    if unsloth_available():
        try:
            from unsloth import FastLanguageModel

            model, _ = FastLanguageModel.from_pretrained(
                MODEL, max_seq_length=MAX_LEN, dtype=dtype, load_in_4bit=False
            )
            model = FastLanguageModel.get_peft_model(
                model,
                r=16,
                lora_alpha=32,
                lora_dropout=0.05,
                target_modules=LORA_TARGETS,
                use_gradient_checkpointing=True,
                random_state=SEED,
            )
            model.config.use_cache = False
            return model, "unsloth"
        except Exception as exc:  # noqa: BLE001 (any Unsloth failure falls back)
            print(f"unsloth path failed ({exc!r}); falling back to transformers+peft", flush=True)

    raise_if_unsloth_imported()

    base = AutoModelForCausalLM.from_pretrained(
        MODEL, torch_dtype=dtype, attn_implementation="sdpa"
    )
    base.config.use_cache = False
    base.gradient_checkpointing_enable(gradient_checkpointing_kwargs={"use_reentrant": False})
    base.enable_input_require_grads()
    model = get_peft_model(
        base,
        LoraConfig(
            r=16,
            lora_alpha=32,
            lora_dropout=0.05,
            task_type="CAUSAL_LM",
            target_modules=LORA_TARGETS,
        ),
    )
    model.config.use_cache = False
    return model, "transformers"


def supervised_ce(transformer, lm_head, input_ids, labels, max_sup_rows):
    """Hidden states at shifted supervised positions only -> (logits, labels).

    Never materializes [B, T, vocab]; windows denser than max_sup_rows are
    stride-subsampled to bound the fp32 CE transient (smoke's memory design)."""
    hidden = transformer(input_ids=input_ids, use_cache=False).last_hidden_state
    shift_labels = labels[:, 1:]
    mask = shift_labels != -100
    sel_hidden = hidden[:, :-1, :][mask]
    sel_labels = shift_labels[mask]
    if max_sup_rows and sel_labels.numel() > max_sup_rows:
        stride = -(-sel_labels.numel() // max_sup_rows)
        sel_hidden = sel_hidden[::stride]
        sel_labels = sel_labels[::stride]
    return lm_head(sel_hidden), sel_labels


class TrainModule(torch.nn.Module):
    """DDP unit: forward computes the supervised-position-only CE of the PEFT model."""

    def __init__(self, model: torch.nn.Module):
        super().__init__()
        self.model = model

    def forward(self, input_ids, labels):
        transformer, lm_head = base_transformer_and_head(self.model)
        logits, sel_labels = supervised_ce(transformer, lm_head, input_ids, labels, MAX_SUP_ROWS)
        return F.cross_entropy(logits.float(), sel_labels, reduction="mean")


def eval_ce_of(module: TrainModule, examples: list[dict], device: str) -> tuple[float, int]:
    """Sum CE and supervised-token count over `examples` (no grad, no stride cap)."""
    transformer, lm_head = base_transformer_and_head(module.model)
    total_ce, total_n = 0.0, 0
    for e in examples:
        ids = torch.tensor([e["input_ids"]], device=device)
        lab = torch.tensor([e["labels"]], device=device)
        logits, sel_labels = supervised_ce(transformer, lm_head, ids, lab, None)
        total_ce += float(F.cross_entropy(logits.float(), sel_labels, reduction="sum"))
        total_n += int(sel_labels.numel())
    return total_ce, total_n


def write_json(path: Path, payload: dict) -> None:
    os.makedirs(path.parent, exist_ok=True)
    tmp = path.with_suffix(path.suffix + ".tmp")
    with open(tmp, "w") as f:
        json.dump(payload, f, indent=1)
    os.replace(tmp, path)


def fmt_ce(ce: float | None) -> str:
    return "n/a" if ce is None else f"{ce:.4f}"


def train(rank: int, world_size: int) -> None:
    torch.manual_seed(SEED + rank)
    t0 = time.time()
    device = "cuda" if torch.cuda.is_available() else "cpu"
    use_ddp = world_size > 1
    if device == "cuda":
        torch.cuda.set_device(rank)
    if use_ddp:
        backend = DDP_BACKEND if device == "cuda" else "gloo"
        try:
            dist.init_process_group(backend, init_method="env://", rank=rank, world_size=world_size)
        except (RuntimeError, ValueError) as exc:
            print(f"{rank}: {backend} init failed ({exc!r}); retrying with gloo", flush=True)
            dist.init_process_group("gloo", init_method="env://", rank=rank, world_size=world_size)

    manifest_path = find_manifest(MANIFEST)
    eval_path = find_eval(EVAL_FILE, "EVAL_PATH", (FINAL_EVAL_FILE,))
    final_eval_path = find_eval(FINAL_EVAL_FILE, "FINAL_EVAL_PATH")
    rows = load_rows(manifest_path)
    if use_ddp:
        n = len(rows)
        rows = rows[rank * n // world_size : (rank + 1) * n // world_size]

    tok = AutoTokenizer.from_pretrained(MODEL)
    tok.pad_token = tok.pad_token or tok.eos_token
    encode = build_encoder(tok, random.Random(SEED))
    enc_t = time.time()
    examples = [e for e in (encode(r) for r in rows) if e]
    tot_tok = sum(len(e["input_ids"]) for e in examples)
    sup_tok = sum(sum(l != -100 for l in e["labels"]) for e in examples)
    print(
        f"[rank {rank}] encoded {len(examples)}/{len(rows)} traces from {manifest_path} in "
        f"{time.time() - enc_t:.0f}s | avg len {tot_tok / max(len(examples), 1):.0f} | "
        f"supervised frac {sup_tok / max(tot_tok, 1):.2f}",
        flush=True,
    )

    def encode_eval(row: dict):
        """Per-trace seeded eval windows: the small eval file is a subset of the full
        one, so a shared trace keeps the same window in both the in-run and final CE."""
        enc = build_encoder(tok, random.Random(f"{SEED + 2}:{row.get('trajectory_id', '')}"))
        return enc(row)

    eval_rows = load_rows(eval_path)
    if EVAL_N and len(eval_rows) > EVAL_N:
        eval_rows = random.Random(SEED + 1).sample(eval_rows, EVAL_N)
    eval_examples = [e for e in (encode_eval(r) for r in eval_rows) if e]
    eval_total = len(eval_examples)
    final_rows = load_rows(final_eval_path)
    final_examples = [e for e in (encode_eval(r) for r in final_rows) if e]
    final_total = len(final_examples)
    if use_ddp:
        eval_examples = eval_examples[rank::world_size]
        final_examples = final_examples[rank::world_size]
    print(
        f"[rank {rank}] evals: periodic {eval_total} traces {eval_path} (local "
        f"{len(eval_examples)}), final {final_total} traces {final_eval_path} (local "
        f"{len(final_examples)})",
        flush=True,
    )

    model, backend_name = build_model(device)
    model.to(device)
    core = TrainModule(model).to(device)
    module = core
    if use_ddp:
        ddp_kwargs = {"device_ids": [rank], "output_device": rank} if device == "cuda" else {}
        module = torch.nn.parallel.DistributedDataParallel(
            core, find_unused_parameters=False, **ddp_kwargs
        )
    module.train()
    if rank == 0 and hasattr(model, "print_trainable_parameters"):
        model.print_trainable_parameters()
    if not examples:
        raise RuntimeError(f"no encodable examples in {manifest_path}")
    params = [p for p in model.parameters() if p.requires_grad]
    opt = torch.optim.AdamW(params, lr=LR, weight_decay=0.0)
    scaler = torch.amp.GradScaler("cuda") if device == "cuda" else None

    metrics = {
        "model": MODEL,
        "manifest": MANIFEST,
        "manifest_path": str(manifest_path),
        "eval_path": str(eval_path),
        "final_eval_path": str(final_eval_path),
        "backend": backend_name,
        "ddp": use_ddp,
        "world_size": world_size,
        "max_len": MAX_LEN,
        "max_steps": MAX_STEPS,
        "grad_accum": GRAD_ACCUM,
        "max_sup_rows": MAX_SUP_ROWS,
        "lr": LR,
        "seed": SEED,
        "eval_every": EVAL_EVERY,
        "eval_windows": EVAL_WINDOWS,
        "eval_n": eval_total,
        "eval_n_local": len(eval_examples),
        "final_eval_n": final_total,
        "final_eval_n_local": len(final_examples),
        "stop_after_seconds": STOP_AFTER_SECONDS,
        "n_traces": len(examples) * world_size,
        "loss_path": "supervised_positions_only",
    }
    log: list[dict] = []
    eval_log: list[dict] = []
    step = 0
    seen_tok = 0
    seen_sup = 0
    windows_seen = 0
    pending_windows = list(EVAL_WINDOWS)
    stop_reason = "max_steps"
    train_t = time.time()
    i = 0

    def snapshot(status: str, **extra) -> dict:
        payload = {
            **metrics,
            "status": status,
            "steps_done": step,
            "windows_seen": windows_seen,
            "loss": log[-1]["loss"] if log else None,
            "tok_per_s": round(seen_tok * world_size / max(time.time() - train_t, 1e-9), 1),
            "sup_tok_per_s": round(seen_sup * world_size / max(time.time() - train_t, 1e-9), 1),
            "elapsed_s": round(time.time() - train_t),
            "total_elapsed_s": round(time.time() - t0),
            "mem_gb": round(torch.cuda.max_memory_allocated() / 1e9, 2)
            if device == "cuda"
            else 0.0,
            "eval_ce": eval_log[-1]["eval_ce"] if eval_log else None,
            "log": log,
            "eval": eval_log,
            "updated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            **extra,
        }
        return payload

    def run_eval(examples: list[dict]) -> float | None:
        """Every rank enters the all_reduce even with an empty local slice, or DDP blocks."""
        local_ce, local_n = 0.0, 0
        if examples:
            core.eval()
            with torch.no_grad():
                local_ce, local_n = eval_ce_of(core, examples, device)
            core.train()
        t = torch.tensor([local_ce, float(local_n)], device=device)
        if use_ddp:
            dist.all_reduce(t)
        return float(t[0] / t[1]) if float(t[1]) > 0 else None

    while step < MAX_STEPS:
        stop = int(time.time() - t0 > STOP_AFTER_SECONDS)
        if use_ddp:
            flag = torch.tensor([stop], device=device)
            dist.all_reduce(flag, op=dist.ReduceOp.MAX)
            stop = int(flag.item())
        if stop:
            stop_reason = "stop_after_seconds"
            break

        opt.zero_grad(set_to_none=True)
        acc_loss = 0.0
        for micro in range(GRAD_ACCUM):
            e = examples[i % len(examples)]
            i += 1
            ids = torch.tensor([e["input_ids"]], device=device)
            lab = torch.tensor([e["labels"]], device=device)
            sync = use_ddp and micro == GRAD_ACCUM - 1
            ctx = nullcontext() if sync else (module.no_sync() if use_ddp else nullcontext())
            with ctx:
                if device == "cuda":
                    with torch.autocast("cuda", dtype=torch.float16):
                        loss = module(ids, lab) / GRAD_ACCUM
                else:
                    loss = module(ids, lab) / GRAD_ACCUM
                if scaler is not None:
                    scaler.scale(loss).backward()
                else:
                    loss.backward()
            acc_loss += float(loss.item())
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
        windows_seen = step * GRAD_ACCUM * world_size
        hit = [w for w in pending_windows if windows_seen >= w]
        is_window_eval = bool(hit)
        is_eval_step = (bool(EVAL_EVERY) and step % EVAL_EVERY == 0) or is_window_eval
        if is_eval_step:
            ce = run_eval(eval_examples)
            eval_log.append(
                {
                    "step": step,
                    "windows_seen": windows_seen,
                    "eval_ce": ce,
                    "event": "windows" if is_window_eval else "step",
                }
            )
            if is_window_eval:
                for w in hit:
                    pending_windows.remove(w)
            print(
                f"[rank {rank}] eval step {step} (windows_seen={windows_seen}"
                + (f", milestone {hit}" if is_window_eval else "")
                + f"): ce={fmt_ce(ce)}",
                flush=True,
            )

        if step == 1 or step % METRICS_EVERY == 0 or step == MAX_STEPS or is_eval_step:
            el = time.time() - train_t
            rec = {
                "step": step,
                "windows_seen": windows_seen,
                "loss": round(acc_loss, 4),
                "tok_per_s": round(seen_tok / el, 1),
                "sup_tok_per_s": round(seen_sup / el, 1),
                "elapsed_s": round(el),
                "mem_gb": round(torch.cuda.max_memory_allocated() / 1e9, 2)
                if device == "cuda"
                else 0.0,
            }
            log.append(rec)
            print(f"[rank {rank}] {rec}", flush=True)
            if rank == 0:
                write_json(Path(OUT) / "metrics.json", snapshot("running"))
            elif use_ddp:
                write_json(Path(OUT) / f"metrics_rank{rank}.json", snapshot("running"))

    # Final audit on the full held-out set (300 traces) before the adapter is saved, so
    # the offline metric exists even if the save fails.
    final_eval_ce = None
    if step:
        final_eval_ce = run_eval(final_examples)
        eval_log.append(
            {
                "step": step,
                "windows_seen": windows_seen,
                "eval_ce": final_eval_ce,
                "event": "final_full",
            }
        )
        print(
            f"[rank {rank}] final eval on {final_total} full-set traces: "
            f"ce={fmt_ce(final_eval_ce)}",
            flush=True,
        )
        if rank == 0:
            write_json(Path(OUT) / "metrics.json", snapshot("running", final_eval_ce=final_eval_ce))
        elif use_ddp:
            write_json(
                Path(OUT) / f"metrics_rank{rank}.json",
                snapshot("running", final_eval_ce=final_eval_ce),
            )

    status = "stopped" if stop_reason == "stop_after_seconds" else "done"
    if rank == 0:
        model.save_pretrained(f"{OUT}/adapter")
        print(f"adapter saved to {OUT}/adapter", flush=True)
    write_json(
        Path(OUT) / "metrics.json" if rank == 0 else Path(OUT) / f"metrics_rank{rank}.json",
        snapshot(
            status,
            stop_reason=stop_reason,
            adapter_path=f"{OUT}/adapter",
            final_eval_ce=final_eval_ce,
        ),
    )
    if use_ddp:
        dist.barrier()
        dist.destroy_process_group()


def worker_entry(rank: int, world_size: int) -> None:
    try:
        train(rank, world_size)
    except Exception as exc:
        import traceback

        traceback.print_exc()
        payload = {
            "status": "error",
            "error": repr(exc),
            "rank": rank,
            "updated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        }
        path = Path(OUT) / ("metrics_error.json" if rank == 0 else f"metrics_error_rank{rank}.json")
        try:
            write_json(path, payload)
        except OSError:
            pass
        raise


def main() -> None:
    n_gpu = torch.cuda.device_count()
    world = 2 if (n_gpu == 2 and not NO_DDP) else 1
    if WORLD_SIZE:
        world = WORLD_SIZE
    elif NO_DDP:
        world = 1
    print(
        json.dumps(
            {
                "model": MODEL,
                "manifest": MANIFEST,
                "n_gpus": n_gpu,
                "world_size": world,
                "max_len": MAX_LEN,
                "max_steps": MAX_STEPS,
                "grad_accum": GRAD_ACCUM,
                "max_sup_rows": MAX_SUP_ROWS,
                "lr": LR,
                "seed": SEED,
                "eval_every": EVAL_EVERY,
                "eval_windows": EVAL_WINDOWS,
                "eval_file": EVAL_FILE,
                "final_eval_file": FINAL_EVAL_FILE,
                "eval_n": EVAL_N,
                "metrics_every": METRICS_EVERY,
                "stop_after_seconds": STOP_AFTER_SECONDS,
                "out": OUT,
            }
        ),
        flush=True,
    )
    if world > 1:
        import torch.multiprocessing as mp

        try:
            mp.set_start_method("spawn")
        except RuntimeError:
            pass
        os.environ.setdefault("MASTER_ADDR", "127.0.0.1")
        os.environ.setdefault("MASTER_PORT", str(29500 + random.randrange(1000)))
        mp.spawn(worker_entry, args=(world,), nprocs=world, join=True)
    else:
        worker_entry(0, 1)


if __name__ == "__main__":
    main()
