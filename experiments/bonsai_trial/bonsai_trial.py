#!/usr/bin/env python3
"""Bonsai T4 trial: can PrismML Ternary-Bonsai-2-27B (PTQ1_0) run on a Kaggle 2xT4, and how fast?

Protocol (all numbers land in /kaggle/working/bonsai_metrics.json and the kernel log):
  1. env probe: nvidia-smi, nvcc discovery, CPU/RAM/disk.
  2. runtime: PrismML llama.cpp fork, prebuilt release prism-b10685 (Linux x64 CUDA 12.4).
     Checked offline with cuobjdump: that build carries native cubins for sm_86/sm_89 plus PTX
     for sm_50/61/70/75/80/90, so a T4 (sm_75) runs the sm_75 PTX through driver JIT. If the
     prebuilt fails on-device, fall back to a source build:
       git clone -b prism https://github.com/PrismML-Eng/llama.cpp
       cmake -B build -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=75 -DGGML_NATIVE=OFF, build -j4
  3. model: prism-ml/Ternary-Bonsai-2-27B-gguf :: Ternary-Bonsai-2-27B-PTQ1_0.gguf (5.93 GB).
  4. llama-bench -ngl 99, pp512 / tg128, on one T4 then on both (-sm layer).
  5. one llama-cli completion on the faster config: ~300-token SWE issue prompt, temperature 0,
     256 max tokens, --jinja single-turn; wall time and raw reply recorded.
  6. optional bonus while time remains: the PQ2_0 file (docs: cheaper unpack -> faster prefill).

The script self-limits to DEADLINE_S (default 2400 s) so the metrics file is always written
before Kaggle's kernel timeout (-t 2700). Steps that do not fit are skipped and recorded.
"""
import glob
import json
import os
import re
import shutil
import subprocess
import sys
import tarfile
import time
import urllib.request
from pathlib import Path

RELEASE = "prism-b10685-7dffb15"
ASSET = f"llama-{RELEASE}-bin-linux-cuda-12.4-x64.tar.gz"
RELEASE_URL = f"https://github.com/PrismML-Eng/llama.cpp/releases/download/{RELEASE}/{ASSET}"
FORK_URL = "https://github.com/PrismML-Eng/llama.cpp"
FORK_BRANCH = "prism"
MODEL_REPO = "prism-ml/Ternary-Bonsai-2-27B-gguf"
MODEL_FILE = "Ternary-Bonsai-2-27B-PTQ1_0.gguf"
BONUS_FILE = "Ternary-Bonsai-2-27B-PQ2_0.gguf"
DEADLINE_S = float(os.environ.get("DEADLINE_S", "2400"))
BENCH_REPS = int(os.environ.get("BENCH_REPS", "3"))
COMPLETION_CTX = int(os.environ.get("COMPLETION_CTX", "16384"))
COMPLETION_TOKENS = int(os.environ.get("COMPLETION_TOKENS", "256"))
ROLLOUT = {"turns": 70, "gen_tokens": 20000, "prefill_tokens": 1_500_000}

OUT = Path("/kaggle/working") if Path("/kaggle/working").is_dir() else Path(__file__).resolve().parent / "out"
BASE = Path("/kaggle/temp") if Path("/kaggle/temp").is_dir() else Path("/tmp")
TMP = BASE / "bonsai"
BIN = TMP / "llama-bin"
MODELS = TMP / "models"

T0 = time.time()
ART = {"status": "started", "release": RELEASE, "model_file": MODEL_FILE, "skipped": []}
COUNTED = []

SWE_PROMPT = """Issue #1482: parse_duration() returns None for compound and ISO-8601 durations

Environment: Python 3.11, pytest 8.2, commit 9f3c1ab

Steps to reproduce:
1. Run `pytest tests/test_duration.py -x`
2. Observe the failures below.

Expected: parse_duration("1h30m") == 5400 and parse_duration("PT2H") == 7200, per the
docstring in utils/timeparse.py.
Actual: both return None. parse_duration("90s") and parse_duration("5m") still pass, so
the regex in _UNIT_RE fails only when the input mixes units or uses the ISO-8601 "PT"
prefix.

Additional context: the retry middleware (middleware/retry.py) treats a None timeout as
"retry forever". After the incident on Tuesday a stuck worker kept retrying for 14 hours
and held a database connection the whole time. The CI log tail is:

FAILED tests/test_duration.py::test_compound - AssertionError: None != 5400
FAILED tests/test_duration.py::test_iso_prefix - AssertionError: None != 7200

Keep the public API unchanged: parse_duration must keep returning int | None.

List the shell commands you would run first.
"""


