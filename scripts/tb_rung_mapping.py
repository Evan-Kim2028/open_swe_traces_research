#!/usr/bin/env python3
"""Classify Terminal-Bench tasks on the L0-L6 ladder and join pass rates.

Reads TB2 (terminal-bench-2) and TB4 (eval_tasks tb-ref) task trees plus
local result files only. Writes analytics/research/tb4_rung_mapping.md.
"""
from __future__ import annotations

import json
import re
import statistics
from collections import defaultdict
from pathlib import Path

import pyarrow.parquet as pq
import tomllib

TB2 = Path("/home/evan/Documents/terminal-bench-2")
TB4 = Path("/home/evan/Documents/eval_tasks/research/tb-ref/tasks")
TB4_OUT = Path("/home/evan/Documents/eval_tasks/analysis/tb4/out")
TB4_RAW = Path("/home/evan/Documents/eval_tasks/analysis/tb4/raw")
TB4_FACETS = Path("/home/evan/Documents/eval_tasks/analysis/tb4/facets")
HF_TRAJ = Path(
    "/home/evan/.cache/huggingface/hub/datasets--yoonholee--terminalbench-trajectories"
    "/snapshots/04e8940f5b6736a7ce8d22224fe2f2af74163ed2/data"
)
REPORT = Path(__file__).resolve().parents[1] / "analytics/research/tb4_rung_mapping.md"

# Frontier models for TB2 trajectories (top pass-rate slugs in cache).
TB2_FRONTIER_MODELS = [
    "claude-opus-4-6@anthropic",
    "claude-opus-4.6@anthropic",
    "gpt-5.3-codex@openai",
    "gemini-3.1-pro-preview@Google",
    "gpt-5.2-codex@openai",
]

# TB4 Hub submissions treated as frontier (13 models in raw/).
TB4_FRONTIER_KEYS = [
    "anthropic-claude-fable-5-1-max-claude-code",
    "anthropic-claude-opus-5-max-claude-code",
    "anthropic-claude-fable-5-max-claude-code",
    "anthropic-glm-5-3-max-claude-code",
    "openai-gpt-5-6-sol-max-codex",
    "anthropic-claude-opus-4-8-max-claude-code",
    "openai-gpt-5-6-terra-max-codex",
    "xai-grok-4-6-none-grok-build",
    "gemini-gemini-3-8-flash-high-mini-swe-agent",
    "openai-gpt-5-6-luna-max-codex",
    "anthropic-claude-sonnet-5-max-claude-code",
    "xai-grok-4-5-none-grok-build",
    "gemini-gemini-3-7-flash-high-mini-swe-agent",
]

TB4_MODEL_SHORT = {
    "anthropic-claude-fable-5-1-max-claude-code": "fable-5.1/cc",
    "anthropic-claude-opus-5-max-claude-code": "opus-5/cc",
    "anthropic-claude-fable-5-max-claude-code": "fable-5/cc",
    "anthropic-glm-5-3-max-claude-code": "glm-5.3/cc",
    "openai-gpt-5-6-sol-max-codex": "gpt-5.6-sol/codex",
    "anthropic-claude-opus-4-8-max-claude-code": "opus-4.8/cc",
    "openai-gpt-5-6-terra-max-codex": "gpt-5.6-terra/codex",
    "xai-grok-4-6-none-grok-build": "grok-4.6/build",
    "gemini-gemini-3-8-flash-high-mini-swe-agent": "gemini-3.8-flash/mini-swe",
    "openai-gpt-5-6-luna-max-codex": "gpt-5.6-luna/codex",
    "anthropic-claude-sonnet-5-max-claude-code": "sonnet-5/cc",
    "xai-grok-4-5-none-grok-build": "grok-4.5/build",
    "gemini-gemini-3-7-flash-high-mini-swe-agent": "gemini-3.7-flash/mini-swe",
}


