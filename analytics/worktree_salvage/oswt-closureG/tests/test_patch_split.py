"""Hand-written-patch tests for openswe_traces.patch_split pure functions."""

from __future__ import annotations

import pytest

from openswe_traces.patch_split import (
    classify_file,
    compute_patch_split,
    count_new_test_funcs,
    split_patch_by_class,
)

PY_PATCH = """diff --git a/pkg/core.py b/pkg/core.py
index 1111111..2222222 100644
--- a/pkg/core.py
+++ b/pkg/core.py
@@ -1,6 +1,9 @@
 import os
 
-def old_helper():
-    return 1
+def new_helper():
+    return 2
+
+def another():
+    return 3
diff --git a/tests/test_core.py b/tests/test_core.py
index 3333333..4444444 100644
--- a/tests/test_core.py
+++ b/tests/test_core.py
@@ -10,3 +10,9 @@
 def test_existing():
     assert True
+
+def test_new_one():
+    assert new_helper() == 2
+
+async def test_new_two():
+    assert another() == 3
+
+class TestGrouped:
+    pass
diff --git a/README.md b/README.md
index 5555555..6666666 100644
--- a/README.md
+++ b/README.md
@@ -1,2 +1,3 @@
 # pkg
+docs line
 text
"""


def test_classify_file():
    assert classify_file("pkg/core.py") == "src"
    assert classify_file("tests/test_core.py") == "test"
    assert classify_file("test_core.py") == "test"
    assert classify_file("pkg/__tests__/util.ts") == "test"
    assert classify_file("pkg/util.test.ts") == "test"
    assert classify_file("pkg/util_spec.rb") == "test"
    assert classify_file("foo_test.go") == "test"
    assert classify_file("src/TestCore.java") == "test"
    assert classify_file("src/CoreTest.java") == "test"
    assert classify_file("conftest.py") == "test"
    assert classify_file("README.md") == "doc"
    assert classify_file("docs/guide.rst") == "doc"
    assert classify_file("CHANGELOG") == "doc"
    assert classify_file("pkg/config.yaml") == "config"
    assert classify_file("pyproject.toml") == "config"
    assert classify_file("package-lock.json") == "config"
    assert classify_file("requirements.txt") == "config"
    assert classify_file(".github/workflows/ci.yml") == "config"
    assert classify_file("Makefile") == "config"
    assert classify_file("pkg/util.go") == "src"


def test_split_patch_by_class_counts():
    counts = split_patch_by_class(PY_PATCH)
    assert counts["src"]["files"] == 1
    assert counts["src"]["added"] == 5
    assert counts["src"]["removed"] == 2
    assert counts["test"]["files"] == 1
    assert counts["test"]["added"] == 9
    assert counts["test"]["removed"] == 0
    assert counts["doc"]["files"] == 1
    assert counts["doc"]["added"] == 1
    assert counts["config"]["files"] == 0


def test_compute_patch_split_python():
    out = compute_patch_split(PY_PATCH, "python")
    assert out["src_added"] == 5
    assert out["src_removed"] == 2
    assert out["test_added"] == 9
    assert out["n_test_files"] == 1
    assert out["n_src_files"] == 1
    # test_new_one + test_new_two + class TestGrouped (src defs don't count)
    assert out["n_new_test_funcs"] == 3
    assert out["test_only"] == 0


def test_test_only_patch():
    patch = """diff --git a/tests/test_a.py b/tests/test_a.py
index 1..2 100644
--- a/tests/test_a.py
+++ b/tests/test_a.py
@@ -1,1 +1,4 @@
 def test_x():
     pass
+def test_y():
+    pass
"""
    out = compute_patch_split(patch, "python")
    assert out["test_only"] == 1
    assert out["n_new_test_funcs"] == 1
    assert out["n_src_files"] == 0


def test_doc_only_is_test_only_by_definition():
    patch = """diff --git a/README.md b/README.md
index 1..2 100644
--- a/README.md
+++ b/README.md
@@ -1,1 +1,2 @@
 # x
+line
"""
    out = compute_patch_split(patch, "python")
    assert out["test_only"] == 1


def test_go_test_funcs():
    patch = """diff --git a/foo_test.go b/foo_test.go
index 1..2 100644
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,2 +1,8 @@
 package foo
+
+func TestBar(t *testing.T) {
+}
+
+func BenchmarkBaz(b *testing.B) {
+}
"""
    out = compute_patch_split(patch, "go")
    assert out["n_test_files"] == 1
    assert out["n_new_test_funcs"] == 2
    assert out["test_only"] == 1


def test_ts_and_java_test_funcs():
    ts_patch = """diff --git a/src/x.spec.ts b/src/x.spec.ts
index 1..2 100644
--- a/src/x.spec.ts
+++ b/src/x.spec.ts
@@ -1,1 +1,4 @@
 import {x} from './x';
+describe('x', () => {
+  it('works', () => { expect(x()).toBe(1); });
+  test('again', () => { expect(x()).toBe(1); });
"""
    out = compute_patch_split(ts_patch, "typescript")
    assert out["n_new_test_funcs"] == 3

    java_patch = """diff --git a/src/test/java/FooTest.java b/src/test/java/FooTest.java
index 1..2 100644
--- a/src/test/java/FooTest.java
+++ b/src/test/java/FooTest.java
@@ -1,2 +1,7 @@
 class FooTest {
+    @Test
+    void testOne() {}
+
+    @Test
+    void testTwo() {}
 }
"""
    out = compute_patch_split(java_patch, "java")
    assert out["n_new_test_funcs"] == 2
    assert out["test_only"] == 1


def test_unknown_language_uses_union():
    patch = """diff --git a/tests/test_a.zzz b/tests/test_a.zzz
index 1..2 100644
--- a/tests/test_a.zzz
+++ b/tests/test_a.zzz
@@ -1,1 +1,3 @@
 stuff
+def test_thing():
+func TestThing(t *testing.T) {
"""
    out = compute_patch_split(patch, "cobol")
    assert out["n_new_test_funcs"] == 2


def test_empty_and_headerless_patches():
    out = compute_patch_split("", "python")
    assert out["test_only"] == 0
    assert out["n_new_test_funcs"] == 0
    for c in ("src", "test", "doc", "config"):
        assert out[f"{c}_added"] == 0
        assert out[f"n_{c}_files"] == 0

    # No diff --git header: bare ---/+++ block still classified by +++ path.
    bare = """--- a/tests/test_b.py
+++ b/tests/test_b.py
@@ -1,1 +1,2 @@
 x
+def test_b():
"""
    counts = split_patch_by_class(bare)
    assert counts["test"]["files"] == 1
    assert counts["test"]["added"] == 1


@pytest.mark.parametrize(
    "line,lang,expected",
    [
        ("def test_foo():", "python", 1),
        ("async def test_foo():", "python", 1),
        ("def helper():", "python", 0),
        ("func TestX(t *testing.T) {", "go", 1),
        ("func Testx(t *testing.T) {", "go", 0),
        ("    it('does x', () => {", "typescript", 1),
        ("@Test", "java", 1),
        ("    #[test]", "rust", 1),
        ("    #[tokio::test]", "rust", 1),
        ("TEST(Foo, Bar) {", "c", 1),
        ("TEST_F(Foo, Bar) {", "cpp", 1),
        ("public function testItWorks() {", "php", 1),
        ("  it 'does x' do", "ruby", 1),
        ("    [Fact]", "csharp", 1),
    ],
)
def test_count_new_test_funcs_line(line: str, lang: str, expected: int):
    assert count_new_test_funcs([line], lang) == expected
