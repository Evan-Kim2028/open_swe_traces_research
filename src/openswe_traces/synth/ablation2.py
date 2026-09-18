"""Package ablation-round-2 VALID units as Harbor A0/A1 tasks.

Does not launch Harbor. Image-proofs buggy-fail / gold-pass (rule A8).
"""

from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import tempfile
from collections.abc import Sequence
from dataclasses import dataclass, field
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.synth.affordance import (
    HiddenTest,
    _copytree,
    build_affordance_levels,
    strip_go_func,
    with_no_web,
)
from openswe_traces.synth.harbor_tasks import instruction_self_check
from openswe_traces.synth.ladder2 import unified_patch, write_patch
from openswe_traces.synth.rules import write_task_validation

EXP = ROOT / "experiments" / "ablation_graph"
HOST = EXP / "repos" / "revive-graph"
DEFAULT_DEST = ROOT / "experiments" / "harbor_nex" / "tasks_ablation2"
PACKAGED_MD = EXP / "PACKAGED.md"
IMAGE_PREFIX = "harbor-ablation2"

_TEST_FUNC_RE = re.compile(r"^func\s+(?:\(.*?\)\s+)?(Test[A-Za-z0-9_]+)\s*\(", re.MULTILINE)
_PKG_RE = re.compile(r"^package\s+\w+", re.MULTILINE)

COPY_IGNORE = (
    ".git",
    ".codegraph",
    "units",
    "bugs",
    "START.txt",
    "START2.txt",
    "__pycache__",
)

FILE_FILTER_INSTRUCTION = """# Missing behavior

A blank (or whitespace-only) pattern matches no path. The patterns `*` and `~`
match every path.

A pattern with no `*` and no leading `~` matches that path and only that path.
Backslashes in the candidate are treated as slashes.

A pattern starting with `~` is the rest of the string as a regular expression.
Compile failure is an error.

The pattern `TEST` matches names ending in `_test.go` and does not match names
like `_test_no.go`.

`*` in a glob matches any run of non-slash characters. `a/b/*.pb.go` matches
files in `a/b` whose names end in `.pb.go`, not `*.nopb.go`.

`**` matches across slashes, including zero directories (`a/**/*.pb.go` matches
`a/xxx.pb.go` and `a/x/y/z/yyy.pb.go`). A `**` glued to non-slash characters on
both sides is invalid.

After a rule's exclude list is compiled, a file is skipped for that rule iff any
pattern matches its path. An empty exclude list never skips. Loading a
configuration document compiles each rule's exclude list so later membership
tests use those compiled patterns. An invalid exclude is a configuration error
rather than a silent match.

Worked cases the hidden tests assert:

- Exact path `a/b/c.go` matches that path and not `a/b/d.go`.
- `~b/[cd].go$` matches `a/b/c.go` and `b/d.go`, not `b/x.go`.
- `TEST` matches `a/b/c_test.go` and not `a/b/c_test_no.go`.
- `a/b/*.pb.go` matches `a/b/xxx.pb.go` and not `a/b/xxx.nopb.go`.
- `a/**/*.pb.go` matches `a/xxx.pb.go` and `a/x/y/z/yyy.pb.go`.
- Empty pattern matches none of those paths; `*` and `~` match all of them.
- A rule with no excludes still runs on a fixture file; a rule whose exclude is
  that fixture path does not.
- Loading a document with a rule-level exclude compiles it so `some/file.go`
  matches and `some/any-other.go` does not.
- A bad exclude string is a configuration error.

A matcher that always returns false is wrong: expected `a/b/c.go` to match
itself, actual no match.

Coverage the hidden checks enforce:

- Exact, regexp, TEST, `*`, `**`, empty, and match-all patterns behave as specified.
- A rule is applied unless a compiled exclude matches the file under test.
- Loading a document with a rule-level exclude compiles it so one path matches
  and a sibling path does not.

Reproduce with:

```
go test -count=1 -timeout 15m ./lint/ ./test/ ./config/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""

REVIVELIB_INSTRUCTION = """# Missing behavior

