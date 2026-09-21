"""Package Harbor L1 (A-1) gapped-contract variants for the L0→L2 flip units.

For each unit that went 0/3 at L0 and passed at L2, build two L1 dirs from the
preflight-PASS L2 task: omit the L0-failing commitment (binding; predicted fail)
or a different commitment the L0 attempts already satisfied (other; predicted pass).
Hidden tests are copied unchanged from L2; ``tests/test.sh`` is preserved.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
from collections.abc import Sequence
from dataclasses import asdict, dataclass
from pathlib import Path

from openswe_traces.synth.affordance import (
    HiddenTest,
    OmittedInvariant,
    build_affordance_levels,
    coerce_hidden_test,
    omit_invariant,
)

L2_ROOT = Path(
    "/home/evan/Documents/oswt-closureK/experiments/pipeline/tasks_composerver"
)
L0_ROOT = Path(
    "/home/evan/Documents/open_swe_traces_research/experiments/dose_response/sweep_L0"
)
DEST_ROOT = Path(
    "/home/evan/Documents/open_swe_traces_research/experiments/dose_response/sweep_L1"
)


@dataclass(frozen=True)
class UnitGap:
    repo: str
    unit: str
    l0_assertion: str
    l0_evidence: str
    l0_rate: str
    l2_rate: str
    binding_row: str | None
    binding_prose: str
    other_row: str | None
    other_prose: str
    binding_row_note: str = ""
    other_row_note: str = ""

    @property
    def family(self) -> str:
        return f"{self.repo}-{self.unit}"

    @property
    def l2_dir(self) -> Path:
        return L2_ROOT / self.repo / f"{self.unit}-L2"

    @property
    def l0_dir(self) -> Path:
        return L0_ROOT / f"{self.family}-L0"


# Binding = the commitment the L0 attempts actually missed. Other = a row the
# same attempts already satisfied (so dropping it should not recreate the L0 fail).
UNITS: tuple[UnitGap, ...] = (
    UnitGap(
        repo="gin",
        unit="jsonrenders",
        l0_assertion="TestJsonpJSONCallbackProperty",
        l0_evidence="jsonp wrap mismatch (3/3)",
        l0_rate="0/3",
        l2_rate="3/3",
        binding_row="TestRenderJsonpJSON",
        binding_prose=(
            "JsonpJSON uses content type 'application/javascript; charset=utf-8': "
            "with an empty callback it writes the raw JSON; otherwise it writes "
            "JSEscapeString(callback) + '(' + json + ');'. "
        ),
        other_row="TestRenderAsciiJSON",
        other_prose=(
            "AsciiJSON uses bare 'application/json' (no charset) and rewrites every "
            "non-ASCII rune in the marshaled output as a \\uXXXX escape (lowercase hex), "
            "leaving ASCII bytes untouched. "
        ),
    ),
    UnitGap(
        repo="gin",
        unit="streamrenders",
        l0_assertion="TestDataContentLengthProperty",
        l0_evidence="empty data must not set Content-Length (3/3)",
        l0_rate="0/3",
        l2_rate="3/3",
        binding_row="TestRenderData/",
        binding_prose=(
            "Data writes Content-Length only when the payload is non-empty and then "
            "writes the bytes. "
        ),
        other_row="TestRenderRedirect",
        other_prose=(
            "Redirect validates the status code — anything outside the 3xx range is a "
            "panic, except 201 Created which is also allowed — then delegates to the "
            "standard redirect helper with the request and location. "
        ),
    ),
    UnitGap(
        repo="kops",
        unit="addonparse",
        l0_assertion="TestAddonParseEmptyInvalidProperty",
        l0_evidence=(
            "case 2 (3/3). Go rng seed 20260919: i=0,1 pad=0 pass; i=2 pad=2 spaces "
            "on a kind-Addons doc. Binding is 'Parse trims the buffer', not empty-input "
            "(empty checks would have failed earlier with 'empty %q')."
        ),
        l0_rate="0/3",
        l2_rate="3/3",
        binding_row="`TestParseAddons` |",
        binding_prose="Parse trims the buffer and loads YAML objects. ",
        other_row="TestParseAddonsEmpty",
        other_prose=(
            "Empty/whitespace/comment-only input yields an empty addons list (not an error). "
        ),
        binding_row_note=(
            "Nearest coverage row is TestParseAddons (kind Addons document parses). "
            "The trim itself is body prose, not its own table row."
        ),
    ),
    UnitGap(
        repo="kops",
        unit="issuecert",
        l0_assertion="TestIssuecertClientServerProperty",
        l0_evidence="IP SAN: [] (3/3)",
        l0_rate="0/3",
        l2_rate="3/3",
        binding_row="(clientServer)",
        binding_prose="; each alternate name is trimmed, IPs vs DNS split, empties skipped",
        other_row="(clientOneYear)",
        other_prose="Optional Validity sets NotAfter from now (UTC). ",
    ),
    UnitGap(
        repo="kops",
        unit="memfs",
        l0_assertion="TestMemFsCreateWriteProperty",
        l0_evidence=(
            "create: file already exists (2/3); 1/3 failed TestMemFsReadDirProperty instead. "
            "Binding is exclusive create (the majority miss)."
        ),
        l0_rate="0/3",
        l2_rate="3/3",
        binding_row="TestMemFsCreateFile",
        binding_prose=(
            "Create is exclusive: a node that already has contents returns exists; "
            "Write always replaces contents. "
        ),
        other_row="TestMemFsReadTree",
        other_prose=(
            "Tree listing recursively yields only leaves (a node is a leaf when it has "
            "no children), including nested files. "
        ),
        other_row_note="Not ReadDir: one L0 attempt failed that assertion.",
    ),
    UnitGap(
        repo="kops",
        unit="oidcdisc",
        l0_assertion="TestOIDCMemoryStoreListProperty",
        l0_evidence=(
            "get missing: <nil> discovery endpoint ns/n not found (2/3); 1/3 failed "
            "TestOIDCDiscoveryDocumentProperty (no oidc spec status 200). Binding is "
            "Get-missing / list-unknown-universe."
        ),
        l0_rate="0/3",
        l2_rate="3/3",
        binding_row=None,
        binding_prose=(
            "List of an unknown universe is empty, not an error. "
            "Get of a missing object returns nil, nil.\n"
        ),
        other_row="TestDiscoveryIsolation",
        other_prose=(
            "Universes do not leak: listing or OIDC in universe A never sees objects "
            "upserted in B.\n"
        ),
        binding_row_note=(
            "No coverage-table row for List-unknown / Get-missing; invariant is body-only. "
            "Other is isolation, not the 404-when-no-OIDC-spec row (that failed 1/3 at L0)."
        ),
    ),
    UnitGap(
        repo="kops",
        unit="templater",
        l0_assertion="TestTemplaterSnippetNameProperty",
        l0_evidence="snippet named mainTemplate must be rejected (3/3)",
        l0_rate="0/3",
        l2_rate="3/3",
        binding_row=None,
        binding_prose=(
            "Each snippet is parsed under its filename; colliding with `mainTemplate` "
            "is an error. "
        ),
        other_row="TestRenderIndent",
        other_prose=(
            "Indent: split on `\\n`, pad every non-first non-empty line with `indent` "
            "spaces, preserve newlines between lines. "
        ),
        binding_row_note=(
            "Hidden suite maps this to 'snippet mainTemplate rejected'; the coverage "
            "table has no reserved-name row. Body-only omit."
        ),
    ),
    UnitGap(
        repo="kops",
        unit="assetsremap",
        l0_assertion="TestAssetsRemapRegistryConvergeProperty",
        l0_evidence="case 0 oracle (3/3 at L0; L2 2/3, same assertion on the miss)",
        l0_rate="0/3",
        l2_rate="2/3",
        binding_row="MappingMultipleTimesConverges",
        binding_prose=(
            "A second pass must not double-prefix (spec assembly calls this until the "
            "cluster spec converges).\n"
        ),
        other_row="TestRemapURLPathDelimiterEscaping",
        other_prose="; commas in the escaped path become `%2C`",
    ),
)


def hidden_digest(task_dir: Path) -> str:
    """sha256 over sorted (relpath, file bytes) under tests/hidden/."""
    root = task_dir / "tests" / "hidden"
    h = hashlib.sha256()
    if not root.is_dir():
        return h.hexdigest()
    for path in sorted(p for p in root.rglob("*") if p.is_file() and not p.is_symlink()):
        rel = path.relative_to(root).as_posix()
        h.update(rel.encode())
        h.update(b"\0")
        h.update(path.read_bytes())
    return h.hexdigest()


def hidden_from_task(task_dir: Path) -> list[HiddenTest]:
    root = task_dir / "tests" / "hidden"
    if not root.is_dir():
        raise FileNotFoundError(f"no tests/hidden in {task_dir}")
    return [
        coerce_hidden_test(path.relative_to(root).as_posix(), task_dir=task_dir)
        for path in sorted(root.rglob("*_test.go"))
    ]


def _packages(hidden: Sequence[HiddenTest]) -> tuple[str, ...]:
    pkgs = sorted({str(Path(h.relpath).parent).replace("\\", "/") or "." for h in hidden})
    return tuple(pkgs) or (".",)


@dataclass(frozen=True)
class PackagedVariant:
    gap: UnitGap
    kind: str  # binding | other
    dest: Path
    omitted: OmittedInvariant
    hidden_sha256: str
    prediction: str


def package_variant(gap: UnitGap, kind: str, dest_root: Path) -> PackagedVariant:
    if kind not in {"binding", "other"}:
        raise ValueError(kind)
    l2 = gap.l2_dir
    if not l2.is_dir():
        raise FileNotFoundError(l2)
    instruction = (l2 / "instruction.md").read_text(encoding="utf-8")
    row = gap.binding_row if kind == "binding" else gap.other_row
    prose = gap.binding_prose if kind == "binding" else gap.other_prose
    gapped, omitted = omit_invariant(instruction, row_contains=row, prose=prose)
    hidden = hidden_from_task(l2)
    built = build_affordance_levels(
        l2,
        hidden,
        levels=(-1,),
        dest_root=dest_root,
        family=gap.family,
        packages=_packages(hidden),
        name_scheme="L",
        variant=kind,
        rewrite_test_sh=False,
        instructions={-1: gapped},
    )
    dest = built[-1]
    (dest / "gap.json").write_text(
        json.dumps(
            {
                "repo": gap.repo,
                "unit": gap.unit,
                "variant": kind,
                "l0_assertion": gap.l0_assertion,
                "l0_evidence": gap.l0_evidence,
                "prediction": "fail" if kind == "binding" else "pass",
                "omitted": asdict(omitted),
                "binding_row_note": gap.binding_row_note if kind == "binding" else gap.other_row_note,
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )
    return PackagedVariant(
        gap=gap,
        kind=kind,
        dest=dest,
        omitted=omitted,
        hidden_sha256=hidden_digest(dest),
        prediction="fail" if kind == "binding" else "pass",
    )


def package_all(dest_root: Path = DEST_ROOT) -> list[PackagedVariant]:
    dest_root.mkdir(parents=True, exist_ok=True)
    out: list[PackagedVariant] = []
    for gap in UNITS:
        for kind in ("binding", "other"):
            out.append(package_variant(gap, kind, dest_root))
    return out


def checksum_row(gap: UnitGap, variants: Sequence[PackagedVariant]) -> dict[str, str]:
    l0 = hidden_digest(gap.l0_dir) if gap.l0_dir.is_dir() else ""
    l2 = hidden_digest(gap.l2_dir) if gap.l2_dir.is_dir() else ""
    by_kind = {v.kind: v.hidden_sha256 for v in variants if v.gap.unit == gap.unit and v.gap.repo == gap.repo}
    binding = by_kind.get("binding", "")
    other = by_kind.get("other", "")
    equal = bool(l0 and l2 and binding and other and len({l0, l2, binding, other}) == 1)
    return {
        "unit": f"{gap.repo}/{gap.unit}",
        "l0": l0,
        "l2": l2,
        "l1binding": binding,
        "l1other": other,
        "equal": "yes" if equal else "NO",
    }


def main(argv: Sequence[str] | None = None) -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--dest", type=Path, default=DEST_ROOT)
    ap.add_argument("--json", action="store_true")
    args = ap.parse_args(list(argv) if argv is not None else None)
    packed = package_all(args.dest)
    rows = [checksum_row(g, packed) for g in UNITS]
    for v in packed:
        print(
            f"PACKAGED {v.dest.name}: omitted_row={v.omitted.original_test or '(body-only)'} "
            f"prediction={v.prediction}",
            file=sys.stderr,
        )
    print("hidden-test sha256:", file=sys.stderr)
    for row in rows:
        print(
            f"  {row['unit']}: equal={row['equal']} l0={row['l0'][:16]} l2={row['l2'][:16]} "
            f"b={row['l1binding'][:16]} o={row['l1other'][:16]}",
            file=sys.stderr,
        )
    if args.json:
        print(
            json.dumps(
                {
                    "variants": [
                        {
                            "dir": str(v.dest),
                            "kind": v.kind,
                            "prediction": v.prediction,
                            "omitted": asdict(v.omitted),
                            "hidden_sha256": v.hidden_sha256,
                        }
                        for v in packed
                    ],
                    "checksums": rows,
                },
                indent=2,
            )
        )
    bad = sum(1 for r in rows if r["equal"] != "yes")
    return 1 if bad else 0


if __name__ == "__main__":
    raise SystemExit(main())
