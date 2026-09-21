"""Closure metrics for excised units.

A unit's *closure* is the set of functions the excision patch removes from a
base tree.  This module measures how self-contained that closure is:

- ``internal_edges``: call references *among* the removed functions (from a
  removed body to another removed function).
- ``boundary_in``: call sites in the remaining tree that reference a removed
  function (sites the solver must satisfy or keep working).
- ``boundary_out``: call references from removed bodies to symbols that remain
  in the tree (everything the removed code depended on that is still there).

``ratio = internal_edges / max(1, boundary_in + boundary_out)`` is the
hypothesis input: units whose difficulty is driven by closure size should show
a high internal-to-boundary edge ratio.

Parsing is done with ``go/ast`` through a small Go helper
(``tools/goclosure/main.go``), not regexes: function decls, call expressions,
light local-variable type inference (params, receivers, ``:=`` composite
literals, ``new``, type assertions, ``var`` decls, one level of struct-field
access) and import aliases.  No ``go/types``: the trees are obfuscated and have
no module cache, so full type checking is unavailable.  Method calls whose
receiver type cannot be resolved locally fall back to name-uniqueness matching:
if every function with that name is removed, the call is internal; if only some
are, it is counted as external (boundary).
"""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import tempfile
from pathlib import Path

TOOLS_DIR = Path(__file__).resolve().parents[3] / "tools"
GO_HELPER = TOOLS_DIR / "goclosure" / "main.go"

_HUNK_RE = re.compile(r"^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@")
_FILE_HDR_RE = re.compile(r"^\+\+\+ b?/?(.+)$")


class ClosureMetricsError(RuntimeError):
    """Raised when the excision patch cannot be applied to the base tree."""


def _run(args: list[str], *, cwd: Path | None = None, input_text: str | None = None, ok_rc: set[int] | None = None) -> str:
    env = dict(os.environ)
    env["GOTOOLCHAIN"] = "local"
    proc = subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=300,
        env=env,
        input=input_text,
        check=False,
    )
    if proc.returncode not in (ok_rc or {0}):
        raise ClosureMetricsError(
            f"command failed ({proc.returncode}): {' '.join(args)}\n"
            f"{(proc.stderr or proc.stdout)[-1200:]}"
        )
    return proc.stdout


# Base trees are re-parsed for every unit in a repo; each parse walks the whole
# tree through goclosure, which is the dominant cost on large repos (kops has
# ~13k .go files). Keyed by resolved path — patched copies live in unique
# tmpdirs, so they never alias a base entry.
_FACTS_CACHE: dict[str, dict] = {}


def parse_tree(root: Path) -> dict:
    """Emit per-file AST facts for a Go tree via tools/goclosure."""
    key = str(Path(root).resolve())
    if key not in _FACTS_CACHE:
        out = _run(["go", "run", str(GO_HELPER)], input_text=json.dumps({"root": key}))
        _FACTS_CACHE[key] = json.loads(out)
    return _FACTS_CACHE[key]


def rebrand_patch(patch: Path | str, rewrites: list[list[str]] | list[tuple[str, str]]) -> str:
    """Apply ordered word-boundary substitutions to a unified diff.

    Used when an excision patch was authored against a differently-branded
    obfuscation of the same commit (e.g. helm's authored tree is
    ``example.internal/chartkit/v4`` while the on-disk ``prepare_repo`` tree is
    ``example.internal/helm``). Rewrites run in order — list the fully-qualified
    module path first so a bare brand token cannot rewrite inside it.
    """
    text = patch.read_text(encoding="utf-8", errors="replace") if isinstance(patch, Path) else patch
    for old, new in rewrites:
        text = re.sub(r"\b" + re.escape(old) + r"\b", new, text)
    return text


def apply_patch(tree: Path, patch: Path | str) -> None:
    """Apply an excision patch in place (git apply, tolerant of the tree not being a repo)."""
    patch_text = patch.read_text(encoding="utf-8", errors="replace") if isinstance(patch, Path) else patch
    with tempfile.NamedTemporaryFile("w", suffix=".patch", delete=False) as fh:
        fh.write(patch_text)
        patch_file = Path(fh.name)
    try:
        _run(["git", "apply", "--whitespace=nowarn", str(patch_file)], cwd=tree)
    finally:
        patch_file.unlink(missing_ok=True)


def _patch_stats(patch: Path | str) -> tuple[int, int]:
    """(n_files, lines_removed) from a unified diff.

    lines_removed counts ``-`` content lines (excluding ``---`` headers);
    n_files counts distinct ``+++`` targets.
    """
    text = patch.read_text(encoding="utf-8", errors="replace") if isinstance(patch, Path) else patch
    files: set[str] = set()
    removed = 0
    for line in text.splitlines():
        if line.startswith("+++ "):
            m = _FILE_HDR_RE.match(line)
            if m:
                files.add(m.group(1).strip())
            continue
        if line.startswith("-") and not line.startswith("--- "):
            removed += 1
    return len(files), removed


