"""Base-tree selection when rebuilding task environments on a fresh machine."""

from __future__ import annotations

from pathlib import Path

import pytest

from openswe_traces.pipeline.config import PipelineConfig, RepoSpec
from openswe_traces.pipeline.materialize import base_tree_for


def _cfg(tmp: Path, spec: RepoSpec) -> PipelineConfig:
    return PipelineConfig(
        state_db=tmp / "state.db",
        logs_dir=tmp / "logs",
        work_dir=tmp / "work",
        authored_dir=tmp / "authored",
        tasks_dir=tmp / "tasks",
        jobs_dir=tmp / "jobs",
        results_parquet=tmp / "results.parquet",
        results_md=tmp / "results.md",
        repos=(spec,),
    )


def _tree(path: Path) -> Path:
    path.mkdir(parents=True, exist_ok=True)
    (path / "go.mod").write_text("module example.internal/x\n\ngo 1.23\n", encoding="utf-8")
    return path


def test_prefers_declared_src_tree(tmp_path: Path) -> None:
    """The `src:` tree is what the excision patches were authored against."""
    src = _tree(tmp_path / "repos2" / "gin" / "src")
    spec = RepoSpec(
        name="gin", url="https://example.invalid/gin.git", commit="cafe", src=str(src)
    )
    assert base_tree_for(spec, _cfg(tmp_path, spec)) == src


def test_declared_src_may_be_repo_relative(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    import openswe_traces.pipeline.materialize as mod

    monkeypatch.setattr(mod, "ROOT", tmp_path)
    src = _tree(tmp_path / "experiments" / "pipeline" / "repos2" / "gin" / "src")
    spec = RepoSpec(
        name="gin",
        url="https://example.invalid/gin.git",
        commit="cafe",
        src="experiments/pipeline/repos2/gin/src",
    )
    assert base_tree_for(spec, _cfg(tmp_path, spec)) == src


def test_falls_back_to_prepare_when_src_absent(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """helm/kops/goa declare a src: tree that is not on a fresh clone; prepare_repo owns those."""
    import openswe_traces.pipeline.materialize as mod

    prepared = _tree(tmp_path / "work" / "helm" / "tree")
    calls: list[str] = []

    def fake_prepare(spec: RepoSpec, cfg: PipelineConfig) -> dict[str, object]:
        calls.append(spec.name)
        return {"tree": str(prepared)}

    monkeypatch.setattr(mod, "prepare_repo", fake_prepare)
    spec = RepoSpec(
        name="helm",
        url="https://example.invalid/helm.git",
        commit="cafe",
        src=str(tmp_path / "nope" / "src"),
    )
    assert base_tree_for(spec, _cfg(tmp_path, spec)) == prepared
    assert calls == ["helm"]


def test_falls_back_when_no_src_declared(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    import openswe_traces.pipeline.materialize as mod

    prepared = _tree(tmp_path / "work" / "mathx" / "tree")
    monkeypatch.setattr(mod, "prepare_repo", lambda s, c: {"tree": str(prepared)})
    spec = RepoSpec(name="mathx", url="https://example.invalid/m.git", commit="cafe")
    assert base_tree_for(spec, _cfg(tmp_path, spec)) == prepared


def test_src_without_go_mod_is_not_used(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """A half-materialised directory must not be mistaken for a prepared tree."""
    import openswe_traces.pipeline.materialize as mod

    empty = tmp_path / "repos2" / "gin" / "src"
    empty.mkdir(parents=True)
    prepared = _tree(tmp_path / "work" / "gin" / "tree")
    monkeypatch.setattr(mod, "prepare_repo", lambda s, c: {"tree": str(prepared)})
    spec = RepoSpec(
        name="gin", url="https://example.invalid/gin.git", commit="cafe", src=str(empty)
    )
    assert base_tree_for(spec, _cfg(tmp_path, spec)) == prepared


def _task_dir(root: Path, repo: str, unit: str, level: int) -> Path:
    td = root / repo / f"{unit}-L{level}"
    td.mkdir(parents=True)
    (td / "task.toml").write_text("schema_version = \"1.3\"\n", encoding="utf-8")
    return td


def _excision(cfg: PipelineConfig, repo: str, unit: str, body: str) -> None:
    d = cfg.authored_dir / repo / unit / "_author" / "excised"
    d.mkdir(parents=True)
    (d / "excision.patch").write_text(body, encoding="utf-8")


def test_one_broken_repo_does_not_stop_the_others(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """client-go cannot be prepared on a fresh machine; helm/gin/goa/kops still must be."""
    import openswe_traces.pipeline.materialize as mod

    good_src = _tree(tmp_path / "trees" / "gin")
    (good_src / "a.go").write_text("package a\n\nfunc F() int { return 1 }\n", encoding="utf-8")
    good = RepoSpec(name="gin", url="https://e.invalid/g.git", commit="c", src=str(good_src))
    bad = RepoSpec(name="client-go", url="https://e.invalid/c.git", commit="c")

    cfg = _cfg(tmp_path, good)
    cfg = type(cfg)(**{**cfg.__dict__, "repos": (bad, good)})

    def fake_prepare(spec: RepoSpec, _cfg_: PipelineConfig) -> dict[str, object]:
        raise RuntimeError(f"docker build ladder-base:{spec.name} failed")

    monkeypatch.setattr(mod, "prepare_repo", fake_prepare)

    _task_dir(cfg.tasks_dir, "client-go", "connarray", 2)
    _task_dir(cfg.tasks_dir, "gin", "bindingdispatch", 2)
    _excision(cfg, "client-go", "connarray", "")
    _excision(
        cfg,
        "gin",
        "bindingdispatch",
        "--- a/a.go\n+++ b/a.go\n@@ -1,3 +1,3 @@\n package a\n \n-func F() int { return 1 }\n+func F() int { panic(\"excised\") }\n",
    )

    result = mod.materialize_all(cfg, roots=[cfg.tasks_dir], only_missing=True)
    assert result.n == 1
    assert set(result.failures) == {"client-go"}
    assert "ladder-base:client-go" in result.failures["client-go"]
    built = cfg.tasks_dir / "gin" / "bindingdispatch-L2" / "environment" / "src" / "a.go"
    assert "excised" in built.read_text(encoding="utf-8")