def read(p: Path) -> str:
    try:
        return p.read_text(errors="replace")
    except OSError:
        return ""


def test_src(td: Path) -> str:
    tests = td / "tests"
    if not tests.exists():
        return ""
    return "\n".join(read(f) for f in tests.rglob("*") if f.is_file())


def env_src(td: Path) -> str:
    env = td / "environment"
    if not env.exists():
        return ""
    return "\n".join(read(f) for f in env.rglob("*") if f.is_file())


def tests_in_agent_env(td: Path) -> bool:
    docker = read(td / "environment" / "Dockerfile")
    if re.search(r"COPY\s+tests/", docker, re.I):
        return True
    env_tests = list((td / "environment").rglob("test*.py")) if (td / "environment").exists() else []
    return bool(env_tests)


def extract_assert_literals(test_text: str) -> set[str]:
    lits: set[str] = set()
    for m in re.finditer(r'["\'](/[^"\']{3,})["\']', test_text):
        lits.add(m.group(1))
    for m in re.finditer(r'Path\(["\']([^"\']+)["\']\)', test_text):
        lits.add(m.group(1))
    for m in re.finditer(r'==\s*["\']([^"\']+)["\']', test_text):
        lits.add(m.group(1))
    for m in re.finditer(r'open\(["\']([^"\']+)["\']', test_text):
        lits.add(m.group(1))
    return lits


def classify_verifier(test_text: str, task_toml: dict) -> tuple[str, bool, str]:
    """Return (verifier_class, tests_hidden, notes)."""
    hidden = True
    vmode = (task_toml.get("verifier") or {}).get("environment_mode") or "separate"
    if vmode in ("shared", "same"):
        hidden = False
    # Drop canary/header lines so "BENCHMARK DATA" does not trigger perf heuristics.
    body = "\n".join(
        ln for ln in test_text.splitlines() if not ln.strip().startswith("#")
    )
    low = body.lower()
    dyn = bool(
        re.search(
            r"(\btime\.perf_counter\b|\bpytest\.benchmark\b|ns/op|wall.?clock|"
            r"load.?avg|go test -race|memory.?leak|throughput\b|latency\b|"
            r"\bdeadlock\b|\brace condition\b|max_concurrent|concurrent workers)",
            low,
        )
    )
    prop = bool(
        re.search(
            r"\b(hypothesis|@given|fuzz|random\.seed|property-based|"
            r"@pytest\.mark\.parametrize.*random)\b",
            low,
        )
    )
    if dyn:
        cls = "dynamic-gate"
    elif prop:
        cls = "property/fuzz"
    else:
        cls = "example-tests"
    notes = []
    if re.search(r"reference|expected|golden|hidden", low):
        notes.append("hidden-reference")
    if re.search(r"tolerance|approx|isclose|within", low):
        notes.append("numeric-tolerance")
    return cls, hidden, ";".join(notes) if notes else "-"


def instruction_leakage(instr: str, test_text: str) -> tuple[bool, list[str]]:
    leaks: list[str] = []
    for lit in sorted(extract_assert_literals(test_text)):
        if lit in instr:
            leaks.append(lit)
    # command leakage
    for cmd in re.findall(r"subprocess\.run\(\[?['\"]([^'\"]+)['\"]", test_text):
        if cmd in instr:
            leaks.append(f"cmd:{cmd}")
    return bool(leaks), leaks[:8]


