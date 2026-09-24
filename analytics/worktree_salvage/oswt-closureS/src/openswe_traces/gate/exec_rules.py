"""Executed-tier rules: run the task's own tests/test.sh inside its own image.

One orchestrator produces every executed verdict so the suites run once:

- bare ×2  -> A8 (must FAIL) and A5 determinism
- gold ×2  -> A1 (must PASS), A5 determinism, A10 (verifier under denied network)
- cheat ×1 -> A3 (must FAIL) when the package ships a cheat.patch

Suites that honour a seed env var (``os.Getenv("HIDDEN_SEED")`` and friends in
the hidden tests) get the second run under a different seed — a verdict that
flips between seeds is a flake, and a flake is a gate FAIL.

Results cache in ``validation.json`` under ``gate_executed`` keyed by
(image id, tests sha256, tree sha256, scheme version): an unchanged package
replays for free, a changed one re-proves once.
"""

from __future__ import annotations

import re
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from openswe_traces.gate.context import GateContext
from openswe_traces.gate.core import (
    EXECUTED,
    Verdict,
    load_validation,
    record_verdicts,
    register,
    write_validation,
)

GATE_RUNS_BLOCK = "gate_executed"
GATE_RUNS_VERSION = 2  # v1 = preflight's bare/gold/cheat single runs
_SEED_ENV_RE = re.compile(r'(?:os\.Getenv|os\.environ(?:\.get)?|process\.env)\W+("|\')?([A-Z_]*SEED[A-Z_]*)')
ALT_SEED = "7301989"


def _seed_var(ctx: GateContext) -> str | None:
    """Env var the hidden suite reads for its random seed, if any."""
    blob = "\n".join(ctx.rules.hidden_files.values()) + "\n" + (ctx.rules.test_sh or "")
    m = _SEED_ENV_RE.search(blob)
    return m.group(2) if m else None


@dataclass(frozen=True)
class SuiteOutcome:
    kind: str  # bare | gold | cheat
    verdict: str  # pass | fail | infra
    evidence: str
    returncode: int
    seconds: float
    seed: str = ""


@dataclass
class GateRuns:
    task_dir: Path
    image: str
    image_id: str
    key: dict[str, str]
    language: str
    seed_var: str
    runs: dict[str, list[SuiteOutcome]] = field(default_factory=dict)
    cached: bool = False
    seconds: float = 0.0

    def verdicts(self) -> list[Verdict]:
        out: list[Verdict] = []
        bare = self.runs.get("bare", [])
        gold = self.runs.get("gold", [])
        cheat = self.runs.get("cheat", [])
        prov = "gate/exec"
        img = self.image or self.image_id or "image-build-failed"

        def v(rid: str, passed: bool, evidence: str, *, skipped: bool = False) -> Verdict:
            return Verdict(rid, passed, skipped, EXECUTED, evidence[:2000], _now(), prov)

        if not bare:
            out.append(v("A8", False, "no bare run executed", skipped=True))
        else:
            ok = all(r.verdict == "fail" for r in bare)
            out.append(
                v(
                    "A8",
                    ok,
                    f"bare excised tree in {img}: "
                    + ", ".join(f"{r.verdict}({r.evidence[:160]})" for r in bare),
                )
            )
        if not gold:
            out.append(v("A1", False, "no gold.patch in package", skipped=True))
        else:
            ok = all(r.verdict == "pass" for r in gold)
            out.append(
                v(
                    "A1",
                    ok,
                    f"gold restore in {img}: "
                    + ", ".join(f"{r.verdict}({r.evidence[:160]})" for r in gold),
                )
            )
        if not cheat:
            out.append(v("A3", False, "no cheat.patch in package", skipped=True))
        else:
            ok = all(r.verdict == "fail" for r in cheat)
            out.append(
                v(
                    "A3",
                    ok,
                    f"cheat patch in {img}: "
                    + ", ".join(f"{r.verdict}({r.evidence[:160]})" for r in cheat),
                )
            )
        # A5 determinism: every repeated run of the same suite must agree.
        if len(bare) >= 2 or len(gold) >= 2:
            flaky: list[str] = []
            for kind in ("bare", "gold"):
                rs = self.runs.get(kind, [])
                if len(rs) >= 2 and len({r.verdict for r in rs}) > 1:
                    flaky.append(
                        f"{kind}: " + ",".join(f"{r.verdict}@seed={r.seed or 'default'}" for r in rs)
                    )
            detail = f"seeds: {self.seed_var or 'none honored'}"
            if flaky:
                out.append(v("A5", False, f"nondeterministic suite ({detail}) — {'; '.join(flaky)}"))
            else:
                out.append(
                    v(
                        "A5",
                        True,
                        f"stable across repeats ({detail}; "
                        + "; ".join(
                            f"{k}={[r.verdict for r in rs]}" for k, rs in self.runs.items() if rs
                        )
                        + ")",
                    )
                )
        else:
            out.append(v("A5", False, "determinism needs 2 runs of a suite", skipped=True))
        # A10 executed: every suite run happens under docker --network=none; a
        # test that silently fetches would fail the gold run.
        if gold:
            ok = all(r.verdict == "pass" for r in gold)
            out.append(
                v(
                    "A10",
                    ok,
                    f"gold suite x{len(gold)} passed with docker --network=none in {img}",
                )
            )
        else:
            out.append(v("A10", False, "no gold run to prove no-network verifier", skipped=True))
        return out