def log(msg):
    print(f"[{time.time() - T0:7.1f}s] {msg}", flush=True)


def remaining():
    return DEADLINE_S - (time.time() - T0)


def skip(step, why):
    log(f"SKIP {step}: {why}")
    ART["skipped"].append({"step": step, "why": why})


def _env(env=None, cvd=None):
    e = os.environ.copy()
    if env:
        e.update(env)
    if cvd is not None:
        e["CUDA_VISIBLE_DEVICES"] = cvd
    return e


def sh(cmd, timeout=None, env=None, cwd=None, cvd=None):
    """Run a command, echo it, return (rc, stdout+stderr, seconds)."""
    t = time.time()
    log("RUN " + " ".join(str(c) for c in cmd)[:400])
    try:
        p = subprocess.run([str(c) for c in cmd], capture_output=True, text=True,
                           timeout=timeout, env=_env(env, cvd), cwd=cwd, check=False)
        out, rc = (p.stdout or "") + (p.stderr or ""), p.returncode
    except subprocess.TimeoutExpired as exc:
        out, rc = (exc.stdout or "") + (exc.stderr or "") + f"\n[timeout after {timeout}s]", -9
    dt = time.time() - t
    tail = [l for l in out.strip().splitlines() if l.strip()]
    log(f"  -> rc={rc} in {dt:.1f}s" + (f" | tail: {tail[-1][:300]}" if tail else ""))
    COUNTED.append({"cmd": " ".join(str(c) for c in cmd)[:200], "rc": rc, "s": round(dt, 1)})
    return rc, out, dt


def sh_split(cmd, timeout=None, env=None, cwd=None, cvd=None):
    """Like sh() but keeps stdout and stderr apart: (rc, stdout, stderr, seconds)."""
    t = time.time()
    log("RUN " + " ".join(str(c) for c in cmd)[:200] + " ...")
    try:
        p = subprocess.run([str(c) for c in cmd], capture_output=True, text=True,
                           timeout=timeout, env=_env(env, cvd), cwd=cwd, stdin=subprocess.DEVNULL,
                           check=False)
        so, se, rc = p.stdout or "", p.stderr or "", p.returncode
    except subprocess.TimeoutExpired as exc:
        so, se = exc.stdout or "", (exc.stderr or "") + f"\n[timeout after {timeout}s]"
        rc = -9
    dt = time.time() - t
    log(f"  -> rc={rc} in {dt:.1f}s, stdout {len(so)} chars, stderr {len(se)} chars")
    COUNTED.append({"cmd": " ".join(str(c) for c in cmd)[:200], "rc": rc, "s": round(dt, 1)})
    return rc, so, se, dt


def download(url, dest, label):
    dest = Path(dest)
    if dest.exists() and dest.stat().st_size > 0:
        log(f"{label}: already present ({dest.stat().st_size / 2**30:.2f} GiB)")
        return dest
    t = time.time()
    log(f"downloading {label}: {url}")
    req = urllib.request.Request(url, headers={"User-Agent": "bonsai-trial"})
    with urllib.request.urlopen(req, timeout=120) as r, open(dest, "wb") as f:
        n = 0
        while True:
            chunk = r.read(1 << 22)
            if not chunk:
                break
            f.write(chunk)
            n += len(chunk)
            if n % (1 << 30) < (1 << 22):
                log(f"  {n / 2**30:.2f} GiB ...")
    dt = time.time() - t
    gb = dest.stat().st_size / 2**30
    log(f"{label}: {gb:.2f} GiB in {dt:.0f}s ({gb * 1024 / dt:.0f} MiB/s)")
    return dest


