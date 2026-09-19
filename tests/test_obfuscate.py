from __future__ import annotations

import subprocess
from pathlib import Path

from openswe_traces.synth.obfuscate import (
    NEW_MODULE,
    OLD_MODULE,
    identity_pass,
    obfuscate_task,
)

GO_MOD = f"""module {OLD_MODULE}

go 1.23
"""

GOLD_GO = """package codec

// EncodeKey packs a TiKV user key for the PD-backed cluster.
func EncodeKey(k []byte) []byte {
	return digest(k)
}

func digest(k []byte) []byte {
	out := make([]byte, 1+len(k))
	out[0] = 1
	copy(out[1:], k)
	return out
}
"""

BUGGY_GO = """package codec

// EncodeKey packs a TiKV user key for the PD-backed cluster.
func EncodeKey(k []byte) []byte {
	return k
}

func digest(k []byte) []byte {
	out := make([]byte, 1+len(k))
	out[0] = 1
	copy(out[1:], k)
	return out
}
"""

TEST_GO = """package codec

import "testing"

func TestRoundTrip(t *testing.T) {
	const keep = "sentinel-keep"
	if keep != "sentinel-keep" {
		t.Fatal("untouched")
	}
	got := EncodeKey([]byte("ab"))
	if string(got) != "\\x01ab" {
		t.Fatalf("got %q", got)
	}
}
"""

README = """# tikv client-go

Copyright PingCAP, Inc. Talks to TiKV and TiDB.
"""

INSTRUCTION = """# Missing behavior

User keys must round-trip through the codec with a one-byte prefix.

Reproduce with:

```
go test -count=1 -timeout 15m ./codec/...
```

Implement the missing behavior so these tests pass.
"""

TASK_TOML = """schema_version = "1.3"

[metadata]
category = "software-engineering"
tags = ["go", "bugfix"]

[verifier]
network_mode = "no-network"
timeout_sec = 1800.0

[agent]
network_mode = "allowlist"
allowed_hosts = ["cursor.com", "*.cursor.com", "*.cursor.sh", "downloads.cursor.com"]
timeout_sec = 14400.0

[environment]
build_timeout_sec = 1800.0
network_mode = "public"
"""

DOCKERFILE = """FROM golang:1.23
WORKDIR /app
COPY src/ /app/
"""

TEST_SH = """#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
if go test -count=1 -timeout 15m -run '^(TestRoundTrip)$' ./codec/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
"""


def _write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text)


def _diff(old: Path, new: Path) -> str:
    proc = subprocess.run(
        ["diff", "-u", str(old), str(new)],
        capture_output=True,
        text=True,
        check=False,
    )
    lines: list[str] = []
    for ln in (proc.stdout or "").splitlines():
        if ln.startswith("--- "):
            lines.append("--- a/codec/codec.go")
            continue
        if ln.startswith("+++ "):
            lines.append("+++ b/codec/codec.go")
            continue
        lines.append(ln)
    body = "\n".join(lines)
    return "diff --git a/codec/codec.go b/codec/codec.go\n" + body + "\n"


