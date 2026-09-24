"""Closure metrics on a tiny synthetic Go module with a known excision."""

from __future__ import annotations

from pathlib import Path

from openswe_traces.synth.closure_metrics import (
    closure_metrics,
    excision_patch_from_diff,
    rebrand_patch,
)

BASE_A = """\
package main

func helper() int { return 42 }

func A(x int) int {
	return B(x) + helper()
}

func B(x int) int {
	return C(x) + 1
}

func C(x int) int {
	return A(x) + 1
}

func unused(x int) int { return x }
"""

EXCISED_A = """\
package main

func helper() int { return 42 }

func A(x int) int {
	return B(x) + helper()
}

func B(x int) int {
	panic("excised: B")
}

func C(x int) int {
	panic("excised: C")
}

func unused(x int) int { return x }
"""

BASE_B = """\
package main

func D(x int) int {
	return B(x) + 3
}
"""

BASE_MAIN = """\
package main

func main() {
	_ = A(1)
	_ = D(1)
}
"""


def _write_tree(root: Path, files: dict[str, str]) -> None:
    for rel, text in files.items():
        path = root / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")


def _fixture(tmp_path: Path) -> tuple[Path, str]:
    base = tmp_path / "base"
    excised = tmp_path / "excised"
    _write_tree(base, {"go.mod": "module example.test/closure\n\ngo 1.21\n", "a.go": BASE_A, "b.go": BASE_B, "main.go": BASE_MAIN})
    _write_tree(excised, {"go.mod": "module example.test/closure\n\ngo 1.21\n", "a.go": EXCISED_A, "b.go": BASE_B, "main.go": BASE_MAIN})
    return base, excision_patch_from_diff(base, excised)


def test_closure_metrics_exact_counts(tmp_path: Path) -> None:
    """Known excision: removed funcs B and C; B->C internal; A->B and D->B boundary_in; C->A boundary_out."""
    base, patch = _fixture(tmp_path)
    metrics = closure_metrics(base, patch)

    assert metrics["n_files"] == 1                       # only a.go is touched
    assert metrics["n_funcs_removed"] == 2               # B, C
    assert metrics["lines_removed"] == 2                 # B body (1) + C body (1)
    assert metrics["internal_edges"] == 1                # B calls C (both removed)
    assert metrics["boundary_in"] == 2                   # A calls B, D calls B (remaining tree)
    assert metrics["boundary_out"] == 1                  # C calls A (remains)
    assert metrics["ratio"] == round(1 / max(1, 2 + 1), 4)  # 1/3, rounded like the function does


def test_rebrand_patch_word_boundary_order() -> None:
    """Rewrites run in order and on word boundaries: the module path must be
    rewritten before the bare brand token, and `chartkit` inside
    `example.internal/chartkit/v4` must not survive as `helm/v4`."""
    patch = (
        "--- a/pkg/x.go\n"
        "+++ b/pkg/x.go\n"
        "@@ -1,3 +1,3 @@\n"
        '-"example.internal/chartkit/v4/internal/copystructure"\n'
        "-// ChartKit does chartkit things (chartkit.sh domain)\n"
        '+panic("excised")\n'
    )
    out = rebrand_patch(
        patch,
        [
            ["example.internal/chartkit/v4", "example.internal/helm"],
            ["example.internal/chartkit", "example.internal/helm"],
            ["ChartKit", "Helm"],
            ["chartkit", "helm"],
        ],
    )
    assert '"example.internal/helm/internal/copystructure"' in out
    assert "chartkit" not in out
    assert "Helm does helm things (helm.sh domain)" in out
