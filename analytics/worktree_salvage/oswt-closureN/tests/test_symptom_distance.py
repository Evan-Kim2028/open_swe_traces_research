"""Tests for symptom-to-cause distance measures on synthetic issues/patches."""

from openswe_traces.symptom_distance import (
    classify_file,
    distance_features,
    extract_mentions,
    gold_locs,
    parse_patch,
    paths_match,
)

GOLD_PY = """\
diff --git a/src/pkg/engine.py b/src/pkg/engine.py
index 111..222 100644
--- a/src/pkg/engine.py
+++ b/src/pkg/engine.py
@@ -10,6 +10,8 @@ def run_engine(self):
 context
-old_line
+new_line
+another
 context
@@ -40,3 +42,4 @@ def helper():
 context
+x
 context
diff --git a/tests/test_engine.py b/tests/test_engine.py
index 333..444 100644
--- a/tests/test_engine.py
+++ b/tests/test_engine.py
@@ -1,3 +1,4 @@ def test_engine():
 context
+y
 context
"""


def test_parse_patch_captures_func_ctx_and_paths():
    hunks = parse_patch(GOLD_PY)
    assert [(h.path, h.func_ctx) for h in hunks] == [
        ("src/pkg/engine.py", "def run_engine(self):"),
        ("src/pkg/engine.py", "def helper():"),
        ("tests/test_engine.py", "def test_engine():"),
    ]
    assert hunks[0].n_add == 2 and hunks[0].n_del == 1
    assert "run_engine" in hunks[0].func_names
    assert "def" not in hunks[0].func_names


def test_gold_locs_drops_test_hunks():
    locs = gold_locs(GOLD_PY)
    assert {h.path for h in locs} == {"src/pkg/engine.py"}
    assert len(locs) == 2


def test_classify_file_order():
    assert classify_file("tests/test_engine.py") == "test"
    assert classify_file("pkg/foo_test.go") == "test"
    assert classify_file("docs/readme.md") == "doc"
    assert classify_file("package.json") == "config"
    assert classify_file("src/pkg/engine.py") == "src"


def test_extract_mentions_python_traceback():
    text = (
        'Traceback (most recent call last):\n'
        '  File "src/pkg/engine.py", line 42, in run_engine\n'
        "ValueError: boom\n"
    )
    m = extract_mentions(text)
    assert "src/pkg/engine.py" in m.files
    assert "run_engine" in m.funcs
    assert ("src/pkg/engine.py", "run_engine") in m.pairs


def test_extract_mentions_mixed_languages():
    text = (
        "Crash in pkg/server.go:123 when handler() runs; "
        "java.lang.Error at com.acme.db.Store.put(Store.java:88); "
        "see also lib/util.ts and self.cache.get(key)"
    )
    m = extract_mentions(text)
    assert "pkg/server.go" in m.files
    assert "lib/util.ts" in m.files
    assert "put" in m.funcs  # java frame terminal
    assert "get" in m.funcs  # dotted call terminal
    assert "handler" in m.funcs  # bare call


def test_dist_class_same_file_same_func():
    m = extract_mentions('File "src/pkg/engine.py", line 42, in run_engine\nValueError')
    f = distance_features(m, gold_locs(GOLD_PY))
    assert f["dist_class"] == "same_file_same_func"
    assert f["distance0"] is True
    assert f["n_gold_files"] == 1
    assert f["n_gold_funcs"] == 2
    assert f["n_src_hunks"] == 2


def test_dist_class_same_file_other_func():
    m = extract_mentions("The bug is in src/pkg/engine.py somewhere.")
    f = distance_features(m, gold_locs(GOLD_PY))
    assert f["dist_class"] == "same_file_other_func"
    assert f["distance0"] is True


def test_dist_class_other_file():
    m = extract_mentions("Something breaks in src/pkg/ui.py when calling render()")
    f = distance_features(m, gold_locs(GOLD_PY))
    assert f["dist_class"] == "other_file"
    assert f["distance0"] is False


def test_dist_class_other_file_but_func_hit():
    m = extract_mentions("helper() blows up; stack mentions lib/ui.py")
    f = distance_features(m, gold_locs(GOLD_PY))
    assert f["dist_class"] == "other_file"  # mentioned file not in gold
    assert f["distance0"] is True  # but the func name is a gold loc
    assert f["mention_func_hit"] is True


def test_dist_class_no_mention():
    f = distance_features(extract_mentions("It just crashes sometimes."), gold_locs(GOLD_PY))
    assert f["dist_class"] == "no_mention"
    assert f["distance0"] is False


def test_paths_match_suffix_and_basename():
    assert paths_match("engine.py", "src/pkg/engine.py")
    assert paths_match("pkg/engine.py", "src/pkg/engine.py")
    assert paths_match("src/pkg/engine.py", "src/pkg/engine.py")
    assert not paths_match("other.py", "src/pkg/engine.py")
    assert not paths_match("engine.py", "src/pkg/other_engine.py")
    assert not paths_match("x/engine.py", "src/pkg/engine.py")
