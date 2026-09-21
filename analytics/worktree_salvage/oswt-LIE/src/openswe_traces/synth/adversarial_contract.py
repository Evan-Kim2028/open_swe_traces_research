"""Package L2-lie Harbor dirs: byte-identical to L2 except one inverted contract row.

The hidden suite still asserts gold's real behaviour. A model that believes the
lie must fail that property. Preflight (bare fail / gold pass / cheat fail)
must not change, because instruction.md is not in the image or the suite.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import sys
from dataclasses import asdict, dataclass
from pathlib import Path

SRC_ROOT = Path(
    "/home/evan/Documents/oswt-closureK/experiments/pipeline/tasks_composerver"
)
DEST_ROOT = Path(
    "/home/evan/Documents/open_swe_traces_research/experiments/dose_response/sweep_lie"
)


@dataclass(frozen=True)
class Lie:
    repo: str
    unit: str
    commitment: str
    l0_miss: str
    hidden_test: str
    prose_old: str
    prose_new: str
    row_old: str
    row_new: str

    @property
    def dest_name(self) -> str:
        return f"{self.repo}-{self.unit}-L2lie"

    @property
    def src_dir(self) -> Path:
        return SRC_ROOT / self.repo / f"{self.unit}-L2"


LIES: tuple[Lie, ...] = (
    Lie(
        repo="gin",
        unit="jsonrenders",
        commitment=(
            "JsonpJSON with a non-empty callback writes "
            "JSEscapeString(callback)+'('+json+');'"
        ),
        l0_miss="TestJsonpJSONCallbackProperty jsonp wrap mismatch (0/3 at L0, 3/3 at L2)",
        hidden_test="TestJsonpJSONCallbackProperty",
        prose_old=(
            "JsonpJSON uses content type 'application/javascript; charset=utf-8': "
            "with an empty callback it writes the raw JSON; otherwise it writes "
            "JSEscapeString(callback) + '(' + json + ');'."
        ),
        prose_new=(
            "JsonpJSON uses content type 'application/javascript; charset=utf-8': "
            "it always writes the raw JSON and never wraps it in callback(...); "
            "the callback value is ignored."
        ),
        row_old=(
            "| `TestRenderJsonpJSON(+Error,+Error2,+Fail)` | callback escaping, "
            "'(' json ');' wrapping, empty-callback passthrough, javascript content type |"
        ),
        row_new=(
            "| `TestRenderJsonpJSON(+Error,+Error2,+Fail)` | callback is ignored; "
            "output is always raw JSON with javascript content type (no wrapping) |"
        ),
    ),
    Lie(
        repo="gin",
        unit="streamrenders",
        commitment="Data writes Content-Length only when the payload is non-empty",
        l0_miss="TestDataContentLengthProperty empty data must not set Content-Length (0/3 at L0, 3/3 at L2)",
        hidden_test="TestDataContentLengthProperty",
        prose_old=(
            "Data writes Content-Length only when the payload is non-empty and then writes the bytes."
        ),
        prose_new=(
            "Data always writes Content-Length (including when the payload is empty, as \"0\") "
            "and then writes the bytes."
        ),
        row_old=(
            "| `TestRenderData/TestRenderDataContentLength/TestRenderDataError` | "
            "bytes written with Content-Length only when non-empty |"
        ),
        row_new=(
            "| `TestRenderData/TestRenderDataContentLength/TestRenderDataError` | "
            "bytes written with Content-Length always set, including \"0\" for empty payloads |"
        ),
    ),
    Lie(
        repo="kops",
        unit="addonparse",
        commitment="Empty/whitespace/comment-only input yields an empty addons list (not an error)",
        l0_miss="TestAddonParseEmptyInvalidProperty empty/invalid input (0/3 at L0, 3/3 at L2)",
        hidden_test="TestAddonParseEmptyInvalidProperty",
        prose_old=(
            "Empty/whitespace/comment-only input yields an empty addons list (not an error)."
        ),
        prose_new="Empty/whitespace/comment-only input is an error.",
        row_old="| `TestParseAddonsEmpty` | empty input is not an error |",
        row_new="| `TestParseAddonsEmpty` | empty input is an error |",
    ),
    Lie(
        repo="kops",
        unit="issuecert",
        commitment="Alternate names are trimmed, then split into IP vs DNS SANs; empties skipped",
        l0_miss="TestIssuecertClientServerProperty IP SAN: [] on ' 10.0.0.1 ' (0/3 at L0, 3/3 at L2)",
        hidden_test="TestIssuecertClientServerProperty",
        prose_old=(
            "Subject is copied; each alternate name is trimmed, IPs vs DNS split, empties skipped."
        ),
        prose_new=(
            "Subject is copied; each alternate name is used as-is without trimming — "
            "a token is an IP SAN only if the entire string parses as an IP, otherwise "
            "it is a DNS SAN (empty strings are kept as empty DNS names)."
        ),
        row_old="| `TestIssueCert` (clientServer) | DNS + IP SANs; both client and server EKU |",
        row_new=(
            "| `TestIssueCert` (clientServer) | untrimmed alternate names: "
            "whitespace-padded IPs are DNS SANs; both client and server EKU |"
        ),
    ),
    Lie(
        repo="kops",
        unit="memfs",
        commitment="Create is exclusive: a second Create on an existing node returns exists",
        l0_miss="TestMemFsCreateWriteProperty create: file already exists (0/3 at L0, 3/3 at L2)",
        hidden_test="TestMemFsCreateWriteProperty",
        prose_old=(
            "Create is exclusive: a node that already has contents returns exists; "
            "Write always replaces contents."
        ),
        prose_new=(
            "Create is not exclusive: a second Create overwrites contents the same way "
            "Write does; it does not return exists."
        ),
        row_old="| `TestMemFsCreateFile` | exclusive create vs overwrite write; exists on second create |",
        row_new="| `TestMemFsCreateFile` | create overwrites like write; a second create succeeds |",
    ),
    Lie(
        repo="kops",
        unit="oidcdisc",
        commitment="OIDC discovery document is 404 unless at least one endpoint has an OIDC spec",
        l0_miss=(
            "TestOIDCDiscoveryDocumentProperty no oidc spec status 200 (1/3 at L0); "
            "the other 2/3 missed Get-missing. L2 3/3."
        ),
        hidden_test="TestOIDCDiscoveryDocumentProperty",
        prose_old=(
            "The document is 404 unless at least one endpoint in the universe has an OIDC spec;"
        ),
        prose_new=(
            "The document is 200 even when no endpoint in the universe has an OIDC spec "
            "(an empty/default discovery document);"
        ),
        row_old=(
            "| `TestOIDCDiscovery` | well-known document issuer/jwks_uri and 404 when no OIDC spec |"
        ),
        row_new=(
            "| `TestOIDCDiscovery` | well-known document issuer/jwks_uri and 200 with a "
            "default document when no OIDC spec |"
        ),
    ),
    Lie(
        repo="kops",
        unit="templater",
        commitment="A snippet whose filename is mainTemplate is rejected",
        l0_miss="TestTemplaterSnippetNameProperty snippet named mainTemplate must be rejected (0/3 at L0, 3/3 at L2)",
        hidden_test="TestTemplaterSnippetNameProperty",
        prose_old=(
            "Each snippet is parsed under its filename; colliding with `mainTemplate` is an error."
        ),
        prose_new=(
            "Each snippet is parsed under its filename; a snippet named `mainTemplate` is "
            "accepted and replaces the main template."
        ),
        row_old="| `TestRenderSnippet` | include named snippet |",
        row_new=(
            "| `TestRenderSnippet` | a snippet named mainTemplate is valid and overrides "
            "the main template |"
        ),
    ),
    Lie(
        repo="kops",
        unit="assetsremap",
        commitment="Commas in a remapped file-repository path are percent-encoded as %2C",
        l0_miss=(
            "L0 0/3 was TestAssetsRemapRegistryConvergeProperty (and 1/3 of L2 still failed it). "
            "Not used: pick a property the 2/3 L2 passes demonstrably got right."
        ),
        hidden_test="TestAssetsRemapFileContractTableProperty",
        prose_old=(
            "If a file repository is set, join repository path with the canonical path; "
            "commas in the escaped path become `%2C`."
        ),
        prose_new=(
            "If a file repository is set, join repository path with the canonical path; "
            "commas in the path are left as literal commas (not percent-encoded)."
        ),
        row_old="| `TestRemapURLPathDelimiterEscaping` | commas in file paths are `%2C` |",
        row_new="| `TestRemapURLPathDelimiterEscaping` | commas in file paths stay as literal commas |",
    ),
)


def dir_digest(root: Path) -> str:
    """sha256 over sorted (relpath, file-sha256); content only, skip symlinks."""
    h = hashlib.sha256()
    if not root.is_dir():
        return h.hexdigest()
    for path in sorted(root.rglob("*")):
        if not path.is_file() or path.is_symlink():
            continue
        rel = path.relative_to(root).as_posix()
        h.update(rel.encode())
        h.update(b"\0")
        h.update(hashlib.sha256(path.read_bytes()).digest())
    return h.hexdigest()


def apply_lie(text: str, lie: Lie) -> str:
    """Rewrite one prose sentence and its coverage-table row. Each must match once."""
    for label, old, new in (
        ("prose", lie.prose_old, lie.prose_new),
        ("row", lie.row_old, lie.row_new),
    ):
        n = text.count(old)
        if n != 1:
            raise ValueError(
                f"{lie.repo}/{lie.unit}: {label} match count {n} (want 1)"
            )
        text = text.replace(old, new, 1)
    return text


def _link_copy(src: str, dst: str, *, follow_symlinks: bool = True) -> str:
    src_p = Path(src)
    if src_p.is_symlink() or not follow_symlinks:
        shutil.copy2(src, dst, follow_symlinks=False)
        return dst
    try:
        os.link(src, dst)
    except OSError:
        shutil.copy2(src, dst)
    return dst


def _unlink_and_write(path: Path, data: str) -> None:
    if path.exists() or path.is_symlink():
        path.unlink()
    path.write_text(data, encoding="utf-8")


def _unlink_and_copy(src: Path, dest: Path) -> None:
    if dest.exists() or dest.is_symlink():
        dest.unlink()
    shutil.copy2(src, dest)


def file_diffs(src: Path, dest: Path) -> list[str]:
    """Relative paths whose bytes differ (regular files only)."""
    diffs: list[str] = []
    src_files = {
        p.relative_to(src).as_posix()
        for p in src.rglob("*")
        if p.is_file() and not p.is_symlink()
    }
    dest_files = {
        p.relative_to(dest).as_posix()
        for p in dest.rglob("*")
        if p.is_file() and not p.is_symlink()
    }
    for rel in sorted(src_files | dest_files):
        a, b = src / rel, dest / rel
        if not a.is_file() or not b.is_file():
            diffs.append(rel)
            continue
        if a.read_bytes() != b.read_bytes():
            diffs.append(rel)
    return diffs


@dataclass
class PackageReport:
    dest_name: str
    src: str
    dest: str
    hidden_sha256: str
    src_tree_sha256: str
    hidden_match: bool
    tree_match: bool
    diffs_vs_l2: list[str]
    instruction_changed: bool

    def as_dict(self) -> dict:
        return asdict(self)


def package_one(lie: Lie, dest_root: Path = DEST_ROOT) -> PackageReport:
    src = lie.src_dir
    dest = dest_root / lie.dest_name
    if not (src / "instruction.md").is_file():
        raise FileNotFoundError(src)
    text = (src / "instruction.md").read_text(encoding="utf-8")
    lied = apply_lie(text, lie)
    if dest.exists():
        shutil.rmtree(dest)
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(src, dest, copy_function=_link_copy, symlinks=True)
    _unlink_and_write(dest / "instruction.md", lied)
    # validation.json will be rewritten by preflight; do not share the inode.
    if (src / "validation.json").is_file():
        _unlink_and_copy(src / "validation.json", dest / "validation.json")
    hidden_src = dir_digest(src / "tests" / "hidden")
    hidden_dest = dir_digest(dest / "tests" / "hidden")
    tree_src = dir_digest(src / "environment" / "src")
    tree_dest = dir_digest(dest / "environment" / "src")
    diffs = file_diffs(src, dest)
    return PackageReport(
        dest_name=lie.dest_name,
        src=str(src),
        dest=str(dest),
        hidden_sha256=hidden_dest,
        src_tree_sha256=tree_dest,
        hidden_match=hidden_src == hidden_dest,
        tree_match=tree_src == tree_dest,
        diffs_vs_l2=diffs,
        instruction_changed="instruction.md" in diffs,
    )


def package_all(
    dest_root: Path = DEST_ROOT, lies: tuple[Lie, ...] = LIES
) -> list[PackageReport]:
    return [package_one(lie, dest_root=dest_root) for lie in lies]


def source_image_tag(lie: Lie) -> str | None:
    from openswe_traces.pipeline_ext.hack_audit import (
        _image_exists,
        audit_image_tag,
    )

    env = (lie.src_dir / "environment").resolve()
    if not (env / "Dockerfile").is_file():
        return None
    tag = audit_image_tag(env)
    return tag if _image_exists(tag) else None


def preflight_one(lie: Lie, dest_root: Path = DEST_ROOT, *, force: bool = False):
    from openswe_traces.pipeline.preflight import ensure_preflight, preflight_task

    dest = dest_root / lie.dest_name
    image = source_image_tag(lie)
    if force:
        return preflight_task(dest, image=image)
    return ensure_preflight(dest, image=image, force=force)


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--dest", type=Path, default=DEST_ROOT)
    ap.add_argument("--package-only", action="store_true")
    ap.add_argument("--preflight-only", action="store_true")
    ap.add_argument("--force-preflight", action="store_true")
    ap.add_argument("--json", action="store_true")
    args = ap.parse_args(argv)
    rows: list[dict] = []
    if not args.preflight_only:
        for report in package_all(dest_root=args.dest):
            if not report.hidden_match or not report.tree_match:
                raise SystemExit(
                    f"checksum mismatch {report.dest_name}: "
                    f"hidden_match={report.hidden_match} tree_match={report.tree_match}"
                )
            extra = [p for p in report.diffs_vs_l2 if p != "instruction.md"]
            if extra:
                raise SystemExit(
                    f"unexpected diffs {report.dest_name}: {extra}"
                )
            rows.append({"stage": "package", **report.as_dict()})
            print(
                f"PACKAGED {report.dest_name} hidden={report.hidden_sha256[:16]} "
                f"tree={report.src_tree_sha256[:16]} diffs={report.diffs_vs_l2}",
                file=sys.stderr,
                flush=True,
            )
    if not args.package_only:
        from openswe_traces.pipeline.preflight import preflight_task

        for lie in LIES:
            dest = args.dest / lie.dest_name
            image = source_image_tag(lie)
            report = preflight_task(dest, image=image)
            payload = report.as_dict() | {
                "stage": "preflight",
                "dest_name": lie.dest_name,
                "reused_image": image or "",
            }
            rows.append(payload)
            mark = "PASS" if report.verdict == "pass" else "FAIL"
            gold = report.gold.verdict if report.gold else "absent"
            cheat = report.cheat.verdict if report.cheat else "absent"
            print(
                f"{mark} {lie.dest_name}: bare={report.bare.verdict} "
                f"gold={gold} cheat={cheat} ({report.seconds:.0f}s) "
                f"image={report.image}",
                file=sys.stderr,
                flush=True,
            )
    if args.json:
        print(json.dumps(rows, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