def _func_key(pkg: str, name: str, recv: str) -> tuple[str, str, str]:
    return (pkg, name, recv)


def _index_facts(facts: dict) -> dict:
    """Build lookup structures from tree facts (see parse_tree)."""
    # The same (pkg, name, recv) can exist in several files guarded by build
    # tags (e.g. gin's binding.go vs binding_nomsgpack.go), so function
    # identities are tracked per file and only collapsed for name resolution.
    funcs_by_file: dict[tuple, dict] = {}  # (file, pkg, name, recv) -> {"hash": h}
    any_funcs: dict[tuple, set[str]] = {}  # (pkg, name, recv) -> {files}
    by_file: dict[str, dict] = facts["files"]
    types_by_pkg: dict[str, dict] = {}     # pkg -> {type_name: {"fields": {fname: ftype}}}
    files = {}
    for rel, ff in by_file.items():
        pkg = ff["pkg"]
        files[rel] = {"pkg": pkg, "imports": ff["imports"], "types": {}, "vars": {}, "funcs": {}}
        for t in ff["types"]:
            files[rel]["types"][t["name"]] = {"kind": t["kind"], "fields": {f["name"]: f["type"] for f in t["fields"]}}
            types_by_pkg.setdefault(pkg, {})[t["name"]] = files[rel]["types"][t["name"]]
        for v in ff["vars"]:
            key = (v["scope"], v["name"])
            files[rel]["vars"][key] = {"type": v["type"], "kind": v["kind"]}
        for fn in ff["funcs"]:
            key = (rel, pkg, fn["name"], fn["recv"])
            funcs_by_file[key] = {"hash": fn["body_hash"]}
            files[rel]["funcs"][(fn["name"], fn["recv"])] = {"hash": fn["body_hash"]}
            any_funcs.setdefault(_func_key(pkg, fn["name"], fn["recv"]), set()).add(rel)
    return {"funcs_by_file": funcs_by_file, "any_funcs": any_funcs, "files": files, "types_by_pkg": types_by_pkg}


def removed_funcs(base: dict, patched: dict) -> tuple[set[tuple], set[tuple]]:
    """Functions whose body the excision patch deleted or stubbed.

    Returns (file_scoped, name_scoped): file_scoped keys are
    (file, pkg, name, recv) — one entry per removed decl; name_scoped keys are
    (pkg, name, recv) — a symbol counts as removed if any of its
    build-tag-guarded variants was removed.
    """
    file_scoped: set[tuple] = set()
    for key, info in base["funcs_by_file"].items():
        p = patched["funcs_by_file"].get(key)
        if p is None or p["hash"] != info["hash"]:
            file_scoped.add(key)
    name_scoped = {k[1:] for k in file_scoped}
    return file_scoped, name_scoped


def _resolve_qual(qual: str, pkg: str, rel: str, index: dict, scope: str) -> str | None:
    """Resolve a selector base expression to a plain type name, or None.

    Handles import aliases (returns "pkg:aliasname"), local vars (via scope
    inference), type names and one level of struct-field access.
    """
    files = index["files"]
    ff = files[rel]
    types_pkg = index["types_by_pkg"]
    if qual == "":
        return None
    if "." not in qual:
        # single ident: import alias? local var? type name?
        if qual in ff["imports"]:
            return "alias:" + ff["imports"][qual].rsplit("/", 1)[-1]
        v = ff["vars"].get((scope, qual))
        if v and v["type"]:
            return v["type"]
        if qual in ff["types"]:
            return qual
        if qual in types_pkg.get(pkg, {}):
            return qual
        return None
    parts = qual.split(".")
    # first segment: alias, var, or type
    head = parts[0]
    target_pkg = pkg
    cur_type: str | None = None
    if head in ff["imports"]:
        target_pkg = ff["imports"][head].rsplit("/", 1)[-1]
    else:
        v = ff["vars"].get((scope, head))
        if v and v["type"]:
            cur_type = v["type"]
        elif head in types_pkg.get(target_pkg, {}):
            cur_type = head
        else:
            return None
    for field in parts[1:]:
        if cur_type is None:
            return None
        tinfo = types_pkg.get(target_pkg, {}).get(cur_type)
        if tinfo is None:
            return None
        ftype = tinfo["fields"].get(field)
        if ftype is None:
            return None
        cur_type = ftype
    return cur_type


def _candidates_for(callee: str, qual_type: str | None, pkg: str, index: dict, removed: set) -> set[tuple]:
    """Removed-func keys a call could reference, or empty if it cannot reference a removed func."""
    if qual_type is None:
        return {k for k in removed if k[1] == callee}
    if qual_type.startswith("alias:"):
        tp = qual_type[len("alias:"):]
        return {k for k in removed if k[0] == tp and k[1] == callee and k[2] == ""}
    # method on a concrete receiver type (same package, or any package with that type)
    return {k for k in removed if k[1] == callee and k[2] == qual_type}