def parse_json_array(text):
    i, j = text.find("["), text.rfind("]")
    if i < 0 or j < 0:
        return None
    try:
        return json.loads(text[i:j + 1])
    except json.JSONDecodeError:
        return None


def bench_summary(records):
    if not records:
        return None

    def flat(r):
        if not r:
            return None
        return {"test": f"pp{r.get('n_prompt')}" if r.get("n_prompt") else f"tg{r.get('n_gen')}",
                "avg_ts": round(r.get("avg_ts") or 0, 2),
                "stddev_ts": round(r.get("stddev_ts") or 0, 2),
                "avg_ms": round((r.get("avg_ns") or 0) / 1e6, 1),
                "samples_ts": [round(x, 2) for x in (r.get("samples_ts") or [])]}

    pp = next((r for r in records if (r.get("n_prompt") or 0) > 0 and not (r.get("n_gen") or 0)), None)
    tg = next((r for r in records if (r.get("n_gen") or 0) > 0 and not (r.get("n_prompt") or 0)), None)
    return {"pp": flat(pp), "tg": flat(tg)}


def find_binaries():
    hits = sorted(BIN.rglob("llama-bench"))
    return hits[0].parent if hits else None


def ld_env(bindir):
    parts = [str(bindir), "/usr/local/cuda/lib64"]
    parts += sorted(glob.glob(str(TMP / "cudalibs" / "nvidia" / "*" / "lib")))
    parts += [p for p in os.environ.get("LD_LIBRARY_PATH", "").split(":") if p]
    return {"LD_LIBRARY_PATH": ":".join(parts)}


def check_libs(bindir):
    missing = {}
    for so in ["libggml-cuda.so", "libggml-base.so"]:
        p = bindir / so
        if not p.exists():
            continue
        _, out, _ = sh(["ldd", str(p)], timeout=60)
        miss = sorted({l.split()[0] for l in out.splitlines() if "not found" in l})
        if miss:
            missing[so] = miss
    if missing:
        log(f"missing shared libs {missing} -> pip nvidia-cuda-runtime-cu12 + nvidia-cublas-cu12")
        sh([sys.executable, "-m", "pip", "install", "-q", "--target", str(TMP / "cudalibs"),
            "nvidia-cuda-runtime-cu12", "nvidia-cublas-cu12"], timeout=600)
    return missing


def find_nvcc():
    """nvcc discovery: PATH -> /usr/local/cuda*/bin -> pip wheel -> torch tree. Records what worked."""
    steps = []
    p = shutil.which("nvcc")
    steps.append({"step": "PATH", "result": p or "not found"})
    if not p:
        hits = sorted(glob.glob("/usr/local/cuda*/bin/nvcc"))
        steps.append({"step": "glob /usr/local/cuda*/bin/nvcc", "result": hits or "none"})
        p = hits[0] if hits else None
    if not p:
        tgt = TMP / "nvccpip"
        sh([sys.executable, "-m", "pip", "install", "-q", "--target", str(tgt),
            "nvidia-cuda-nvcc-cu12"], timeout=600)
        hits = [x for x in tgt.rglob("bin/*") if x.is_file()]
        steps.append({"step": "pip nvidia-cuda-nvcc-cu12", "bin_contents": sorted(x.name for x in hits)})
        p = next((str(x) for x in hits if x.name == "nvcc"), None)
    if not p:
        try:
            import torch
            base = Path(torch.__file__).parent
            hits = [str(x) for x in base.rglob("bin/nvcc")]
            ptx = sorted(x.name for x in base.rglob("bin/ptxas"))
            steps.append({"step": "torch tree", "nvcc": hits or None, "ptxas": ptx or None})
            p = hits[0] if hits else None
        except Exception as exc:  # noqa: BLE001
            steps.append({"step": "torch tree", "result": f"torch unavailable: {exc}"})
    return {"path": p, "discovery": steps}