def _now() -> str:
    from openswe_traces.gate.core import _now_iso

    return _now_iso()


def _to_outcome(row: dict[str, Any]) -> SuiteOutcome:
    return SuiteOutcome(
        kind=str(row.get("kind") or ""),
        verdict=str(row.get("verdict") or "infra"),
        evidence=str(row.get("evidence") or ""),
        returncode=int(row.get("returncode") or 1),
        seconds=float(row.get("seconds") or 0.0),
        seed=str(row.get("seed") or ""),
    )


def _run_outcome(oc: SuiteOutcome) -> dict[str, Any]:
    return {
        "kind": oc.kind,
        "verdict": oc.verdict,
        "evidence": oc.evidence[:600],
        "returncode": oc.returncode,
        "seconds": round(oc.seconds, 1),
        "seed": oc.seed,
    }


def _digests(td: Path) -> dict[str, str]:
    from openswe_traces.pipeline.preflight import _dir_digest

    return {
        "tests_sha256": _dir_digest(td / "tests"),
        "tree_sha256": _dir_digest(td / "environment" / "src"),
    }


def _usable_recorded_image(td: Path, digests: dict[str, str], runner: Any) -> str:
    """Image id from a previous proof whose content key matches — skip rebuild."""
    from openswe_traces.pipeline.preflight import _image_id

    data = load_validation(td)
    for block in (data.get(GATE_RUNS_BLOCK), data.get("preflight")):
        if not isinstance(block, dict):
            continue
        key = block.get("key") or {}
        if (
            key.get("tests_sha256") == digests["tests_sha256"]
            and key.get("tree_sha256") == digests["tree_sha256"]
        ):
            ident = str(key.get("image") or block.get("image") or "")
            if ident and _image_id(ident, runner):
                return ident
    return ""


def _find_patch(td: Path, name: str) -> Path | None:
    for cand in (td / "tests" / name, td / "patches" / name):
        if cand.is_file():
            return cand
    return None


def ensure_gate_runs(ctx: GateContext) -> GateRuns:
    """Cached or fresh: bare x2, gold x2 (second seed where honored), cheat x1."""
    from openswe_traces.pipeline.preflight import (
        _image_id,
        detect_language,
        run_suite,
    )
    from openswe_traces.pipeline_ext.hack_audit import default_docker_run

    def _build_image() -> str:
        from openswe_traces.pipeline_ext.hack_audit import task_image

        def run(argv: list, **kw: Any) -> Any:
            return runner(list(argv), timeout=kw.get("timeout"))

        return task_image(td, run=run) or ""

    td = ctx.task_dir
    runner = ctx.docker_run or default_docker_run
    t0 = time.monotonic()
    lang = ctx.language or detect_language(td)
    digests = _digests(td)

    data = load_validation(td)
    prev = data.get(GATE_RUNS_BLOCK)
    prev_key = (prev or {}).get("key") or {}
    image_id = ""
    image = ""

    cached = (
        isinstance(prev, dict)
        and prev.get("version") == GATE_RUNS_VERSION
        and prev_key.get("tests_sha256") == digests["tests_sha256"]
        and prev_key.get("tree_sha256") == digests["tree_sha256"]
        and not ctx.force
    )
    if cached:
        runs = {
            kind: [_to_outcome(r) for r in (prev.get("runs") or {}).get(kind) or []]
            for kind in ("bare", "gold", "cheat")
        }
        rep = GateRuns(
            task_dir=td,
            image=str(prev.get("image") or ""),
            image_id=str(prev_key.get("image") or ""),
            key=dict(prev_key),
            language=lang,
            seed_var=str(prev.get("seed_var") or ""),
            runs={k: v for k, v in runs.items() if v},
            cached=True,
            seconds=0.0,
        )
        return rep

    image = ctx.image or _usable_recorded_image(td, digests, runner) or _build_image()
    image_id = _image_id(image, runner) if image else ""
    key = {"image": image_id, **digests}
    rep = GateRuns(
        task_dir=td, image=image, image_id=image_id, key=key, language=lang,
        seed_var=_seed_var(ctx) or "",
    )
    if not image:
        rep.seconds = time.monotonic() - t0
        rep.runs["bare"] = [SuiteOutcome("bare", "infra", "task image build failed", 125, 0.0)]
        _persist(rep)
        return rep

    def suite(kind: str, patch: Path | None, seed: str = "") -> SuiteOutcome:
        env = {rep.seed_var: seed} if (seed and rep.seed_var) else {}
        oc = run_suite(
            image, td, patch=patch, kind=kind, language=lang, docker_run=runner, env=env
        )
        return SuiteOutcome(kind, oc.verdict, oc.evidence, oc.returncode, oc.seconds, seed)

    gold_p = _find_patch(td, "gold.patch")
    cheat_p = _find_patch(td, "cheat.patch")
    rep.runs["bare"] = [
        suite("bare", None),
        suite("bare", None, seed=ALT_SEED),
    ]
    if gold_p:
        rep.runs["gold"] = [
            suite("gold", gold_p),
            suite("gold", gold_p, seed=ALT_SEED),
        ]
    if cheat_p:
        rep.runs["cheat"] = [suite("cheat", cheat_p)]
    rep.seconds = time.monotonic() - t0
    _persist(rep)
    return rep