def classify_call(call: dict, pkg: str, rel: str, index: dict, removed: set, all_funcs: dict) -> str:
    """Classify one call site: 'internal' (references a removed func) or 'external'."""
    callee = call["name"]
    qual = call["qual"]
    scope = (call["caller"], call["caller_recv"])
    qual_type = _resolve_qual(qual, pkg, rel, index, scope) if qual else None
    cands = _candidates_for(callee, qual_type, pkg, index, removed)
    if not cands:
        return "external"
    if qual_type is not None:
        # qualifier resolved to a concrete receiver/package: candidates are
        # already filtered to that target, so a hit is unambiguous
        return "internal"
    # unqualified or unresolved qualifier: if EVERY function with that name is
    # removed, the call is internal; if only some are, ambiguous -> external
    same_name = {k for k in all_funcs if k[1] == callee}
    if same_name and same_name <= removed:
        return "internal"
    return "external"


def _calls_in_removed_bodies(facts: dict, removed_file_scoped: set) -> list[tuple[dict, str]]:
    """Call facts from the base tree that lie inside a removed function's body."""
    out: list[tuple[dict, str]] = []
    for rel, ff in facts["files"].items():
        pkg = ff["pkg"]
        removed_ranges = []
        for fn in ff["funcs"]:
            if (rel, pkg, fn["name"], fn["recv"]) in removed_file_scoped:
                removed_ranges.append((fn["body_start"], fn["body_end"]))
        for call in ff["calls"]:
            if any(rs <= call["line"] <= re for rs, re in removed_ranges):
                out.append((call, rel))
    return out


def closure_metrics(
    base_tree: Path,
    excision_patch: Path | str,
    *,
    patch_rewrites: list[list[str]] | list[tuple[str, str]] | None = None,
) -> dict:
    """Compute closure metrics for an excised unit.

    base_tree: the tree the excision patch was authored against.
    excision_patch: unified diff that deletes or stubs the removed functions.
    patch_rewrites: optional ordered (old, new) word-boundary substitutions
        applied to the patch text before use (see rebrand_patch).
    """
    base_tree = Path(base_tree)
    patch_text = rebrand_patch(excision_patch, patch_rewrites or [])
    base_facts_raw = parse_tree(base_tree)
    base = _index_facts(base_facts_raw)

    with tempfile.TemporaryDirectory(prefix="closureA-") as tmp:
        patched_tree = Path(tmp) / "src"
        shutil.copytree(base_tree, patched_tree, symlinks=True)
        apply_patch(patched_tree, patch_text)
        patched_facts_raw = parse_tree(patched_tree)

    patched = _index_facts(patched_facts_raw)
    removed_file_scoped, removed = removed_funcs(base, patched)

    all_funcs = dict(base["any_funcs"])
    n_files, lines_removed = _patch_stats(patch_text)

    internal = 0
    boundary_out = 0
    for call, rel in _calls_in_removed_bodies(base_facts_raw, removed_file_scoped):
        pkg = base_facts_raw["files"][rel]["pkg"]
        if classify_call(call, pkg, rel, base, removed, all_funcs) == "internal":
            internal += 1
        else:
            boundary_out += 1

    boundary_in = 0
    for rel, ff in patched_facts_raw["files"].items():
        pkg = ff["pkg"]
        for call in ff["calls"]:
            if classify_call(call, pkg, rel, patched, removed, all_funcs) == "internal":
                boundary_in += 1

    return {
        "n_files": n_files,
        "n_funcs_removed": len(removed_file_scoped),
        "lines_removed": lines_removed,
        "internal_edges": internal,
        "boundary_in": boundary_in,
        "boundary_out": boundary_out,
        "ratio": round(internal / max(1, boundary_in + boundary_out), 4),
    }


def excision_patch_from_diff(base_tree: Path, excised_tree: Path) -> str:
    """Derive an excision patch by diffing a base tree against its excised copy.

    git diff --no-index emits paths relative to the two arguments, which are not
    the tree-relative paths the patch must carry; rewrite them so ``git apply``
    works from the base tree root.
    """
    diff = _run(
        ["git", "diff", "--no-index", "--no-color", str(base_tree), str(excised_tree)],
        ok_rc={0, 1},
    )
    base_tree, excised_tree = Path(base_tree).resolve(), Path(excised_tree).resolve()

    def rel_of(path: str, prefix: str, anchor: Path) -> str:
        p = Path("/") / path[len(prefix):]
        try:
            return p.relative_to(anchor).as_posix()
        except ValueError:
            return p.as_posix()

    out: list[str] = []
    for line in diff.splitlines():
        if line.startswith("--- "):
            out.append("--- a/" + rel_of(line[4:], "a/", base_tree))
        elif line.startswith("+++ "):
            out.append("+++ b/" + rel_of(line[4:], "b/", excised_tree))
        else:
            out.append(line)
    return "\n".join(out) + "\n"