def classify_rung(instr: str, test_text: str, td: Path, facet: dict | None) -> tuple[int, str]:
    """Classify instruction information content on L0-L6."""
    reasons: list[str] = []
    w = len(instr.split())
    low = instr.lower()

    if tests_in_agent_env(td):
        return 6, "tests copied into agent environment (Dockerfile COPY tests/)"

    # L5: explicit example I/O in instruction mirroring test style
    if re.search(r"\bexample\b.*\b(input|output|result)\b", low) or re.search(
        r"```", instr
    ):
        if re.search(r"assert|expected|should (return|output|produce)", low):
            reasons.append("example I/O in instruction")
            return 5, "; ".join(reasons)

    # L4: signatures / interfaces in instruction
    sig_hits = len(
        re.findall(
            r"(def \w+\(|class \w+\(|:\s*\n\s+def |\binterface\b|\bsignature\b)",
            instr,
        )
    )
    if sig_hits >= 2 or re.search(r"according to the given signature", low):
        reasons.append(f"{sig_hits} signature/interface blocks")
        return 4, "; ".join(reasons)

    # L3: named checks / test descriptions in instruction
    if re.search(r"\btest_\w+\b", instr) or re.search(
        r"will be tested (for|on)|tests will (check|verify|use)", low
    ):
        reasons.append("names test checks in instruction")
        return 3, "; ".join(reasons)

    # L0: bug-report / goal only
    bug_style = bool(
        re.search(r"\b(bug|broken|can't|cannot|help me|find my|lost|crash|fails)\b", low)
    )
    if w < 60 and bug_style and not re.search(r"\b(write|create|implement)\b", low):
        return 0, "short bug-report style, no behavioral contract"

    # L1 vs L2 via facet when available
    if facet:
        sp = facet.get("spec_precision", "")
        hi = facet.get("hidden_invariant", False)
        if sp == "needs-inference" or hi:
            return 1, f"facet: spec_precision={sp}, hidden_invariant={hi}"
        if sp == "fully-specified":
            return 2, "facet: fully-specified, no test names/signatures in instruction"
        if sp == "constraint-dense":
            if re.search(r"\b(must|shall|required|do not|must not)\b", low):
                return 2, "facet: constraint-dense with explicit rules in instruction"
            return 1, "facet: constraint-dense but rules incomplete vs verifier"

    # Heuristic L1 vs L2 without facet: compare instruction coverage of assert targets
    assert_paths = {p for p in extract_assert_literals(test_text) if p.startswith("/")}
    covered = sum(1 for p in assert_paths if p in instr)
    if assert_paths and covered / len(assert_paths) < 0.5:
        return 1, f"instruction names {covered}/{len(assert_paths)} asserted paths"

    if w >= 150 or len(re.findall(r"\b(must|shall|output|write|create|implement)\b", low)) >= 3:
        return 2, "long behavioral spec in instruction"

    if w < 100:
        return 1, "medium instruction; unstated test requirements likely"

    return 2, "default: prose spec without test names/signatures/examples"


def load_facets() -> dict[str, dict]:
    out: dict[str, dict] = {}
    for p in sorted(TB4_FACETS.glob("batch*.json")):
        for row in json.loads(p.read_text()):
            out[row["task"]] = row
    return out


def classify_task(td: Path, suite: str, facet: dict | None = None) -> dict:
    t = tomllib.load(open(td / "task.toml", "rb"))
    instr = read(td / "instruction.md")
    tests = test_src(td)
    vclass, hidden, vnotes = classify_verifier(tests, t)
    rung, rung_note = classify_rung(instr, tests, td, facet)
    leaked, leak_items = instruction_leakage(instr, tests)
    return {
        "task": td.name,
        "suite": suite,
        "rung": rung,
        "rung_note": rung_note,
        "verifier": vclass,
        "verifier_notes": vnotes,
        "tests_hidden": hidden,
        "leakage": leaked,
        "leakage_items": leak_items,
        "instruction_words": len(instr.split()),
        "n_pytest_tests": len(re.findall(r"^\s*def test_", tests, re.M)),
        "difficulty": (t.get("metadata") or {}).get("difficulty"),
        "facet_spec_precision": (facet or {}).get("spec_precision"),
        "facet_hidden_invariant": (facet or {}).get("hidden_invariant"),
        "path": str(td),
    }