# ---------------------------------------------------------------- step 1: env
def step_env():
    log("=== step 1: environment ===")
    _, out, _ = sh(["nvidia-smi"], timeout=120)
    ART["env"] = {"nvidia_smi": out.strip()[:2500]}
    for pat, key in ((r"Driver Version:\s*([\d.]+)", "driver"),
                     (r"CUDA Version:\s*([\d.]+)", "cuda_version_smi")):
        ART["env"][key] = (re.search(pat, out) or [None, "?"])[1]
    _, out, _ = sh(["nvidia-smi", "-L"], timeout=120)
    ART["env"]["gpus"] = [l for l in out.splitlines() if l.strip()]
    ART["env"]["n_gpus"] = len(ART["env"]["gpus"])
    _, out, _ = sh(["bash", "-lc", "nproc; free -g | head -2; df -h /kaggle/temp /kaggle/working | tail -3"],
                   timeout=60)
    ART["env"]["host"] = out.strip()
    ART["env"]["nvcc"] = find_nvcc()
    if ART["env"]["nvcc"].get("path"):
        _, out, _ = sh([ART["env"]["nvcc"]["path"], "--version"], timeout=60)
        ART["env"]["nvcc_version"] = out.strip().splitlines()[-1] if out.strip() else "?"
    try:
        import torch
        ART["env"]["torch"] = f"{torch.__version__} cuda={torch.version.cuda} available={torch.cuda.is_available()}"
    except Exception as exc:  # noqa: BLE001
        ART["env"]["torch"] = f"unavailable: {exc}"


# --------------------------------------------------- step 2a: prebuilt runtime
def step_prebuilt():
    log("=== step 2: prebuilt PrismML runtime (CUDA 12.4 tarball) ===")
    t = time.time()
    tar = download(RELEASE_URL, TMP / ASSET, "runtime tarball")
    BIN.mkdir(parents=True, exist_ok=True)
    try:
        with tarfile.open(tar) as tf:
            tf.extractall(BIN, filter="data")
    except TypeError:
        with tarfile.open(tar) as tf:
            tf.extractall(BIN)
    bindir = find_binaries()
    ART["runtime"] = {"path": "prebuilt", "release": RELEASE, "asset": ASSET,
                      "setup_s": round(time.time() - t, 1), "bin_dir": str(bindir)}
    if bindir is None:
        ART["runtime"]["error"] = "llama-bench not found in tarball"
        return None
    check_libs(bindir)
    ART["runtime"]["bins"] = {b: str(bindir / b) if (bindir / b).exists() else None
                              for b in ["llama-bench", "llama-cli", "llama-completion"]}
    rc, out, _ = sh([str(bindir / "llama-bench"), "--version"], timeout=180, env=ld_env(bindir))
    ART["runtime"]["version_line"] = out.strip().splitlines()[0] if out.strip() else f"rc={rc}"
    return bindir


# -------------------------------------------------- step 2b: source-build fallback
def step_build():
    log("=== step 2 fallback: source build for sm_75 ===")
    nvcc = ART.get("env", {}).get("nvcc", {})
    if not nvcc.get("path"):
        ART["build"] = {"attempted": False, "reason": "no nvcc found", "discovery": nvcc.get("discovery")}
        log("build skipped: no nvcc")
        return None
    if remaining() < 1500:
        ART["build"] = {"attempted": False, "reason": f"insufficient time ({remaining():.0f}s left)"}
        return None
    src = TMP / "fork"
    if not src.exists():
        rc, out, _ = sh(["git", "clone", "--depth", "1", "-b", FORK_BRANCH, FORK_URL, str(src)], timeout=600)
        if rc != 0:
            ART["build"] = {"attempted": True, "ok": False, "stage": "clone", "tail": out.splitlines()[-80:]}
            return None
    ENV = {"PATH": f"{Path(nvcc['path']).parent}:" + os.environ["PATH"],
           "CUDACXX": nvcc["path"], "CUDA_HOME": str(Path(nvcc["path"]).parent.parent)}
    rc, out, _ = sh(["cmake", "-B", "build", "-DCMAKE_BUILD_TYPE=Release", "-DGGML_CUDA=ON",
                     "-DCMAKE_CUDA_ARCHITECTURES=75", "-DGGML_NATIVE=OFF", "-DLLAMA_CURL=OFF",
                     "-DLLAMA_BUILD_TESTS=OFF", "-DGGML_CUDA_FA_ALL_QUANTS=OFF"],
                    timeout=900, cwd=src, env=ENV)
    if rc != 0:
        ART["build"] = {"attempted": True, "ok": False, "stage": "configure", "tail": out.splitlines()[-80:]}
        return None
    t = time.time()
    rc, out, _ = sh(["cmake", "--build", "build", "-j4", "--target", "llama-bench", "llama-cli",
                     "llama-completion"], timeout=3000, cwd=src, env=ENV)
    build_s = time.time() - t
    log(f"build rc={rc} in {build_s:.0f}s")
    if rc != 0:
        ART["build"] = {"attempted": True, "ok": False, "stage": "compile", "build_s": round(build_s, 1),
                        "tail": out.splitlines()[-80:]}
        return None
    bindir = src / "build" / "bin"
    if not (bindir / "llama-bench").exists():
        ART["build"] = {"attempted": True, "ok": False, "stage": "locate", "build_s": round(build_s, 1)}
        return None
    ART["build"] = {"attempted": True, "ok": True, "build_s": round(build_s, 1),
                    "flags": "-DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=75 -DGGML_NATIVE=OFF -DLLAMA_CURL=OFF -j4"}
    return bindir