Constructing a runner from a configuration, an optional "treat all findings as
failing exit", a max-open-files cap, and extra rules yields an object that can
scan path patterns and render the resulting findings.

Extra rules whose names are not already in the configuration are inserted with
their default configuration. Scanning accepts include and exclude patterns; if
the caller passes no excludes, the configuration's excludes are used; if those
are also empty, `vendor/...` is excluded. Includes default to the working
directory when none remain after trimming. The result is a channel of findings.

Rendering named after a known formatter drains the channel, drops findings below
the configured confidence, and computes an exit code: 0 if nothing passed the
filter; the warning code on the first kept finding; the error code if any kept
finding is configured as error. With "set exit status" construction, both codes
are 1, so any kept finding exits 1. The formatter runs concurrently with the
drain.

Worked cases the hidden tests assert:

- Scanning the if-return sample under default rules plus if-return produces five
  findings.
- Stylish rendering of those five findings contains each file/line/rule/message
  and exits 1.
- Construction with set-exit-status and an extra named rule loads configuration,
  that extra rule, file-cap 2048, and codes 1/1.

A constructor that returns a nil runner is wrong: expected five findings from
the if-return sample, actual a nil channel or error.

Coverage the hidden checks enforce:

- Scanning the if-return sample under default rules plus if-return produces five findings.
- Stylish rendering of those five findings contains each file/line/rule/message and exits 1.
- Construction with set-exit-status and an extra named rule loads configuration, that extra rule, file-cap 2048, and codes 1/1.

Reproduce with:

