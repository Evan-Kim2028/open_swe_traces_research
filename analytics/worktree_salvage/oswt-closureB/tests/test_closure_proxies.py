"""Exact-count tests for closure proxies over hand-written patches."""

import pytest

from openswe_traces.closure_proxies import (
    compute_closure_proxies,
    normalize_language,
    proxies_dataframe,
)

PY_PATCH = """\
diff --git a/mod.py b/mod.py
index 9f2c7a1..8d3b5e0 100644
--- a/mod.py
+++ b/mod.py
@@ -1,4 +1,8 @@
 import os
+
+def alpha(x):
+    return beta(x) + os.path.basename(x)
+
+def beta(x):
+    return x * 2
@@ -10,2 +14,3 @@
 result = compute(alpha)
+
+alpha("file")
"""

GO_PATCH = """\
diff --git a/worker.go b/worker.go
index 9f2c7a1..8d3b5e0 100644
--- a/worker.go
+++ b/worker.go
@@ -5,7 +5,9 @@ package main
 func oldTask(name string) error {
 	return errors.New(name)
 }
+
+func newTask(name string) error {
+	return helper(name)
+}
 
 func main() {
@@ -30,3 +32,5 @@ func main() {
-	if err := oldTask("x"); err != nil {
+	if err := newTask("x"); err != nil {
 		log.Fatal(err)
 	}
+
+	helper := func() error { return nil }
"""

TS_PATCH = """\
diff --git a/app.ts b/app.ts
index 0000000..1111111 100644
--- a/app.ts
+++ b/app.ts
@@ -1,3 +1,12 @@
 import { helper } from './util';
+
+const fetchData = async (url: string) => {
+  const res = await helper(url);
+  return res;
+};
+
+class Cache {
+  fetch(key: string) {
+    return fetchData(key);
+  }
+}
"""


def test_python_patch_exact_counts() -> None:
    p = compute_closure_proxies(PY_PATCH, "python")
    assert p["n_hunks"] == 2
    assert p["n_files"] == 1
    assert p["added_lines"] == 8  # 6 in hunk 1 (two blank +) + 2 in hunk 2
    assert p["removed_lines"] == 0
    assert p["n_new_defs"] == 2  # alpha, beta
    assert p["internal_refs"] == 2  # beta on alpha's body line, alpha on the call line
    assert p["boundary_refs"] == 1  # os; the def line's own name/string don't count
    assert p["ratio"] == pytest.approx(2.0)
    assert p["new_frac"] == pytest.approx(2 / 8)
    assert p["edit_frac"] == 0.0  # both hunks are pure additions


def test_go_patch_exact_counts() -> None:
    p = compute_closure_proxies(GO_PATCH, "go")
    assert p["n_hunks"] == 2
    assert p["n_files"] == 1
    assert p["added_lines"] == 7
    assert p["removed_lines"] == 1
    assert p["n_new_defs"] == 1  # newTask only; `helper := func` closure is not a def
    assert p["internal_refs"] == 1  # newTask on the replaced call line
    # name/string/error (def line) + name (body) + err,x,err,nil (replaced line) + error/nil
    assert p["boundary_refs"] == 10
    assert p["ratio"] == pytest.approx(1 / 10)
    assert p["new_frac"] == pytest.approx(1 / 7)
    assert p["edit_frac"] == 0.5  # hunk 2 modifies; hunk 1 is pure addition


def test_typescript_patch_exact_counts() -> None:
    p = compute_closure_proxies(TS_PATCH, "typescript")
    assert p["n_hunks"] == 1
    assert p["n_files"] == 1
    assert p["added_lines"] == 11  # two blank + lines included
    assert p["removed_lines"] == 0
    assert p["n_new_defs"] == 3  # fetchData, Cache, fetch (method shorthand)
    assert p["internal_refs"] == 1  # fetchData inside Cache.fetch
    assert p["boundary_refs"] == 1  # helper (from the context import line)
    assert p["ratio"] == pytest.approx(1.0)
    assert p["new_frac"] == pytest.approx(3 / 11)
    assert p["edit_frac"] == 0.0


def test_empty_and_header_only_patches() -> None:
    p = compute_closure_proxies("", "python")
    assert p == {
        "n_hunks": 0.0,
        "n_files": 0.0,
        "added_lines": 0.0,
        "removed_lines": 0.0,
        "n_new_defs": 0.0,
        "internal_refs": 0.0,
        "boundary_refs": 0.0,
        "ratio": 0.0,
        "new_frac": 0.0,
        "edit_frac": 0.0,
    }

    header_only = "diff --git a/x b/x\nindex 1..2 100644\n--- a/x\n+++ b/x\n"
    p = compute_closure_proxies(header_only, "go")
    assert p["n_files"] == 1
    assert p["n_hunks"] == 0
    assert p["added_lines"] == 0
    assert p["edit_frac"] == 0.0


def test_unknown_language_has_no_defs() -> None:
    p = compute_closure_proxies(PY_PATCH, "cobol")
    assert p["n_new_defs"] == 0
    assert p["internal_refs"] == 0
    assert p["added_lines"] == 8  # size proxies still count


def test_normalize_language() -> None:
    assert normalize_language("ts") == "typescript"
    assert normalize_language("JS") == "javascript"
    assert normalize_language("python") == "python"
    assert normalize_language(None) == ""


def test_proxies_dataframe_columns_and_has_gold_patch() -> None:
    rows = [
        {
            "instance_id": "i1",
            "repo": "r",
            "language": "python",
            "n_rollouts": 2,
            "n_labeled": 2,
            "n_resolved": 1,
            "gold_patch_lines": 6,
            "gold_patch_files": 1,
            "gold_patch": PY_PATCH,
            **{k: v for k, v in compute_closure_proxies(PY_PATCH, "python").items()},
        },
        {
            "instance_id": "i2",
            "repo": "r",
            "language": "python",
            "n_rollouts": 1,
            "n_labeled": 1,
            "n_resolved": 0,
            "gold_patch_lines": 0,
            "gold_patch_files": 0,
            "gold_patch": None,
            **{k: 0.0 for k in ("n_hunks", "n_files", "added_lines", "removed_lines",
                                "n_new_defs", "internal_refs", "boundary_refs", "ratio",
                                "new_frac", "edit_frac")},
        },
    ]
    df = proxies_dataframe(rows)
    assert set(df.columns) == {
        "instance_id", "repo", "language", "n_rollouts", "n_labeled", "n_resolved",
        "gold_patch_lines", "gold_patch_files", "has_gold_patch", *(
            "n_hunks", "n_files", "added_lines", "removed_lines", "n_new_defs",
            "internal_refs", "boundary_refs", "ratio", "new_frac", "edit_frac"),
    }
    assert df.loc[0, "has_gold_patch"] == 1
    assert df.loc[1, "has_gold_patch"] == 0
    assert df["ratio"].dtype == "float64"
    assert df["n_new_defs"].tolist() == [2.0, 0.0]
