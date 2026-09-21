"""Unit tests for the shadow gate's pure functions: Go span scanning,
signature digests, response parsing, and the splice. No docker, no network."""

from __future__ import annotations

import textwrap
from pathlib import Path

import pytest

from openswe_traces.gate.shadow import (
    ShadowCode,
    apply_shadow,
    build_context,
    find_excised_files,
    go_func_spans,
    parse_response,
    signature_digest,
)

GO_FILE = textwrap.dedent('''
    package demo

    import (
    	"errors"
    	_ "sort"
    	_ "github.com/Masterminds/semver/v3"
    )

    var ErrBad = errors.New("bad")

    type Item struct {
    	Name string `json:"name"`
    }

    // Len returns length. A brace in a comment: { } must not matter.
    func (i Item) Len() int {
    	return len(i.Name) // and a "{" in a comment
    }

    func NewItem(name string) Item {
    	return Item{Name: name + "}"}
    }

    // Do excised thing.
    func (i *Item) Do(flag bool) error {
    	panic("excised: Item.Do")
    }

    func helper() {
    	panic("excised: helper")
    }

    func Pick(x interface {
    	M() int
    }, y int) int {
    	return x.M() + y
    }
''').lstrip()


def test_func_spans_finds_stubs_and_plain_funcs():
    spans = go_func_spans(GO_FILE)
    by_name = {s.name: s for s in spans}
    assert set(by_name) == {"Len", "NewItem", "Do", "helper", "Pick"}
    assert by_name["Do"].excised and by_name["helper"].excised
    assert not by_name["Len"].excised and not by_name["NewItem"].excised
    assert not by_name["Pick"].excised
    assert by_name["Do"].recv == "Item"
    assert by_name["helper"].recv == ""
    # Pick has a multi-line interface type in its signature; its body span
    # must still be right.
    pick = GO_FILE[by_name["Pick"].sig_end:by_name["Pick"].end]
    assert "return x.M() + y" in pick


def test_signature_digest_omits_bodies_keeps_stubs():
    d = signature_digest(GO_FILE)
    assert "/* body omitted */" in d
    assert 'panic("excised: Item.Do")' in d
    assert 'panic("excised: helper")' in d
    assert "return len(i.Name)" not in d
    assert "return Item{" not in d
    # decls and comments survive
    assert "type Item struct" in d
    assert "var ErrBad" in d
    assert "Len returns length" in d


def test_parse_response_fence_imports_funcs():
    resp = """Here you go.
```go
import (
	"fmt"
	"sort"
)

func (i *Item) Do(flag bool) error {
	if flag {
		return fmt.Errorf("x")
	}
	return nil
}

func helper() {
	sort.Ints(nil)
}
```"""
    sc = parse_response(resp)
    assert ("Item", "Do") in sc.funcs
    assert ("", "helper") in sc.funcs
    assert "fmt" in sc.imports and "sort" in sc.imports
    assert "return fmt.Errorf" in sc.funcs[("Item", "Do")]


def _unit(tmp_path: Path) -> Path:
    u = tmp_path / "demo-L2"
    (u / "environment" / "src" / "pkg").mkdir(parents=True)
    (u / "tests" / "hidden").mkdir(parents=True)
    (u / "environment" / "src" / "pkg" / "demo.go").write_text(GO_FILE)
    (u / "environment" / "src" / "pkg" / "demo_test.go").write_text(
        "package demo\n// test secrets the generator must never see\n")
    (u / "instruction.md").write_text("# Contract (L2) — demo\nDo the thing.\n")
    (u / "tests" / "gold.patch").write_text("diff --git a/x b/x\n")
    return u


def test_find_excised_skips_tests(tmp_path):
    u = _unit(tmp_path)
    found = find_excised_files(u / "environment" / "src")
    assert list(found) == [Path("pkg/demo.go")]


def test_context_excludes_tests_and_gold(tmp_path):
    u = _unit(tmp_path)
    ctx = build_context(u)
    sources = [m["source"] for m in ctx.manifest]
    assert "instruction.md" in sources
    assert sources == ["instruction.md", "environment/src/pkg/demo.go"]
    assert "test secrets" not in ctx.prompt
    assert "gold.patch" not in str(sources)
    assert "CONTRACT:" in ctx.prompt and "FUNCTIONS TO IMPLEMENT" in ctx.prompt
    assert ctx.excised == {"pkg/demo.go": [("Item", "Do"), ("", "helper")]}


