"""Build the itsdangerous/signer unit: authored patches + L0/L2 task dirs.

Generalisation proof for the pipeline beyond Go: the only new *pipeline*
surface is repos.yaml config (`image`, `install_cmd`, `test_cmd`,
`test_glob`); this module carries the unit content itself (excision
surgery, hidden suite, instructions) and drives the same packaging
abstractions the Go repos use — ``materialize.materialize_task`` for
``environment/src`` and ``synth.affordance.build_affordance_levels`` for
the L0/L2 dirs.

The unit: ``src/itsdangerous/signer.py`` (pinned tag 2.2.0). Three
methods are excised — ``Signer.derive_key`` (returns a wrong constant,
ignoring salt + derivation scheme), ``Signer.get_signature`` (empty
signature), ``Signer.verify_signature`` (always rejects). Gold restores
the three bodies; the cheat special-cases only the contract's worked
examples and must fail the seeded-random hidden suite.
"""

from __future__ import annotations

import re
import shutil
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.pipeline.config import PipelineConfig, load_config
from openswe_traces.pipeline.materialize import materialize_task
from openswe_traces.synth.affordance import (
    HiddenTest,
    build_affordance_levels,
    render_hidden_test_sh,
    render_ladder_base_dockerfile,
    render_unsolv_task_toml,
)

REPO = "itsdangerous"
FAMILY = "signer"
UNIT_DIR = Path("src/itsdangerous/signer.py")
# Hidden suite relpath under /app; it must collect inside the repo's own
# package-style test dir (tests/test_itsdangerous/__init__.py exists).
HIDDEN_REL = "tests/test_itsdangerous/test_signer_bb.py"
BASE_IMAGE = "ladder-base:itsdangerous"

# Worked-example constants (secret "secret-key", default salt, SHA-1) —
# contract.md examples a cheat may hardcode; the seeded properties defeat it.
_WORKED = {
    b"my string": b"wh6tMHxLgJqB6oY1uT73iMlyrOA",
    b"value": b"RI9crmxKpuX1wTw2mg_FHFGXRic",
    b"hello": b"GIG_I5genEs4rf1bcDRWLO8Yso4",
    b"": b"b4-og8DGknlZ-AXdHwT5DOiFoFs",
}
_DERIVE_KEY_HEX = "95c1413ec52400db8a22b50e2f318e9204636448"


def _replace_method_body(text: str, header: str, new_body: str) -> str:
    """Replace a method body with ``new_body`` (no trailing newline), keeping
    the signature and docstring; the following member stays untouched."""
    m = re.search(re.escape(header), text)
    if not m:
        raise ValueError(f"method header not found: {header!r}")
    open_dq = text.find('"""', m.end())
    if open_dq < 0:
        raise ValueError(f"no docstring after {header!r}")
    close = text.find('"""', open_dq + 3)
    if close < 0:
        raise ValueError(f"unterminated docstring after {header!r}")
    rest = text[close + 3:]
    nxt = re.search(r"\n\n    (?:def |@|class )", rest)
    end = close + 3 + (nxt.start() if nxt else len(rest))
    if new_body.endswith("\n"):
        raise ValueError("new_body must not end with a newline")
    return text[: close + 3] + "\n" + new_body + text[end:]


def excise_signer(text: str) -> str:
    """Stub the three closure methods with wrong constants (still importable)."""
    text = _replace_method_body(
        text,
        "    def derive_key(self, secret_key: str | bytes | None = None) -> bytes:",
        '        return b"broken-key"',
    )
    text = _replace_method_body(
        text,
        "    def get_signature(self, value: str | bytes) -> bytes:",
        '        return b""',
    )
    text = _replace_method_body(
        text,
        "    def verify_signature(self, value: str | bytes, sig: str | bytes) -> bool:",
        "        return False",
    )
    return text


def cheat_signer(excised: str) -> str:
    """Special-case only the contract's worked examples in ``Signer.get_signature``.

    Applied on top of the excised tree: ``derive_key`` and
    ``verify_signature`` stay broken, so random inputs still fail; the four
    literal values from contract.md round-trip.
    """
    lines = [
        "        value = want_bytes(value)",
    ]
    for value, sig in _WORKED.items():
        lines.append(f"        if value == {value!r}:")
        lines.append(f"            return {sig!r}")
    lines.append('        return b""')
    return _replace_method_body(
        excised,
        "    def get_signature(self, value: str | bytes) -> bytes:",
        "\n".join(lines),
    )


def _unified_diff(old: str, new: str, old_label: str, new_label: str) -> str:
    import difflib

    diff = difflib.unified_diff(
        old.splitlines(keepends=True),
        new.splitlines(keepends=True),
        fromfile=old_label,
        tofile=new_label,
        n=3,
    )
    return "".join(diff)


