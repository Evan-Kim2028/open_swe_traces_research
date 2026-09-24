from __future__ import annotations

import pytest

from openswe_traces.synth.xrepo_pairs import (
    UNITS,
    blank_imports,
    build_buggy_file,
    stub_go_func,
    unit_for,
)

GO_SRC = """package dep

import (
	"fmt"
	"slices"
)

func alpha(x int) int {
	if x > 0 {
		return x + 1
	}
	return x
}

func beta() []int {
	return slices.Clone([]int{1})
}
"""


def test_stub_go_func_replaces_body() -> None:
    out = stub_go_func(GO_SRC, "alpha")
    assert 'panic("excised: alpha")' in out
    assert "return x + 1" not in out
    assert "func alpha(x int) int {" in out  # signature preserved
    assert "func beta()" in out  # sibling untouched


def test_stub_go_func_unknown() -> None:
    with pytest.raises(KeyError):
        stub_go_func(GO_SRC, "gamma")


def test_blank_imports() -> None:
    out = blank_imports(GO_SRC, ["slices"])
    assert '\t_ "slices"' in out
    assert '"fmt"' in out


def test_build_buggy_file_stubs_and_blanks() -> None:
    out = build_buggy_file(GO_SRC, ["alpha", "beta"], ["slices"])
    assert out.count('panic("excised:') == 2
    assert '\t_ "slices"' in out


def test_units_declared() -> None:
    fams = [u.family for u in UNITS]
    assert fams == ["condreq", "fieldcmp", "oneofuniq"]
    for u in UNITS:
        assert u.dep_funcs, u.family
        assert u.hidden_relpath.endswith("_test.go")
        assert all(f.startswith("deps/validator/") for f in u.changed_files)
        assert unit_for(u.family) is u
    with pytest.raises(KeyError):
        unit_for("nope")