def _persist(rep: GateRuns) -> None:
    """Write the executed block + the executed verdicts through the registry."""
    data = load_validation(rep.task_dir)
    data[GATE_RUNS_BLOCK] = {
        "version": GATE_RUNS_VERSION,
        "key": rep.key,
        "image": rep.image,
        "language": rep.language,
        "seed_var": rep.seed_var,
        "seconds": round(rep.seconds, 1),
        "runs": {k: [_run_outcome(o) for o in v] for k, v in rep.runs.items()},
        "at": _now(),
    }
    write_validation(rep.task_dir, data)
    record_verdicts(rep.task_dir, rep.verdicts())
    # compat: keep the preflight block shape K's tooling reads
    data = load_validation(rep.task_dir)
    checks = data.setdefault("checks", {})
    if isinstance(checks, dict):
        bare = rep.runs.get("bare", [])
        gold = rep.runs.get("gold", [])
        cheat = rep.runs.get("cheat", [])
        if bare:
            checks["buggy_fails"] = all(r.verdict == "fail" for r in bare)
        if gold:
            checks["gold_restore"] = checks["gold_pass"] = all(
                r.verdict == "pass" for r in gold
            )
        if cheat:
            checks["cheat_rejected"] = all(r.verdict == "fail" for r in cheat)
        first = lambda rs: rs[0] if rs else None
        data["preflight"] = {
            "key": rep.key,
            "verdict": "pass"
            if _all_expected(rep)
            else "fail",
            "image": rep.image,
            "cached": rep.cached,
            "seconds": round(rep.seconds, 1),
            "checks": {
                "bare": (first(rep.runs.get("bare", [])) or SuiteOutcome("bare", "absent", "", 0, 0)).verdict,
                "gold": (first(rep.runs.get("gold", [])) or SuiteOutcome("gold", "absent", "", 0, 0)).verdict,
                "cheat": (first(rep.runs.get("cheat", [])) or SuiteOutcome("cheat", "absent", "", 0, 0)).verdict,
            },
            "evidence": {
                k: (rs[0].evidence if rs else "") for k, rs in rep.runs.items()
            },
            "at": _now(),
        }
        write_validation(rep.task_dir, data)


def _all_expected(rep: GateRuns) -> bool:
    bare = rep.runs.get("bare", [])
    if not bare or any(r.verdict != "fail" for r in bare):
        return False
    gold = rep.runs.get("gold", [])
    if gold and any(r.verdict != "pass" for r in gold):
        return False
    cheat = rep.runs.get("cheat", [])
    return not (cheat and any(r.verdict != "fail" for r in cheat))


# --- registered executed impls --------------------------------------------------


def _runs(ctx: GateContext) -> GateRuns:
    if ctx._runs is None:
        rep = ensure_gate_runs(ctx)
        if rep.cached:
            # cached replay never persisted this session's verdicts; re-record
            # so a documentary overwrite in between is repaired.
            record_verdicts(ctx.task_dir, rep.verdicts())
        ctx._runs = rep
    return ctx._runs


def _pick(rep: GateRuns, rule_id: str) -> Verdict:
    for v in rep.verdicts():
        if v.rule_id == rule_id:
            return v
    return Verdict(rule_id, False, True, EXECUTED, "impl produced no verdict", _now(), "gate/exec")


@register("A8", EXECUTED, provenance="gate/exec", description="bare excised tree fails in-image, x2")
def _a8(ctx: GateContext) -> Verdict:
    return _pick(_runs(ctx), "A8")


@register("A1", EXECUTED, provenance="gate/exec", description="gold patch passes in-image, x2")
def _a1(ctx: GateContext) -> Verdict:
    return _pick(_runs(ctx), "A1")


@register("A3", EXECUTED, provenance="gate/exec", description="cheat patch fails in-image")
def _a3(ctx: GateContext) -> Verdict:
    return _pick(_runs(ctx), "A3")


@register("A5", EXECUTED, provenance="gate/exec", description="suite outcome stable across 2 runs/seeds")
def _a5(ctx: GateContext) -> Verdict:
    return _pick(_runs(ctx), "A5")


@register("A10", EXECUTED, provenance="gate/exec", description="gold passes under docker --network=none")
def _a10(ctx: GateContext) -> Verdict:
    return _pick(_runs(ctx), "A10")
