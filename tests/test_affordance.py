from __future__ import annotations

from pathlib import Path

from openswe_traces.synth.affordance import (
    HiddenTest,
    a1_appendix,
    build_affordance_levels,
    coerce_hidden_test,
    instruction_for_level,
    render_hidden_test_sh,
    render_unsolv_task_toml,
    strip_go_func,
    stub_expo,
    with_no_web,
)
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE


def _mini_task(tmp_path: Path) -> Path:
    task = tmp_path / "demo-A0"
    src = task / "environment" / "src" / "retry"
    src.mkdir(parents=True)
    (src / "backoff.go").write_text("package retry\n\nfunc expo(base, cap, n int) int { return base }\n")
    (task / "instruction.md").write_text("# Missing behavior\n\nWaits must grow.\n")
    (task / "task.toml").write_text(render_unsolv_task_toml())
    (task / "environment" / "Dockerfile").write_text("FROM golang:1.23\n")
    hidden_go = """package retry

import "testing"

func TestExpo(t *testing.T) {
	if expo(2, 8, 1) != 4 {
		t.Fatal("no")
	}
}

func TestCap(t *testing.T) {
	if expo(2, 4, 8) != 4 {
		t.Fatal("no")
	}
}
"""
    dest = task / "tests" / "hidden" / "retry" / "backoff_test.go"
    dest.parent.mkdir(parents=True)
    dest.write_text(hidden_go)
    return task


def test_build_affordance_levels_a0_to_a4(tmp_path: Path) -> None:
    task = _mini_task(tmp_path)
    hidden = [
        coerce_hidden_test("retry/backoff_test.go", task_dir=task),
        HiddenTest(
            relpath="retry/extra_test.go",
            content="package retry\n\nimport \"testing\"\n\nfunc TestExtra(t *testing.T) {}\n",
            one_liner="extra adversarial cases",
            test_names=("TestExtra",),
        ),
    ]
    stubs = {
        "retry/backoff.go": (
            "package retry\n\n"
            "// expo is min(cap, base*2^n).\n"
            "func expo(base, cap, n int) int { return base }\n"
        )
    }
    out = build_affordance_levels(
        task,
        hidden,
        dest_root=tmp_path / "levels",
        family="demo",
        instruction_a0="Waits must grow. expected 4 actual 2.\n\nReproduce with:\n\n```\ngo test -count=1 -timeout 15m ./retry/\n```\n",
        stubs=stubs,
        representative="retry/backoff_test.go",
        packages=("retry",),
        changed_symbols=("expo",),
        changed_files=("backoff.go",),
    )
    assert set(out) == {0, 1, 2, 3, 4}
    a0 = out[0]
    a1 = out[1]
    a2 = out[2]
    a3 = out[3]
    a4 = out[4]
    assert not (a0 / "environment" / "src" / "retry" / "backoff_test.go").exists()
    assert (a0 / "tests" / "hidden" / "retry" / "backoff_test.go").is_file()
    instr0 = (a0 / "instruction.md").read_text()
    assert NO_WEB_CLAUSE in instr0
    assert "TestExpo" not in instr0
    instr1 = (a1 / "instruction.md").read_text()
    assert "`TestExpo`" in instr1
    assert "extra adversarial cases" in instr1
    assert "expo is min" in (a2 / "environment" / "src" / "retry" / "backoff.go").read_text()
    assert not (a2 / "environment" / "src" / "retry" / "backoff_test.go").exists()
    assert (a3 / "environment" / "src" / "retry" / "backoff_test.go").is_file()
    assert not (a3 / "environment" / "src" / "retry" / "extra_test.go").exists()
    assert (a4 / "environment" / "src" / "retry" / "backoff_test.go").is_file()
    assert (a4 / "environment" / "src" / "retry" / "extra_test.go").is_file()
    toml = (a0 / "task.toml").read_text()
    assert 'network_mode = "allowlist"' in toml
    assert 'network_mode = "no-network"' in toml
    sh = (a0 / "tests" / "test.sh").read_text()
    assert "install_hidden" in sh
    assert "TestExpo" in sh
    assert "TestExtra" in sh


def test_hidden_test_sh_checksums_and_race() -> None:
    hidden = [
        HiddenTest(
            relpath="pkg/a_test.go",
            content="package pkg\nfunc TestA() {}\n",
            one_liner="a",
            test_names=("TestA",),
        )
    ]
    sh = render_hidden_test_sh(hidden, ("pkg",), race=True, perf_bench="BenchmarkA", perf_limit_ns=100.0)
    assert "go test -race" in sh
    assert "BenchmarkA" in sh
    assert "sha256sum -c" in sh


