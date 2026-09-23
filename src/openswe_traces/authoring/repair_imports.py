"""Point hidden tests at the module path their repository actually declares.

The renaming pass rewrote module paths inside the task repositories (goa -> apikit/v3,
helm -> chartkit/v4, kops -> clustkit) after the tasks were validated, and did not rewrite
the checksum-locked hidden tests to match; the pilot copies had the reverse mismatch. The
test then fails at `go test` setup, fetching a module that does not exist, whatever the
agent wrote: 64 verdicts on 13 tasks measured nothing (see ladder.ledger.unmeasured).

The repair is mechanical and touches only import paths:

  * an import is broken when no module the go.mod declares (module or replace) covers it;
  * its old prefix is the one whose remainder names a package directory that exists in
    the repository, and it is rewritten to the declared module;
  * every copy of the hidden test is rewritten - tests/hidden/ and, at L5/L6 where the
    agent is shown the file, environment/src/ - and the sha256 guards in tests/test.sh
    are recomputed for the new bytes.

A repaired copy is not trusted until it is re-validated in Docker (buggy fails, gold
passes, cheat is rejected); ``--verify`` does that with stage_units.verify_staged.

Usage::

    uv run python -m openswe_traces.authoring.repair_imports TASK_DIR... [--check] [--verify]
"""
from __future__ import annotations

import hashlib
import re
import sys
from pathlib import Path

IMPORT = re.compile(r'"(example\.internal/[^"\s]+)"')


def declared(src: Path) -> list[str]:
    txt = (src / "go.mod").read_text(errors="replace")
    mod = re.search(r"^module\s+(\S+)", txt, re.M)
    reps = re.findall(r"^\s*replace\s+(example\.internal/\S+)", txt, re.M)
    reps += re.findall(r"^\s+(example\.internal/\S+)\s+=>", txt, re.M)  # replace ( ... ) block
    return ([mod.group(1)] if mod else []) + reps


def covered(imp: str, mods: list[str]) -> bool:
    return any(imp == m or imp.startswith(m + "/") for m in mods)


def hidden_files(task: Path) -> list[Path]:
    return sorted((task / "tests/hidden").rglob("*.go"))


PATCHES = ("tests/gold.patch", "tests/cheat.patch")
PATH = re.compile(r"example\.internal/[A-Za-z0-9_.\-/]+")


def broken_imports(task: Path) -> dict[str, str]:
    """{old module prefix: declared module} for every example.internal path, in the hidden
    tests or in the gold/cheat patches, that the repository's go.mod does not cover. The
    patches count too: their context lines quote import blocks, so a stale prefix there
    stops the answer key applying at all."""
    src = task / "environment/src"
    mods = declared(src)
    module = mods[0]
    texts = [f.read_text(errors="replace") for f in hidden_files(task)]
    texts += [(task / p).read_text(errors="replace") for p in PATCHES if (task / p).exists()]
    out: dict[str, str] = {}
    for imp in sorted({p.rstrip("/.") for t in texts for p in PATH.findall(t)}):
        if covered(imp, mods) or any(imp == o or imp.startswith(o + "/") for o in out):
            continue
        parts = imp.split("/")
        for k in range(2, len(parts) + 1):
            rest = "/".join(parts[k:])
            if rest and (src / rest).is_dir():
                out["/".join(parts[:k])] = module
                break
        # a path naming no directory here (a file, a vendored module) is left alone
    return out


def _rewrite(text: str, mapping: dict[str, str]) -> str:
    for old, new in sorted(mapping.items(), key=lambda kv: -len(kv[0])):
        text = re.sub(re.escape(old) + r'(?=[/"\s`]|$)', new, text, flags=re.M)
    return text


def repair(task: Path) -> dict[str, str]:
    """Rewrite stale module prefixes in the hidden tests (every copy) and the patches;
    return the prefix mapping used."""
    mapping = broken_imports(task)
    if not mapping:
        return mapping
    for p in PATCHES:
        if (task / p).exists():
            (task / p).write_text(_rewrite((task / p).read_text(), mapping))
    test_sh = (task / "tests/test.sh").read_text()
    for f in hidden_files(task):
        rel = f.relative_to(task / "tests/hidden").as_posix()
        new = _rewrite(f.read_text(), mapping)
        f.write_text(new)
        shown = task / "environment/src" / rel
        if shown.exists():
            shown.write_text(_rewrite(shown.read_text(), mapping))
        digest = hashlib.sha256(new.encode()).hexdigest()
        for where in (r"\$HIDDEN/", "/app/"):
            test_sh = re.sub(rf'(?<=echo ")[0-9a-f]{{64}}(?=  {where}{re.escape(rel)}")',
                             digest, test_sh)
    (task / "tests/test.sh").write_text(test_sh)
    ins = task / "instruction.md"
    if ins.exists():
        ins.write_text(_rewrite(ins.read_text(), mapping))
    return mapping


def verify(task: Path) -> dict[str, bool]:
    from openswe_traces.synth.stage_units import verify_staged

    tag = "repairimports-" + re.sub(r"[^a-z0-9]+", "-", f"{task.parent.name}-{task.name}".lower())
    checks, details = verify_staged(task, tag)
    for k, v in details.items():
        if not checks.get(k, True):
            print(f"    {k}: {v[-240:]}", flush=True)
    return checks


def main(argv: list[str] | None = None) -> int:
    argv = sys.argv[1:] if argv is None else argv
    check, do_verify = "--check" in argv, "--verify" in argv
    tasks = [Path(a) for a in argv if not a.startswith("--")]
    bad = 0
    for t in tasks:
        mapping = broken_imports(t) if check else repair(t)
        line = f"{t.parent.name}/{t.name}: " + (", ".join(f"{a} -> {b}" for a, b in
                                                          sorted(mapping.items())) or "clean")
        if do_verify and not check:
            checks = verify(t)
            ok = all(checks.values())
            bad += not ok
            line += "  VALID" if ok else f"  INVALID {checks}"
        print(line, flush=True)
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