def test_context_asserts_on_forbidden_source(tmp_path, monkeypatch):
    u = _unit(tmp_path)
    # a "hidden" test path inside environment/src must trip the assert
    bad = u / "environment" / "src" / "pkg" / "gold.go"
    bad.write_text('package pkg\nfunc x() { panic("excised: x") }\n')
    with pytest.raises(AssertionError):
        build_context(u)


def test_apply_shadow_splices_and_unblanks(tmp_path):
    src = tmp_path / "src" / "pkg"
    src.mkdir(parents=True)
    f = src / "demo.go"
    f.write_text(GO_FILE)
    impl_do = """func (i *Item) Do(flag bool) error {
	if flag {
		return errors.New("no")
	}
	sort.Ints(nil)
	return nil
}"""
    impl_h = "func helper() {\n\tsort.Strings(nil)\n}"
    sc = ShadowCode(imports=[], funcs={("Item", "Do"): impl_do, ("", "helper"): impl_h})
    rep = apply_shadow(tmp_path / "src", {Path("pkg/demo.go"): GO_FILE}, sc)
    assert rep.ok and not rep.missing
    out = f.read_text()
    assert 'panic("excised' not in out
    assert "return errors.New" in out
    assert '\t"sort"' in out            # blank import un-blanked
    assert '_ "github.com/Masterminds/semver/v3"' in out  # unused blank stays
    assert "return len(i.Name)" in out  # non-excised body preserved


def test_merge_imports_filters_unreferenced_and_phantom(tmp_path):
    """A merged import must be referenced by the file's code AND resolvable
    offline; a phantom module path is dropped, never fetched."""
    src = tmp_path / "src"
    (src / "pkg").mkdir(parents=True)
    (src / "go.mod").write_text(
        "module example.internal/demo\n\ngo 1.26\n\nrequire (\n"
        "\tgo.yaml.in/yaml/v3 v3.0.5\n)\n")
    text = ("package pkg\n\nfunc F() {\n\tpanic(\"excised: F\")\n}\n")
    f = src / "pkg" / "f.go"
    f.write_text(text)
    impl = ("func F() {\n\t_ = yaml.Marshal(1)\n\t_ = corev1.Taint{}\n}")
    sc = ShadowCode(
        imports=["fmt", "gopkg.in/yaml.v3", "go.yaml.in/yaml/v3",
                 "k8s.io/api/core/v1", "example.internal/chartkit/v4/internal/x"],
        funcs={("", "F"): impl})
    rep = apply_shadow(src, {Path("pkg/f.go"): text}, sc)
    out = f.read_text()
    assert rep.ok
    assert '"go.yaml.in/yaml/v3"' in out          # referenced + in go.mod
    # corev1.Taint{} does not match candidates {v1, core} -> unreferenced here;
    # and k8s.io/api is not in go.mod anyway
    assert '"k8s.io/api/core/v1"' not in out
    assert '"fmt"' not in out                     # not referenced
    assert '"gopkg.in/yaml.v3"' not in out        # yaml.v3 not in go.mod
    assert "chartkit" not in out                  # different module, phantom


def test_apply_shadow_reports_missing(tmp_path):
    src = tmp_path / "src" / "pkg"
    src.mkdir(parents=True)
    (src / "demo.go").write_text(GO_FILE)
    sc = ShadowCode(imports=[], funcs={("Item", "Do"): "func (i *Item) Do(flag bool) error { return nil }"})
    rep = apply_shadow(tmp_path / "src", {Path("pkg/demo.go"): GO_FILE}, sc)
    assert not rep.ok
    assert rep.missing == ["pkg/demo.go:helper"]
    # the unmatched helper the model emitted lands nowhere unless provided —
    # leftover stub stays a panic, which is exactly the defect signal we want
    out = (src / "demo.go").read_text()
    assert 'panic("excised: helper")' in out