def test_strip_go_func_and_stub_expo() -> None:
    src = (
        "package p\n\nfunc keep() {}\n\n"
        "func (s *suite) TestGone() {\n\tx := 1\n\tif x > 0 {\n\t\tx++\n\t}\n}\n\n"
        "func after() {}\n"
    )
    out = strip_go_func(src, "func (s *suite) TestGone(")
    assert "TestGone" not in out
    assert "func keep()" in out
    assert "func after()" in out
    gold = "func expo(base, cap, n int) int {\n\treturn int(math.Min(float64(cap), float64(base)*math.Pow(2.0, float64(n))))\n}"
    assert "return base" in stub_expo(gold)


def test_instruction_levels_append_names() -> None:
    hidden = [
        HiddenTest("a_test.go", "func TestFoo() {}", "foo cases", ("TestFoo",)),
    ]
    a0 = instruction_for_level("bare contract\n", hidden, 0)
    a1 = instruction_for_level("bare contract\n", hidden, 1)
    assert "TestFoo" not in a0
    assert "`TestFoo`" in a1
    assert NO_WEB_CLAUSE in with_no_web("x")
    assert "foo cases" in a1_appendix(hidden)


def test_negative_levels_keep_base_and_name_dirs(tmp_path: Path) -> None:
    task = _mini_task(tmp_path)
    hidden = [coerce_hidden_test("retry/backoff_test.go", task_dir=task)]
    gapped = "Waits grow except we never said they cap. expected 4 actual 2.\n\nReproduce with:\n\n```\ngo test ./retry/\n```\n"
    report = (
        "A wait stayed at 2 after the second attempt. expected 4, actual 2.\n\n"
        "Reproduce with:\n\n```\ngo test ./retry/\n```\n"
    )
    out = build_affordance_levels(
        task,
        hidden,
        levels=(-2, -1),
        dest_root=tmp_path / "deep",
        family="demo",
        instruction_a0=gapped,
        packages=("retry",),
        changed_symbols=("expo",),
        changed_files=("backoff.go",),
    )
    assert set(out) == {-2, -1}
    assert out[-1].name == "demo-A-1"
    assert out[-2].name == "demo-A-2"
    assert "TestExpo" not in (out[-1] / "instruction.md").read_text()
    assert "TestExpo" not in (out[-2] / "instruction.md").read_text()
    assert NO_WEB_CLAUSE in (out[-1] / "instruction.md").read_text()
    (out[-2] / "instruction.md").write_text(
        instruction_for_level(report, hidden, -2), encoding="utf-8"
    )
    assert "TestExpo" not in (out[-2] / "instruction.md").read_text()
    assert "expected 4" in (out[-2] / "instruction.md").read_text()


def test_l_scheme_dirs_and_per_level_instructions(tmp_path: Path) -> None:
    from openswe_traces.synth.affordance import dest_dir_name, render_ladder_base_dockerfile

    assert dest_dir_name("unit", -2, name_scheme="L") == "unit-L0"
    assert dest_dir_name("unit", 0, name_scheme="L") == "unit-L2"
    assert "ladder-base:client-go-obf" in render_ladder_base_dockerfile()
    task = _mini_task(tmp_path)
    hidden = [coerce_hidden_test("retry/backoff_test.go", task_dir=task)]
    report = "A wait stayed at 2. expected 4, actual 2.\n\nReproduce with:\n\n```\ngo test ./retry/\n```\n"
    contract = "Waits must grow. expected 4 actual 2.\n\nReproduce with:\n\n```\ngo test ./retry/\n```\n"
    out = build_affordance_levels(
        task,
        hidden,
        levels=(-2, 0),
        dest_root=tmp_path / "ladder",
        family="demo-obf",
        instruction_a0=contract,
        packages=("retry",),
        changed_symbols=("expo",),
        changed_files=("backoff.go",),
        name_scheme="L",
        dockerfile_from="ladder-base:client-go-obf",
        instructions={-2: report, 0: contract},
    )
    assert out[-2].name == "demo-obf-L0"
    assert out[0].name == "demo-obf-L2"
    assert "stayed at 2" in (out[-2] / "instruction.md").read_text()
    assert "Waits must grow" in (out[0] / "instruction.md").read_text()
    assert "TestExpo" not in (out[-2] / "instruction.md").read_text()
    docker = (out[-2] / "environment" / "Dockerfile").read_text()
    assert docker.startswith("FROM ladder-base:client-go-obf")
    toml = (out[-2] / "task.toml").read_text()
    assert 'network_mode = "no-network"' in toml
    assert "cursor.com" in toml
