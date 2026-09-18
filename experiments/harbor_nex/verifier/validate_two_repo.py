"""Host validation for the client-go → integration_tests two-repo bug."""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

from openswe_traces.synth.codegraph_bugs import hops_to_test_names, parse_go_test_output
from openswe_traces.synth.harbor_tasks import snapshot_bugged_tree
from openswe_traces.synth.two_repo import stitch_impact_files

ROOT = Path("/home/evan/Documents/open_swe_traces_research")
REPO = ROOT / "experiments/codegraph_bugs/repos/client-go"
BUGS = ROOT / "experiments/codegraph_bugs/bugs/client-go-two-repo"
BASE = "190f0cce536f835b72481f7bcd0a9448bb1e5202"
OUT = ROOT / "experiments/harbor_nex/verifier"
PROGRESS = ROOT / "experiments/harbor_nex/TWO_REPO_PROGRESS.md"
GO = "/usr/local/go/bin/go"
LD = "-ldflags=-checklinkname=0"
FLAKE = "TestTiKVRecoveredFromDown"
F2P_TOP = "TestOnePC"


def _go(args: list[str], cwd: Path, timeout: int) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env["PATH"] = "/usr/local/go/bin:" + env.get("PATH", "")
    env.setdefault("GOTOOLCHAIN", "local")
    return subprocess.run(
        [GO, *args],
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=env,
    )


def _append_progress(text: str) -> None:
    with PROGRESS.open("a", encoding="utf-8") as fh:
        fh.write(text)
        if not text.endswith("\n"):
            fh.write("\n")


def _apply_on_tree(tree: Path, patch: Path) -> None:
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(patch.resolve()), "-d", str(tree)],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"apply {patch} failed: {proc.stdout}\n{proc.stderr}")