```
go test -count=1 -timeout 15m ./revivelib/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.
"""


@dataclass(frozen=True)
class ValidUnit:
    cond: str
    name: str
    units_root: Path
    listed_tests: tuple[str, ...]
    test_files: tuple[str, ...]
    packages: tuple[str, ...]
    changed_symbols: tuple[str, ...]
    changed_files: tuple[str, ...]
    instruction: str
    one_liners: dict[str, str] = field(default_factory=dict)

    @property
    def family(self) -> str:
        return f"{self.cond}-{self.name}"

    @property
    def unit_dir(self) -> Path:
        return self.units_root / self.name

    @property
    def excision(self) -> Path:
        return self.unit_dir / "excision.patch"


def valid_units() -> tuple[ValidUnit, ...]:
    """Judge VALID units from RESULT2.md (both (a) and (b) and (c))."""
    graph_units = EXP / "repos" / "revive-graph" / "units"
    nograph_units = EXP / "repos" / "revive-nograph" / "units"
    file_liners = {
        "lint/filefilter_test.go": (
            "Exact, regexp, TEST, `*`, `**`, empty, and match-all filename patterns."
        ),
        "test/file_filter_test.go": (
            "A rule runs unless a compiled exclude matches the file under test."
        ),
        "config/getconfig_bb_test.go": (
            "Loading a document with a rule-level exclude compiles it so one path "
            "matches and a sibling does not."
        ),
    }
    return (
        ValidUnit(
            cond="graph",
            name="file-exclude-filter",
            units_root=graph_units,
            listed_tests=("TestFileFilter", "TestFileExcludeFilterAtRuleLevel", "TestGetConfig"),
            test_files=(
                "lint/filefilter_test.go",
                "test/file_filter_test.go",
                "config/config_test.go",
            ),
            packages=("lint", "test", "config"),
            changed_symbols=(
                "ParseFileFilter",
                "prepareRegexp",
                "MatchFileName",
                "Initialize",
                "MustExclude",
            ),
            changed_files=("filefilter.go", "config.go"),
            instruction=FILE_FILTER_INSTRUCTION,
            one_liners=file_liners,
        ),
        ValidUnit(
            cond="nograph",
            name="revivelib-runner",
            units_root=nograph_units,
            listed_tests=("TestReviveLint", "TestReviveFormat", "TestReviveCreateInstance"),
            test_files=("revivelib/core_test.go", "revivelib/core_internal_test.go"),
            packages=("revivelib",),
            changed_symbols=("New", "Lint", "Format", "getPackages", "normalizeSplit", "Include", "Exclude"),
            changed_files=("core.go", "pattern.go"),
            instruction=REVIVELIB_INSTRUCTION,
            one_liners={
                "revivelib/core_test.go": (
                    "If-return sample yields five findings; stylish rendering contains "
                    "each file/line/rule/message and exits 1."
                ),
                "revivelib/core_internal_test.go": (
                    "Construction with set-exit-status and an extra named rule loads "
                    "configuration, that extra rule, file-cap 2048, and codes 1/1."
                ),
            },
        ),
        ValidUnit(
            cond="nograph",
            name="file-filter",
            units_root=nograph_units,
            listed_tests=("TestFileFilter", "TestFileExcludeFilterAtRuleLevel", "TestGetConfig"),
            test_files=(
                "lint/filefilter_test.go",
                "test/file_filter_test.go",
                "config/config_test.go",
            ),
            packages=("lint", "test", "config"),
            changed_symbols=(
                "ParseFileFilter",
                "prepareRegexp",
                "MatchFileName",
                "Initialize",
                "MustExclude",
            ),
            changed_files=("filefilter.go", "config.go"),
            instruction=FILE_FILTER_INSTRUCTION,
            one_liners=file_liners,
        ),
    )


def apply_excision(tree: Path, patch: Path) -> None:
    text = patch.read_text(encoding="utf-8")
    pflag = "1" if "\n--- a/" in text or text.startswith("--- a/") else "0"
    proc = subprocess.run(
        ["patch", f"-p{pflag}", "--forward", "--batch", "-i", str(patch.resolve()), "-d", str(tree)],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        err = (proc.stderr or proc.stdout or "").strip()
        raise RuntimeError(f"excision {patch} failed (p{pflag}): {err}")


def extract_go_func(text: str, signature_prefix: str) -> str:
    """Return one top-level Go func whose header starts with ``signature_prefix``."""
    idx = text.find(signature_prefix)
    if idx < 0:
        raise ValueError(f"function not found: {signature_prefix[:80]}")
    start = text.rfind("\n", 0, idx)
    start = 0 if start < 0 else start + 1
    brace = text.find("{", idx)
    if brace < 0:
        end = text.find("\n", idx)
        end = len(text) if end < 0 else end + 1
        return text[start:end]
    depth = 0
    i = brace
    while i < len(text):
        ch = text[i]
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                end = i + 1
                if end < len(text) and text[end] == "\n":
                    end += 1
                return text[start:end]
        i += 1
    raise ValueError(f"unclosed function: {signature_prefix[:80]}")


def _package_and_imports(text: str) -> str:
    m = _PKG_RE.search(text)
    if not m:
        return "package extracted\n\n"
    start = m.start()
    import_m = re.search(r"^import\s+\(", text, re.MULTILINE)
    if not import_m:
        single = re.search(r"^import\s+\".+\"\s*$", text, re.MULTILINE)
        if single:
            return text[start : single.end()] + "\n\n"
        return text[start : m.end()] + "\n\n"
    depth = 0
    i = import_m.start()
    while i < len(text):
        if text[i] == "(":
            depth += 1
        elif text[i] == ")":
            depth -= 1
            if depth == 0:
                return text[start : i + 1] + "\n\n"
        i += 1
    return text[start : m.end()] + "\n\n"


def hidden_tests_for(unit: ValidUnit, src: Path) -> list[HiddenTest]:
    listed = set(unit.listed_tests)
    hidden: list[HiddenTest] = []
    for rel in unit.test_files:
        path = src / rel
        if not path.is_file():
            raise FileNotFoundError(path)
        text = path.read_text(encoding="utf-8")
        names = tuple(_TEST_FUNC_RE.findall(text))
        listed_here = tuple(n for n in names if n in listed)
        if not listed_here:
            continue
        if set(names) <= listed:
            hidden.append(
                HiddenTest(
                    relpath=rel,
                    content=text,
                    one_liner=unit.one_liners.get(rel, listed_here[0]),
                    test_names=listed_here,
                )
            )
            continue
        # Mixed file: extract listed tests into a sibling hidden file.
        dest_rel = str(Path(rel).with_name("getconfig_bb_test.go"))
        if rel.endswith("config_test.go"):
            dest_rel = "config/getconfig_bb_test.go"
        chunks = [_package_and_imports(text)]
        kept: list[str] = []
        for name in listed_here:
            chunks.append(extract_go_func(text, f"func {name}("))
            kept.append(name)
            text = strip_go_func(text, f"func {name}(")
        path.write_text(text, encoding="utf-8")
        hidden.append(
            HiddenTest(
                relpath=dest_rel,
                content="".join(chunks),
                one_liner=unit.one_liners.get(dest_rel, ", ".join(kept)),
                test_names=tuple(kept),
            )
        )
    if not hidden:
        raise ValueError(f"no hidden tests for {unit.family}")
    return hidden


def gold_patch_from(original: Path, excised: Path) -> str:
    hunks: list[str] = []
    for src in excised.rglob("*"):
        if not src.is_file():
            continue
        rel = src.relative_to(excised).as_posix()
        if rel.endswith("_test.go") or "/tests/" in rel:
            continue
        orig = original / rel
        if not orig.is_file():
            continue
        old = src.read_text(encoding="utf-8", errors="replace")
        new = orig.read_text(encoding="utf-8", errors="replace")
        if old == new:
            continue
        hunks.append(unified_patch(rel, old, new))
    return "".join(h for h in hunks if h)


def render_revive_dockerfile() -> str:
    return """FROM golang:1.23

RUN apt-get update && apt-get install -y --no-install-recommends \\
        git tmux ca-certificates patch \\
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY src/go.mod src/go.sum /app/
RUN GOPROXY=https://proxy.golang.org,direct go mod download
COPY src/ /app/
RUN GOPROXY=https://proxy.golang.org,direct go mod download || \\
    GOPROXY=https://proxy.golang.org,direct go mod tidy || true
RUN go version
"""


def copy_host_tree(host: Path, dest: Path) -> None:
    if dest.exists():
        shutil.rmtree(dest)
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(host, dest, symlinks=True, ignore=shutil.ignore_patterns(*COPY_IGNORE))


def stage_a0(unit: ValidUnit, host: Path, dest_root: Path) -> tuple[Path, list[HiddenTest]]:
    family_a0 = dest_root / f"{unit.family}-A0"
    if family_a0.exists():
        shutil.rmtree(family_a0)
    src = family_a0 / "environment" / "src"
    copy_host_tree(host, src)
    original = dest_root / f"_original_{unit.family}"
    if original.exists():
        shutil.rmtree(original)
    shutil.copytree(src, original, symlinks=True)
    apply_excision(src, unit.excision)
    hidden = hidden_tests_for(unit, src)
    gold = gold_patch_from(original, src)
    tests = family_a0 / "tests"
    tests.mkdir(parents=True, exist_ok=True)
    write_patch(tests / "gold.patch", [gold])
    (family_a0 / "instruction.md").write_text(with_no_web(unit.instruction), encoding="utf-8")
    (family_a0 / "environment" / "Dockerfile").write_text(render_revive_dockerfile(), encoding="utf-8")
    return family_a0, hidden


def _docker(*args: str, timeout: int = 600) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["docker", *args],
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
    )


def build_task_image(task_dir: Path, tag: str) -> None:
    proc = _docker(
        "build",
        "-t",
        tag,
        "-f",
        str(task_dir / "environment" / "Dockerfile"),
        str(task_dir / "environment"),
        timeout=1200,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"docker build {tag} failed:\n{proc.stdout}\n{proc.stderr}")


def validate_harness(tag: str) -> dict[str, object]:
    go = _docker("run", "--rm", tag, "bash", "-c", "command -v go && go version")
    false_p = _docker("run", "--rm", tag, "bash", "-c", "false")
    true_p = _docker("run", "--rm", tag, "bash", "-c", "true")
    ok = go.returncode == 0 and false_p.returncode != 0 and true_p.returncode == 0
    return {
        "ok": ok,
        "go_on_path": go.returncode == 0,
        "go_out": (go.stdout + go.stderr)[-500:],
        "false_exit": false_p.returncode,
        "true_exit": true_p.returncode,
    }


def run_test_sh(tag: str, src: Path, tests: Path, *, timeout: int = 900) -> tuple[int, str, str]:
    logs = tempfile.mkdtemp(prefix="ablation2-logs-")
    try:
        proc = _docker(
            "run",
            "--rm",
            "--network=none",
            "-v",
            f"{src}:/app",
            "-v",
            f"{tests}:/tests",
            "-v",
            f"{logs}:/logs",
            "-e",
            "GOCACHE=/tmp/gocache",
            "-e",
            "GOMODCACHE=/go/pkg/mod",
            "-e",
            "GOPROXY=off",
            "-e",
            "GOTOOLCHAIN=local",
            tag,
            "bash",
            "/tests/test.sh",
            timeout=timeout,
        )
        reward = Path(logs) / "verifier" / "reward.txt"
        reward_s = reward.read_text(encoding="utf-8").strip() if reward.is_file() else ""
        return proc.returncode, reward_s, (proc.stdout or "") + (proc.stderr or "")
    finally:
        shutil.rmtree(logs, ignore_errors=True)


def prove_unit(
    unit: ValidUnit,
    a0: Path,
    dest_root: Path,
    tag: str,
) -> dict[str, object]:
    harness = validate_harness(tag)
    out: dict[str, object] = {"harness": harness, "harness_ok": bool(harness["ok"]), "ok": True}
    if not harness["ok"]:
        out["ok"] = False
        return out

    work = dest_root / f"_proof_{unit.family}_A0"
    if work.exists():
        shutil.rmtree(work)
    _copytree(a0 / "environment" / "src", work)
    tests = a0 / "tests"
    rc, reward, blob = run_test_sh(tag, work, tests)
    buggy_fail = rc != 0 or reward != "1"
    out["buggy_rc"] = rc
    out["buggy_reward"] = reward
    out["buggy_fails"] = buggy_fail
    out["buggy_tail"] = blob[-2500:]

    gold_tree = dest_root / f"_proof_{unit.family}_A0_gold"
    if gold_tree.exists():
        shutil.rmtree(gold_tree)
    _copytree(work, gold_tree)
    gold_patch = tests / "gold.patch"
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(gold_patch.resolve()), "-d", str(gold_tree)],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        out["ok"] = False
        out["gold_patch_err"] = (proc.stderr or proc.stdout or "")[-1500:]
        return out
    rc, reward, blob = run_test_sh(tag, gold_tree, tests)
    gold_pass = rc == 0 and reward == "1"
    out["gold_rc"] = rc
    out["gold_reward"] = reward
    out["gold_pass"] = gold_pass
    out["gold_restore"] = gold_pass
    out["gold_tail"] = blob[-2500:]
    out["ok"] = bool(buggy_fail and gold_pass and harness["ok"])
    return out


def validation_payload(
    unit: ValidUnit,
    level: int,
    hidden: Sequence[HiddenTest],
    instruction: str,
    test_sh: str,
    proof: MappingLike | None,
) -> dict[str, object]:
    check = instruction_self_check(
        instruction,
        test_sh=test_sh,
        f2p_tests=[n for h in hidden for n in h.names()],
        changed_symbols=unit.changed_symbols,
        changed_files=unit.changed_files,
        locality=0 if level >= 1 else 2,
        packages=unit.packages,
    )
    proof = dict(proof or {})
    checks: dict[str, object] = {
        "buggy_fails": bool(proof.get("buggy_fails")),
        "gold_restore": bool(proof.get("gold_pass") or proof.get("gold_restore")),
        "gold_pass": bool(proof.get("gold_pass") or proof.get("gold_restore")),
        "harness_ok": bool(proof.get("harness_ok")),
        "rest_suite_green": True,  # judge2 others_ok on VALID units
        "f2p_in_impact": True,
        "black-box": True,  # judge2 (c) on every unit
        "patches_skip_tests": True,
    }
    return {
        "family": unit.family,
        "cond": unit.cond,
        "unit": unit.name,
        "level": level,
        "instruction_self_check": check,
        "changed_symbols": list(unit.changed_symbols),
        "changed_files": list(unit.changed_files),
        "impact_files": list(unit.test_files),
        "f2p_tests": [n for h in hidden for n in h.names()],
        "checks": checks,
        "harness_ok": bool(proof.get("harness_ok")),
        "patches_skip_tests": True,
        "proof": {
            k: proof[k]
            for k in (
                "ok",
                "harness_ok",
                "buggy_fails",
                "gold_pass",
                "buggy_rc",
                "buggy_reward",
                "gold_rc",
                "gold_reward",
                "buggy_tail",
                "gold_tail",
            )
            if k in proof
        },
    }


MappingLike = dict[str, object]


def write_packaged_md(
    dest_root: Path,
    results: dict[str, dict[int, Path]],
    proofs: dict[str, dict[str, object]],
) -> str:
    lines = [
        "# Ablation round 2 — packaged Harbor tasks",
        "",
        "Date: 2026-09-18. VALID units from `experiments/ablation_graph/RESULT2.md`",
        "(judge: (a) closure ∧ (b) self-containment ∧ (c) black-box). Host tree:",
        "`experiments/ablation_graph/repos/revive-graph` (no `.git` / `.codegraph` /",
        "`units/` / `bugs/`). Gold = revert the unit's `excision.patch`.",
        "Affordance A0 and A1 only. **No Harbor jobs were launched.**",
        "",
        "Build:",
        "",
        "```",
        "uv run python scripts/build_ablation2.py",
        "```",
        "",
        "Layout: `experiments/harbor_nex/tasks_ablation2/<cond>-<unit>-A{0,1}/`.",
        "Hidden tests live under `tests/hidden/` and are checksum-guarded by `tests/test.sh`.",
        "`task.toml` `[agent]` allowlist + `[verifier]` no-network. Instruction is the",
        "unit `contract.md` rewritten at locality L2 (no test/symbol/file names) plus a",
        "reproduce command and the no-web clause. A1 adds hidden test names and one-liners.",
        "",
        "| task | cond | unit | level | hidden tests | image | buggy | gold |",
        "|---|---|---|---:|---|---|---|---|",
    ]
    for unit in valid_units():
        proof = proofs.get(unit.family) or {}
        hidden_names: list[str] = []
        a0 = results.get(unit.family, {}).get(0)
        if a0 is not None:
            sh = (a0 / "tests" / "test.sh").read_text(encoding="utf-8") if (a0 / "tests" / "test.sh").is_file() else ""
            m = re.search(r"-run '?\^\(?([A-Za-z0-9_|]+)", sh)
            if m:
                hidden_names = [p for p in m.group(1).split("|") if p.startswith("Test")]
        for level in (0, 1):
            slug = f"{unit.family}-A{level}"
            img = f"{IMAGE_PREFIX}-{unit.family}:latest"
            buggy = "FAIL (want)" if proof.get("buggy_fails") else ("skipped" if not proof else "FAIL unexpected")
            gold = "PASS (want)" if proof.get("gold_pass") else ("skipped" if not proof else "FAIL unexpected")
            if proof and not proof.get("ok"):
                if not proof.get("buggy_fails"):
                    buggy = "PASS unexpected"
                if not proof.get("gold_pass"):
                    gold = "FAIL unexpected"
            lines.append(
                f"| `{slug}` | {unit.cond} | `{unit.name}` | {level} | "
                f"{', '.join(f'`{n}`' for n in hidden_names) or '—'} | `{img}` | {buggy} | {gold} |"
            )
    lines += [
        "",
        "## Proofs (built image, A8)",
        "",
        "Each A0 environment was `docker build` from `golang:1.23` with modules",
        "pre-downloaded (`GOTOOLCHAIN=auto`). Verifier ran with `--network=none`.",
        "A1 shares the A0 image (same tree, same `tests/test.sh`).",
        "",
    ]
    for unit in valid_units():
        proof = proofs.get(unit.family) or {}
        lines += [
            f"### `{unit.family}`",
            "",
            f"- harness_ok: `{proof.get('harness_ok')}`",
            f"- buggy_fails: `{proof.get('buggy_fails')}` (rc={proof.get('buggy_rc')} reward={proof.get('buggy_reward')!r})",
            f"- gold_restore: `{proof.get('gold_pass')}` (rc={proof.get('gold_rc')} reward={proof.get('gold_reward')!r})",
            "",
        ]
        if proof.get("buggy_tail"):
            lines += ["Buggy tail:", "", "```", str(proof["buggy_tail"])[-800:], "```", ""]
        if proof.get("gold_tail"):
            lines += ["Gold tail:", "", "```", str(proof["gold_tail"])[-800:], "```", ""]
    lines += [
        "## Rules registry",
        "",
        "`validation.json` is written by `openswe_traces.synth.rules.write_task_validation`.",
        "A2/A3/A9/A11 skipped (no alt/cheat/perf). B5 recorded as fail: hidden tests are",
        "the unit's existing example black-box tests, not seeded properties.",
        "",
    ]
    text = "\n".join(lines) + "\n"
    PACKAGED_MD.parent.mkdir(parents=True, exist_ok=True)
    PACKAGED_MD.write_text(text, encoding="utf-8")
    (dest_root / "PACKAGED.md").write_text(text, encoding="utf-8")
    return text


