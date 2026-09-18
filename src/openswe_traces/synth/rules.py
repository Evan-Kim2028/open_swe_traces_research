"""Per-rule verdicts for synthetic Harbor tasks (verifier_rules.md A1–A10, B1–B8).

Each rule is a function ``check_<id>(ctx) -> RuleVerdict``. Live task builders
write ``<task_dir>/validation.json`` with a ``rule_verdicts`` list. Existing
task dirs are backfilled from artifacts (patches, RESULT.md / ITER_*.md tables,
verifier logs, task.toml / test.sh / instruction.md) without re-running Harbor.
"""

from __future__ import annotations

import argparse
import csv
import json
import re
from collections.abc import Callable, Mapping, Sequence
from dataclasses import asdict, dataclass, field
from glob import glob as glob_paths
from pathlib import Path
from typing import Any

from openswe_traces.data import ROOT
from openswe_traces.synth.harbor_tasks import instruction_self_check
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE

RULE_IDS: tuple[str, ...] = (
    tuple(f"A{i}" for i in range(1, 13))
    + tuple(f"B{i}" for i in range(1, 9))
    + ("C5",)
)

_TEST_FUNC_RE = re.compile(r"^func\s+(?:\(.*?\)\s+)?(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)
_RUN_RE = re.compile(r"-run\s+'?\^?\(?([A-Za-z0-9_|]+)")
_REWARD_RE = re.compile(r"REWARD=(\S+)")
_SLUG_RE = re.compile(r"`([A-Za-z0-9_.-]+)`")
_WHITEBOX_TOKENS = (
    "ThornSlot",
    "NimbusCore",
    "codecV2",
    "*codecV2",
    "memCodec",
    "PebbleUnit",
    "IvoryLink",
    "RidgeWire",
    "suite.codec.",
)
_OK_LOWER_CALLS = {
    "fatalf",
    "errorf",
    "logf",
    "helper",
    "run",
    "parallel",
    "skip",
    "fatal",
    "error",
    "log",
    "skipf",
    "skipnow",
    "failnow",
    "fail",
    "cleanup",
    "tempdir",
    "setenv",
    "deadline",
    "name",
}
_FAMILY_KEYS = {
    "spec-reimpl",
    "property-backoff",
    "dynamic-pipeline",
    "spec-reimpl-bb",
    "spec-bb-chain",
    "spec-bb-bucket",
    "property-policy",
    "property-1pc",
    "dynamic-snapshot",
    "dynamic-latch",
}


@dataclass(frozen=True)
class RuleVerdict:
    rule_id: str
    passed: bool
    evidence: str
    skipped: bool = False


def _v(rule_id: str, passed: bool, evidence: str, *, skipped: bool = False) -> RuleVerdict:
    return RuleVerdict(rule_id=rule_id, passed=passed, evidence=evidence[:2000], skipped=skipped)


def _skip(rule_id: str, reason: str) -> RuleVerdict:
    return _v(rule_id, False, reason, skipped=True)


@dataclass
class RuleContext:
    """Read-only view of one Harbor task dir plus nearby notes."""

    task_dir: Path
    extra: dict[str, Any] = field(default_factory=dict)
    instruction: str = ""
    task_toml: str = ""
    test_sh: str = ""
    dockerfile: str = ""
    patches: dict[str, Path] = field(default_factory=dict)
    hidden_files: dict[str, str] = field(default_factory=dict)
    obfuscate: dict[str, Any] = field(default_factory=dict)
    affordance: dict[str, Any] = field(default_factory=dict)
    validation: dict[str, Any] = field(default_factory=dict)
    family_checks: list[dict[str, Any]] = field(default_factory=list)
    md_checks: dict[str, str] = field(default_factory=dict)
    verifier_hits: list[str] = field(default_factory=list)
    src_test_names: set[str] = field(default_factory=set)


RuleFn = Callable[[RuleContext], RuleVerdict]
RULES: dict[str, RuleFn] = {}


def rule(rule_id: str) -> Callable[[RuleFn], RuleFn]:
    def deco(fn: RuleFn) -> RuleFn:
        RULES[rule_id] = fn
        return fn

    return deco


def is_harbor_task(path: Path) -> bool:
    path = Path(path)
    if not path.is_dir() or path.name.startswith("_"):
        return False
    return (path / "instruction.md").is_file() and (path / "task.toml").is_file()


def _read(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return ""


def _load_json(path: Path) -> dict[str, Any]:
    if not path.is_file():
        return {}
    try:
        data = json.loads(_read(path) or "{}")
    except json.JSONDecodeError:
        return {}
    return data if isinstance(data, dict) else {}


def _looks_like_task_slug(s: str) -> bool:
    if not s or "/" in s or s.startswith("Test") or len(s) < 3:
        return False
    return bool(re.fullmatch(r"[A-Za-z0-9_.-]+", s))


def _normalize_slug(name: str) -> str:
    s = name.strip().strip("`").lower().replace("_", "-")
    s = re.sub(r"-a-?\d+$", "", s)
    s = re.sub(r"-l\d+$", "", s)
    s = re.sub(r"-obf$", "", s)
    return s


def _slug_keys(name: str) -> set[str]:
    keys = {_normalize_slug(name), name.lower()}
    rest = _normalize_slug(name)
    for prefix in ("client-go-", "dailycodingproblem-go-"):
        rest = rest.removeprefix(prefix)
    keys.add(rest)
    keys.add(rest.replace("-", ""))
    return {k for k in keys if k}


def _family_key(name: str) -> str | None:
    base = re.sub(r"-A-?\d+$", "", name)
    base = re.sub(r"-L\d+$", "", base)
    base = re.sub(r"-obf$", "", base)
    return base if base in _FAMILY_KEYS else None


def _toml_section(text: str, header: str) -> str:
    m = re.search(rf"^\[{re.escape(header)}\]\s*$", text, re.MULTILINE)
    if not m:
        return ""
    rest = text[m.end() :]
    nxt = re.search(r"^\[", rest, re.MULTILINE)
    return rest[: nxt.start()] if nxt else rest


def _f2p_from_test_sh(test_sh: str) -> list[str]:
    m = _RUN_RE.search(test_sh or "")
    if not m:
        return []
    return [p for p in m.group(1).split("|") if p.startswith("Test")]


def _find_notes_root(task_dir: Path) -> Path | None:
    for parent in [task_dir, *task_dir.parents]:
        if (parent / "RESULT.md").is_file() or any(parent.glob("ITER_*.md")):
            return parent
    return None


def _store_md(index: dict[str, dict[str, str]], slug: str, check: str, result: str) -> None:
    check = check.strip().lower()
    result = result.strip()
    if not slug or not check or not result:
        return
    for key in _slug_keys(slug):
        index.setdefault(key, {})[check] = result


def _parse_pipe_row(line: str) -> list[str]:
    if not line.strip().startswith("|"):
        return []
    cells = [c.strip() for c in line.strip().strip("|").split("|")]
    if cells and set(cells[0].replace(":", "")) <= {"-"}:
        return []
    return cells


def _skip_table_sep(lines: list[str], i: int) -> int:
    if i < len(lines) and re.match(r"^\|[\s:|-]+\|$", lines[i].strip()):
        return i + 1
    return i


def parse_harbor_notes(root: Path) -> dict[str, dict[str, str]]:
    """slug → {check name: result text} from RESULT.md, ITER_*.md, verifier/table.csv."""
    index: dict[str, dict[str, str]] = {}
    md_files = [root / "RESULT.md", *sorted(root.glob("ITER_*.md"))]
    md_files.extend(sorted(root.glob("FAILURE_AUDIT*.md")))
    md_files.extend(sorted(root.glob("BLACKBOX.md")))
    md_files.extend(sorted(root.glob("UNSOLVABLE.md")))
    for path in md_files:
        if not path.is_file():
            continue
        current = ""
        lines = _read(path).splitlines()
        i = 0
        while i < len(lines):
            line = lines[i]
            if line.startswith("#"):
                m = _SLUG_RE.search(line)
                if m and _looks_like_task_slug(m.group(1)):
                    current = m.group(1)
            cells = _parse_pipe_row(line)
            header_l = [c.lower() for c in cells]
            if cells and header_l[:2] == ["check", "result"] and current:
                i = _skip_table_sep(lines, i + 1)
                while i < len(lines):
                    row = _parse_pipe_row(lines[i])
                    if len(row) < 2:
                        break
                    _store_md(index, current, row[0], row[1])
                    i += 1
                continue
            if cells and header_l[:1] == ["bug"] and "f2p + alt" in header_l:
                i = _skip_table_sep(lines, i + 1)
                while i < len(lines):
                    row = _parse_pipe_row(lines[i])
                    if len(row) < 5:
                        break
                    slug = row[0]
                    _store_md(index, slug, "alt accepted", row[2])
                    _store_md(index, slug, "cheat rejected", row[4])
                    i += 1
                continue
            if cells and header_l[:2] == ["task", "buggy"] and "fixed" in header_l:
                i = _skip_table_sep(lines, i + 1)
                while i < len(lines):
                    row = _parse_pipe_row(lines[i])
                    if len(row) < 3:
                        break
                    _store_md(index, row[0], "docker buggy", row[1])
                    _store_md(index, row[0], "docker gold", row[2])
                    i += 1
                continue
            i += 1
    table = root / "verifier" / "table.csv"
    if table.is_file():
        try:
            with table.open(encoding="utf-8", newline="") as fh:
                for row in csv.DictReader(fh):
                    slug = (row.get("symbol") or "").strip()
                    if not slug:
                        continue
                    _store_md(index, slug, "alt accepted", row.get("alt") or "")
                    _store_md(index, slug, "cheat rejected", row.get("cheat") or "")
        except OSError:
            pass
    return index


_NOTES_CACHE: dict[Path, dict[str, dict[str, str]]] = {}


def _notes_for(task_dir: Path) -> dict[str, str]:
    root = _find_notes_root(task_dir)
    if root is None:
        return {}
    cached = _NOTES_CACHE.get(root)
    if cached is None:
        cached = parse_harbor_notes(root)
        _NOTES_CACHE[root] = cached
    merged: dict[str, str] = {}
    for key in _slug_keys(task_dir.name):
        merged.update(cached.get(key) or {})
    fam = _family_key(task_dir.name)
    if fam:
        merged.update(cached.get(_normalize_slug(fam)) or {})
    return merged


def _load_family_checks(task_dir: Path) -> list[dict[str, Any]]:
    parent = task_dir.parent
    fam = _family_key(task_dir.name)
    out: list[dict[str, Any]] = []
    val = _load_json(parent / "validation.json")
    if fam and fam != "spec-reimpl-bb":
        families = val.get("families") if isinstance(val.get("families"), dict) else {}
        slot = families.get(fam) if isinstance(families, dict) else None
        if isinstance(slot, dict):
            checks = slot.get("checks") or []
            if isinstance(checks, list):
                out.extend(c for c in checks if isinstance(c, dict))
    if fam == "spec-reimpl-bb":
        bb = _load_json(parent / "spec-reimpl-bb-validation.json")
        checks = bb.get("checks") or []
        if isinstance(checks, list):
            out.extend(c for c in checks if isinstance(c, dict))
    return out


def _collect_patches(task_dir: Path) -> dict[str, Path]:
    found: dict[str, Path] = {}
    for folder in (task_dir / "patches", task_dir / "tests"):
        if not folder.is_dir():
            continue
        for path in folder.glob("*.patch"):
            stem = path.stem.lower()
            found[stem] = path
    return found


def _collect_hidden(task_dir: Path) -> dict[str, str]:
    hidden = task_dir / "tests" / "hidden"
    out: dict[str, str] = {}
    if not hidden.is_dir():
        return out
    for path in hidden.rglob("*.go"):
        out[path.relative_to(hidden).as_posix()] = _read(path)
    return out


def _src_test_names(task_dir: Path) -> set[str]:
    src = task_dir / "environment" / "src"
    names: set[str] = set()
    if not src.is_dir():
        return names
    for path in src.rglob("*_test.go"):
        names.update(_TEST_FUNC_RE.findall(_read(path)))
    return names


def _verifier_hits(task_dir: Path) -> list[str]:
    root = _find_notes_root(task_dir)
    if root is None:
        return []
    vdir = root / "verifier"
    if not vdir.is_dir():
        return []
    tokens = [t for t in _slug_keys(task_dir.name) if len(t) >= 4]
    hits: list[str] = []
    try:
        for path in vdir.iterdir():
            if not path.is_file():
                continue
            low = path.name.lower()
            if any(tok in low for tok in tokens):
                hits.append(_read(path)[-4000:])
    except OSError:
        return []
    return hits


def load_context(task_dir: Path, extra: Mapping[str, Any] | None = None) -> RuleContext:
    task_dir = Path(task_dir)
    extra_d = dict(extra or {})
    existing = _load_json(task_dir / "validation.json")
    # Prefer caller payload; keep on-disk fields that the payload does not override.
    merged = dict(existing)
    merged.update(extra_d)
    ctx = RuleContext(task_dir=task_dir, extra=merged)
    ctx.instruction = _read(task_dir / "instruction.md")
    ctx.task_toml = _read(task_dir / "task.toml")
    ctx.test_sh = _read(task_dir / "tests" / "test.sh")
    ctx.dockerfile = _read(task_dir / "environment" / "Dockerfile")
    ctx.patches = _collect_patches(task_dir)
    ctx.hidden_files = _collect_hidden(task_dir)
    ctx.obfuscate = _load_json(task_dir / "obfuscate_result.json")
    ctx.affordance = _load_json(task_dir / "affordance.json")
    ctx.validation = existing
    ctx.family_checks = _load_family_checks(task_dir)
    ctx.md_checks = _notes_for(task_dir)
    ctx.verifier_hits = _verifier_hits(task_dir)
    ctx.src_test_names = _src_test_names(task_dir)
    return ctx


def _check_named(ctx: RuleContext, *aliases: str) -> tuple[bool, str] | None:
    want = {a.lower() for a in aliases}
    for row in ctx.family_checks:
        name = str(row.get("check") or "").lower()
        if name in want:
            ok = bool(row.get("ok"))
            return ok, f"{name} ok={ok}: {str(row.get('detail') or '')[:400]}"
    extra_checks = ctx.extra.get("checks")
    if isinstance(extra_checks, dict):
        for alias in aliases:
            if alias in extra_checks:
                ok = bool(extra_checks[alias])
                return ok, f"payload.checks.{alias}={ok}"
    if isinstance(extra_checks, list):
        for row in extra_checks:
            if not isinstance(row, dict):
                continue
            name = str(row.get("check") or "").lower()
            if name in want:
                ok = bool(row.get("ok"))
                return ok, f"{name} ok={ok}"
    for key, result in ctx.md_checks.items():
        if key in want:
            interpreted = _interpret_md(key, result)
            if interpreted is None:
                continue
            return interpreted, f"notes[{key}]={result[:300]}"
    return None


def _interpret_md(check: str, result: str) -> bool | None:
    r = result.lower()
    c = check.lower()
    if any(tok in r for tok in ("n/a", "infeasible", "not shipped", "never reached")):
        return None
    if "cheat" in c:
        if "reject" in r or ("fail" in r and "accept" not in r):
            return True
        if "accept" in r or "pass" in r:
            return False
        return None
    if "single-site" in c or c.endswith("fails"):
        if "fail" in r or "reject" in r:
            return True
        if "pass" in r or "accept" in r:
            return False
        return None
    if "buggy" in c:
        return "fail" in r or "reward=0" in r
    if "gold" in c or "alt accepted" in c or "gold revert" in c:
        return "pass" in r or "accept" in r or "yes" in r
    if "not flaky" in c or "3×" in c or "3x" in c:
        return "fail" in r or "pass" in r or "×3" in r or "x3" in r
    if "no collateral" in c or "rest_suite" in c:
        return "green" in r or "except" in r or "only" in r or "pass" in r or "ok" in r
    if re.search(r"\b(voided|hung test|rejected the task)\b", r):
        return False
    if r.strip():
        return True
    return None


def _obf_reward(ctx: RuleContext, key: str) -> str | None:
    val = ctx.obfuscate.get("validation")
    if not isinstance(val, dict):
        val = ctx.extra.get("validation") if isinstance(ctx.extra.get("validation"), dict) else None
    if not isinstance(val, dict):
        return None
    row = val.get(key)
    if isinstance(row, dict) and row.get("reward") is not None:
        return str(row["reward"])
    return None


def _docker_pair(ctx: RuleContext) -> tuple[bool, bool] | None:
    """(buggy_fails, gold_passes) if known."""
    br, gr = _obf_reward(ctx, "buggy"), _obf_reward(ctx, "gold")
    if br is not None and gr is not None:
        return br != "1", gr == "1"
    buggy_md = ctx.md_checks.get("docker buggy")
    gold_md = ctx.md_checks.get("docker gold")
    if buggy_md and gold_md:
        b_fail = "fail" in buggy_md.lower()
        g_pass = "pass" in gold_md.lower()
        return b_fail, g_pass
    docker = ctx.md_checks.get("docker") or ""
    if docker:
        low = docker.lower()
        if "buggy" in low and "gold" in low:
            return ("fail" in low or "reward=0" in low), ("pass" in low or "reward=1" in low)
    blob = "\n".join(ctx.verifier_hits)
    rewards = _REWARD_RE.findall(blob)
    if len(rewards) >= 2:
        return rewards[0] != "1", rewards[1] == "1"
    b = _check_named(ctx, "buggy_fails", "docker_a0_buggy_fails", "docker_buggy_fails")
    g = _check_named(ctx, "gold_pass", "gold_restore", "docker_a0_gold_pass", "docker_gold_pass")
    if b and g:
        return b[0], g[0]
    return None


def _self_check(ctx: RuleContext) -> dict[str, Any]:
    cached = ctx.extra.get("instruction_self_check")
    if isinstance(cached, dict) and cached:
        return cached
    aff = ctx.affordance.get("instruction_self_check")
    if isinstance(aff, dict) and aff:
        return aff
    f2p = _f2p_from_test_sh(ctx.test_sh)
    site = ctx.extra.get("site") if isinstance(ctx.extra.get("site"), dict) else {}
    symbols = [str(s) for s in (ctx.extra.get("changed_symbols") or []) if s]
    if site.get("name"):
        symbols.append(str(site["name"]))
    files = [str(f) for f in (ctx.extra.get("changed_files") or []) if f]
    if site.get("file_path"):
        files.append(str(site["file_path"]))
    names_in = any(n in ctx.instruction for n in f2p if n)
    locality = 0 if names_in else 2
    return instruction_self_check(
        ctx.instruction,
        test_sh=ctx.test_sh or "go test ./...",
        f2p_tests=f2p,
        changed_symbols=symbols,
        changed_files=files,
        locality=locality,
    )


# --- validity ---------------------------------------------------------------


@rule("A1")
def check_a1(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(
        ctx, "gold_restore", "gold_pass", "gold revert", "docker_a0_gold_pass"
    )
    if hit is not None:
        return _v("A1", hit[0], hit[1])
    gold_r = _obf_reward(ctx, "gold")
    if gold_r is not None:
        return _v("A1", gold_r == "1", f"gold reward={gold_r}")
    gold_md = ctx.md_checks.get("docker gold")
    if gold_md:
        ok = "pass" in gold_md.lower()
        return _v("A1", ok, f"docker gold: {gold_md[:300]}")
    pair = _docker_pair(ctx)
    if pair is not None:
        return _v("A1", pair[1], f"docker pair gold_passes={pair[1]}")
    if "gold" not in ctx.patches and "gold.patch" not in {p.name for p in ctx.patches.values()}:
        return _skip("A1", "no gold.patch or gold-restore proof in artifacts")
    return _skip("A1", "gold.patch present but no pass/fail evidence")


@rule("A2")
def check_a2(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(ctx, "two_func_alt_accepted", "alt accepted", "alt_accepted")
    if hit is not None:
        return _v("A2", hit[0], hit[1])
    alt_r = _obf_reward(ctx, "alt")
    notes = ctx.md_checks.get("alt accepted") or ""
    if "n/a" in notes.lower() or "infeasible" in notes.lower():
        return _skip("A2", notes[:400] or "alt N/A at this size")
    if alt_r is not None:
        expected_fail = bool(ctx.extra.get("alt_expected_pass") is False)
        if expected_fail:
            return _skip("A2", f"alt expected not to pass (reward={alt_r})")
        return _v("A2", alt_r == "1", f"alt reward={alt_r}")
    if "alt" not in ctx.patches:
        return _skip("A2", "no alt.patch or alt-accept proof in artifacts")
    return _skip("A2", "alt.patch present but no pass/fail evidence")


@rule("A3")
def check_a3(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(ctx, "cheat_rejected", "cheat rejected", "cheat")
    if hit is not None:
        return _v("A3", hit[0], hit[1])
    cheat_r = _obf_reward(ctx, "cheat")
    if cheat_r is not None:
        return _v("A3", cheat_r != "1", f"cheat reward={cheat_r} (must not be 1)")
    if "cheat" not in ctx.patches:
        return _skip("A3", "no cheat.patch or cheat-reject proof in artifacts")
    return _skip("A3", "cheat.patch present but no pass/fail evidence")


@rule("A4")
def check_a4(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(ctx, "f2p files in impact", "f2p_in_impact", "impact")
    if hit is not None:
        return _v("A4", hit[0], hit[1])
    impact = ctx.extra.get("impact_files") or ctx.validation.get("impact_files")
    f2p = _f2p_from_test_sh(ctx.test_sh)
    if isinstance(impact, list) and impact and f2p:
        test_paths = [x for x in impact if str(x).endswith("_test.go")]
        if test_paths:
            return _v("A4", True, f"impact test files: {test_paths}")
        return _skip("A4", f"impact_files={impact[:8]} do not include test paths; f2p={f2p}")
    return _skip("A4", "no impact-set evidence in notes or validation.json")


@rule("A5")
def check_a5(ctx: RuleContext) -> RuleVerdict:
    coll = _check_named(ctx, "no collateral", "rest_suite_green", "collateral")
    flake = _check_named(ctx, "3× not flaky", "3x not flaky", "not flaky", "f2p_stable")
    if coll is None and flake is None:
        return _skip("A5", "no collateral/flake evidence in artifacts")
    ok = True
    bits: list[str] = []
    if coll is not None:
        ok = ok and coll[0]
        bits.append(coll[1])
    if flake is not None:
        ok = ok and flake[0]
        bits.append(flake[1])
    return _v("A5", ok, "; ".join(bits))


@rule("A6")
def check_a6(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(ctx, "single-site-fix-fails", "single_site_fails", "site_fix_fails")
    if hit is not None:
        return _v("A6", hit[0], hit[1])
    site_patches = [k for k in ctx.patches if k.startswith("site_")]
    two = ctx.extra.get("two_site") or ctx.validation.get("two_site")
    sites_req = ctx.extra.get("sites_requested") or ctx.validation.get("sites_requested")
    multi = bool(site_patches) or bool(two) or (isinstance(sites_req, int) and sites_req >= 2)
    if not multi:
        return _skip("A6", "not a multi-site bug (no site_* patches or two_site design)")
    return _skip("A6", f"multi-site artifacts {site_patches or two} but no single-site-fail proof")


@rule("A7")
def check_a7(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(ctx, "decoy", "decoys", "wrong fix breaks guard")
    if hit is not None:
        return _v("A7", hit[0], hit[1])
    decoys = ctx.extra.get("decoys") or ctx.validation.get("decoys")
    decoy_patches = [k for k in ctx.patches if "decoy" in k]
    if not decoys and not decoy_patches:
        return _skip("A7", "no decoys on this task")
    if isinstance(decoys, list) and decoys:
        guards = []
        for d in decoys:
            if isinstance(d, dict):
                guards.extend(d.get("guard_tests") or [])
        if guards:
            return _v("A7", True, f"decoys with guard_tests={guards[:8]}")
        return _v("A7", True, "decoys listed in design (unmodified by construction)")
    return _skip("A7", "decoy artifacts present but no guard-test proof")


@rule("A8")
def check_a8(ctx: RuleContext) -> RuleVerdict:
    pair = _docker_pair(ctx)
    if pair is not None:
        ok = pair[0] and pair[1]
        return _v("A8", ok, f"image proof buggy_fails={pair[0]} gold_passes={pair[1]}")
    hit = _check_named(ctx, "docker", "builds")
    if hit is not None:
        return _v("A8", hit[0], hit[1])
    return _skip("A8", "no built-image buggy-fail/gold-pass proof")


@rule("A9")
def check_a9(ctx: RuleContext) -> RuleVerdict:
    sh = ctx.test_sh
    has_gate = bool(sh) and (
        " -race" in sh or "-race " in sh or "ns/op" in sh or "-bench" in sh or "perf gate" in sh
    )
    hit = _check_named(
        ctx,
        "naive_fails_throughput_gate",
        "naive_constant_fails",
        "dynamic_gate",
        "race 10/10",
    )
    if hit is not None:
        return _v("A9", hit[0], hit[1])
    if not has_gate:
        return _skip("A9", "no race/perf/property gate in test.sh")
    naive = ctx.extra.get("f3_naive_ns_op")
    limit = ctx.extra.get("f3_perf_limit_ns")
    gold = ctx.extra.get("f3_gold_ns_op")
    if isinstance(naive, (int, float)) and isinstance(limit, (int, float)):
        ok = naive > limit
        return _v("A9", ok, f"naive_ns={naive} limit={limit} gold_ns={gold}")
    return _skip("A9", "dynamic gate in test.sh but no proven non-passer")


@rule("A10")
def check_a10(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(ctx, "verifier_no_network")
    if hit is not None:
        return _v("A10", hit[0], hit[1])
    section = _toml_section(ctx.task_toml, "verifier")
    if 'network_mode = "no-network"' in section:
        return _v("A10", True, "task.toml [verifier] network_mode=no-network")
    if 'network_mode = "no-network"' in ctx.task_toml:
        # present but maybe under another section
        if 'network_mode = "no-network"' in section or not section:
            return _v("A10", True, "task.toml has verifier no-network")
        return _v("A10", False, "[verifier] section lacks no-network (found elsewhere)")
    return _v("A10", False, "[verifier] network_mode is not no-network")


@rule("A11")
def check_a11(ctx: RuleContext) -> RuleVerdict:
    """Timing gates: idle host, recorded load, margin; measure_gold.sh present."""
    hit = _check_named(ctx, "perf_load_recorded", "a11_load", "gold_ns_op_load")
    sh = ctx.test_sh or ""
    has_perf = bool(sh) and ("ns/op" in sh or "-bench" in sh or "perf gate" in sh)
    measure = (ctx.task_dir / "tests" / "measure_gold.sh").is_file()
    load = ctx.extra.get("perf_load_avg")
    if load is None:
        load = ctx.extra.get("load_avg")
    gold_ns = ctx.extra.get("gold_ns_op") or ctx.extra.get("f3_gold_ns_op")
    if not has_perf:
        return _skip("A11", "no timing gate in test.sh")
    if hit is not None:
        ok = hit[0] and measure
        return _v("A11", ok, hit[1] + f"; measure_gold.sh={measure}")
    if gold_ns is None:
        return _skip("A11", "perf gate present but gold ns/op not recorded")
    try:
        load_f = float(load) if load is not None else None
    except (TypeError, ValueError):
        load_f = None
    idle = load_f is not None and load_f < 2.0
    ok = idle and measure
    return _v(
        "A11",
        ok,
        f"gold_ns={gold_ns} load_avg={load} idle={idle} measure_gold.sh={measure}",
    )


_TEST_PATCH_RE = re.compile(r"^diff --git a/(.+\.go) b/", re.MULTILINE)


@rule("A12")
def check_a12(ctx: RuleContext) -> RuleVerdict:
    """gold/alt/cheat patches must not touch checksum-guarded test files."""
    hit = _check_named(ctx, "patches_skip_tests", "a12", "gold_skips_tests")
    if hit is not None:
        return _v("A12", hit[0], hit[1])
    if not ctx.patches:
        return _skip("A12", "no gold/alt/cheat patches")
    bad: list[str] = []
    for stem, path in ctx.patches.items():
        if stem not in {"gold", "alt", "cheat"} and not stem.endswith(
            ("gold", "alt", "cheat")
        ):
            continue
        text = _read(path)
        for m in _TEST_PATCH_RE.finditer(text):
            rel = m.group(1)
            if rel.endswith("_test.go") or "/tests/" in rel or rel.startswith("tests/"):
                bad.append(f"{path.name}:{rel}")
    if not any(k in {"gold", "alt", "cheat"} or k.endswith(("gold", "alt", "cheat")) for k in ctx.patches):
        return _skip("A12", "no gold/alt/cheat patches")
    return _v(
        "A12",
        not bad,
        "no test-file hunks in gold/alt/cheat" if not bad else f"test hunks: {bad[:8]}",
    )


@rule("C5")
def check_c5(ctx: RuleContext) -> RuleVerdict:
    """A gold-failing proof is a harness failure until the toolchain is validated."""
    hit = _check_named(ctx, "harness_ok", "proof_harness", "c5")
    if hit is not None:
        return _v("C5", hit[0], hit[1])
    extra_ok = ctx.extra.get("harness_ok")
    if extra_ok is not None:
        return _v("C5", bool(extra_ok), f"harness_ok={extra_ok}")
    pair = _docker_pair(ctx)
    if pair is not None and not pair[1]:
        return _v(
            "C5",
            False,
            "gold reported failing in image proof; treat as harness until go-on-PATH confirmed",
        )
    if pair is not None and pair[1]:
        return _v("C5", True, "gold passed in image proof (harness exit codes trusted)")
    return _skip("C5", "no proof-harness evidence")


# --- fairness ---------------------------------------------------------------


@rule("B1")
def check_b1(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(ctx, "checksum_guard")
    if hit is not None:
        return _v("B1", hit[0], hit[1])
    if "sha256sum -c" in (ctx.test_sh or ""):
        return _v("B1", True, "tests/test.sh checksum-guards files")
    return _v("B1", False, "tests/test.sh has no sha256sum checksum guard")


@rule("B2")
def check_b2(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(ctx, "agent_allowlist")
    if hit is not None:
        return _v("B2", hit[0], hit[1])
    section = _toml_section(ctx.task_toml, "agent")
    allow = 'network_mode = "allowlist"' in section
    hosts = "cursor.com" in section or "allowed_hosts" in section
    if allow:
        return _v(
            "B2",
            True,
            "task.toml [agent] allowlist"
            + ("; trial web-tool use not scored (jobs/ excluded)" if hosts else ""),
        )
    return _v("B2", False, "[agent] network_mode is not allowlist")


@rule("B3")
def check_b3(ctx: RuleContext) -> RuleVerdict:
    if ctx.hidden_files:
        return _v("B3", True, f"hidden verifier ({len(ctx.hidden_files)} files)")
    f2p = set(_f2p_from_test_sh(ctx.test_sh))
    if not f2p:
        return _skip("B3", "could not parse fail-to-pass names from test.sh")
    overlap = f2p & ctx.src_test_names
    if overlap:
        return _v(
            "B3",
            False,
            f"in-tree example tests are the verifier: {sorted(overlap)[:8]}",
        )
    if ctx.src_test_names:
        return _skip("B3", "f2p names not found in src tests; cannot confirm in-tree verifier")
    return _skip("B3", "no src tests and no tests/hidden")


@rule("B4")
def check_b4(ctx: RuleContext) -> RuleVerdict:
    hit = _check_named(ctx, "blackbox_hygiene", "black-box", "blackbox")
    if hit is not None:
        return _v("B4", hit[0], hit[1])
    if not ctx.hidden_files:
        return _skip("B4", "no hidden tests")
    blob = "\n".join(ctx.hidden_files.values())
    hits = [tok for tok in _WHITEBOX_TOKENS if tok in blob]
    if hits:
        return _v("B4", False, f"hidden tests mention internal names: {hits}")
    # unexported method calls on non-testing receivers
    internal = []
    for m in re.finditer(r"\.([a-z][A-Za-z0-9]{2,})\(", blob):
        name = m.group(1)
        if name not in _OK_LOWER_CALLS:
            internal.append(name)
    internal = sorted(set(internal))
    if internal:
        return _v("B4", False, f"hidden tests call unexported names: {internal[:12]}")
    return _v("B4", True, "hidden tests look black-box (exported API / testing helpers)")


@rule("B5")
def check_b5(ctx: RuleContext) -> RuleVerdict:
    blob = "\n".join(ctx.hidden_files.values()) + "\n" + (ctx.test_sh or "")
    markers = (
        "rand.New",
        "rand.NewSource",
        "testing/quick",
        "Property",
        "property",
        "seeded",
        "adversarial",
        "-bench",
        "ns/op",
        " -race",
    )
    found = [m for m in markers if m in blob]
    if found:
        return _v("B5", True, f"property/dynamic markers: {found[:8]}")
    if ctx.hidden_files:
        return _v("B5", False, "hidden tests look like example assertions (no property/gate markers)")
    return _skip("B5", "no hidden tests or dynamic-gate markers")


@rule("B6")
def check_b6(ctx: RuleContext) -> RuleVerdict:
    if not ctx.instruction:
        return _skip("B6", "no instruction.md")
    chk = _self_check(ctx)
    ok = bool(chk.get("has_symptom_paraphrase")) and bool(chk.get("can_reproduce_one_command"))
    return _v(
        "B6",
        ok,
        f"symptom={chk.get('has_symptom_paraphrase')} repro={chk.get('can_reproduce_one_command')} "
        f"cmd={chk.get('reproduce_command')!s:.200}",
    )


@rule("B7")
def check_b7(ctx: RuleContext) -> RuleVerdict:
    if not ctx.instruction:
        return _skip("B7", "no instruction.md")
    chk = _self_check(ctx)
    leaked_s = list(chk.get("leaked_symbols") or [])
    leaked_f = list(chk.get("leaked_files") or [])
    line_nos = bool(chk.get("has_line_numbers"))
    has_diff = bool(chk.get("has_diff"))
    tokens = list(chk.get("diff_hunk_tokens_in_instruction") or [])
    ok = not leaked_s and not leaked_f and not line_nos and not has_diff and not tokens
    return _v(
        "B7",
        ok,
        f"leaked_symbols={leaked_s} leaked_files={leaked_f} line_nos={line_nos} "
        f"diff={has_diff} hunk_tokens={tokens[:8]}",
    )


@rule("B8")
def check_b8(ctx: RuleContext) -> RuleVerdict:
    if not ctx.instruction:
        return _skip("B8", "no instruction.md")
    if NO_WEB_CLAUSE in ctx.instruction:
        return _v("B8", True, "instruction contains the no-web clause")
    low = ctx.instruction.lower()
    if "do not use web search" in low or "do not use web" in low:
        return _v("B8", True, "instruction has an equivalent no-web warning")
    return _v("B8", False, "instruction.md has no no-web clause")


def evaluate_rules(
    task_dir: Path | str,
    extra: Mapping[str, Any] | None = None,
) -> list[RuleVerdict]:
    ctx = load_context(Path(task_dir), extra=extra)
    out: list[RuleVerdict] = []
    for rid in RULE_IDS:
        fn = RULES.get(rid)
        if fn is None:
            out.append(_skip(rid, "not registered"))
            continue
        out.append(fn(ctx))
    return out


def verdicts_to_dicts(verdicts: Sequence[RuleVerdict]) -> list[dict[str, Any]]:
    return [asdict(v) for v in verdicts]


def attach_rule_verdicts(
    payload: Mapping[str, Any] | None,
    task_dir: Path | str,
) -> dict[str, Any]:
    data = dict(payload or {})
    data["rule_verdicts"] = verdicts_to_dicts(evaluate_rules(task_dir, extra=data))
    return data


def write_task_validation(
    task_dir: Path | str,
    payload: Mapping[str, Any] | None = None,
) -> dict[str, Any]:
    """Write ``<task_dir>/validation.json`` including ``rule_verdicts``."""
    task_dir = Path(task_dir)
    task_dir.mkdir(parents=True, exist_ok=True)
    existing = _load_json(task_dir / "validation.json")
    data = dict(existing)
    if payload:
        data.update(payload)
    data = attach_rule_verdicts(data, task_dir)
    (task_dir / "validation.json").write_text(
        json.dumps(data, indent=2, default=str) + "\n", encoding="utf-8"
    )
    return data


def iter_task_dirs(tasks_glob: str) -> list[Path]:
    raw = [Path(p) for p in glob_paths(tasks_glob)]
    if not raw:
        raw = list(ROOT.glob(tasks_glob))
    dirs = [p.resolve() for p in raw if is_harbor_task(p)]
    return sorted(set(dirs))


def backfill_task(task_dir: Path) -> dict[str, Any] | None:
    """Write rule_verdicts if missing. Returns the payload when it wrote, else None."""
    path = task_dir / "validation.json"
    existing = _load_json(path)
    if existing.get("rule_verdicts"):
        return None
    return write_task_validation(task_dir, existing or None)


def backfill_tasks(tasks_glob: str) -> list[Path]:
    written: list[Path] = []
    for task_dir in iter_task_dirs(tasks_glob):
        if backfill_task(task_dir) is not None:
            written.append(task_dir)
    return written


@dataclass
class RuleReportRow:
    rule_id: str
    n_evaluated: int
    n_failed: int
    n_skipped: int
    killed_tasks: list[str]


def _rel_task(path: Path) -> str:
    try:
        return path.resolve().relative_to(ROOT).as_posix()
    except ValueError:
        return path.as_posix()


def aggregate_rules(task_dirs: Sequence[Path]) -> list[RuleReportRow]:
    killed: dict[str, list[str]] = {rid: [] for rid in RULE_IDS}
    n_eval = {rid: 0 for rid in RULE_IDS}
    n_fail = {rid: 0 for rid in RULE_IDS}
    n_skip = {rid: 0 for rid in RULE_IDS}
    for task_dir in task_dirs:
        data = _load_json(task_dir / "validation.json")
        verdicts = data.get("rule_verdicts") or []
        by_id = {
            str(v.get("rule_id")): v
            for v in verdicts
            if isinstance(v, dict) and v.get("rule_id")
        }
        label = _rel_task(task_dir)
        for rid in RULE_IDS:
            row = by_id.get(rid)
            if not row:
                n_skip[rid] += 1
                continue
            if row.get("skipped"):
                n_skip[rid] += 1
                continue
            n_eval[rid] += 1
            if not row.get("passed"):
                n_fail[rid] += 1
                killed[rid].append(label)
    return [
        RuleReportRow(
            rule_id=rid,
            n_evaluated=n_eval[rid],
            n_failed=n_fail[rid],
            n_skipped=n_skip[rid],
            killed_tasks=killed[rid],
        )
        for rid in RULE_IDS
    ]


def format_report_table(rows: Sequence[RuleReportRow], *, n_tasks: int) -> str:
    lines = [
        f"n_tasks = {n_tasks}",
        "",
        "| # | n_evaluated | n_failed (kills) | n_skipped | killed tasks |",
        "|---|---:|---:|---:|---|",
    ]
    for row in rows:
        killed = ", ".join(f"`{t}`" for t in row.killed_tasks) if row.killed_tasks else "—"
        lines.append(
            f"| {row.rule_id} | {row.n_evaluated} | {row.n_failed} | {row.n_skipped} | {killed} |"
        )
    return "\n".join(lines) + "\n"


def rules_report(
    tasks_glob: str,
    *,
    backfill: bool = True,
) -> tuple[str, list[RuleReportRow]]:
    if backfill:
        backfill_tasks(tasks_glob)
    task_dirs = iter_task_dirs(tasks_glob)
    rows = aggregate_rules(task_dirs)
    table = format_report_table(rows, n_tasks=len(task_dirs))
    return table, rows


def rules_report_main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        prog="openswe-synth rules-report",
        description="Aggregate per-rule kill counts from Harbor task validation.json files",
    )
    parser.add_argument(
        "--tasks-glob",
        default="experiments/harbor_nex/tasks*/*",
        help="Glob of task directories (relative to cwd or repo root)",
    )
    parser.add_argument(
        "--no-backfill",
        action="store_true",
        help="Do not write missing rule_verdicts before aggregating",
    )
    args = parser.parse_args(argv)
    table, _rows = rules_report(args.tasks_glob, backfill=not args.no_backfill)
    print(table, end="")
    return 0


def main(argv: list[str] | None = None) -> int:
    return rules_report_main(argv)


if __name__ == "__main__":
    raise SystemExit(main())
