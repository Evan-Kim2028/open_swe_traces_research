from __future__ import annotations

import os
import shutil
import subprocess
from pathlib import Path

import pytest

from openswe_traces.data import ROOT
from openswe_traces.synth.codegraph_bugs import (
    TASK_DIFFICULTY,
    go_package,
    hops_to_files,
    hops_to_test_names,
    is_go_exported,
    parse_cover_func,
    parse_go_test_output,
    plausible_fix_sites,
    rank_candidates,
    select_go_hosts,
    shortest_caller_path,
)

FIXTURE = ROOT / "experiments" / "codegraph_bugs" / "fixture_host"


def test_go_package_and_export() -> None:
    assert go_package("internal/locate/region_cache.go") == "internal/locate"
    assert go_package("mathx/mathx.go") == "mathx"
    assert is_go_exported("Add")
    assert not is_go_exported("add")
    assert is_go_exported("Backoff", False)


def test_parse_cover_func(tmp_path: Path) -> None:
    p = tmp_path / "cover.func"
    p.write_text(
        "fixturehost/mathx/mathx.go:4:\tAdd\t100.0%\n"
        "fixturehost/mathx/mathx.go:9:\tClamp\t50.0%\n"
        "total:\t\t(statements)\t80.0%\n"
    )
    cover = parse_cover_func(p)
    assert cover[("mathx.go", "Add")] == 100.0
    assert cover[("fixturehost/mathx/mathx.go", "Clamp")] == 50.0


def test_parse_go_test_output() -> None:
    text = (
        "--- FAIL: TestAdd (0.00s)\n"
        "--- FAIL: TestSumClamped (0.00s)\n"
        "FAIL\tfixturehost/mathx\t0.002s\n"
        "FAIL\tfixturehost/app\t0.001s\n"
        "ok  \tfixturehost/other\t0.001s\n"
    )
    tests, pkgs = parse_go_test_output(text)
    assert tests == ["TestAdd", "TestSumClamped"]
    assert pkgs == ["fixturehost/mathx", "fixturehost/app"]


def test_select_go_hosts_from_parquet() -> None:
    if not TASK_DIFFICULTY.exists():
        pytest.skip("outputs/task_difficulty.parquet missing")
    hosts = select_go_hosts(n=2, min_mid_share=0.20)
    assert [h.repo for h in hosts] == [
        "vaskoz/dailycodingproblem-go",
        "tikv/client-go",
    ]
    assert hosts[0].mid_share >= 0.20
    assert hosts[1].n_instances >= 1


def _ensure_fixture_indexed() -> None:
    if not shutil.which("codegraph"):
        pytest.skip("codegraph CLI not on PATH")
    if not (FIXTURE / ".codegraph").exists():
        proc = subprocess.run(
            ["codegraph", "init", "-y", str(FIXTURE)],
            capture_output=True,
            text=True,
            check=False,
        )
        if proc.returncode != 0:
            pytest.skip(f"codegraph init failed: {proc.stderr}")


def test_fixture_cross_package_candidate(tmp_path: Path) -> None:
    _ensure_fixture_indexed()
    env = os.environ.copy()
    env["PATH"] = "/usr/local/go/bin:" + env.get("PATH", "")
    cover_out = tmp_path / "fixture.cover"
    cover_text = subprocess.check_output(
        ["go", "test", "./...", "-count=1", f"-coverprofile={cover_out}", "-coverpkg=./..."],
        cwd=FIXTURE,
        text=True,
        env=env,
    )
    assert "FAIL" not in cover_text
    func = subprocess.check_output(
        ["go", "tool", "cover", f"-func={cover_out}"],
        cwd=FIXTURE,
        text=True,
        env=env,
    )
    func_file = tmp_path / "fixture.func"
    func_file.write_text(func)
    cover = parse_cover_func(func_file)
    ranked = rank_candidates(FIXTURE, cover=cover, top_n=10)
    names = [c.symbol for c in ranked]
    assert "Add" in names
    add = next(c for c in ranked if c.symbol == "Add")
    assert add.n_caller_packages >= 2
    assert "mathx" in add.caller_packages
    assert "app" in add.caller_packages
    hops = hops_to_files(FIXTURE, "Add", {"app/app_test.go", "mathx/mathx_test.go"})
    assert hops is not None and hops >= 1
    hops_fn = hops_to_test_names(FIXTURE, "Add", {"TestSumClamped", "TestAdd"})
    assert hops_fn is not None and hops_fn >= 1
    path = shortest_caller_path(FIXTURE, "Add", {"TestSumClamped"})
    assert path is not None
    assert path[0] == "Add"
    assert path[-1] == "TestSumClamped"
    sites = plausible_fix_sites(path)
    assert "Add" in sites
    assert "TestSumClamped" not in sites