# ------------------------------------------------------------ step 3: the model
def step_model(fname=MODEL_FILE):
    try:
        from huggingface_hub import hf_hub_download
    except ImportError:
        subprocess.check_call([sys.executable, "-m", "pip", "install", "-q", "huggingface_hub"])
        from huggingface_hub import hf_hub_download
    t = time.time()
    p = hf_hub_download(MODEL_REPO, fname, local_dir=str(MODELS))
    dt = time.time() - t
    gb = Path(p).stat().st_size / 2**30
    log(f"model {fname}: {gb:.2f} GiB in {dt:.0f}s ({gb * 1024 / dt:.0f} MiB/s)")
    return {"path": str(p), "file": fname, "bytes": Path(p).stat().st_size, "gib": round(gb, 2),
            "download_s": round(dt, 1), "mbps": round(gb * 1024 / dt, 1) if dt else None}


def run_bench(bindir, model_path, label, extra, cvd, reps=BENCH_REPS, p=512, n=128, timeout=1200):
    rc, out, dt = sh([str(bindir / "llama-bench"), "-m", model_path, "-ngl", "99",
                      "-p", str(p), "-n", str(n), "-r", str(reps), "-o", "json"] + extra,
                     timeout=timeout, env=ld_env(bindir), cvd=cvd)
    recs = parse_json_array(out)
    res = {"label": label, "rc": rc, "wall_s": round(dt, 1), "cuda_visible_devices": cvd,
           "flags": extra, "summary": bench_summary(recs), "raw": recs}
    if recs is None:
        res["tail"] = out.splitlines()[-25:]
    (OUT / f"bonsai_bench_{label}.json").write_text(json.dumps(res, indent=2))
    return res