def main() -> int:
    OUT.mkdir(parents=True, exist_ok=True)
    bug = BUGS / "checkOnePC.patch"
    alt = BUGS / "checkOnePC.alt.patch"
    cheat = BUGS / "checkOnePC.cheat.patch"
    rows: dict[str, object] = {}

    with tempfile.TemporaryDirectory(prefix="two_repo_val_") as tmp:
        buggy = Path(tmp) / "buggy"
        snapshot_bugged_tree(REPO, BASE, bug, buggy)
        consumer = buggy / "integration_tests"

        print("=== library build ===", flush=True)
        lib_build = _go(["build", "./..."], buggy, 180)
        rows["library_build"] = lib_build.returncode == 0
        if lib_build.returncode != 0:
            print(lib_build.stdout[-2000:], lib_build.stderr[-2000:])
            _append_progress(f"\n- library build: FAIL\n```\n{(lib_build.stderr or lib_build.stdout)[-1500:]}\n```\n")
            Path(OUT / "two_repo_validate.json").write_text(json.dumps(rows, indent=2))
            return 1
        print("library build ok", flush=True)
        _append_progress("\n- library build: **Y**\n")

        print("=== consumer build ===", flush=True)
        cons_bin = Path(tmp) / "consumer.test"
        cons_build = _go(["test", "-c", "-o", str(cons_bin), LD, "."], consumer, 600)
        rows["consumer_build"] = cons_build.returncode == 0
        if cons_build.returncode != 0:
            print(cons_build.stdout[-3000:], cons_build.stderr[-3000:])
            _append_progress(
                f"\n- consumer build: FAIL\n```\n{(cons_build.stderr or cons_build.stdout)[-1500:]}\n```\n"
            )
            Path(OUT / "two_repo_validate.json").write_text(json.dumps(rows, indent=2))
            return 1
        print("consumer build ok", flush=True)
        _append_progress("- consumer build: **Y**\n")

        print("=== library go test ./... ===", flush=True)
        lib_test = _go(["test", LD, "-count=1", "-timeout", "14m", "./..."], buggy, 900)
        lib_out = (lib_test.stdout or "") + (lib_test.stderr or "")
        (OUT / "two_repo_library.log").write_text(lib_out)
        fail_tests, fail_pkgs = parse_go_test_output(lib_out)
        real_fails = [t for t in fail_tests if t != FLAKE]
        rows["library_test_ok"] = lib_test.returncode == 0 or (not real_fails)
        rows["library_failing"] = fail_tests
        rows["library_failing_real"] = real_fails
        print("library failing:", fail_tests, "rc", lib_test.returncode, flush=True)
        _append_progress(
            f"- library `go test ./...`: **{'Y' if rows['library_test_ok'] else 'N'}** "
            f"(fails={fail_tests or 'none'}; real={real_fails or 'none'})\n"
        )
        if real_fails:
            Path(OUT / "two_repo_validate.json").write_text(json.dumps(rows, indent=2))
            return 1

        print("=== consumer f2p TestOnePC x3 ===", flush=True)
        f2p_names: list[str] = []
        flaky = False
        last_out = ""
        for i in range(3):
            ct = _go(
                ["test", LD, "-count=1", "-timeout", "15m", "-run", f"^({F2P_TOP})$", "."],
                consumer,
                400,
            )
            out = (ct.stdout or "") + (ct.stderr or "")
            last_out = out
            (OUT / f"two_repo_f2p_{i+1}.log").write_text(out)
            names, _ = parse_go_test_output(out)
            print(f"  run {i+1} rc={ct.returncode} fail={names}", flush=True)
            if ct.returncode == 0 or not names:
                flaky = True
            if not f2p_names:
                f2p_names = names
            elif set(names) != set(f2p_names):
                flaky = True
        (OUT / "two_repo_f2p.txt").write_text(last_out)
        rows["f2p"] = f2p_names
        rows["f2p_count"] = len(f2p_names)
        rows["flaky"] = flaky
        _append_progress(
            f"- consumer f2p: {f2p_names} (n={len(f2p_names)}); flaky={flaky}\n"
        )
        if flaky or len(f2p_names) < 1:
            Path(OUT / "two_repo_validate.json").write_text(json.dumps(rows, indent=2))
            return 1

        print("=== collateral TestCommitRollback (expect PASS) ===", flush=True)
        col = _go(
            ["test", LD, "-count=1", "-timeout", "10m", "-run", "^(TestCommitRollback)$", "."],
            consumer,
            300,
        )
        col_out = (col.stdout or "") + (col.stderr or "")
        (OUT / "two_repo_collateral.log").write_text(col_out)
        col_fail, _ = parse_go_test_output(col_out)
        rows["collateral"] = col_fail
        rows["collateral_ok"] = col.returncode == 0
        _append_progress(
            f"- collateral TestCommitRollback: **{'none' if col.returncode == 0 else col_fail}**\n"
        )
        if col.returncode != 0:
            print(col_out[-2000:])
            Path(OUT / "two_repo_validate.json").write_text(json.dumps(rows, indent=2))
            return 1

        impact = stitch_impact_files(REPO, REPO / "integration_tests", "checkOnePC")
        f2p_files = {"integration_tests/1pc_test.go"}
        inside = f2p_files <= impact
        hops = hops_to_test_names(REPO, "checkOnePC", {"Test1PC", "TestOnePC"})
        rows["impact_n"] = len(impact)
        rows["f2p_inside_impact"] = inside
        rows["hops"] = hops
        _append_progress(
            f"- f2p files ⊆ stitched impact: **{'Y' if inside else 'N'}** "
            f"(impact {len(impact)} files; hops={hops})\n"
        )
        if not inside:
            Path(OUT / "two_repo_validate.json").write_text(json.dumps(rows, indent=2))
            return 1

        print("=== alt on buggy (expect PASS) ===", flush=True)
        _apply_on_tree(buggy, alt)
        alt_t = _go(
            ["test", LD, "-count=1", "-timeout", "15m", "-run", f"^({F2P_TOP})$", "."],
            consumer,
            400,
        )
        alt_out = (alt_t.stdout or "") + (alt_t.stderr or "")
        (OUT / "two_repo_alt.log").write_text(alt_out)
        rows["alt_ok"] = alt_t.returncode == 0
        _append_progress(f"- alt: **{'accept' if alt_t.returncode == 0 else 'REJECT'}**\n")
        if alt_t.returncode != 0:
            print(alt_out[-2000:])
            Path(OUT / "two_repo_validate.json").write_text(json.dumps(rows, indent=2))
            return 1

        print("=== revert alt, apply cheat (expect FAIL) ===", flush=True)
        subprocess.run(
            ["patch", "-p1", "-R", "--forward", "--batch", "-i", str(alt.resolve()), "-d", str(buggy)],
            check=True,
            capture_output=True,
        )
        _apply_on_tree(buggy, cheat)
        ch_t = _go(
            ["test", LD, "-count=1", "-timeout", "15m", "-run", f"^({F2P_TOP})$", "."],
            consumer,
            400,
        )
        ch_out = (ch_t.stdout or "") + (ch_t.stderr or "")
        (OUT / "two_repo_cheat.log").write_text(ch_out)
        rows["cheat_rejected"] = ch_t.returncode != 0
        _append_progress(
            f"- cheat: **{'reject' if ch_t.returncode != 0 else 'ACCEPT (bad)'}**\n"
        )
        if ch_t.returncode == 0:
            Path(OUT / "two_repo_validate.json").write_text(json.dumps(rows, indent=2))
            return 1

    Path(OUT / "two_repo_validate.json").write_text(json.dumps(rows, indent=2, default=str))
    print(json.dumps(rows, indent=2, default=str))
    return 0


if __name__ == "__main__":
    sys.exit(main())
