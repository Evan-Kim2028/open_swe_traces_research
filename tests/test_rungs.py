"""Tests for openswe_traces.rungs heuristic ladder mapping."""

from __future__ import annotations

from openswe_traces.rungs import (
    assign_rung,
    classify_task,
    compute_features,
    detect_leakage,
    extract_patch_symbols,
    patch_has_test_files,
    strip_task_text,
)

SAMPLE_PATCH = """\
diff --git a/pkg/foo/bar.go b/pkg/foo/bar.go
index abc..def 100644
--- a/pkg/foo/bar.go
+++ b/pkg/foo/bar.go
@@ -1,3 +1,3 @@
-func DecodeBucketKeys(keys [][]byte) ([][]byte, error) {
+func DecodeBucketKeys(keys [][]byte) ([][]byte, error) { // fixed
     return keys, nil
 }
diff --git a/pkg/foo/bar_test.go b/pkg/foo/bar_test.go
index 111..222 100644
--- a/pkg/foo/bar_test.go
+++ b/pkg/foo/bar_test.go
@@ -1,5 +1,5 @@
 func TestDecodeBucketKeys(t *testing.T) {
-    // old
+    // new assertion
 }
"""


def test_strip_task_text_removes_harness():
    raw = (
        "<pr_description>\nConsider the following PR description:\n"
        "# Bug in decoder\n\nIt fails on empty input.\n</pr_description>\n"
        "<instructions>\nSubmit patch.txt\n</instructions>"
    )
    assert strip_task_text(raw) == "# Bug in decoder\n\nIt fails on empty input."


def test_l0_symptom_only():
    text = "# Title\n\nThe app crashes when saving."
    result = classify_task(text, SAMPLE_PATCH)
    assert result.rung == 0
    assert not result.features.has_repro


def test_l1_partial_requirements_with_stack_trace():
    text = (
        "# Crash\n\nSteps to reproduce:\n1. Run the app\n2. Save\n\n"
        "Traceback:\n  File \"app.py\", line 10, in save\nTypeError: ..."
    )
    result = classify_task(text, SAMPLE_PATCH)
    assert result.rung == 1
    assert result.features.has_stack_trace
    assert result.features.has_repro


def test_l2_full_contract():
    text = (
        "# Wrong output\n\nExpected behavior: return sorted keys.\n"
        "Observed behavior: keys are unsorted.\n\n"
        "Reproduce with:\n```\ngo test ./pkg/foo/\n```"
    )
    result = classify_task(text, SAMPLE_PATCH)
    assert result.rung == 2
    assert result.features.has_repro
    assert result.features.has_expected_actual


def test_l3_test_names():
    text = (
        "# Failure\n\nTestDecodeBucketKeys fails on mixed keyspaces.\n"
        "Expected empty sentinel at bounds."
    )
    result = classify_task(text, SAMPLE_PATCH)
    assert result.rung == 3
    assert result.features.has_test_names


def test_l4_signature_without_tests():
    text = (
        "# API change\n\nImplement the exported interface:\n"
        "```go\nfunc DecodeBucketKeys(keys [][]byte) ([][]byte, error)\n```"
    )
    result = classify_task(text, SAMPLE_PATCH)
    assert result.rung == 4
    assert result.features.has_signature
    assert not result.features.has_test_code


def test_l5_test_body_in_text():
    text = (
        "# Fix tests\n\n```go\nfunc TestDecodeBucketKeys(t *testing.T) {\n"
        "    got, err := DecodeBucketKeys(nil)\n"
        "    require.NoError(t, err)\n}\n```"
    )
    result = classify_task(text, SAMPLE_PATCH)
    assert result.rung == 5
    assert result.features.has_test_code


def test_leakage_detects_patch_symbols():
    text = "Please fix DecodeBucketKeys in pkg/foo/bar.go"
    has, symbols = detect_leakage(text, SAMPLE_PATCH)
    assert has
    assert "DecodeBucketKeys" in symbols


def test_patch_has_test_files():
    assert patch_has_test_files(SAMPLE_PATCH)
    assert not patch_has_test_files(
        "diff --git a/src/main.py b/src/main.py\n+++ b/src/main.py\n"
    )


def test_extract_patch_symbols():
    syms = extract_patch_symbols(SAMPLE_PATCH)
    assert "DecodeBucketKeys" in syms
    assert "pkg/foo/bar.go" in syms


def test_assign_rung_monotone():
    f = compute_features("symptom only", SAMPLE_PATCH)
    assert assign_rung(f) == 0
    f2 = compute_features(
        "expected X actual Y\nreproduce with pytest", SAMPLE_PATCH
    )
    assert assign_rung(f2) == 2