def step_completion(bindir, model_path, cvd, extra, label, n_tokens=COMPLETION_TOKENS, tag="main"):
    log(f"=== step 5: completion on {label} ({tag}, n={n_tokens}) ===")
    env = ld_env(bindir)
    base = [str(bindir / "llama-cli"), "-m", model_path, "-c", str(COMPLETION_CTX), "-ngl", "99",
            "--temp", "0", "--seed", "0", "-n", str(n_tokens), "--no-warmup", "-p", SWE_PROMPT]
    modes = [("jinja_single_turn", ["--jinja", "-st"] + extra), ("no_cnv_completion", ["-no-cnv"] + extra)]
    attempts = []
    for mode, flags in modes:
        rc, so, se, dt = sh_split(base + flags, timeout=1500, env=env, cvd=cvd)
        (OUT / f"bonsai_completion_{tag}_{mode}.txt").write_text(so)
        (OUT / f"bonsai_completion_{tag}_{mode}.stderr.txt").write_text(se)
        # every gen-side pattern is anchored on "runs" so it cannot match the prompt-eval line
        # ("prompt eval time =" contains the substring "eval time =")
        perf = {
            "load_ms": grab(se, r"load time =\s*([\d.]+) ms"),
            "prompt_ms": grab(se, r"prompt eval time =\s*([\d.]+) ms"),
            "prompt_tokens": grab(se, r"prompt eval time =.*?/\s*(\d+) tokens"),
            "prompt_tok_s": grab(se, r"prompt eval time =.*?([\d.]+) tokens per second\)"),
            "gen_ms": grab(se, r"eval time =\s*([\d.]+) ms /\s*\d+ runs"),
            "gen_tokens": grab(se, r"/\s*(\d+) runs"),
            "gen_tok_s": grab(se, r"runs\s+\(\s*[\d.]+ ms per token,\s*([\d.]+) tokens per second\)"),
            "total_ms": grab(se, r"total time =\s*([\d.]+) ms"),
        }
        attempt = {"mode": mode, "flags": flags, "rc": rc, "wall_s": round(dt, 1), "perf": perf,
                   "reply": so.strip(), "stderr_tail": se.strip().splitlines()[-25:]}
        attempts.append(attempt)
        if rc == 0 and perf["gen_tokens"] and so.strip():
            break
    best = attempts[-1]
    for a in attempts:
        if a["rc"] == 0 and a["perf"]["gen_tokens"] and a["reply"]:
            best = a
            break
    return {"gpu_config": label, "cuda_visible_devices": cvd, "flags": extra, "n_tokens": n_tokens,
            "attempts": [{k: v for k, v in a.items() if k != "reply"} for a in attempts],
            "mode_used": best["mode"], "wall_s": best["wall_s"], "perf": best["perf"],
            "prompt": SWE_PROMPT, "reply": best["reply"]}


def grab(text, pat):
    m = re.search(pat, text)
    return float(m.group(1)) if m else None