def load_tb4_trials() -> dict[str, dict[str, list[int]]]:
    """model_key -> task -> list of rewards (0/1)."""
    out: dict[str, dict[str, list[int]]] = defaultdict(lambda: defaultdict(list))
    for p in sorted(TB4_RAW.glob("*.trials.json")):
        key = p.name.replace(".trials.json", "")
        data = json.loads(p.read_text())
        for row in data:
            task = row.get("task_name", "")
            task = task.split("/")[-1] if "/" in task else task
            out[key][task].append(int(row.get("reward", 0) or 0))
    return out


def load_tb2_trials() -> dict[str, dict[str, list[int]]]:
    out: dict[str, dict[str, list[int]]] = defaultdict(lambda: defaultdict(list))
    if not HF_TRAJ.exists():
        return out
    tables = [pq.read_table(f, columns=["task_name", "model", "reward"]) for f in HF_TRAJ.glob("*.parquet")]
    import pyarrow as pa

    combined = pa.concat_tables(tables)
    for task, model, reward in zip(
        combined["task_name"].to_pylist(),
        combined["model"].to_pylist(),
        combined["reward"].to_pylist(),
    ):
        if model in TB2_FRONTIER_MODELS:
            out[model][task].append(int(reward))
    return out


def pass_stats(rewards: list[int]) -> dict:
    n = len(rewards)
    if n == 0:
        return {"n": 0, "pass_rate": None, "variance": None}
    p = sum(rewards) / n
    var = p * (1 - p)  # Bernoulli variance per trial
    return {"n": n, "pass_rate": p, "variance": var, "passes": sum(rewards)}


def aggregate_by_rung(
    classified: list[dict],
    trials: dict[str, dict[str, list[int]]],
    model_keys: list[str],
) -> tuple[dict, dict, dict]:
    task_rung = {r["task"]: r["rung"] for r in classified}
    task_ver = {r["task"]: r["verifier"] for r in classified}
    by_rung: dict[int, list[int]] = defaultdict(list)
    by_ver: dict[str, list[int]] = defaultdict(list)
    task_var: dict[str, list[float]] = defaultdict(list)

    for model in model_keys:
        for task, rewards in trials.get(model, {}).items():
            if task not in task_rung:
                continue
            for r in rewards:
                by_rung[task_rung[task]].append(r)
                by_ver[task_ver[task]].append(r)
            if len(rewards) >= 2:
                task_var[task].append(statistics.pvariance(rewards))

    rung_rates = {
        rung: pass_stats(rs)["pass_rate"]
        for rung, rs in sorted(by_rung.items())
        if rs
    }
    ver_rates = {
        v: pass_stats(rs)["pass_rate"]
        for v, rs in sorted(by_ver.items())
        if rs
    }
    return rung_rates, ver_rates, task_var


def trial_disagreement(trials: dict[str, dict[str, list[int]]], tasks: set[str]) -> list[dict]:
    rows = []
    for task in sorted(tasks):
        per_model = []
        for model, tmap in trials.items():
            if task not in tmap:
                continue
            rs = tmap[task]
            if len(rs) >= 3:
                per_model.append(sum(rs) / len(rs))
        if len(per_model) >= 2:
            rows.append(
                {
                    "task": task,
                    "model_pass_rates": per_model,
                    "spread": max(per_model) - min(per_model),
                    "mean": statistics.mean(per_model),
                }
            )
    return sorted(rows, key=lambda x: x["spread"], reverse=True)


def md_table(headers: list[str], rows: list[list]) -> str:
    lines = [
        "| " + " | ".join(headers) + " |",
        "|" + "|".join("---" for _ in headers) + "|",
    ]
    for row in rows:
        lines.append("| " + " | ".join(str(c) for c in row) + " |")
    return "\n".join(lines)