def test_obfuscate_tiny_go_module(tmp_path: Path) -> None:
    upstream = tmp_path / "upstream"
    _write(upstream / "codec" / "codec.go", GOLD_GO)
    _write(upstream / "codec" / "codec_test.go", TEST_GO)
    _write(upstream / "go.mod", GO_MOD)

    task = tmp_path / "task"
    src = task / "environment" / "src"
    _write(src / "go.mod", GO_MOD)
    _write(src / "README.md", README)
    _write(src / "codec" / "codec.go", BUGGY_GO)
    _write(src / "codec" / "codec_test.go", TEST_GO)
    _write(task / "instruction.md", INSTRUCTION)
    _write(task / "task.toml", TASK_TOML)
    _write(task / "environment" / "Dockerfile", DOCKERFILE)
    _write(task / "tests" / "test.sh", TEST_SH)
    (task / "tests" / "test.sh").chmod(0o755)

    gold_file = tmp_path / "gold.go"
    bug_file = tmp_path / "bug.go"
    gold_file.write_text(GOLD_GO)
    bug_file.write_text(BUGGY_GO)
    bug_patch = tmp_path / "bug.patch"
    bug_patch.write_text(_diff(gold_file, bug_file))

    alt_go = GOLD_GO.replace("return digest(k)", "out := digest(k)\n\treturn out")
    alt_src = tmp_path / "alt.go"
    alt_src.write_text(alt_go)
    alt_patch = tmp_path / "alt.patch"
    alt_patch.write_text(_diff(bug_file, alt_src))

    cheat_go = BUGGY_GO.replace(
        "return k",
        'if string(k) == "ab" {\n\t\treturn []byte{1, \'a\', \'b\'}\n\t}\n\treturn k',
    )
    cheat_src = tmp_path / "cheat.go"
    cheat_src.write_text(cheat_go)
    cheat_patch = tmp_path / "cheat.patch"
    cheat_patch.write_text(_diff(bug_file, cheat_src))

    dest = tmp_path / "task-obf"
    result = obfuscate_task(
        task,
        dest,
        bug_patch=bug_patch,
        alt_patch=alt_patch,
        cheat_patch=cheat_patch,
        upstream=upstream,
        skip_docker=True,
    )
    assert result.mapping_size >= 2
    ob_src = dest / "environment" / "src"
    gom = (ob_src / "go.mod").read_text()
    assert NEW_MODULE in gom
    assert OLD_MODULE not in gom
    codec = (ob_src / "codec" / "codec.go").read_text()
    assert "EncodeKey" not in codec
    assert "TiKV" not in codec
    assert "PD" not in codec
    assert "digest" not in codec
    readme = (ob_src / "README.md").read_text()
    assert "TiKV" not in readme
    assert "PingCAP" not in readme
    assert "tikv" not in readme.lower()
    test = (ob_src / "codec" / "codec_test.go").read_text()
    assert "sentinel-keep" in test
    instr = (dest / "instruction.md").read_text()
    assert NEW_MODULE in instr or "go test" in instr
    assert "Do NOT use web search" in instr
    toml = (dest / "task.toml").read_text()
    assert 'network_mode = "allowlist"' in toml
    assert 'network_mode = "no-network"' in toml
    tidy = subprocess.run(
        ["go", "mod", "tidy"],
        cwd=ob_src,
        capture_output=True,
        text=True,
        check=False,
        env={
            **dict(__import__("os").environ),
            "GOPROXY": "off",
            "GOSUMDB": "off",
            "GOFLAGS": "-mod=mod",
        },
    )
    assert tidy.returncode == 0, tidy.stderr
    assert (dest / "mapping.json").is_file()
    assert (dest / "patches" / "gold.patch").is_file()


def test_identity_pass_module_and_brands_only(tmp_path: Path) -> None:
    src = tmp_path / "src"
    _write(
        src / "go.mod",
        "module github.com/helm/helm/v3\n\ngo 1.23\n",
    )
    _write(
        src / "pkg/util.go",
        'package pkg\n\n// Helm chart helper used by helm CLI.\nfunc EncodeKey(k []byte) []byte { return k }\n',
    )
    _write(
        src / "pkg/util_test.go",
        'package pkg\n\nimport "testing"\n\nfunc TestKeep(t *testing.T) {\n\tif "sentinel-keep" != "sentinel-keep" {\n\t\tt.Fatal("untouched")\n\t}\n}\n',
    )
    _write(src / "README.md", "# helm\n\nThe Helm project.\n")
    result = identity_pass(
        src,
        new_module="example.internal/chartkit",
        brand_pairs=(("Helm", "ChartKit"), ("helm", "chartkit")),
    )
    gom = (src / "go.mod").read_text()
    assert "example.internal/chartkit/v3" in gom
    assert "github.com/helm/helm" not in gom
    util = (src / "pkg/util.go").read_text()
    assert "func EncodeKey" in util
    assert "Helm" not in util
    assert "helm" not in util
    assert "ChartKit" in util
    readme = (src / "README.md").read_text()
    assert "helm" not in readme.lower()
    assert "sentinel-keep" in (src / "pkg/util_test.go").read_text()
    assert result["modules"]["github.com/helm/helm/v3"] == "example.internal/chartkit/v3"


def test_map_modules_keeps_nested_suffix() -> None:
    from openswe_traces.synth.obfuscate import map_modules

    mapping = map_modules(
        ["knative.dev/client", "knative.dev/client/pkg"],
        "example.internal/eventkit",
    )
    assert mapping["knative.dev/client"] == "example.internal/eventkit"
    assert mapping["knative.dev/client/pkg"] == "example.internal/eventkit/pkg"
    mapping = map_modules(
        ["goa.design/goa/v3", "goa.design/goa/v3/jsonrpc/integration_tests"],
        "example.internal/apikit",
    )
    assert mapping["goa.design/goa/v3"] == "example.internal/apikit/v3"
    assert (
        mapping["goa.design/goa/v3/jsonrpc/integration_tests"]
        == "example.internal/apikit/v3/jsonrpc/integration_tests"
    )


def test_strip_go_file_keeps_imports_when_comment_has_paren() -> None:
    from openswe_traces.synth.obfuscate import _strip_go_file

    src = """package main

import (
	// note (see argoproj docs)
	"github.com/argoproj/pkg/stats"
)

func main() {}
"""
    out, _hits = _strip_go_file(src, set(), (("argoproj", "flowkit"),))
    assert "github.com/argoproj/pkg/stats" in out
    assert "github.com/flowkit/pkg/stats" not in out