def main():
    for d in (TMP, BIN, MODELS):
        d.mkdir(parents=True, exist_ok=True)
    ART["dirs"] = {"tmp": str(TMP), "out": str(OUT)}
    step_env()

    model = None
    bindir = step_prebuilt()
    if bindir:
        model = step_model()
        log("=== step 4a: smoke bench (also pays the sm_75 PTX JIT cost) ===")
        smoke = run_bench(bindir, model["path"], "smoke_prebuilt", [], cvd="0", reps=1, p=32, n=8, timeout=900)
        ART["bench"] = {"smoke": smoke}
    else:
        ART["bench"] = {}

    if bindir and not (smoke["rc"] == 0 and smoke.get("summary")):
        log("prebuilt runtime failed on-device -> source build fallback")
        ART["runtime"]["failed_on_device"] = {"rc": smoke["rc"], "tail": smoke.get("tail")}
        bindir = step_build()
        if bindir:
            model = model or step_model()
            s2 = run_bench(bindir, model["path"], "smoke_source_build", [], cvd="0", reps=1, p=32, n=8, timeout=900)
            ART["bench"]["smoke_source_build"] = s2
            smoke = s2
    elif not bindir:
        bindir = step_build()
        if bindir:
            model = step_model()
            s2 = run_bench(bindir, model["path"], "smoke_source_build", [], cvd="0", reps=1, p=32, n=8, timeout=900)
            ART["bench"]["smoke_source_build"] = s2
            smoke = s2

    if not bindir or not (smoke["rc"] == 0 and smoke.get("summary")):
        ART["status"] = "runtime_unavailable" if not bindir else "smoke_failed"
        return

    ART["runtime"]["active_bin_dir"] = str(bindir)
    ART["model"] = model
    ART["bench"]["active_smoke"] = smoke["label"]

    log("=== step 4b: bench 1xT4 ===")
    ART["bench"]["single_gpu"] = run_bench(bindir, model["path"], "single_gpu", [], cvd="0")
    if ART["env"]["n_gpus"] >= 2 and remaining() > 700:
        log("=== step 4c: bench 2xT4 (-sm layer) ===")
        ART["bench"]["dual_gpu"] = run_bench(bindir, model["path"], "dual_gpu", ["-sm", "layer"], cvd="0,1")
    else:
        skip("bench_dual_gpu", f"n_gpus={ART['env']['n_gpus']}, {remaining():.0f}s left")

    def ts(name, which):
        return ((ART["bench"].get(name) or {}).get("summary") or {}).get(which) or {}

    s_pp, s_tg = ts("single_gpu", "pp").get("avg_ts"), ts("single_gpu", "tg").get("avg_ts")
    d_pp, d_tg = ts("dual_gpu", "pp").get("avg_ts"), ts("dual_gpu", "tg").get("avg_ts")
    if d_tg and s_tg and d_tg >= s_tg:
        best = ("dual_gpu", "0,1", ["-sm", "layer"])
    else:
        best = ("single_gpu", "0", [])
    ART["bench"]["best_for_completion"] = best[0]

    ART["rollout_estimate"] = {}
    for name, pp, tg in (("single_gpu", s_pp, s_tg), ("dual_gpu", d_pp, d_tg)):
        if pp and tg:
            sec = ROLLOUT["prefill_tokens"] / pp + ROLLOUT["gen_tokens"] / tg
            ART["rollout_estimate"][name] = {
                "prefill_s": round(ROLLOUT["prefill_tokens"] / pp, 1),
                "gen_s": round(ROLLOUT["gen_tokens"] / tg, 1),
                "rollout_min": round(sec / 60, 1),
                "rollouts_per_12h": round(43200 / sec, 1),
                "assumption": (f"{ROLLOUT['turns']} turns, {ROLLOUT['gen_tokens']} generated tokens, "
                               f"{ROLLOUT['prefill_tokens']} prefill tokens (no KV-reuse credit)")}

    if remaining() > 400:
        ART["completion"] = step_completion(bindir, model["path"], best[1], best[2], best[0])
        if remaining() > 600:
            # the model thinks by default (xhigh effort): 256 tokens is often all reasoning,
            # so a longer run shows whether the answer actually arrives and stays coherent
            ART["completion_extended"] = step_completion(bindir, model["path"], best[1], best[2],
                                                         best[0], n_tokens=768, tag="extended")
    else:
        skip("completion", f"only {remaining():.0f}s left")

    # the vendor quickstart passes "-fa on"; the llama-bench default is "auto", so measure the delta
    if remaining() > 700:
        try:
            ART["bench"]["fa_on_single_gpu"] = run_bench(bindir, model["path"], "fa_on_single_gpu",
                                                         ["-fa", "on"], cvd="0", reps=1)
        except Exception as exc:  # noqa: BLE001
            ART["bench"]["fa_on_single_gpu"] = {"error": repr(exc)}
    else:
        skip("fa_on_bench", f"{remaining():.0f}s left")

    free_gib = shutil.disk_usage(str(TMP)).free / 2**30
    if remaining() > 900 and smoke["rc"] == 0 and free_gib > 9:
        log(f"=== bonus: PQ2_0 file (docs: cheaper unpack -> faster prefill); {free_gib:.1f} GiB free ===")
        try:
            m2 = step_model(BONUS_FILE)
            ART["bonus_model"] = m2
            ART["bench"]["pq2_bonus"] = run_bench(bindir, m2["path"], "pq2_single_gpu", [], cvd="0", reps=2)
        except Exception as exc:  # noqa: BLE001
            ART["bench"]["pq2_bonus"] = {"error": repr(exc)}
    else:
        skip("pq2_bonus", f"{remaining():.0f}s left, {free_gib:.1f} GiB free, smoke_rc={smoke['rc']}")


if __name__ == "__main__":
    try:
        main()
        if ART.get("status") == "started":
            ART["status"] = "ok"
    except Exception as exc:  # noqa: BLE001
        import traceback
        traceback.print_exc()
        ART["status"] = "exception"
        ART["error"] = repr(exc)
    ART["elapsed_s"] = round(time.time() - T0, 1)
    ART["wall_min"] = round((time.time() - T0) / 60, 2)
    ART["gpu_quota_note"] = ("wall_min is the T4x2 session wall time; if Kaggle bills the T4x2 shape at "
                             "2 GPU-hours per wall hour the session cost ~2x wall_min GPU-minutes.")
    ART["timings"] = COUNTED
    p = OUT / "bonsai_metrics.json"
    p.write_text(json.dumps(ART, indent=2, default=str))
    print("\n================ bonsai_metrics.json ================", flush=True)
    print(json.dumps(ART, indent=2, default=str), flush=True)
    print(f"written: {p}", flush=True)