def construct_ablation2(
    dest_root: Path | str | None = None,
    *,
    host: Path | str | None = None,
    skip_docker: bool = False,
    units: Sequence[ValidUnit] | None = None,
) -> tuple[dict[str, dict[int, Path]], dict[str, dict[str, object]]]:
    dest_root = Path(dest_root) if dest_root is not None else DEFAULT_DEST
    host = Path(host) if host is not None else HOST
    dest_root.mkdir(parents=True, exist_ok=True)
    units = tuple(units) if units is not None else valid_units()
    results: dict[str, dict[int, Path]] = {}
    proofs: dict[str, dict[str, object]] = {}
    hidden_by: dict[str, list[HiddenTest]] = {}

    for unit in units:
        a0, hidden = stage_a0(unit, host, dest_root)
        hidden_by[unit.family] = hidden
        results[unit.family] = build_affordance_levels(
            a0,
            hidden,
            levels=(0, 1),
            dest_root=dest_root,
            family=unit.family,
            instruction_a0=unit.instruction,
            packages=unit.packages,
            changed_symbols=unit.changed_symbols,
            changed_files=unit.changed_files,
        )
        dockerfile = render_revive_dockerfile()
        for path in results[unit.family].values():
            (path / "environment" / "Dockerfile").write_text(dockerfile, encoding="utf-8")
            gold_src = a0 / "tests" / "gold.patch"
            gold_dst = path / "tests" / "gold.patch"
            if gold_src.is_file() and gold_src.resolve() != gold_dst.resolve():
                shutil.copy2(gold_src, gold_dst)

    if not skip_docker:
        for unit in units:
            a0 = results[unit.family][0]
            tag = f"{IMAGE_PREFIX}-{unit.family}:latest"
            print(f"docker build {tag}", flush=True)
            build_task_image(a0, tag)
            print(f"prove {unit.family}", flush=True)
            proofs[unit.family] = prove_unit(unit, a0, dest_root, tag)
    else:
        for unit in units:
            proofs[unit.family] = {"ok": False, "harness_ok": False, "skipped": True}

    for unit in units:
        hidden = hidden_by[unit.family]
        proof = proofs[unit.family]
        for level, path in results[unit.family].items():
            instr = (path / "instruction.md").read_text(encoding="utf-8")
            sh = (path / "tests" / "test.sh").read_text(encoding="utf-8")
            write_task_validation(
                path,
                validation_payload(unit, level, hidden, instr, sh, proof),
            )

    write_packaged_md(dest_root, results, proofs)
    return results, proofs


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Package ablation2 VALID units as Harbor A0/A1")
    parser.add_argument("--dest", default=str(DEFAULT_DEST))
    parser.add_argument("--host", default=str(HOST))
    parser.add_argument("--skip-docker", action="store_true")
    args = parser.parse_args(argv)
    construct_ablation2(args.dest, host=args.host, skip_docker=args.skip_docker)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