def write_patches(author_dir: Path, base_text: str) -> dict[str, Path]:
    """Write excision/gold/cheat patches for the unit; returns their paths."""
    excised = excise_signer(base_text)
    if excised == base_text:
        raise ValueError("excision produced no change")
    authored = author_dir / "_author"
    excised_dir = authored / "excised"
    excised_dir.mkdir(parents=True, exist_ok=True)
    old_label = f"a/{UNIT_DIR}"
    new_label = f"b/{UNIT_DIR}"
    excision = _unified_diff(base_text, excised, old_label, new_label)
    gold = _unified_diff(excised, base_text, old_label, new_label)
    cheat = _unified_diff(excised, cheat_signer(excised), old_label, new_label)
    paths = {
        "excision": excised_dir / "excision.patch",
        "gold": authored / "gold.patch",
        "cheat": authored / "cheat.patch",
    }
    for name, content in (
        ("excision", excision),
        ("gold", gold),
        ("cheat", cheat),
    ):
        paths[name].write_text(content, encoding="utf-8")
    return paths


def hidden_test() -> HiddenTest:
    src = ROOT / "experiments" / "pipeline" / "authored" / REPO / FAMILY / "_author" / "hidden" / "test_signer_bb.py"
    content = src.read_text(encoding="utf-8")
    names = tuple(
        m.group(1)
        for m in re.finditer(
            r"^[ \t]*(?:async[ \t]+)?def[ \t]+(test_[A-Za-z0-9_]+)[ \t]*\(",
            content,
            re.MULTILINE,
        )
    )
    if not names:
        raise ValueError("hidden suite defines no test functions")
    return HiddenTest(
        relpath=HIDDEN_REL,
        content=content,
        one_liner="Seeded-random sign/unsign round-trips, tamper + wrong-key rejection, rotation scan order, and contract edges.",
        test_names=names,
    )


def build_python_unit(cfg: PipelineConfig | None = None) -> dict[str, object]:
    """Materialise the excised tree and package signer-L0 / signer-L2."""
    cfg = cfg or load_config()
    spec = cfg.repo(REPO)
    base_tree = Path(spec.src)
    if not base_tree.is_absolute():
        base_tree = ROOT / base_tree
    base_file = base_tree / UNIT_DIR
    base_text = base_file.read_text(encoding="utf-8")

    authored = cfg.authored_dir / REPO / FAMILY
    write_patches(authored, base_text)

    skel = cfg.tasks_dir / REPO / "_signer_skel"
    if skel.exists():
        shutil.rmtree(skel)
    (skel / "environment").mkdir(parents=True)
    (skel / "environment" / "Dockerfile").write_text(
        render_ladder_base_dockerfile(BASE_IMAGE), encoding="utf-8"
    )
    (skel / "task.toml").write_text(render_unsolv_task_toml(language="python"), encoding="utf-8")
    hidden = hidden_test()
    (skel / "tests" / "hidden" / HIDDEN_REL).parent.mkdir(parents=True, exist_ok=True)
    (skel / "tests" / "hidden" / HIDDEN_REL).write_text(hidden.content, encoding="utf-8")
    test_sh = render_hidden_test_sh([hidden], ("tests/test_itsdangerous",), test_cmd=spec.test_cmd)
    (skel / "tests" / "test.sh").write_text(test_sh, encoding="utf-8")
    (skel / "tests" / "test.sh").chmod(0o755)
    patches = write_patches(authored, base_text)
    shutil.copy2(patches["gold"], skel / "tests" / "gold.patch")
    shutil.copy2(patches["cheat"], skel / "tests" / "cheat.patch")
    # The excised environment/src the preflight and solver run against.
    materialize_task(cfg, base_tree, REPO, FAMILY, skel)
    for pycache in (skel / "environment" / "src").rglob("__pycache__"):
        shutil.rmtree(pycache, ignore_errors=True)

    bugreport = (authored / "_author" / "bugreport.md").read_text(encoding="utf-8")
    contract = (authored / "_author" / "contract.md").read_text(encoding="utf-8")
    dest_root = cfg.tasks_dir / REPO
    out = build_affordance_levels(
        skel,
        [hidden],
        levels=(-2, 0),
        dest_root=dest_root,
        family=FAMILY,
        instruction_a0=contract,
        instructions={-2: bugreport},
        language="python",
        test_cmd=spec.test_cmd,
        dockerfile_from=BASE_IMAGE,
        packages=("tests/test_itsdangerous",),
        changed_symbols=("derive_key", "get_signature", "verify_signature"),
        changed_files=(str(UNIT_DIR),),
        name_scheme="L",
    )
    return {str(level): str(path) for level, path in out.items()}


if __name__ == "__main__":
    raise SystemExit(build_python_unit())