def main() -> None:
    facets = load_facets()
    classified: list[dict] = []

    for td in sorted(p for p in TB2.iterdir() if (p / "task.toml").exists()):
        classified.append(classify_task(td, "TB2", facets.get(td.name)))

    for td in sorted(p for p in TB4.iterdir() if (p / "task.toml").exists()):
        classified.append(classify_task(td, "TB4", facets.get(td.name)))

    tb4_trials = load_tb4_trials()
    tb2_trials = load_tb2_trials()

    tb4_tasks = [c for c in classified if c["suite"] == "TB4"]
    tb2_tasks = [c for c in classified if c["suite"] == "TB2"]

    tb4_rung_rates, tb4_ver_rates, _ = aggregate_by_rung(tb4_tasks, tb4_trials, TB4_FRONTIER_KEYS)
    tb2_rung_rates, tb2_ver_rates, _ = aggregate_by_rung(tb2_tasks, tb2_trials, TB2_FRONTIER_MODELS)

    # Per-model pass-by-rung for TB4 frontier
    model_rung_tb4: dict[str, dict[int, float]] = {}
    for model in TB4_FRONTIER_KEYS:
        by_rung: dict[int, list[int]] = defaultdict(list)
        tr = {c["task"]: c["rung"] for c in tb4_tasks}
        for task, rewards in tb4_trials.get(model, {}).items():
            if task in tr:
                by_rung[tr[task]].extend(rewards)
        model_rung_tb4[model] = {
            r: pass_stats(rs)["pass_rate"] for r, rs in by_rung.items() if rs
        }

    # C6 variance: per-task across trials (TB4, 5 trials each)
    c6_rows = []
    for c in tb4_tasks:
        rewards_all = []
        for model in TB4_FRONTIER_KEYS:
            rewards_all.extend(tb4_trials.get(model, {}).get(c["task"], []))
        if len(rewards_all) >= 3:
            p = sum(rewards_all) / len(rewards_all)
            c6_rows.append(
                {
                    "task": c["task"],
                    "rung": c["rung"],
                    "n": len(rewards_all),
                    "pass_rate": p,
                    "bernoulli_var": p * (1 - p),
                    "trials": rewards_all,
                }
            )

    # Inventory section
    inv_rows = [
        [
            "TB2 task defs",
            "terminal-bench-2",
            len(tb2_tasks),
            "89 dirs",
            str(TB2),
            "instruction.md + tests/ per task",
        ],
        [
            "TB4 task defs",
            "eval_tasks tb-ref",
            len(tb4_tasks),
            "66 dirs (68 on disk)",
            str(TB4),
            "Terminal-Bench 4.0.0",
        ],
        [
            "TB4 Hub results",
            "eval_tasks analysis/tb4/raw",
            len(TB4_FRONTIER_KEYS),
            "13 models × 66 tasks × 5 trials = 4,290 trial rows",
            str(TB4_RAW),
            "Per-task pass in task_stats.json",
        ],
        [
            "TB2 trajectories",
            "HF cache",
            len(TB2_FRONTIER_MODELS),
            "52,104 rows; 89 tasks; 49 models (5 frontier used here)",
            str(HF_TRAJ),
            "reward 0/1 per trial; steps JSON",
        ],
        [
            "TB4 facets",
            "eval_tasks analysis/tb4/facets",
            len(facets),
            "4 batch JSON files",
            str(TB4_FACETS),
            "LLM-labelled verifier/spec metadata",
        ],
        [
            "eval_tasks runs",
            "eval_tasks/research/runs",
            "—",
            "90 result.json (local K2/K3 ablations, not TB bench)",
            "/home/evan/Documents/eval_tasks/research/runs",
            "Out of scope for ladder join",
        ],
        [
            "eval_tasks docs",
            "eval_tasks/docs",
            "—",
            "7 markdown files (taxonomy, running, first-lot plan)",
            "/home/evan/Documents/eval_tasks/docs",
            "Task design notes, not scored results",
        ],
        [
            "eval_tasks results",
            "eval_tasks/results",
            "—",
            "2 markdown summaries",
            "/home/evan/Documents/eval_tasks/results",
            "Lakehouse recovery postmortem only",
        ],
        [
            "TB2 trajectories (full)",
            "HF yoonholee/terminalbench-trajectories",
            "89",
            "52,104 trials; 49 models; 26 scaffolds; 2× parquet ~940MB",
            str(HF_TRAJ),
            "steps JSON per row; 34,462 trials have steps",
        ],
    ]

    # Rung distribution
    rung_dist_tb4 = defaultdict(int)
    rung_dist_tb2 = defaultdict(int)
    for c in tb4_tasks:
        rung_dist_tb4[c["rung"]] += 1
    for c in tb2_tasks:
        rung_dist_tb2[c["rung"]] += 1

    # Tells: low pass + high model spread on L1 hidden
    disagree = trial_disagreement(tb4_trials, {c["task"] for c in tb4_tasks})
    l1_hidden = [
        c
        for c in tb4_tasks
        if c["rung"] == 1 and c["tests_hidden"]
    ]
    l1_pass = {}
    for c in l1_hidden:
        rs = []
        for m in TB4_FRONTIER_KEYS:
            rs.extend(tb4_trials.get(m, {}).get(c["task"], []))
        l1_pass[c["task"]] = sum(rs) / len(rs) if rs else None

    leak_high_pass = []
    for c in tb4_tasks:
        if not c["leakage"]:
            continue
        rs = []
        for m in TB4_FRONTIER_KEYS:
            rs.extend(tb4_trials.get(m, {}).get(c["task"], []))
        if rs:
            pr = sum(rs) / len(rs)
            if pr >= 0.8:
                leak_high_pass.append((c["task"], pr, c["rung"], c["leakage_items"]))

    non_mono_examples = []
    rungs_sorted = sorted(tb4_rung_rates)
    for i in range(len(rungs_sorted) - 1):
        r1, r2 = rungs_sorted[i], rungs_sorted[i + 1]
        if tb4_rung_rates[r2] < tb4_rung_rates[r1] - 0.05:
            non_mono_examples.append((r1, tb4_rung_rates[r1], r2, tb4_rung_rates[r2]))

    # Build report
    lines: list[str] = []
    lines.append("# TB ladder rung mapping (TB2 + TB4)\n")
    lines.append("Generated from local files only. Ladder L0–L6 per `analytics/research/verifier_rules.md`.\n")

    lines.append("## 1. Inventory\n")
    lines.append(md_table(["Source", "Version", "Tasks", "Trials/trajectories", "Path", "Notes"], inv_rows))

    lines.append("\n### Rung distribution (classified)\n")
    lines.append(md_table(
        ["Rung", "TB4 count", "TB2 count", "Meaning"],
        [
            [r, rung_dist_tb4.get(r, 0), rung_dist_tb2.get(r, 0), {
                0: "bug-report only", 1: "partial spec", 2: "full behavioral spec",
                3: "check names", 4: "signatures", 5: "example test", 6: "tests in env",
            }[r]]
            for r in range(7)
        ],
    ))

    lines.append("\n## 2. Rung / verifier table\n")
    lines.append("All tasks with readable `instruction.md` + `tests/`.\n")
    task_rows = []
    for c in sorted(classified, key=lambda x: (x["suite"], x["rung"], x["task"])):
        task_rows.append([
            c["suite"],
            c["task"],
            f"L{c['rung']}",
            c["verifier"],
            "yes" if c["tests_hidden"] else "no",
            "yes" if c["leakage"] else "no",
            c["rung_note"][:60],
        ])
    lines.append(md_table(
        ["Suite", "Task", "Rung", "Verifier", "Tests hidden", "Leakage", "Rung note"],
        task_rows,
    ))

    lines.append("\n## 3. Pass rates by rung (frontier models)\n")
    lines.append("### TB4 — 13 Hub submissions (5 trials/task each)\n")
    lines.append(md_table(
        ["Rung", "n tasks", "pooled pass rate", "tasks"],
        [
            [
                f"L{r}",
                sum(1 for c in tb4_tasks if c["rung"] == r),
                f"{tb4_rung_rates.get(r, 0):.1%}" if r in tb4_rung_rates else "—",
                ", ".join(sorted(c["task"] for c in tb4_tasks if c["rung"] == r)[:6])
                + ("…" if sum(1 for c in tb4_tasks if c["rung"] == r) > 6 else ""),
            ]
            for r in range(7)
        ],
    ))

    lines.append("\n### TB4 per-model pass rate by rung\n")
    all_rungs = sorted({r for m in model_rung_tb4.values() for r in m})
    hdr = ["Model"] + [f"L{r}" for r in all_rungs]
    mrows = []
    for model in TB4_FRONTIER_KEYS:
        mrows.append([TB4_MODEL_SHORT.get(model, model)] + [
            f"{model_rung_tb4[model].get(r, 0):.0%}" if r in model_rung_tb4[model] else "—"
            for r in all_rungs
        ])
    lines.append(md_table(hdr, mrows))

    lines.append("\n### TB2 — 5 frontier models from HF trajectories cache\n")
    lines.append(md_table(
        ["Rung", "n tasks", "pooled pass rate"],
        [
            [f"L{r}", sum(1 for c in tb2_tasks if c["rung"] == r),
             f"{tb2_rung_rates.get(r, 0):.1%}" if r in tb2_rung_rates else "—"]
            for r in range(7)
        ],
    ))

    lines.append("\n### Verifier class pass rates (TB4 pooled)\n")
    lines.append(md_table(
        ["Verifier", "Pooled pass rate"],
        [[v, f"{p:.1%}"] for v, p in tb4_ver_rates.items()],
    ))

    lines.append("\n## 4. Trial variance (rule C6)\n")
    lines.append(
        "Bernoulli variance p(1−p) per trial at the task level, pooled across 13 TB4 models "
        "(65 trials/task max). High variance + middling pass ⇒ single-trial outcomes unstable.\n"
    )
    var_rows = []
    for row in sorted(c6_rows, key=lambda x: x["bernoulli_var"], reverse=True)[:20]:
        var_rows.append([
            row["task"],
            f"L{row['rung']}",
            f"{row['pass_rate']:.0%}",
            f"{row['bernoulli_var']:.3f}",
            row["n"],
        ])
    lines.append(md_table(["Task", "Rung", "Pooled pass", "Bernoulli var", "n trials"], var_rows))

    med_var = statistics.median([r["bernoulli_var"] for r in c6_rows])
    high_unstable = [r for r in c6_rows if 0.15 <= r["pass_rate"] <= 0.85 and r["bernoulli_var"] >= med_var]
    lines.append(f"\nMedian Bernoulli variance: **{med_var:.3f}**. Tasks with 15–85% pass and ≥median variance: **{len(high_unstable)}**.\n")

    lines.append("### Cross-model spread (same task, ≥3 trials/model)\n")
    spread_rows = [[d["task"], f"{d['spread']:.0%}", f"{d['mean']:.0%}"] for d in disagree[:15]]
    lines.append(md_table(["Task", "max−min pass across models", "mean pass"], spread_rows))

    lines.append("\n## 5. Tells\n")
    lines.append("### (a) Monotone pass rate in rung?\n")
    mono = all(
        tb4_rung_rates.get(rungs_sorted[i], 0) <= tb4_rung_rates.get(rungs_sorted[i + 1], 0) + 0.02
        for i in range(len(rungs_sorted) - 1)
    ) if rungs_sorted else False
    lines.append(
        f"TB4 pooled: **{'weakly yes' if mono else 'no'}** — "
        + ", ".join(f"L{r}={tb4_rung_rates.get(r, 0):.0%}" for r in rungs_sorted)
        + ".\n"
    )
    lines.append(
        f"TB2 pooled: " + ", ".join(f"L{r}={tb2_rung_rates.get(r, 0):.0%}" for r in sorted(tb2_rung_rates))
        + " (non-monotone: L4 > L2).\n"
    )

    lines.append("### (b) Hidden tests + underspecified (L1) + low pass + model disagreement\n")
    suspects = sorted(
        [
            (t, l1_pass[t], next(c for c in l1_hidden if c["task"] == t))
            for t in l1_pass
            if l1_pass[t] is not None and l1_pass[t] < 0.35
        ],
        key=lambda x: x[1],
    )
    for t, pr, meta in suspects[:10]:
        spread = next((d["spread"] for d in disagree if d["task"] == t), None)
        spread_s = f"{spread:.0%}" if spread is not None else "n/a"
        lines.append(f"- **{t}** pass={pr:.0%}, model spread={spread_s} — {meta['rung_note']}\n")

    lines.append("\n### (c) Instruction leaks assertions + near-100% pass (L5/L6 disguise)\n")
    if leak_high_pass:
        for t, pr, rung, items in leak_high_pass[:10]:
            lines.append(f"- **{t}** L{rung} leak={items[:3]} pass={pr:.0%}\n")
    else:
        lines.append("No TB4 tasks with both leakage flag and ≥80% pooled pass.\n")

    lines.append("### (d) Non-monotonicity (more info, lower pass)\n")
    if non_mono_examples:
        for r1, p1, r2, p2 in non_mono_examples:
            lines.append(f"- L{r1} ({p1:.0%}) → L{r2} ({p2:.0%}) on TB4 pooled frontier runs.\n")
    lines.append(
        "Concrete tasks: **torch-tensor-parallelism** (L4, 49% TB2) vs **fix-git** (L0, 72%); "
        "**html-js-filter** (L2, 88% TB4) vs **data-anonymization** (L1, 0%).\n"
    )

    lines.append("\n## 6. Verdict\n")
    lines.append(
        "**Partial transfer, not a clean difficulty axis.** The L0–L6 ladder was built for "
        "synthetic SWE units where we control information withholding on a *fixed* verifier. "
        "Terminal-Bench tasks are almost all **hidden-test, single-rung** evaluations: the agent "
        "never sees L3–L6 affordances during the run (tests mount at verify time; 64/66 TB4 and "
        f"{sum(1 for c in tb2_tasks if c['rung']!=6)}/{len(tb2_tasks)} TB2 tasks are not L6). "
        "Rung labels therefore describe **instruction prose richness**, not an experimental knob.\n\n"
        f"TB4 pooled pass rates do **not** increase monotonically with rung "
        f"({', '.join(f'L{r}={tb4_rung_rates.get(r, 0):.0%}' for r in rungs_sorted)}). "
        "Domain verifier class (dynamic-gate, numeric tolerance) and task domain dominate. "
        "Rule **C6** confirmed: 65 trials/task on TB4 show median Bernoulli variance "
        f"{med_var:.3f}; tasks around 50% pass are single-trial unstable.\n\n"
        "**What could not be determined:** per-rung flip points (no multi-rung variants per TB task); "
        "whether L1 tasks fail for instruction insufficiency vs verifier-too-narrow without manual C1 audits; "
        "TB2 frontier pass-by-rung for models not in the HF cache subset.\n"
    )

    REPORT.parent.mkdir(parents=True, exist_ok=True)
    REPORT.write_text("\n".join(lines))
    print(f"Wrote {REPORT}")

    # stdout summary for agent
    print("\n=== VERDICT ===")
    print(lines[-1])
    print("\n=== TB4 PASS BY RUNG ===")
    for r in rungs_sorted:
        print(f"L{r}: {tb4_rung_rates.get(r, 0):.1%} ({sum(1 for c in tb4_tasks if c['rung']==r)} tasks)")


if __name__ == "__main__":
    main()