def test_resolve_imports_rebinds_blank_to_alias(tmp_path):
    """kops-issuecert shape: the file imports ``_ "crypto/rand"`` while the
    shadow references ``crypto_rand.`` — the blank spec is rebound to the
    alias the code uses rather than left unreferenceable."""
    src = tmp_path / "src"
    (src / "pkg").mkdir(parents=True)
    text = ('package pkg\n\nimport (\n\t_ "crypto/rand"\n)\n\n'
            'func F() {\n\tpanic("excised: F")\n}\n')
    f = src / "pkg" / "f.go"
    f.write_text(text)
    impl = "func F() {\n\t_ = crypto_rand.Reader\n}"
    sc = ShadowCode(imports=["crypto/rand"], aliases={"crypto/rand": "crypto_rand"},
                    funcs={("", "F"): impl})
    rep = apply_shadow(src, {Path("pkg/f.go"): text}, sc)
    out = f.read_text()
    assert rep.ok
    assert 'crypto_rand "crypto/rand"' in out
    assert '_ "crypto/rand"' not in out


def test_resolve_imports_one_name_one_path(tmp_path):
    """kops-templater shape: two blank imports share a base name — only the
    deepest (most specific) path may claim it; the other stays blank rather
    than produce ``redeclared in this block``."""
    src = tmp_path / "src"
    (src / "pkg").mkdir(parents=True)
    text = ('package pkg\n\nimport (\n\t_ "example.internal/kops"\n'
            '\t_ "example.internal/kops/pkg/apis/kops"\n)\n\n'
            'func F() {\n\tpanic("excised: F")\n}\n')
    f = src / "pkg" / "f.go"
    f.write_text(text)
    impl = "func F() {\n\t_ = kops.FindKubernetesVersionSpec\n}"
    sc = ShadowCode(imports=[], funcs={("", "F"): impl})
    rep = apply_shadow(src, {Path("pkg/f.go"): text}, sc)
    out = f.read_text()
    assert rep.ok
    assert '"example.internal/kops/pkg/apis/kops"' in out
    assert '_ "example.internal/kops"' in out          # root stays blank
    assert out.count('"example.internal/kops/pkg/apis/kops"') == 1


def test_stdlib_fixup_respects_bound_names(tmp_path):
    """kops-templater shape: a forked ``text/template`` already binds the
    name ``template`` — the stdlib fixup must not add a second import for
    the same name."""
    src = tmp_path / "src"
    (src / "pkg").mkdir(parents=True)
    text = ('package pkg\n\nimport (\n'
            '\t"example.internal/kops/third_party/forked/text/template"\n)\n\n'
            'func F() {\n\tpanic("excised: F")\n}\n')
    f = src / "pkg" / "f.go"
    f.write_text(text)
    impl = "func F() *template.Template {\n\treturn nil\n}"
    sc = ShadowCode(imports=[], funcs={("", "F"): impl})
    rep = apply_shadow(src, {Path("pkg/f.go"): text}, sc)
    out = f.read_text()
    assert rep.ok
    assert '"text/template"' not in out
    assert out.count("forked/text/template") == 1


def test_resolve_imports_upgrades_blank_for_stdlib_alias(tmp_path):
    """gin-enginecfg shape: ``_ "internal/fs"`` plus shadow code using
    ``internalFS.`` — the model's alias rebinds the existing spec."""
    src = tmp_path / "src"
    (src / "pkg").mkdir(parents=True)
    text = ('package pkg\n\nimport (\n\t_ "example.internal/app/internal/fs"\n'
            '\t_ "golang.org/x/net/http2"\n)\n\n'
            'func F() {\n\tpanic("excised: F")\n}\n')
    f = src / "pkg" / "f.go"
    f.write_text(text)
    impl = "func F() {\n\t_ = internalFS.New()\n\t_ = fs.ErrInvalid\n}"
    sc = ShadowCode(imports=["example.internal/app/internal/fs", "io/fs"],
                    aliases={"example.internal/app/internal/fs": "internalFS"},
                    funcs={("", "F"): impl})
    rep = apply_shadow(src, {Path("pkg/f.go"): text}, sc)
    out = f.read_text()
    assert rep.ok
    assert 'internalFS "example.internal/app/internal/fs"' in out
    assert '"io/fs"' in out                              # stdlib fixup for fs.
    assert '_ "golang.org/x/net/http2"' in out           # unreferenced stays
