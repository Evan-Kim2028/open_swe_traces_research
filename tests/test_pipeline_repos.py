from __future__ import annotations

from pathlib import Path

from openswe_traces.synth.pipeline_repos import (
    parse_go_test_json,
    primary_mod_dir,
    render_repos_md,
    render_repos_yaml,
)


def test_parse_go_test_json_package_actions() -> None:
    text = """
{"Action":"run","Test":"TestFoo","Package":"example.internal/apikit"}
{"Action":"pass","Test":"TestFoo","Package":"example.internal/apikit","Elapsed":0.01}
{"Action":"pass","Package":"example.internal/apikit","Elapsed":0.2}
{"Action":"fail","Package":"example.internal/apikit/cmd","Elapsed":1.5}
{"Action":"skip","Package":"example.internal/apikit/docs","Elapsed":0}
not json
"""
    pkgs = parse_go_test_json(text)
    assert pkgs["example.internal/apikit"]["action"] == "pass"
    assert pkgs["example.internal/apikit/cmd"]["action"] == "fail"
    assert pkgs["example.internal/apikit/docs"]["action"] == "skip"
    assert sum(1 for r in pkgs.values() if r["action"] == "pass") == 1


def test_primary_mod_dir_picks_latest_version(tmp_path: Path) -> None:
    (tmp_path / "v60" / "go.mod").parent.mkdir(parents=True)
    (tmp_path / "v60" / "go.mod").write_text("module github.com/google/go-github/v60\n")
    (tmp_path / "v73" / "go.mod").parent.mkdir(parents=True)
    (tmp_path / "v73" / "go.mod").write_text("module github.com/google/go-github/v73\n")
    assert primary_mod_dir(tmp_path).name == "v73"


def test_render_repos_yaml_and_md() -> None:
    kept = [
        {
            "name": "goa",
            "url": "https://github.com/goadesign/goa",
            "commit": "abc123def456",
            "base_image": "ladder-base:goa",
            "packages_passing": 12,
            "notes": "parquet mid+hard=11",
        }
    ]
    dropped = [
        {
            "name": "kops",
            "url": "https://github.com/kubernetes/kops",
            "drop_reason": "offline tests: 2 packages passed (need >=5)",
        }
    ]
    yaml_text = render_repos_yaml(kept, dropped)
    assert "packages_passing: 12" in yaml_text
    assert "ladder-base:goa" in yaml_text
    md = render_repos_md(kept, dropped)
    assert "`goa`" in md
    assert "2 packages passed" in md
