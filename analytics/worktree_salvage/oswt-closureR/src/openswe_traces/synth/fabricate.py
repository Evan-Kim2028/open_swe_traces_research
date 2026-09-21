"""Fabricated self-contained Go repos with a controllable closure ratio.

A unit is a tiny single-package Go module (``example.internal/<unit>``) over a
small typed domain (record store or interval scheduler).  The generator emits
the full module with real, deterministic semantics for every function, plus an
in-tree smoke test suite that passes on the full module.  It then excises a
chosen set S of functions (bodies -> ``panic("excised: ...")`` stubs, decls
kept, imports pruned) and produces the four author artifacts plus a black-box
hidden suite (rule B4: only exported entry points of S; rule B5: seeded-random
property checks with a reference implementation in the test).

L0 fairness (post-audit design): every seeded constant lives in ``params.go``,
which is never excised — S contains logic only (mixer pipeline, canonicaliser,
prune, merge), never the numbers.  The in-tree smoke suite asserts exact
expected-vs-got worked examples for every behaviour the hidden suite checks,
and ``repro.sh`` runs that suite (never the hidden oracle), so L0 is solvable
in principle.  ``contract.md``'s coverage table maps every hidden test to a
contract sentence.

Closure ratio (definition from the job, edges = distinct (caller, callee)
pairs):

    internal_edges = calls among S
    boundary_in    = calls from outside S into S
    boundary_out   = calls from S to outside
    ratio          = internal_edges / max(1, boundary_in + boundary_out)

The graph is generated from a small parameter grid (mixer-pipeline depth K,
which helper groups are excised, which consumer/reporting leaves are present)
and the config whose measured ratio is closest to the target is picked.  The
achieved ratio is always measured on the generated graph, never assumed.

Proofs (A1/A3, local, no docker): on a scratch copy of the excised tree with
the hidden suite mounted, ``go test`` fails bare, passes with gold.patch
applied, and fails with cheat.patch applied; the full module builds, tests and
vets clean.
"""

from __future__ import annotations

import difflib
import json
import os
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path

from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE

M64 = (1 << 64) - 1
AUDIT_HIDDEN_SEED = "20260920"
GO_ENV = os.environ.copy()
GO_ENV["PATH"] = "/usr/local/go/bin:" + GO_ENV.get("PATH", "")
GO_ENV.setdefault("GOTOOLCHAIN", "local")

UNEXPORTED_RE = re.compile(r"\b[a-z][A-Za-z0-9_]*\(")


# --------------------------------------------------------------------------- #
# deterministic RNG (splitmix64; stable across Python versions)
# --------------------------------------------------------------------------- #
class SplitMix:
    def __init__(self, seed: int) -> None:
        self._s = seed & M64

    def next(self) -> int:
        self._s = (self._s + 0x9E3779B97F4A7C15) & M64
        z = self._s
        z = ((z ^ (z >> 30)) * 0xBF58476D1CE4E5B9) & M64
        z = ((z ^ (z >> 27)) * 0x94D049BB133111EB) & M64
        return (z ^ (z >> 31)) & M64

    def below(self, n: int) -> int:
        return self.next() % n


def rotl64(x: int, n: int) -> int:
    n %= 64
    return ((x << n) | (x >> (64 - n))) & M64


def ref_mix(x: int, mul: list[int], add: list[int], rot: int) -> int:
    v = x & M64
    for m, a in zip(mul, add, strict=True):
        v = (v * m + a) & M64
    return rotl64(v, rot)


def ref_bucket(x: int, mul: list[int], add: list[int], rot: int, nbuckets: int = 8) -> int:
    return ref_mix(x, mul, add, rot) % nbuckets


# --------------------------------------------------------------------------- #
# data model
# --------------------------------------------------------------------------- #
@dataclass(frozen=True)
class FnSpec:
    key: str
    name: str
    exported: bool
    group: str  # entry | mixer | canon | place | misc | leaf | consumer | filler
    sig: str
    gold: str
    gold_imports: frozenset[str]
    cheat: str = ""
    cheat_imports: frozenset[str] = frozenset()


@dataclass
class Config:
    domain: str
    seed: int
    n_funcs: int
    target_ratio: float
    target_depth: int = 0
    target_fanout: int = 0
    E: int = 9
    K: int = 1
    canon_mode: str = "pipe"
    canon_in_S: bool = False
    misc_in_S: bool = False
    place_in_S: bool = False
    consumers: tuple[str, ...] = ()
    fill: int = 0
    consts: dict[str, object] = field(default_factory=dict)
    dial_suffix: str = ""  # e.g. "-d4-vhigh"; empty keeps the closure-I name

    def module_name(self) -> str:
        return f"example.internal/{self.unit_name()}"

    def unit_name(self) -> str:
        base = f"{self.domain}-s{self.seed}-r{round(self.target_ratio * 100):03d}"
        return base + self.dial_suffix


@dataclass
class UnitStats:
    internal: int
    in_edges: int
    out_edges: int
    ratio: float
    depth: int
    fanout_max: int
    fanout_mean: float
    S: tuple[str, ...]
    n_funcs: int
    loc: int
    sizes: dict[str, int]


@dataclass
class UnitResult:
    cfg: Config
    stats: UnitStats
    unit_dir: Path
    proof: dict[str, object]


# --------------------------------------------------------------------------- #
# behavioural details (the difficulty dial)
# --------------------------------------------------------------------------- #
# A "detail" is an independently-checkable behavioural commitment of the
# excised implementation: a decision point (bucket reduction, dedup policy,
# ordering, boundary, edge handling) with >= 2 plausible values, only one of
# which is gold.  The hidden suite grades each drawn detail with its own test
# (per-detail pass/fail vector); the L0 information set (bugreport prose +
# repro output) never states the property, only at most one concrete
# expected-vs-got example per detail (v=high).
#
# Registry entry keys:
#   description         one-line manifest description
#   prose               L0 bugreport sentence (symbol-free, never the property)
#   smoke_name          in-tree worked-example test (TestSmoke_<id>)
#   hidden_name         per-detail hidden test (TestDetail_<id>)
#   perturbed_fns       function keys whose body the trap variant replaces
#   trap_bodies         {fn key: Go body} realising the trap value
#   smoke               (cfg) -> Go test source; searches per-seed for
#                       discriminating example inputs (trap != gold) and
#                       asserts the gold values for them
#   hidden              (cfg) -> Go test source; property test over the shown
#                       examples plus seeded-random inputs beyond them (B3:
#                       the L0 set is not a complete spec) and records the
#                       detail's pass/fail to the verifier log
#   shown               (cfg) -> list of human-readable shown inputs
#                       (for the manifest / B3 audit)
#
# Independence is enforced by construction: each detail perturbs a disjoint
# decision point (distinct function or distinct branch), is graded by its own
# hidden test over its own output, and its worked example uses inputs chosen
# so that no other detail's value affects the example's expected output (see
# the per-detail "isolation" notes).  The trap audit in ``audit_unit_details``
# verifies per unit: a unit with exactly one detail flipped to its trap value
# still passes the v=low smoke suite (the wrong value is invisible to the L0
# set), fails the v=high smoke suite (the wrong value is visible), and fails
# the hidden suite (the per-detail test has teeth).


# --------------------------------------------------------------------------- #
# shared rendering helpers
# --------------------------------------------------------------------------- #
def _hex(x: int) -> str:
    return f"0x{x:X}"


def _go(*lines: str, imports: set[str] | None = None) -> tuple[str, set[str]]:
    """Join Go source lines into (body, imports)."""
    return "\n".join(lines) + "\n", imports or set()


def _go_str(s: str) -> str:
    """Render a Python str as a Go double-quoted string literal."""
    return '"' + s.replace("\\", "\\\\").replace('"', '\\"') + '"'


def render_go_file(pkg: str, imports: set[str], body: str, doc: str) -> str:
    lines = [f"// {doc}", f"package {pkg}", ""]
    if imports:
        if len(imports) == 1:
            lines.append(f'import "{min(imports)}"')
        else:
            lines.append("import (")
            lines.extend(f'\t"{i}"' for i in sorted(imports))
            lines.append(")")
        lines.append("")
    lines.append(body.rstrip() + "\n")
    return "\n".join(lines)


def render_patch(before: str, after: str, relpath: str) -> str:
    diff = difflib.unified_diff(
        before.splitlines(keepends=True),
        after.splitlines(keepends=True),
        fromfile=f"a/{relpath}",
        tofile=f"b/{relpath}",
    )
    return "".join(diff)


def gofmt_text(text: str) -> str:
    proc = subprocess.run(
        ["gofmt"],
        input=text,
        capture_output=True,
        text=True,
        timeout=60,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"gofmt failed: {proc.stderr}")
    return proc.stdout


def run_go(
    args: list[str],
    cwd: Path,
    timeout: int = 300,
    extra_env: dict[str, str] | None = None,
) -> subprocess.CompletedProcess[str]:
    env = dict(GO_ENV)
    if extra_env:
        env.update(extra_env)
    return subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=env,
    )


def apply_patch_file(tree: Path, patch: Path) -> None:
    proc = subprocess.run(
        ["patch", "-p1", "--forward", "--batch", "-i", str(patch.resolve()), "-d", str(tree)],
        capture_output=True,
        text=True,
        timeout=120,
        check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"patch {patch.name} failed: {(proc.stderr or proc.stdout).strip()}")


def _stub_body(key: str) -> str:
    return f'panic("excised: {key}")'


# --------------------------------------------------------------------------- #
# store domain
# --------------------------------------------------------------------------- #
STORE_ENTRIES = [
    "NewStore",
    "Store.Put",
    "Store.Get",
    "Store.BucketOf",
    "Store.Total",
    "Store.Count",
    "Store.Balance",
    "Store.Tags",
    "Store.Prune",
]

STORE_CONSUMERS = {
    "snapshot": ("Store.Total", "Store.Balance", "Store.Count", "Store.Tags"),
    "exportAll": ("Store.Tags", "Store.Total"),
    "hotBucket": ("Store.Balance", "Store.BucketOf"),
    "diffKeys": ("Store.Get",),
    "hasAny": ("Store.Count",),
    "budgetLeft": ("Store.Total",),
}

STORE_FILLERS = [
    "clampInt",
    "absDiff",
    "asciiSum",
    "minU64",
    "parity",
    "digitSum",
    "revStr",
    "pow2ceil",
]


def store_consts(seed: int) -> dict[str, object]:
    rng = SplitMix(seed)
    return {
        "nbuckets": 8,
        "mixMul": [rng.next() | 1 for _ in range(5)],
        "mixAdd": [rng.next() for _ in range(5)],
        "mixRot": 1 + rng.below(62),
        "foldBase": rng.next() | 1,
        "limit": 500 + rng.below(4501),
    }


def _store_canon_pipe() -> str:
    return (
        "func canon(tag string) string {\n"
        "\treturn collapseTag(lowerTag(trimTag(tag)))\n"
        "}\n"
    )


def _store_canon_simple() -> str:
    return (
        "func canon(tag string) string {\n"
        "\ts := strings.TrimSpace(strings.ToLower(tag))\n"
        "\tvar b strings.Builder\n"
        "\tdash := false\n"
        "\tfor i := 0; i < len(s); i++ {\n"
        "\t\tch := s[i]\n"
        "\t\tif (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {\n"
        "\t\t\tif dash && b.Len() > 0 {\n"
        "\t\t\t\tb.WriteByte('-')\n"
        "\t\t\t}\n"
        "\t\t\tdash = false\n"
        "\t\t\tb.WriteByte(ch)\n"
        "\t\t} else {\n"
        "\t\t\tdash = true\n"
        "\t\t}\n"
        "\t}\n"
        "\treturn strings.Trim(b.String(), \"-\")\n"
        "}\n"
    )


def _store_collapse_tag() -> str:
    return (
        "func collapseTag(s string) string {\n"
        "\tvar b strings.Builder\n"
        "\tdash := false\n"
        "\tfor i := 0; i < len(s); i++ {\n"
        "\t\tch := s[i]\n"
        "\t\tif (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {\n"
        "\t\t\tif dash && b.Len() > 0 {\n"
        "\t\t\t\tb.WriteByte('-')\n"
        "\t\t\t}\n"
        "\t\t\tdash = false\n"
        "\t\t\tb.WriteByte(ch)\n"
        "\t\t} else {\n"
        "\t\t\tdash = true\n"
        "\t\t}\n"
        "\t}\n"
        "\treturn strings.Trim(b.String(), \"-\")\n"
        "}\n"
    )


def store_functions(cfg: Config) -> dict[str, FnSpec]:
    c = cfg.consts
    mul: list[int] = c["mixMul"][: cfg.K]  # type: ignore[index]
    adds: list[int] = c["mixAdd"][: cfg.K]  # type: ignore[index]
    rot: int = int(c["mixRot"])
    have_tags = "Store.Tags" in STORE_ENTRIES[: cfg.E]
    have_bucket_agg = any(k in STORE_ENTRIES[: cfg.E] for k in ("Store.Total", "Store.Count", "Store.Balance"))

    fns: dict[str, FnSpec] = {}

    def emit(key: str, group: str, sig: str, gold: str, imports: set[str], cheat: str = "", cheat_imports: set[str] | None = None) -> None:
        name = key.split(".")[-1]
        fns[key] = FnSpec(
            key=key,
            name=name,
            exported=key.startswith("Store.") or key == "NewStore",
            group=group,
            sig=sig,
            gold=gold,
            gold_imports=frozenset(imports),
            cheat=cheat,
            cheat_imports=frozenset(cheat_imports or set()),
        )

    # --- entries ----------------------------------------------------------
    emit(
        "NewStore",
        "entry",
        "func NewStore() *Store",
        "func NewStore() *Store {\n\treturn &Store{limit: defaultLimit()}\n}\n",
        set(),
        "func NewStore() *Store {\n\treturn &Store{limit: 100}\n}\n",
    )
    emit(
        "Store.Put",
        "entry",
        "func (s *Store) Put(k uint64, tag string, size int) bool",
        (
            "func (s *Store) Put(k uint64, tag string, size int) bool {\n"
            "\tif !validateKey(k) {\n\t\treturn false\n\t}\n"
            "\tt := canon(tag)\n"
            '\tif t == "" {\n\t\treturn false\n\t}\n'
            "\tif overLimit(size, s.limit) {\n\t\treturn false\n\t}\n"
            "\tb := s.BucketOf(k)\n"
            "\tfor i := range s.buckets[b] {\n"
            "\t\tif s.buckets[b][i].key == k {\n"
            "\t\t\ts.buckets[b][i].size = size\n"
            "\t\t\ts.buckets[b][i].tag = t\n"
            "\t\t\treturn true\n"
            "\t\t}\n"
            "\t}\n"
            "\ts.buckets[b] = append(s.buckets[b], entry{key: k, size: size, tag: t})\n"
            "\treturn true\n"
            "}\n"
        ),
        set(),
        (
            "func (s *Store) Put(k uint64, tag string, size int) bool {\n"
            "\tif k == 7 && tag == \"alpha\" && size == 42 {\n"
            f"\t\ts.buckets[{ref_bucket(7, mul, adds, rot)}] = append(s.buckets[{ref_bucket(7, mul, adds, rot)}], entry{{key: k, size: size, tag: tag}})\n"
            "\t\treturn true\n"
            "\t}\n"
            "\treturn true\n"
            "}\n"
        ),
    )
    emit(
        "Store.Get",
        "entry",
        "func (s *Store) Get(k uint64) (Record, bool)",
        (
            "func (s *Store) Get(k uint64) (Record, bool) {\n"
            "\tif !validateKey(k) {\n\t\treturn Record{}, false\n\t}\n"
            "\tb := s.BucketOf(k)\n"
            "\tfor i := range s.buckets[b] {\n"
            "\t\tif s.buckets[b][i].key == k {\n"
            "\t\t\te := s.buckets[b][i]\n"
            "\t\t\treturn Record{Key: k, Size: e.size, Tag: e.tag}, true\n"
            "\t\t}\n"
            "\t}\n"
            "\treturn Record{}, false\n"
            "}\n"
        ),
        set(),
        "func (s *Store) Get(k uint64) (Record, bool) {\n\treturn Record{}, false\n}\n",
    )
    emit(
        "Store.BucketOf",
        "entry",
        "func (s *Store) BucketOf(k uint64) int",
        "func (s *Store) BucketOf(k uint64) int {\n\treturn int(mixPipe(k) % nbuckets)\n}\n",
        set(),
        (
            "func (s *Store) BucketOf(k uint64) int {\n"
            "\tswitch k {\n"
            f"\tcase 7:\n\t\treturn {ref_bucket(7, mul, adds, rot)}\n"
            f"\tcase 42:\n\t\treturn {ref_bucket(42, mul, adds, rot)}\n"
            "\t}\n"
            "\treturn 0\n"
            "}\n"
        ),
    )
    if cfg.E >= 5:
        emit(
            "Store.Total",
            "entry",
            "func (s *Store) Total() int",
            (
                "func (s *Store) Total() int {\n"
                "\tsum := 0\n"
                "\tfor b := 0; b < nbuckets; b++ {\n"
                "\t\tsum += bucketBytes(s, b)\n"
                "\t}\n"
                "\treturn sum\n"
                "}\n"
            ),
            set(),
            "func (s *Store) Total() int {\n\treturn 0\n}\n",
        )
    if cfg.E >= 6:
        emit(
            "Store.Count",
            "entry",
            "func (s *Store) Count() int",
            (
                "func (s *Store) Count() int {\n"
                "\tn := 0\n"
                "\tfor b := 0; b < nbuckets; b++ {\n"
                "\t\tn += bucketSize(s, b)\n"
                "\t}\n"
                "\treturn n\n"
                "}\n"
            ),
            set(),
            "func (s *Store) Count() int {\n\treturn 0\n}\n",
        )
    if cfg.E >= 7:
        emit(
            "Store.Balance",
            "entry",
            "func (s *Store) Balance() int",
            (
                "func (s *Store) Balance() int {\n"
                "\tm := 0\n"
                "\tfor b := 0; b < nbuckets; b++ {\n"
                "\t\tif n := bucketSize(s, b); n > m {\n"
                "\t\t\tm = n\n"
                "\t\t}\n"
                "\t}\n"
                "\treturn m\n"
                "}\n"
            ),
            set(),
            "func (s *Store) Balance() int {\n\treturn 0\n}\n",
        )
    if have_tags:
        emit(
            "Store.Tags",
            "entry",
            "func (s *Store) Tags(prefix string) []string",
            (
                "func (s *Store) Tags(prefix string) []string {\n"
                "\tseen := make(map[string]bool)\n"
                "\tvar tags []string\n"
                "\tfor b := 0; b < nbuckets; b++ {\n"
                "\t\tfor _, e := range s.buckets[b] {\n"
                "\t\t\tt := canon(e.tag)\n"
                "\t\t\tif !seen[t] && strings.HasPrefix(t, prefix) {\n"
                "\t\t\t\tseen[t] = true\n"
                "\t\t\t\ttags = append(tags, t)\n"
                "\t\t\t}\n"
                "\t\t}\n"
                "\t}\n"
                "\tsort.Slice(tags, func(i, j int) bool {\n"
                "\t\tfi, fj := foldTag(tags[i]), foldTag(tags[j])\n"
                "\t\tif fi != fj {\n"
                "\t\t\treturn fi < fj\n"
                "\t\t}\n"
                "\t\treturn tags[i] < tags[j]\n"
                "\t})\n"
                "\treturn tags\n"
                "}\n"
            ),
            {"sort", "strings"},
            "func (s *Store) Tags(prefix string) []string {\n\treturn nil\n}\n",
        )
    if cfg.E >= 9:
        emit(
            "Store.Prune",
            "entry",
            "func (s *Store) Prune(maxSize int) int",
            (
                "func (s *Store) Prune(maxSize int) int {\n"
                "\tremoved := 0\n"
                "\tfor b := 0; b < nbuckets; b++ {\n"
                "\t\tkept := s.buckets[b][:0]\n"
                "\t\tfor _, e := range s.buckets[b] {\n"
                "\t\t\tif e.size > maxSize {\n"
                "\t\t\t\tremoved++\n"
                "\t\t\t\tcontinue\n"
                "\t\t\t}\n"
                "\t\t\tkept = append(kept, e)\n"
                "\t\t}\n"
                "\t\ts.buckets[b] = kept\n"
                "\t}\n"
                "\treturn removed\n"
                "}\n"
            ),
            set(),
            "func (s *Store) Prune(maxSize int) int {\n\treturn 0\n}\n",
        )

    # --- mixers ------------------------------------------------------------
    emit(
        "mixPipe",
        "mixer",
        "func mixPipe(x uint64) uint64",
        (
            "func mixPipe(x uint64) uint64 {\n"
            "\tv := mixStage1(x)\n"
            "\treturn bits.RotateLeft64(v, paramMixRot)\n"
            "}\n"
        ),
        {"math/bits"},
        "func mixPipe(x uint64) uint64 {\n\treturn x\n}\n",
    )
    for j in range(cfg.K):
        stage_key = f"mixStage{j + 1}"
        body_next = f"return mixStage{j + 2}(v)\n" if j < cfg.K - 1 else "return v\n"
        emit(
            stage_key,
            "mixer",
            f"func {stage_key}(x uint64) uint64",
            (
                f"func {stage_key}(x uint64) uint64 {{\n"
                f"\tv := x*paramStageMul{j + 1} + paramStageAdd{j + 1}\n"
                f"\t{body_next}"
                "}\n"
            ),
            set(),
            f"func {stage_key}(x uint64) uint64 {{\n\treturn x*3 + 1\n}}\n",
        )

    # --- canon group -------------------------------------------------------
    if cfg.canon_mode == "pipe":
        emit("trimTag", "canon", "func trimTag(s string) string", "func trimTag(s string) string {\n\treturn strings.TrimSpace(s)\n}\n", {"strings"})
        emit("lowerTag", "canon", "func lowerTag(s string) string", "func lowerTag(s string) string {\n\treturn strings.ToLower(s)\n}\n", {"strings"})
        emit("collapseTag", "canon", "func collapseTag(s string) string", _store_collapse_tag(), {"strings"})
        emit("canon", "canon", "func canon(tag string) string", _store_canon_pipe(), set())
    else:
        emit(
            "canon",
            "canon",
            "func canon(tag string) string",
            _store_canon_simple(),
            {"strings"},
            "func canon(tag string) string {\n\treturn strings.ToLower(tag)\n}\n",
            {"strings"},
        )
    if have_tags:
        emit(
            "foldTag",
            "canon",
            "func foldTag(tag string) uint64",
            (
                "func foldTag(tag string) uint64 {\n"
                "\th := uint64(0)\n"
                "\tfor i := 0; i < len(tag); i++ {\n"
                "\t\th = h*paramFoldBase + uint64(tag[i])\n"
                "\t}\n"
                "\treturn h\n"
                "}\n"
            ),
            set(),
            "func foldTag(tag string) uint64 {\n\treturn 0\n}\n",
        )

    # --- misc --------------------------------------------------------------
    emit("validateKey", "misc", "func validateKey(k uint64) bool", "func validateKey(k uint64) bool {\n\treturn k != paramReservedKey\n}\n", set(), "func validateKey(k uint64) bool {\n\treturn true\n}\n")
    emit("overLimit", "misc", "func overLimit(size, limit int) bool", "func overLimit(size, limit int) bool {\n\treturn size > limit\n}\n", set(), "func overLimit(size, limit int) bool {\n\treturn false\n}\n")
    if have_bucket_agg:
        emit("bucketSize", "misc", "func bucketSize(s *Store, b int) int", "func bucketSize(s *Store, b int) int {\n\treturn len(s.buckets[b])\n}\n", set(), "func bucketSize(s *Store, b int) int {\n\treturn 0\n}\n")
        emit(
            "bucketBytes",
            "misc",
            "func bucketBytes(s *Store, b int) int",
            (
                "func bucketBytes(s *Store, b int) int {\n"
                "\tn := 0\n"
                "\tfor _, e := range s.buckets[b] {\n"
                "\t\tn += e.size\n"
                "\t}\n"
                "\treturn n\n"
                "}\n"
            ),
            set(),
            "func bucketBytes(s *Store, b int) int {\n\treturn 0\n}\n",
        )

    # --- leaf --------------------------------------------------------------
    emit(
        "defaultLimit",
        "leaf",
        "func defaultLimit() int",
        "func defaultLimit() int {\n\treturn paramStoreLimit\n}\n",
        set(),
        "func defaultLimit() int {\n\treturn 100\n}\n",
    )

    # --- consumers ---------------------------------------------------------
    present = set(STORE_ENTRIES[: cfg.E])
    consumer_bodies = {
        "snapshot": _go(
            "func snapshot(s *Store) string {",
            '\treturn fmt.Sprintf("n=%d total=%d bal=%d tags=%d", s.Count(), s.Total(), s.Balance(), len(s.Tags("")))',
            "}",
            imports={"fmt"},
        ),
        "exportAll": _go(
            "func exportAll(s *Store) ([]string, int) {",
            '\treturn s.Tags(""), s.Total()',
            "}",
        ),
        "hotBucket": _go(
            "func hotBucket(s *Store) (int, int) {",
            "\treturn s.Balance(), s.BucketOf(1)",
            "}",
        ),
        "diffKeys": _go(
            "func diffKeys(s *Store, keys []uint64) int {",
            "\tn := 0",
            "\tfor _, k := range keys {",
            "\t\tif _, ok := s.Get(k); ok {",
            "\t\t\tn++",
            "\t\t}",
            "\t}",
            "\treturn n",
            "}",
        ),
        "hasAny": _go(
            "func hasAny(s *Store) bool {",
            "\treturn s.Count() > 0",
            "}",
        ),
        "budgetLeft": _go(
            "func budgetLeft(s *Store, budget int) int {",
            "\treturn budget - s.Total()",
            "}",
        ),
    }
    consumer_sigs = {
        "snapshot": "func snapshot(s *Store) string",
        "exportAll": "func exportAll(s *Store) ([]string, int)",
        "hotBucket": "func hotBucket(s *Store) (int, int)",
        "diffKeys": "func diffKeys(s *Store, keys []uint64) int",
        "hasAny": "func hasAny(s *Store) bool",
        "budgetLeft": "func budgetLeft(s *Store, budget int) int",
    }
    for cname in cfg.consumers:
        if not set(STORE_CONSUMERS[cname]).issubset(present):
            continue
        body, imports = consumer_bodies[cname]
        emit(cname, "consumer", consumer_sigs[cname], body, imports)

    # --- fillers -----------------------------------------------------------
    filler_bodies = {
        "clampInt": "func clampInt(x, lo, hi int) int {\n\tif x < lo {\n\t\treturn lo\n\t}\n\tif x > hi {\n\t\treturn hi\n\t}\n\treturn x\n}\n",
        "absDiff": "func absDiff(a, b int) int {\n\tif a > b {\n\t\treturn a - b\n\t}\n\treturn b - a\n}\n",
        "asciiSum": "func asciiSum(s string) int {\n\tn := 0\n\tfor i := 0; i < len(s); i++ {\n\t\tn += int(s[i])\n\t}\n\treturn n\n}\n",
        "minU64": "func minU64(a, b uint64) uint64 {\n\tif a < b {\n\t\treturn a\n\t}\n\treturn b\n}\n",
        "parity": "func parity(x uint64) int {\n\treturn int(x & 1)\n}\n",
        "digitSum": "func digitSum(x uint64) int {\n\tn := 0\n\tfor x > 0 {\n\t\tn += int(x % 10)\n\t\tx /= 10\n\t}\n\treturn n\n}\n",
        "revStr": "func revStr(s string) string {\n\tb := []byte(s)\n\tfor i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {\n\t\tb[i], b[j] = b[j], b[i]\n\t}\n\treturn string(b)\n}\n",
        "pow2ceil": "func pow2ceil(x int) int {\n\tif x <= 1 {\n\t\treturn 1\n\t}\n\tp := 1\n\tfor p < x {\n\t\tp <<= 1\n\t}\n\treturn p\n}\n",
    }
    for fname in STORE_FILLERS[: cfg.fill]:
        body = filler_bodies[fname]
        sig = body.splitlines()[0].removesuffix("{")
        emit(fname, "filler", sig, body, set())

    return fns


def store_edges(cfg: Config) -> list[tuple[str, str]]:
    present = set(STORE_ENTRIES[: cfg.E])
    edges: list[tuple[str, str]] = [
        ("mixPipe", "mixStage1"),
        *[(f"mixStage{j + 1}", f"mixStage{j + 2}") for j in range(cfg.K - 1)],
        ("Store.BucketOf", "mixPipe"),
        ("Store.Put", "validateKey"),
        ("Store.Put", "canon"),
        ("Store.Put", "overLimit"),
        ("Store.Put", "Store.BucketOf"),
        ("Store.Get", "validateKey"),
        ("Store.Get", "Store.BucketOf"),
        ("NewStore", "defaultLimit"),
    ]
    if "Store.Total" in present:
        edges.append(("Store.Total", "bucketBytes"))
    if "Store.Count" in present:
        edges.append(("Store.Count", "bucketSize"))
    if "Store.Balance" in present:
        edges.append(("Store.Balance", "bucketSize"))
    if "Store.Tags" in present:
        edges.extend([("Store.Tags", "canon"), ("Store.Tags", "foldTag")])
    if "Store.Prune" in present:
        edges.append(("Store.Prune", "overLimit"))
    if cfg.canon_mode == "pipe":
        edges.extend([("canon", "trimTag"), ("canon", "lowerTag"), ("canon", "collapseTag")])
    for cname in cfg.consumers:
        if not set(STORE_CONSUMERS[cname]).issubset(present):
            continue
        for target in STORE_CONSUMERS[cname]:
            if target in present:
                edges.append((cname, target))
    return edges


def store_groups_in_S(cfg: Config) -> set[str]:
    groups = {"entry", "mixer"}
    if cfg.canon_in_S:
        groups.add("canon")
    if cfg.misc_in_S:
        groups.add("misc")
    return groups


STORE_TYPES = """type Record struct {
	Key  uint64
	Size int
	Tag  string
}

type entry struct {
	key  uint64
	size int
	tag  string
}

type Store struct {
	limit   int
	buckets [nbuckets][]entry
}
"""

STORE_PKG_DOC = "Package store is a fabricated record store (generated by openswe_traces.synth.fabricate)."


# --------------------------------------------------------------------------- #
# sched domain
# --------------------------------------------------------------------------- #
SCHED_ENTRIES = [
    "NewScheduler",
    "Sched.Slot",
    "Sched.Add",
    "Sched.Overlaps",
    "Sched.At",
    "Sched.Len",
    "Sched.Total",
    "Sched.Gap",
    "Sched.Merge",
]

SCHED_CONSUMERS = {
    "coverage": ("Sched.Total", "Sched.Len"),
    "busy": ("Sched.Len", "Sched.At"),
    "bandStats": ("Sched.Slot",),
    "density": ("Sched.Total", "Sched.Gap"),
    "freeAt": ("Sched.At",),
    "mergeCount": ("Sched.Merge",),
    "touch": ("Sched.Len", "Sched.Merge"),
    "span": ("Sched.Gap", "Sched.Total"),
}

SCHED_FILLERS = STORE_FILLERS


def sched_consts(seed: int) -> dict[str, object]:
    rng = SplitMix(seed)
    return {
        "nbuckets": 8,
        "mixMul": [rng.next() | 1 for _ in range(5)],
        "mixAdd": [rng.next() for _ in range(5)],
        "mixRot": 1 + rng.below(62),
        "cap": 1000 + rng.below(9001),
    }


def sched_functions(cfg: Config) -> dict[str, FnSpec]:
    have_total = "Sched.Total" in SCHED_ENTRIES[: cfg.E]
    have_gap = "Sched.Gap" in SCHED_ENTRIES[: cfg.E]
    have_merge = "Sched.Merge" in SCHED_ENTRIES[: cfg.E]
    present = set(SCHED_ENTRIES[: cfg.E])

    fns: dict[str, FnSpec] = {}

    def emit(key: str, group: str, sig: str, gold: str, imports: set[str], cheat: str = "", cheat_imports: set[str] | None = None) -> None:
        name = key.split(".")[-1]
        fns[key] = FnSpec(
            key=key,
            name=name,
            exported=key.startswith("Sched.") or key == "NewScheduler",
            group=group,
            sig=sig,
            gold=gold,
            gold_imports=frozenset(imports),
            cheat=cheat,
            cheat_imports=frozenset(cheat_imports or set()),
        )

    emit(
        "NewScheduler",
        "entry",
        "func NewScheduler() *Scheduler",
        "func NewScheduler() *Scheduler {\n\treturn &Scheduler{cap: defaultCap()}\n}\n",
        set(),
        "func NewScheduler() *Scheduler {\n\treturn &Scheduler{cap: 100}\n}\n",
    )
    emit(
        "Sched.Slot",
        "entry",
        "func (sc *Scheduler) Slot(x int) int",
        "func (sc *Scheduler) Slot(x int) int {\n\treturn band(x, sc.cap)\n}\n",
        set(),
        "func (sc *Scheduler) Slot(x int) int {\n\treturn x % 8\n}\n",
    )
    emit(
        "Sched.Add",
        "entry",
        "func (sc *Scheduler) Add(iv Interval) bool",
        (
            "func (sc *Scheduler) Add(iv Interval) bool {\n"
            "\tn := norm(iv, sc.cap)\n"
            "\tif n.Start >= n.End {\n\t\treturn false\n\t}\n"
            "\tif !canPlace(sc, n) {\n\t\treturn false\n\t}\n"
            "\tinsertAt(sc, n)\n"
            "\treturn true\n"
            "}\n"
        ),
        set(),
        "func (sc *Scheduler) Add(iv Interval) bool {\n\treturn true\n}\n",
    )
    emit(
        "Sched.Overlaps",
        "entry",
        "func (sc *Scheduler) Overlaps(iv Interval) bool",
        (
            "func (sc *Scheduler) Overlaps(iv Interval) bool {\n"
            "\tn := norm(iv, sc.cap)\n"
            "\tfor _, s := range sc.slots {\n"
            "\t\tif overlaps(s, n) {\n\t\t\treturn true\n\t\t}\n"
            "\t}\n"
            "\treturn false\n"
            "}\n"
        ),
        set(),
        "func (sc *Scheduler) Overlaps(iv Interval) bool {\n\treturn false\n}\n",
    )
    if "Sched.At" in present:
        emit(
            "Sched.At",
            "entry",
            "func (sc *Scheduler) At(x int) bool",
            "func (sc *Scheduler) At(x int) bool {\n\treturn findSlot(sc, x) >= 0\n}\n",
            set(),
            "func (sc *Scheduler) At(x int) bool {\n\treturn false\n}\n",
        )
    if "Sched.Len" in present:
        emit("Sched.Len", "entry", "func (sc *Scheduler) Len() int", "func (sc *Scheduler) Len() int {\n\treturn len(sc.slots)\n}\n", set(), "func (sc *Scheduler) Len() int {\n\treturn 0\n}\n")
    if have_total:
        emit(
            "Sched.Total",
            "entry",
            "func (sc *Scheduler) Total() int",
            (
                "func (sc *Scheduler) Total() int {\n"
                "\tt := 0\n"
                "\tfor _, s := range sc.slots {\n"
                "\t\tt += length(s)\n"
                "\t}\n"
                "\treturn t\n"
                "}\n"
            ),
            set(),
            "func (sc *Scheduler) Total() int {\n\treturn 0\n}\n",
        )
    if have_gap:
        emit(
            "Sched.Gap",
            "entry",
            "func (sc *Scheduler) Gap() int",
            (
                "func (sc *Scheduler) Gap() int {\n"
                "\tbest := 0\n"
                "\tprev := 0\n"
                "\tfor _, s := range sc.slots {\n"
                "\t\tif s.Start > prev && length(Interval{prev, s.Start}) > best {\n"
                "\t\t\tbest = length(Interval{prev, s.Start})\n"
                "\t\t}\n"
                "\t\tif s.End > prev {\n"
                "\t\t\tprev = s.End\n"
                "\t\t}\n"
                "\t}\n"
                "\tif cap0 := sc.cap; cap0 > prev && length(Interval{prev, cap0}) > best {\n"
                "\t\tbest = length(Interval{prev, cap0})\n"
                "\t}\n"
                "\treturn best\n"
                "}\n"
            ),
            set(),
            "func (sc *Scheduler) Gap() int {\n\treturn 0\n}\n",
        )
    if have_merge:
        emit(
            "Sched.Merge",
            "entry",
            "func (sc *Scheduler) Merge() int",
            (
                "func (sc *Scheduler) Merge() int {\n"
                "\tif len(sc.slots) < 2 {\n\t\treturn 0\n\t}\n"
                "\tmerged := 0\n"
                "\tout := sc.slots[:1]\n"
                "\tfor _, iv := range sc.slots[1:] {\n"
                "\t\tlast := &out[len(out)-1]\n"
                "\t\tif overlaps(*last, iv) || last.End >= iv.Start {\n"
                "\t\t\tif iv.End > last.End {\n"
                "\t\t\t\tlast.End = iv.End\n"
                "\t\t\t}\n"
                "\t\t\tmerged++\n"
                "\t\t} else {\n"
                "\t\t\tout = append(out, iv)\n"
                "\t\t}\n"
                "\t}\n"
                "\tsc.slots = out\n"
                "\treturn merged\n"
                "}\n"
            ),
            set(),
            "func (sc *Scheduler) Merge() int {\n\treturn 0\n}\n",
        )

    # --- mixers ------------------------------------------------------------
    emit(
        "mixPipe",
        "mixer",
        "func mixPipe(x uint64) uint64",
        (
            "func mixPipe(x uint64) uint64 {\n"
            "\tv := mixStage1(x)\n"
            "\treturn bits.RotateLeft64(v, paramMixRot)\n"
            "}\n"
        ),
        {"math/bits"},
        "func mixPipe(x uint64) uint64 {\n\treturn x\n}\n",
    )
    for j in range(cfg.K):
        stage_key = f"mixStage{j + 1}"
        body_next = f"return mixStage{j + 2}(v)\n" if j < cfg.K - 1 else "return v\n"
        emit(
            stage_key,
            "mixer",
            f"func {stage_key}(x uint64) uint64",
            (
                f"func {stage_key}(x uint64) uint64 {{\n"
                f"\tv := x*paramStageMul{j + 1} + paramStageAdd{j + 1}\n"
                f"\t{body_next}"
                "}\n"
            ),
            set(),
            f"func {stage_key}(x uint64) uint64 {{\n\treturn x*3 + 1\n}}\n",
        )

    # --- norm group --------------------------------------------------------
    if cfg.canon_mode == "pipe":
        emit("swapOrder", "canon", "func swapOrder(iv Interval) Interval", "func swapOrder(iv Interval) Interval {\n\tif iv.Start > iv.End {\n\t\treturn Interval{Start: iv.End, End: iv.Start}\n\t}\n\treturn iv\n}\n", set())
        emit(
            "clampBoth",
            "canon",
            "func clampBoth(iv Interval, cap int) Interval",
            (
                "func clampBoth(iv Interval, cap int) Interval {\n"
                "\tif iv.Start < 0 {\n\t\tiv.Start = 0\n\t}\n"
                "\tif iv.End > cap {\n\t\tiv.End = cap\n\t}\n"
                "\tif iv.End < iv.Start {\n\t\tiv.End = iv.Start\n\t}\n"
                "\treturn iv\n"
                "}\n"
            ),
            set(),
        )
        emit("norm", "canon", "func norm(iv Interval, cap int) Interval", "func norm(iv Interval, cap int) Interval {\n\treturn clampBoth(swapOrder(iv), cap)\n}\n", set())
    else:
        emit(
            "norm",
            "canon",
            "func norm(iv Interval, cap int) Interval",
            (
                "func norm(iv Interval, cap int) Interval {\n"
                "\tif iv.Start > iv.End {\n"
                "\t\tiv.Start, iv.End = iv.End, iv.Start\n"
                "\t}\n"
                "\tif iv.Start < 0 {\n\t\tiv.Start = 0\n\t}\n"
                "\tif iv.End > cap {\n\t\tiv.End = cap\n\t}\n"
                "\tif iv.End < iv.Start {\n\t\tiv.End = iv.Start\n\t}\n"
                "\treturn iv\n"
                "}\n"
            ),
            set(),
        )

    # --- place + misc ------------------------------------------------------
    emit(
        "canPlace",
        "place",
        "func canPlace(sc *Scheduler, iv Interval) bool",
        (
            "func canPlace(sc *Scheduler, iv Interval) bool {\n"
            "\tfor _, s := range sc.slots {\n"
            "\t\tif overlaps(s, iv) {\n\t\t\treturn false\n\t\t}\n"
            "\t}\n"
            "\treturn true\n"
            "}\n"
        ),
        set(),
        "func canPlace(sc *Scheduler, iv Interval) bool {\n\treturn true\n}\n",
    )
    emit(
        "overlaps",
        "misc",
        "func overlaps(a, b Interval) bool",
        "func overlaps(a, b Interval) bool {\n\treturn a.Start < b.End && b.Start < a.End\n}\n",
        set(),
        "func overlaps(a, b Interval) bool {\n\treturn false\n}\n",
    )
    if have_total or have_gap:
        emit("length", "misc", "func length(iv Interval) int", "func length(iv Interval) int {\n\tif iv.End < iv.Start {\n\t\treturn 0\n\t}\n\treturn iv.End - iv.Start\n}\n", set(), "func length(iv Interval) int {\n\treturn 0\n}\n")
    emit(
        "band",
        "misc",
        "func band(x, cap int) int",
        (
            "func band(x, cap int) int {\n"
            "\tif cap <= 0 {\n\t\treturn 0\n\t}\n"
            "\tif x < 0 {\n\t\tx = 0\n\t}\n"
            "\tif x >= cap {\n\t\tx = cap - 1\n\t}\n"
            "\treturn int(mixPipe(uint64(x)) % nbuckets)\n"
            "}\n"
        ),
        set(),
        "func band(x, cap int) int {\n\treturn x % 8\n}\n",
    )
    emit(
        "insertAt",
        "misc",
        "func insertAt(sc *Scheduler, iv Interval)",
        (
            "func insertAt(sc *Scheduler, iv Interval) {\n"
            "\ti := 0\n"
            "\tfor i < len(sc.slots) && sc.slots[i].Start < iv.Start {\n"
            "\t\ti++\n"
            "\t}\n"
            "\tsc.slots = append(sc.slots, Interval{})\n"
            "\tcopy(sc.slots[i+1:], sc.slots[i:])\n"
            "\tsc.slots[i] = iv\n"
            "}\n"
        ),
        set(),
    )
    if "Sched.At" in present:
        emit(
            "findSlot",
            "misc",
            "func findSlot(sc *Scheduler, x int) int",
            (
                "func findSlot(sc *Scheduler, x int) int {\n"
                "\tlo, hi := 0, len(sc.slots)\n"
                "\tfor lo < hi {\n"
                "\t\tmid := (lo + hi) / 2\n"
                "\t\tif sc.slots[mid].End <= x {\n"
                "\t\t\tlo = mid + 1\n"
                "\t\t} else {\n"
                "\t\t\thi = mid\n"
                "\t\t}\n"
                "\t}\n"
                "\tif lo < len(sc.slots) && sc.slots[lo].Start <= x && x < sc.slots[lo].End {\n"
                "\t\treturn lo\n"
                "\t}\n"
                "\treturn -1\n"
                "}\n"
            ),
            set(),
            "func findSlot(sc *Scheduler, x int) int {\n\treturn -1\n}\n",
        )

    # --- leaf --------------------------------------------------------------
    emit(
        "defaultCap",
        "leaf",
        "func defaultCap() int",
        "func defaultCap() int {\n\treturn paramCap\n}\n",
        set(),
        "func defaultCap() int {\n\treturn 100\n}\n",
    )

    # --- consumers ---------------------------------------------------------
    consumer_bodies = {
        "coverage": "func coverage(sc *Scheduler) float64 {\n\tif sc.Len() == 0 {\n\t\treturn 0\n\t}\n\treturn float64(sc.Total()) / float64(sc.cap)\n}\n",
        "busy": "func busy(sc *Scheduler, x int) bool {\n\treturn sc.Len() > 0 && sc.At(x)\n}\n",
        "bandStats": "func bandStats(sc *Scheduler) int {\n\treturn sc.Slot(0)\n}\n",
        "density": "func density(sc *Scheduler) int {\n\treturn sc.Total() + sc.Gap()\n}\n",
        "freeAt": "func freeAt(sc *Scheduler, x int) bool {\n\treturn !sc.At(x)\n}\n",
        "mergeCount": "func mergeCount(sc *Scheduler) int {\n\treturn sc.Merge()\n}\n",
        "touch": "func touch(sc *Scheduler) int {\n\treturn sc.Len() + sc.Merge()\n}\n",
        "span": "func span(sc *Scheduler) int {\n\treturn sc.Gap() + sc.Total()\n}\n",
    }
    consumer_sigs = {
        "coverage": "func coverage(sc *Scheduler) float64",
        "busy": "func busy(sc *Scheduler, x int) bool",
        "bandStats": "func bandStats(sc *Scheduler) int",
        "density": "func density(sc *Scheduler) int",
        "freeAt": "func freeAt(sc *Scheduler, x int) bool",
        "mergeCount": "func mergeCount(sc *Scheduler) int",
        "touch": "func touch(sc *Scheduler) int",
        "span": "func span(sc *Scheduler) int",
    }
    for cname in cfg.consumers:
        if not set(SCHED_CONSUMERS[cname]).issubset(present):
            continue
        emit(cname, "consumer", consumer_sigs[cname], consumer_bodies[cname], set())

    # --- fillers -----------------------------------------------------------
    filler_bodies = {
        "clampInt": "func clampInt(x, lo, hi int) int {\n\tif x < lo {\n\t\treturn lo\n\t}\n\tif x > hi {\n\t\treturn hi\n\t}\n\treturn x\n}\n",
        "absDiff": "func absDiff(a, b int) int {\n\tif a > b {\n\t\treturn a - b\n\t}\n\treturn b - a\n}\n",
        "asciiSum": "func asciiSum(s string) int {\n\tn := 0\n\tfor i := 0; i < len(s); i++ {\n\t\tn += int(s[i])\n\t}\n\treturn n\n}\n",
        "minU64": "func minU64(a, b uint64) uint64 {\n\tif a < b {\n\t\treturn a\n\t}\n\treturn b\n}\n",
        "parity": "func parity(x uint64) int {\n\treturn int(x & 1)\n}\n",
        "digitSum": "func digitSum(x uint64) int {\n\tn := 0\n\tfor x > 0 {\n\t\tn += int(x % 10)\n\t\tx /= 10\n\t}\n\treturn n\n}\n",
        "revStr": "func revStr(s string) string {\n\tb := []byte(s)\n\tfor i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {\n\t\tb[i], b[j] = b[j], b[i]\n\t}\n\treturn string(b)\n}\n",
        "pow2ceil": "func pow2ceil(x int) int {\n\tif x <= 1 {\n\t\treturn 1\n\t}\n\tp := 1\n\tfor p < x {\n\t\tp <<= 1\n\t}\n\treturn p\n}\n",
    }
    for fname in SCHED_FILLERS[: cfg.fill]:
        body = filler_bodies[fname]
        sig = body.splitlines()[0].removesuffix("{")
        emit(fname, "filler", sig, body, set())

    return fns


def sched_edges(cfg: Config) -> list[tuple[str, str]]:
    present = set(SCHED_ENTRIES[: cfg.E])
    edges: list[tuple[str, str]] = [
        ("mixPipe", "mixStage1"),
        *[(f"mixStage{j + 1}", f"mixStage{j + 2}") for j in range(cfg.K - 1)],
        ("Sched.Slot", "band"),
        ("band", "mixPipe"),
        ("Sched.Add", "norm"),
        ("Sched.Add", "canPlace"),
        ("Sched.Add", "insertAt"),
        ("canPlace", "overlaps"),
        ("Sched.Overlaps", "norm"),
        ("Sched.Overlaps", "overlaps"),
        ("Sched.At", "findSlot"),
        ("NewScheduler", "defaultCap"),
    ]
    if "Sched.Total" in present:
        edges.append(("Sched.Total", "length"))
    if "Sched.Gap" in present:
        edges.append(("Sched.Gap", "length"))
    if "Sched.Merge" in present:
        edges.append(("Sched.Merge", "overlaps"))
    if cfg.canon_mode == "pipe":
        edges.extend([("norm", "swapOrder"), ("norm", "clampBoth")])
    for cname in cfg.consumers:
        if not set(SCHED_CONSUMERS[cname]).issubset(present):
            continue
        for target in SCHED_CONSUMERS[cname]:
            if target in present:
                edges.append((cname, target))
    return edges


def sched_groups_in_S(cfg: Config) -> set[str]:
    groups = {"entry", "mixer"}
    if cfg.canon_in_S:
        groups.add("canon")
    if cfg.place_in_S:
        groups.add("place")
    if cfg.misc_in_S:
        groups.add("misc")
    return groups


SCHED_TYPES = """type Interval struct {
	Start int
	End   int
}

type Scheduler struct {
	cap   int
	slots []Interval
}
"""

SCHED_PKG_DOC = "Package sched is a fabricated interval scheduler (generated by openswe_traces.synth.fabricate)."


# --------------------------------------------------------------------------- #
# domain registry
# --------------------------------------------------------------------------- #
DOMAINS = {
    "store": {
        "pkg": "store",
        "types": STORE_TYPES,
        "pkg_doc": STORE_PKG_DOC,
        "entries": STORE_ENTRIES,
        "consumers": STORE_CONSUMERS,
        "consts": store_consts,
        "functions": store_functions,
        "edges": store_edges,
        "groups_in_S": store_groups_in_S,
    },
    "sched": {
        "pkg": "sched",
        "types": SCHED_TYPES,
        "pkg_doc": SCHED_PKG_DOC,
        "entries": SCHED_ENTRIES,
        "consumers": SCHED_CONSUMERS,
        "consts": sched_consts,
        "functions": sched_functions,
        "edges": sched_edges,
        "groups_in_S": sched_groups_in_S,
    },
}


# --------------------------------------------------------------------------- #
# detail registry (the difficulty dial)
# --------------------------------------------------------------------------- #
def _mix_consts(cfg: Config) -> tuple[list[int], list[int], int]:
    c = cfg.consts
    return (
        [int(x) for x in c["mixMul"][: cfg.K]],  # type: ignore[index]
        [int(x) for x in c["mixAdd"][: cfg.K]],  # type: ignore[index]
        int(c["mixRot"]),
    )


def _pick_where(cfg: Config, gold, trap, pool: list[int], want: int) -> list[int]:
    out = [k for k in pool if gold(k) != trap(k)]
    if not out:
        raise ValueError(f"no discriminating input in pool (seed {cfg.seed})")
    return out[:want]


def _canon_order_set(cfg: Config) -> tuple[list[str], list[str]]:
    """(chosen canonical tags, fold-hash order) with fold order != lex order."""
    fold_base = int(cfg.consts["foldBase"])
    pool = ["alpha", "beta", "gamma", "delta", "zeta", "x1", "q1", "zebra", "iota", "mu"]
    rng = SplitMix(cfg.seed ^ 0x0D1A1)
    for _ in range(200):
        chosen = sorted(pool, key=lambda _t: rng.next())[:6]
        lex = sorted(chosen)
        fold = sorted(chosen, key=lambda t: (_fold_tag(t, fold_base), t))
        if fold != lex:
            return chosen, fold
    raise ValueError(f"could not find a fold-vs-lex order set (seed {cfg.seed})")


# --- store details -----------------------------------------------------------
def _detail_store_bucket_reduction(cfg: Config) -> dict[str, object]:
    mul, adds, rot = _mix_consts(cfg)
    nb = int(cfg.consts["nbuckets"])

    def gold(k: int) -> int:
        return ref_bucket(k, mul, adds, rot, nb)

    def trap(k: int) -> int:
        return (ref_mix(k, mul, adds, rot) >> 61) & (nb - 1)

    pool = [0, 1, 2, 3, 7, 42, 999, 1000, 1 << 40, (1 << 64) - 2]
    shown = _pick_where(cfg, gold, trap, pool, 3)
    cases = ", ".join(f"{k}: {gold(k)}" for k in shown)
    smoke = (
        "func TestSmoke_bucket_reduction(t *testing.T) {\n"
        "\ts := NewStore()\n"
        f"\tfor k, want := range map[uint64]int{{{cases}}} {{\n"
        "\t\tif got := s.BucketOf(k); got != want {\n"
        '\t\t\tt.Fatalf("BucketOf(%d) = %d, want %d", k, got, want)\n'
        "\t\t}\n"
        "\t}\n"
        "}\n"
    )
    fixed = [0, 1, 2, 3, 7, 42, 999, 1000, 1 << 40, (1 << 64) - 1, (1 << 64) - 2, 5000, 12345, 1 << 33]
    keys_src = ", ".join(str(k) for k in fixed)
    hidden = (
        "func TestDetail_bucket_reduction(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("bucket_reduction", pass) }()\n'
        "\ts := NewStore()\n"
        "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
        f"\tkeys := []uint64{{{keys_src}}}\n"
        "\tfor i := 0; i < 400; i++ {\n"
        "\t\tkeys = append(keys, rng.Uint64())\n"
        "\t}\n"
        "\tfor _, k := range keys {\n"
        "\t\tif got := s.BucketOf(k); got != refBucket(k) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("BucketOf(%d) = %d, want %d", k, got, refBucket(k))\n'
        "\t\t}\n"
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, {"math/bits"}),
        "shown": [f"key {k} -> bucket {gold(k)}" for k in shown],
    }


def _detail_store_tag_normalization(cfg: Config) -> dict[str, object]:
    pairs = [('"  Alpha beta "', '"alpha-beta"'), ('"x  y   z"', '"x-y-z"'), ('"  q-1  "', '"q-1"')]
    checks = "".join(
        f"\tif !s.Put({i + 1}, {raw}, 1) {{\n\t\tt.Fatal(\"put rejected\")\n\t}}\n"
        f"\tif rec, ok := s.Get({i + 1}); !ok || rec.Tag != {want} {{\n"
        f"\t\tt.Fatalf(\"Get({i + 1}) tag = %q %v, want %s\", rec.Tag, ok, {want})\n\t}}\n"
        for i, (raw, want) in enumerate(pairs)
    )
    smoke = (
        "func TestSmoke_tag_normalization(t *testing.T) {\n"
        "\ts := NewStore()\n"
        f"{checks}"
        "}\n"
    )
    raws = ["  Alpha beta ", "x  y   z", "UPPER", "a--b", "under_score", " q-1 ", "TAG_1", "  c  "]
    raws_src = ", ".join(_go_str(t) for t in raws)
    hidden = (
        "func TestDetail_tag_normalization(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("tag_normalization", pass) }()\n'
        "\ts := NewStore()\n"
        "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
        f"\traws := []string{{{raws_src}}}\n"
        "\tfor i := 0; i < 200; i++ {\n"
        "\t\traws = append(raws, raws[rng.Intn(len(raws))])\n"
        "\t}\n"
        "\tseen := 0\n"
        "\tfor i, raw := range raws {\n"
        '\t\tif refCanon(raw) == "" {\n'
        "\t\t\tcontinue\n"
        "\t\t}\n"
        "\t\tseen++\n"
        "\t\tif !s.Put(uint64(i+1), raw, 1) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Put(%q) rejected", raw)\n'
        "\t\t}\n"
        "\t\trec, ok := s.Get(uint64(i + 1))\n"
        "\t\tif !ok {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Get(%d) missing", i+1)\n'
        "\t\t}\n"
        "\t\tif rec.Tag != refCanon(raw) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Get(%d).Tag = %q, want %q", i+1, rec.Tag, refCanon(raw))\n'
        "\t\t}\n"
        "\t}\n"
        '\tif seen == 0 {\n\t\tt.Fatal("no live inputs")\n\t}\n'
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, {"strings"}),
        "shown": [f"{raw} -> {want}" for raw, want in pairs],
    }


def _detail_store_empty_tag(cfg: Config) -> dict[str, object]:
    # The rejected inputs are normalization-invariant empties (empty string and
    # whitespace), so this detail's test never depends on the normalization
    # rule: every plausible normalization collapses them to "".
    smoke = (
        "func TestSmoke_empty_tag(t *testing.T) {\n"
        "\ts := NewStore()\n"
        '\tif s.Put(1, "", 1) {\n\t\tt.Fatal("empty tag accepted")\n\t}\n'
        '\tif s.Put(1, "   ", 1) {\n\t\tt.Fatal("whitespace-only tag accepted")\n\t}\n'
        '\tif !s.Put(2, "ok", 1) {\n\t\tt.Fatal("valid tag rejected")\n\t}\n'
        "}\n"
    )
    hidden = (
        "func TestDetail_empty_tag(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("empty_tag_rejection", pass) }()\n'
        "\ts := NewStore()\n"
        '\tfor _, tag := range []string{"", "   ", "\\t", " \\t ", "      "} {\n'
        "\t\tif s.Put(1, tag, 1) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("empty tag %q accepted", tag)\n'
        "\t\t}\n"
        "\t}\n"
        '\tif !s.Put(2, "ok", 1) {\n\t\tpass = false\n\t\tt.Fatal("valid tag rejected")\n\t}\n'
        '\tif !s.Put(3, "a b", 1) {\n\t\tpass = false\n\t\tt.Fatal("normalizable tag rejected")\n\t}\n'
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": ['"" rejected', '"   " rejected', '"ok" accepted'],
    }


def _detail_store_reserved_key(cfg: Config) -> dict[str, object]:
    smoke = (
        "func TestSmoke_reserved_key(t *testing.T) {\n"
        "\ts := NewStore()\n"
        '\tif s.Put(paramReservedKey, "x", 1) {\n\t\tt.Fatal("reserved key accepted")\n\t}\n'
        '\tif !s.Put(0, "x", 1) {\n\t\tt.Fatal("key 0 rejected")\n\t}\n'
        '\tif !s.Put(paramReservedKey-1, "x", 1) {\n\t\tt.Fatal("second-largest key rejected")\n\t}\n'
        "}\n"
    )
    hidden = (
        "func TestDetail_reserved_key(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("reserved_key_rejection", pass) }()\n'
        "\ts := NewStore()\n"
        "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
        "\tkeys := []uint64{0, 1, ^uint64(0), ^uint64(0) - 1, 1 << 40}\n"
        "\tfor i := 0; i < 300; i++ {\n"
        "\t\tkeys = append(keys, rng.Uint64())\n"
        "\t}\n"
        "\tfor _, k := range keys {\n"
        '\t\tif ok := s.Put(k, "x", 1); ok == (k == ^uint64(0)) {\n'
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Put(%d) accepted=%v, want rejected only for the reserved key", k, ok)\n'
        "\t\t}\n"
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": ["reserved key rejected", "key 0 accepted", "second-largest key accepted"],
    }


def _detail_store_size_cap(cfg: Config) -> dict[str, object]:
    limit = int(cfg.consts["limit"])
    smoke = (
        "func TestSmoke_size_cap(t *testing.T) {\n"
        "\ts := NewStore()\n"
        '\tif !s.Put(1, "ok", paramStoreLimit) {\n\t\tt.Fatal("at-limit record rejected")\n\t}\n'
        '\tif s.Put(1, "ok", paramStoreLimit+1) {\n\t\tt.Fatal("oversize record accepted")\n\t}\n'
        '\tif !s.Put(2, "ok", 0) {\n\t\tt.Fatal("zero-size record rejected")\n\t}\n'
        "}\n"
    )
    hidden = (
        "func TestDetail_size_cap(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("size_cap", pass) }()\n'
        "\ts := NewStore()\n"
        "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
        "\tsizes := []int{0, refLimit - 1, refLimit, refLimit + 1}\n"
        "\tfor i := 0; i < 200; i++ {\n"
        "\t\tsizes = append(sizes, rng.Intn(refLimit+10))\n"
        "\t}\n"
        "\tfor i, size := range sizes {\n"
        '\t\tif ok := s.Put(uint64(i+1), "ok", size); ok == (size > refLimit) {\n'
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Put size %d accepted=%v, want %v", size, ok, size <= refLimit)\n'
        "\t\t}\n"
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": [f"size {limit} accepted", f"size {limit + 1} rejected", "size 0 accepted"],
    }


def _detail_store_tag_dedup(cfg: Config) -> dict[str, object]:
    smoke = (
        "func TestSmoke_tag_dedup(t *testing.T) {\n"
        "\ts := NewStore()\n"
        '\ts.Put(1, "zebra", 1)\n'
        '\ts.Put(2, "zebra", 1)\n'
        '\ts.Put(3, "alpha", 1)\n'
        '\tgot := s.Tags("")\n'
        '\tif len(got) != 2 {\n\t\tt.Fatalf("Tags() = %v, want 2 distinct tags", got)\n\t}\n'
        '\tif !slices.Contains(got, "zebra") || !slices.Contains(got, "alpha") {\n'
        '\t\tt.Fatalf("Tags() = %v, missing zebra or alpha", got)\n\t}\n'
        "}\n"
    )
    hidden = (
        "func TestDetail_tag_dedup(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("tag_dedup", pass) }()\n'
        "\ts := NewStore()\n"
        "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
        '\tpool := []string{"zebra", "alpha", "beta", "gamma", "delta"}\n'
        "\tvar stored []string\n"
        "\tfor i := 0; i < 120; i++ {\n"
        "\t\tstored = append(stored, pool[rng.Intn(len(pool))])\n"
        "\t}\n"
        "\tfor i, tag := range stored {\n"
        "\t\ts.Put(uint64(i+1), tag, 1)\n"
        "\t}\n"
        "\twant := map[string]bool{}\n"
        "\tfor _, tag := range stored {\n"
        "\t\twant[tag] = true\n"
        "\t}\n"
        '\tgot := s.Tags("")\n'
        "\tif len(got) != len(want) {\n"
        "\t\tpass = false\n"
        '\t\tt.Fatalf("Tags() = %v, want %d distinct tags", got, len(want))\n'
        "\t}\n"
        "\tfor _, tag := range got {\n"
        "\t\tif !want[tag] {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("unexpected tag %q in listing", tag)\n'
        "\t\t}\n"
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, {"slices"}),
        "shown": ['storing "zebra" twice and "alpha" once lists exactly {zebra, alpha}'],
    }


def _detail_store_tag_ordering(cfg: Config) -> dict[str, object]:
    chosen, fold = _canon_order_set(cfg)
    # The hidden test stores strictly more tags than the smoke example (B3:
    # the L0 examples never pin the ordering on all inputs).
    extras = [t for t in ("iota", "mu", "zeta") if t not in chosen][:3]
    hidden_tags = chosen + extras
    hidden_fold = sorted(hidden_tags, key=lambda t: (_fold_tag(t, int(cfg.consts["foldBase"])), t))
    puts = "".join(f'\ts.Put({i + 1}, {_go_str(t)}, 1)\n' for i, t in enumerate(chosen))
    want_src = ", ".join(_go_str(t) for t in fold)
    smoke = (
        "func TestSmoke_tag_ordering(t *testing.T) {\n"
        "\ts := NewStore()\n"
        f"{puts}"
        f'\twant := []string{{{want_src}}}\n'
        '\tif got := s.Tags(""); !reflect.DeepEqual(got, want) {\n'
        '\t\tt.Fatalf("Tags() = %v, want %v", got, want)\n\t}\n'
        "}\n"
    )
    hidden_puts = "".join(f'\ts.Put({i + 1}, {_go_str(t)}, 1)\n' for i, t in enumerate(hidden_tags))
    hidden_want = ", ".join(_go_str(t) for t in hidden_fold)
    hidden = (
        "func TestDetail_tag_ordering(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("tag_ordering", pass) }()\n'
        "\ts := NewStore()\n"
        f"{hidden_puts}"
        f'\twant := []string{{{hidden_want}}}\n'
        '\tif got := s.Tags(""); !reflect.DeepEqual(got, want) {\n'
        "\t\tpass = false\n"
        '\t\tt.Fatalf("Tags() = %v, want %v", got, want)\n\t}\n'
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, {"reflect"}),
        "shown": [f"storing {chosen} lists {fold}"],
    }


def _detail_store_overwrite(cfg: Config) -> dict[str, object]:
    smoke = (
        "func TestSmoke_overwrite(t *testing.T) {\n"
        "\ts := NewStore()\n"
        '\tif !s.Put(1, "a", 10) {\n\t\tt.Fatal("put rejected")\n\t}\n'
        '\tif !s.Put(1, "b", 20) {\n\t\tt.Fatal("overwrite rejected")\n\t}\n'
        "\trec, ok := s.Get(1)\n"
        '\tif !ok {\n\t\tt.Fatal("record missing after overwrite")\n\t}\n'
        '\tif rec.Size != 20 || rec.Tag != "b" {\n'
        '\t\tt.Fatalf("Get(1) = %+v, want size 20 tag \\\"b\\\"", rec)\n\t}\n'
        '\tif n := s.Count(); n != 1 {\n\t\tt.Fatalf("Count = %d, want 1", n)\n\t}\n'
        "}\n"
    )
    hidden = (
        "func TestDetail_overwrite(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("overwrite_semantics", pass) }()\n'
        "\ts := NewStore()\n"
        "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
        "\tkeyPool := []uint64{0, 1, 2, 42, 999, 1 << 33}\n"
        "\tstate := map[uint64]string{}\n"
        "\tfor i := 0; i < 250; i++ {\n"
        "\t\tk := keyPool[rng.Intn(len(keyPool))]\n"
        "\t\tif rng.Intn(5) == 0 {\n"
        "\t\t\tk = rng.Uint64() &^ 1\n"
        "\t\t}\n"
        '\t\ttag := "t" + strconv.Itoa(i%7)\n'
        "\t\ts.Put(k, tag, 1)\n"
        "\t\tstate[k] = tag\n"
        "\t}\n"
        "\tfor k, want := range state {\n"
        "\t\trec, ok := s.Get(k)\n"
        "\t\tif !ok {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Get(%d) missing", k)\n'
        "\t\t}\n"
        "\t\tif rec.Tag != want {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Get(%d).Tag = %q, want %q", k, rec.Tag, want)\n'
        "\t\t}\n"
        "\t}\n"
        "\tif n := s.Count(); n != len(state) {\n"
        "\t\tpass = false\n"
        '\t\tt.Fatalf("Count = %d, want %d", n, len(state))\n'
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, {"strconv"}),
        "shown": ['storing key 1 twice keeps one record with the second values'],
    }


def _detail_store_prune(cfg: Config) -> dict[str, object]:
    smoke = (
        "func TestSmoke_prune(t *testing.T) {\n"
        "\ts := NewStore()\n"
        "\tfor i := 1; i <= 10; i++ {\n"
        '\t\ts.Put(uint64(i), "k", i*10)\n'
        "\t}\n"
        '\tif got := s.Prune(50); got != 5 {\n\t\tt.Fatalf("Prune(50) = %d, want 5", got)\n\t}\n'
        '\tif n := s.Count(); n != 5 {\n\t\tt.Fatalf("Count = %d, want 5", n)\n\t}\n'
        '\tif _, ok := s.Get(6); ok {\n\t\tt.Fatal("oversize record still present")\n\t}\n'
        '\tif _, ok := s.Get(5); !ok {\n\t\tt.Fatal("at-limit record removed")\n\t}\n'
        "}\n"
    )
    hidden = (
        "func TestDetail_prune(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("prune_boundary", pass) }()\n'
        "\ts := NewStore()\n"
        "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
        "\tsizes := make([]int, 80)\n"
        "\tfor i := range sizes {\n"
        "\t\tsizes[i] = rng.Intn(1000)\n"
        "\t}\n"
        "\tfor i, size := range sizes {\n"
        '\t\ts.Put(uint64(i+1), "k", size)\n'
        "\t}\n"
        "\tmax := 500\n"
        "\twant := 0\n"
        "\tfor _, size := range sizes {\n"
        "\t\tif size > max {\n"
        "\t\t\twant++\n"
        "\t\t}\n"
        "\t}\n"
        "\t// one record exactly at the boundary: a >= trap must remove it too\n"
        '\tif !s.Put(999, "k", max) {\n\t\tpass = false\n\t\tt.Fatal("at-limit record rejected")\n\t}\n'
        "\tif got := s.Prune(max); got != want {\n"
        "\t\tpass = false\n"
        '\t\tt.Fatalf("Prune(%d) = %d, want %d", max, got, want)\n'
        "\t}\n"
        "\tif n := s.Count(); n != len(sizes)+1-want {\n"
        "\t\tpass = false\n"
        '\t\tt.Fatalf("Count after Prune = %d, want %d", n, len(sizes)+1-want)\n'
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": ["sizes 10..100, Prune(50) removes exactly the 5 records above 50"],
    }


# --- sched details -----------------------------------------------------------
def _detail_sched_band_mapping(cfg: Config) -> dict[str, object]:
    # Clamp and reduction share the band() function and their observable
    # values entangle (an out-of-range worked example necessarily shows the
    # reduction too), so they are one detail: the whole band() contract.
    mul, adds, rot = _mix_consts(cfg)
    nb = int(cfg.consts["nbuckets"])
    cap = int(cfg.consts["cap"])

    def gold(x: int) -> int:
        return ref_slot(x, cap, mul, adds, rot, nb)

    def trap(x: int) -> int:
        x = min(max(x, 0), cap - 1)
        return (ref_mix(x, mul, adds, rot) >> 61) & (nb - 1)

    in_range = [0, 1, 2, 7, 42, cap - 1, cap // 2, cap // 3, cap - 5]
    out_range = [-1, -5, cap * 10, cap + 7, cap + 3, cap * 2 + 1]
    shown = _pick_where(cfg, gold, trap, in_range + out_range, 4)
    cases = ", ".join(f"{x}: {gold(x)}" for x in shown)
    smoke = (
        "func TestSmoke_band_mapping(t *testing.T) {\n"
        "\tsc := NewScheduler()\n"
        f"\tfor x, want := range map[int]int{{{cases}}} {{\n"
        "\t\tif got := sc.Slot(x); got != want {\n"
        '\t\t\tt.Fatalf("Slot(%d) = %d, want %d", x, got, want)\n'
        "\t\t}\n"
        "\t}\n"
        "}\n"
    )
    xs_src = ", ".join(str(x) for x in in_range + out_range)
    hidden = (
        "func TestDetail_band_mapping(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("band_mapping", pass) }()\n'
        "\tsc := NewScheduler()\n"
        "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
        f"\txs := []int{{{xs_src}}}\n"
        "\tfor i := 0; i < 400; i++ {\n"
        "\t\txs = append(xs, rng.Intn(refCap))\n"
        "\t}\n"
        "\tfor _, x := range xs {\n"
        "\t\tif got := sc.Slot(x); got != refSlot(x, refCap) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Slot(%d) = %d, want %d", x, got, refSlot(x, refCap))\n'
        "\t\t}\n"
        "\t}\n"
        "\t// equality of distinct out-of-range inputs: clamping maps both to the\n"
        "\t// same in-range point, so a no-clamp implementation diverges here\n"
        "\tif got := sc.Slot(-1); got != sc.Slot(-5) {\n"
        "\t\tpass = false\n"
        '\t\tt.Fatalf("Slot(-1) = %d != Slot(-5) = %d; expected clamping to one value", got, sc.Slot(-5))\n'
        "\t}\n"
        "\tif got := sc.Slot(refCap*2 + 1); got != sc.Slot(refCap*2 + 9) {\n"
        "\t\tpass = false\n"
        '\t\tt.Fatalf("Slot(2*cap+1) = %d != Slot(2*cap+9) = %d; expected clamping to one value", got, sc.Slot(refCap*2+9))\n'
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, {"math/bits"}),
        "shown": [f"point {x} -> band {gold(x)}" for x in shown],
    }


def _detail_sched_endpoint_swap(cfg: Config) -> dict[str, object]:
    smoke = (
        "func TestSmoke_endpoint_swap(t *testing.T) {\n"
        "\tsc := NewScheduler()\n"
        "\tif !sc.Add(Interval{Start: 30, End: 20}) {\n\t\tt.Fatal(\"reversed interval rejected\")\n\t}\n"
        '\tif n := sc.Len(); n != 1 {\n\t\tt.Fatalf("Len = %d, want 1", n)\n\t}\n'
        "\tif !sc.At(25) {\n\t\tt.Fatal(\"normalized interval not covered\")\n\t}\n"
        "\tif sc.At(15) {\n\t\tt.Fatal(\"coverage outside normalized interval\")\n\t}\n"
        "}\n"
    )
    hidden = (
        "func TestDetail_endpoint_swap(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("endpoint_swap", pass) }()\n'
        "\tsc := NewScheduler()\n"
        "\tfor _, iv := range []Interval{{Start: 30, End: 20}, {Start: 50, End: 40}, {Start: 70, End: 60}} {\n"
        "\t\tif !sc.Add(iv) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("reversed interval %+v rejected", iv)\n'
        "\t\t}\n"
        "\t}\n"
        '\tif n := sc.Len(); n != 3 {\n\t\tpass = false\n\t\tt.Fatalf("Len = %d, want 3", n)\n\t}\n'
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": ["[30,20) accepted and covers 25 but not 15"],
    }


def _detail_sched_clamp_capacity(cfg: Config) -> dict[str, object]:
    cap = int(cfg.consts["cap"])
    smoke = (
        "func TestSmoke_clamp_capacity(t *testing.T) {\n"
        "\tsc := NewScheduler()\n"
        "\tif !sc.Add(Interval{Start: -5, End: 5}) {\n\t\tt.Fatal(\"negative-start interval rejected\")\n\t}\n"
        "\tif !sc.At(2) {\n\t\tt.Fatal(\"clamped interval not covered\")\n\t}\n"
        "\tif !sc.Add(Interval{Start: paramCap - 5, End: paramCap + 10}) {\n"
        "\t\tt.Fatal(\"beyond-capacity interval rejected\")\n\t}\n"
        "\tif sc.At(paramCap + 2) {\n\t\tt.Fatal(\"coverage beyond capacity\")\n\t}\n"
        "}\n"
    )
    hidden = (
        "func TestDetail_clamp_capacity(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("clamp_capacity", pass) }()\n'
        "\tsc := NewScheduler()\n"
        "\tif !sc.Add(Interval{Start: -5, End: 5}) {\n\t\tpass = false\n\t\tt.Fatal(\"negative-start interval rejected\")\n\t}\n"
        "\tif !sc.Add(Interval{Start: 100, End: 200}) {\n"
        "\t\tpass = false\n\t\tt.Fatal(\"in-range interval rejected\")\n\t}\n"
        "\tif !sc.Add(Interval{Start: refCap - 5, End: refCap + 10}) {\n"
        "\t\tpass = false\n\t\tt.Fatal(\"beyond-capacity interval rejected\")\n\t}\n"
        '\tif n := sc.Len(); n != 3 {\n\t\tpass = false\n\t\tt.Fatalf("Len = %d, want 3", n)\n\t}\n'
        "\tfor x, want := range map[int]bool{2: true, 150: true, refCap - 3: true, refCap + 2: false, -1: false} {\n"
        "\t\tif got := sc.At(x); got != want {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("At(%d) = %v, want %v", x, got, want)\n'
        "\t\t}\n"
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": [
            "[-5,5) accepted and covers 2",
            f"[{cap}-5,{cap}+10) accepted and does not cover {cap}+2",
            f"[{cap}+10,{cap}+20) rejected",
        ],
    }


def _detail_sched_empty_interval(cfg: Config) -> dict[str, object]:
    smoke = (
        "func TestSmoke_empty_interval(t *testing.T) {\n"
        "\tsc := NewScheduler()\n"
        "\tif sc.Add(Interval{Start: 10, End: 10}) {\n\t\tt.Fatal(\"empty interval accepted\")\n\t}\n"
        "\tif sc.Add(Interval{Start: 5, End: 5}) {\n\t\tt.Fatal(\"empty interval accepted\")\n\t}\n"
        "}\n"
    )
    hidden = (
        "func TestDetail_empty_interval(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("empty_interval_reject", pass) }()\n'
        "\tsc := NewScheduler()\n"
        "\tfor _, iv := range []Interval{{Start: 10, End: 10}, {Start: 5, End: 5}, {Start: 0, End: 0}, {Start: refCap - 1, End: refCap - 1}} {\n"
        "\t\tif sc.Add(iv) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("empty interval %+v accepted", iv)\n'
        "\t\t}\n"
        "\t}\n"
        "\tif !sc.Add(Interval{Start: 10, End: 20}) {\n"
        "\t\tpass = false\n"
        '\t\tt.Fatal("non-empty interval rejected")\n\t}\n'
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": ["[10,10) rejected", "[5,5) rejected"],
    }


def _detail_sched_overlap_semantics(cfg: Config) -> dict[str, object]:
    # The half-open convention is one commitment with two observable faces:
    # Add accepts touching intervals (rejects only true overlaps) and Merge
    # coalesces touching-or-overlapping runs.  Folding them into one detail
    # keeps the registry free of root-cause entanglements (a schedule can only
    # ever contain disjoint or touching intervals, so merge behaviour is only
    # testable on touching pairs, which in turn requires the Add convention).
    smoke = (
        "func TestSmoke_overlap_semantics(t *testing.T) {\n"
        "\tsc := NewScheduler()\n"
        "\tif !sc.Add(Interval{Start: 0, End: 10}) {\n\t\tt.Fatal(\"first interval rejected\")\n\t}\n"
        "\tif !sc.Add(Interval{Start: 10, End: 20}) {\n\t\tt.Fatal(\"touching interval rejected\")\n\t}\n"
        "\tif sc.Add(Interval{Start: 5, End: 15}) {\n\t\tt.Fatal(\"overlapping interval accepted\")\n\t}\n"
        "\tsc.Add(Interval{Start: 30, End: 40})\n"
        '\tif got := sc.Merge(); got != 1 {\n\t\tt.Fatalf("Merge = %d, want 1", got)\n\t}\n'
        '\tif n := sc.Len(); n != 2 {\n\t\tt.Fatalf("Len = %d, want 2", n)\n\t}\n'
        "}\n"
    )
    hidden = (
        "func TestDetail_overlap_semantics(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("overlap_semantics", pass) }()\n'
        "\tsc := NewScheduler()\n"
        "\tadds := []Interval{{Start: 0, End: 10}, {Start: 20, End: 30}, {Start: 10, End: 20}, {Start: 15, End: 25}, {Start: 30, End: 40}, {Start: 5, End: 35}}\n"
        "\twant := []bool{true, true, true, false, true, false}\n"
        "\tfor i, iv := range adds {\n"
        "\t\tif got := sc.Add(iv); got != want[i] {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Add(%+v) = %v, want %v", iv, got, want[i])\n'
        "\t\t}\n"
        "\t}\n"
        "\t// touching runs coalesce into one (schedules can only hold disjoint\n"
        "\t// or touching intervals, so merge is testable on touching pairs only)\n"
        "\tsc2 := NewScheduler()\n"
        "\tfor _, iv := range []Interval{{Start: 0, End: 10}, {Start: 10, End: 20}, {Start: 25, End: 35}, {Start: 35, End: 45}, {Start: 50, End: 60}} {\n"
        "\t\tif !sc2.Add(iv) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Add(%+v) rejected", iv)\n'
        "\t\t}\n"
        "\t}\n"
        '\tif got := sc2.Merge(); got != 2 {\n\t\tpass = false\n\t\tt.Fatalf("Merge = %d, want 2", got)\n\t}\n'
        '\tif n := sc2.Len(); n != 3 {\n\t\tpass = false\n\t\tt.Fatalf("Len = %d, want 3", n)\n\t}\n'
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": ["[0,10) then [10,20) both accepted; [5,15) rejected; touching runs merge"],
    }


def _detail_sched_insertion_order(cfg: Config) -> dict[str, object]:
    smoke = (
        "func TestSmoke_insertion_order(t *testing.T) {\n"
        "\tsc := NewScheduler()\n"
        "\tif !sc.Add(Interval{Start: 30, End: 40}) {\n\t\tt.Fatal(\"first interval rejected\")\n\t}\n"
        "\tif !sc.Add(Interval{Start: 10, End: 20}) {\n\t\tt.Fatal(\"second interval rejected\")\n\t}\n"
        '\tif n := sc.Len(); n != 2 {\n\t\tt.Fatalf("Len = %d, want 2", n)\n\t}\n'
        "\tif !sc.At(15) {\n\t\tt.Fatal(\"earlier interval not covered\")\n\t}\n"
        "\tif !sc.At(35) {\n\t\tt.Fatal(\"later interval not covered\")\n\t}\n"
        "}\n"
    )
    hidden = (
        "func TestDetail_insertion_order(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("insertion_order", pass) }()\n'
        "\tsc := NewScheduler()\n"
        "\tfor _, iv := range []Interval{{Start: 30, End: 40}, {Start: 10, End: 20}, {Start: 50, End: 60}} {\n"
        "\t\tif !sc.Add(iv) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("Add(%+v) rejected", iv)\n'
        "\t\t}\n"
        "\t}\n"
        '\tif n := sc.Len(); n != 3 {\n\t\tpass = false\n\t\tt.Fatalf("Len = %d, want 3", n)\n\t}\n'
        "\tfor _, x := range []int{15, 35, 55} {\n"
        "\t\tif !sc.At(x) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("At(%d) = false, want true", x)\n'
        "\t\t}\n"
        "\t}\n"
        "\tfor _, x := range []int{5, 25, 45, 65} {\n"
        "\t\tif sc.At(x) {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("At(%d) = true, want false", x)\n'
        "\t\t}\n"
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": ["adding [30,40) then [10,20) leaves both covered"],
    }


def _detail_sched_gap(cfg: Config) -> dict[str, object]:
    cap = int(cfg.consts["cap"])
    smoke = (
        "func TestSmoke_gap(t *testing.T) {\n"
        "\tsc := NewScheduler()\n"
        "\tsc.Add(Interval{Start: 0, End: 10})\n"
        "\tsc.Add(Interval{Start: 20, End: 30})\n"
        '\tif got := sc.Gap(); got != paramCap-30 {\n\t\tt.Fatalf("Gap = %d, want %d", got, paramCap-30)\n\t}\n'
        "}\n"
    )
    hidden = (
        "func TestDetail_gap(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("gap_definition", pass) }()\n'
        "\tsc := NewScheduler()\n"
        "\tsc.Add(Interval{Start: 0, End: 10})\n"
        "\tsc.Add(Interval{Start: 20, End: 30})\n"
        '\tif got := sc.Gap(); got != refCap-30 {\n'
        "\t\tpass = false\n"
        '\t\tt.Fatalf("Gap = %d, want %d (trailing span counts)", got, refCap-30)\n\t}\n'
        "\tsc2 := NewScheduler()\n"
        "\tsc2.Add(Interval{Start: refCap - 20, End: refCap})\n"
        '\tif got := sc2.Gap(); got != refCap-20 {\n'
        "\t\tpass = false\n"
        '\t\tt.Fatalf("leading gap = %d, want %d", got, refCap-20)\n\t}\n'
        "\tsc3 := NewScheduler()\n"
        '\tif got := sc3.Gap(); got != refCap {\n'
        "\t\tpass = false\n"
        '\t\tt.Fatalf("empty-schedule gap = %d, want %d", got, refCap)\n\t}\n'
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": [f"[0,10) and [20,30): Gap = {cap}-30"],
    }


def _detail_sched_at_coverage(cfg: Config) -> dict[str, object]:
    smoke = (
        "func TestSmoke_at_coverage(t *testing.T) {\n"
        "\tsc := NewScheduler()\n"
        "\tif !sc.Add(Interval{Start: 10, End: 20}) {\n\t\tt.Fatal(\"interval rejected\")\n\t}\n"
        "\tif !sc.At(10) || !sc.At(19) {\n\t\tt.Fatal(\"start or inside point not covered\")\n\t}\n"
        "\tif sc.At(20) {\n\t\tt.Fatal(\"end point covered\")\n\t}\n"
        "\tif sc.At(9) {\n\t\tt.Fatal(\"outside point covered\")\n\t}\n"
        "}\n"
    )
    hidden = (
        "func TestDetail_at_coverage(t *testing.T) {\n"
        "\tpass := true\n"
        '\tdefer func() { recordDetail("at_coverage", pass) }()\n'
        "\tsc := NewScheduler()\n"
        "\tif !sc.Add(Interval{Start: 10, End: 20}) {\n\t\tpass = false\n\t\tt.Fatal(\"interval rejected\")\n\t}\n"
        "\tif !sc.Add(Interval{Start: 30, End: 40}) {\n\t\tpass = false\n\t\tt.Fatal(\"interval rejected\")\n\t}\n"
        "\tfor x, want := range map[int]bool{10: true, 19: true, 20: false, 9: false, 30: true, 39: true, 40: false, 29: false} {\n"
        "\t\tif got := sc.At(x); got != want {\n"
        "\t\t\tpass = false\n"
        '\t\t\tt.Fatalf("At(%d) = %v, want %v", x, got, want)\n'
        "\t\t}\n"
        "\t}\n"
        "}\n"
    )
    return {
        "smoke": smoke,
        "hidden": (hidden, set()),
        "shown": ["[10,20): 10 and 19 covered, 20 and 9 not"],
    }


STORE_DETAILS: dict[str, dict[str, object]] = {
    "bucket_reduction": {
        "requires": ("NewStore", "Store.BucketOf"),
        "description": "bucket index is the seeded key hash reduced modulo the bucket count (not a power-of-two mask)",
        "prose": "a record's bucket is derived from the seeded hash of its key, spread across all 8 buckets so that most distinct keys land in different buckets",
        "smoke_name": "TestSmoke_bucket_reduction",
        "hidden_name": "TestDetail_bucket_reduction",
        "perturbed_fns": ("Store.BucketOf",),
        "trap_bodies": {
            "Store.BucketOf": (
                "func (s *Store) BucketOf(k uint64) int {\n"
                "\treturn int((mixPipe(k) >> 61) & (nbuckets - 1))\n"
                "}\n"
            )
        },
        "gen": _detail_store_bucket_reduction,
    },
    "tag_normalization": {
        "requires": ("NewStore", "Store.Put", "Store.Get"),
        "description": "tag canonicalization: trim, lowercase, collapse non-alphanumeric runs to a single dash, trim leading/trailing dashes",
        "prose": "tags are normalized before storage: surrounding whitespace is trimmed, letters are lowercased, and every run of spaces or punctuation is collapsed to a single dash",
        "smoke_name": "TestSmoke_tag_normalization",
        "hidden_name": "TestDetail_tag_normalization",
        "perturbed_fns": ("canon",),
        "trap_bodies": {
            "canon": "func canon(tag string) string {\n\treturn strings.ToLower(strings.TrimSpace(tag))\n}\n"
        },
        "gen": _detail_store_tag_normalization,
    },
    "empty_tag_rejection": {
        "requires": ("NewStore", "Store.Put"),
        "description": "a tag whose canonical form is empty (whitespace/punctuation only) is rejected",
        "prose": "a tag that normalizes to nothing — only whitespace or punctuation — is rejected",
        "smoke_name": "TestSmoke_empty_tag",
        "hidden_name": "TestDetail_empty_tag",
        "perturbed_fns": ("Store.Put",),
        "trap_bodies": {
            "Store.Put": (
                "func (s *Store) Put(k uint64, tag string, size int) bool {\n"
                "\tif !validateKey(k) {\n\t\treturn false\n\t}\n"
                "\tt := canon(tag)\n"
                "\tif overLimit(size, s.limit) {\n\t\treturn false\n\t}\n"
                "\tb := s.BucketOf(k)\n"
                "\tfor i := range s.buckets[b] {\n"
                "\t\tif s.buckets[b][i].key == k {\n"
                "\t\t\ts.buckets[b][i].size = size\n"
                "\t\t\ts.buckets[b][i].tag = t\n"
                "\t\t\treturn true\n"
                "\t\t}\n"
                "\t}\n"
                "\ts.buckets[b] = append(s.buckets[b], entry{key: k, size: size, tag: t})\n"
                "\treturn true\n"
                "}\n"
            )
        },
        "gen": _detail_store_empty_tag,
    },
    "reserved_key_rejection": {
        "requires": ("NewStore", "Store.Put"),
        "description": "the all-ones key is rejected, every other key accepted",
        "prose": "the reserved key (2^64 − 1) is always rejected, while key 0 and the second-largest key are accepted",
        "smoke_name": "TestSmoke_reserved_key",
        "hidden_name": "TestDetail_reserved_key",
        "perturbed_fns": ("validateKey",),
        "trap_bodies": {
            "validateKey": "func validateKey(k uint64) bool {\n\treturn true\n}\n"
        },
        "gen": _detail_store_reserved_key,
    },
    "size_cap": {
        "requires": ("NewStore", "Store.Put"),
        "description": "size limit boundary: > limit rejected, == limit and 0 accepted",
        "prose": "a record whose size exceeds the store's limit is rejected, and a record exactly at the limit is accepted",
        "smoke_name": "TestSmoke_size_cap",
        "hidden_name": "TestDetail_size_cap",
        "perturbed_fns": ("overLimit",),
        "trap_bodies": {
            "overLimit": "func overLimit(size, limit int) bool {\n\treturn size >= limit\n}\n"
        },
        "gen": _detail_store_size_cap,
    },
    "tag_dedup": {
        "requires": ("NewStore", "Store.Put", "Store.Tags"),
        "description": "the tag listing returns each distinct tag at most once",
        "prose": "the tag listing returns each distinct tag at most once, even when many stored records share it",
        "smoke_name": "TestSmoke_tag_dedup",
        "hidden_name": "TestDetail_tag_dedup",
        "perturbed_fns": ("Store.Tags",),
        "trap_bodies": {
            "Store.Tags": (
                "func (s *Store) Tags(prefix string) []string {\n"
                "\tvar tags []string\n"
                "\tfor b := 0; b < nbuckets; b++ {\n"
                "\t\tfor _, e := range s.buckets[b] {\n"
                "\t\t\tt := canon(e.tag)\n"
                "\t\t\tif strings.HasPrefix(t, prefix) {\n"
                "\t\t\t\ttags = append(tags, t)\n"
                "\t\t\t}\n"
                "\t\t}\n"
                "\t}\n"
                "\tsort.Slice(tags, func(i, j int) bool {\n"
                "\t\tfi, fj := foldTag(tags[i]), foldTag(tags[j])\n"
                "\t\tif fi != fj {\n"
                "\t\t\treturn fi < fj\n"
                "\t\t}\n"
                "\t\treturn tags[i] < tags[j]\n"
                "\t})\n"
                "\treturn tags\n"
                "}\n"
            )
        },
        "gen": _detail_store_tag_dedup,
    },
    "tag_ordering": {
        "requires": ("NewStore", "Store.Put", "Store.Tags"),
        "description": "the tag listing order is the ascending tag hash (ties lexical), not insertion or plain lexical order",
        "prose": "the tag listing is ordered by an ascending seeded tag hash",
        "smoke_name": "TestSmoke_tag_ordering",
        "hidden_name": "TestDetail_tag_ordering",
        "perturbed_fns": ("Store.Tags",),
        "trap_bodies": {
            "Store.Tags": (
                "func (s *Store) Tags(prefix string) []string {\n"
                "\tseen := make(map[string]bool)\n"
                "\tvar tags []string\n"
                "\tfor b := 0; b < nbuckets; b++ {\n"
                "\t\tfor _, e := range s.buckets[b] {\n"
                "\t\t\tt := canon(e.tag)\n"
                "\t\t\tif !seen[t] && strings.HasPrefix(t, prefix) {\n"
                "\t\t\t\tseen[t] = true\n"
                "\t\t\t\ttags = append(tags, t)\n"
                "\t\t\t}\n"
                "\t\t}\n"
                "\t\t}\n"
                "\t\tsort.Strings(tags)\n"
                "\treturn tags\n"
                "}\n"
            )
        },
        "gen": _detail_store_tag_ordering,
    },
    "overwrite_semantics": {
        "requires": ("NewStore", "Store.Put", "Store.Get", "Store.Count"),
        "description": "storing an existing key replaces the record (last write wins) and never grows the count",
        "prose": "storing a key that is already present replaces the previous record rather than adding a second one",
        "smoke_name": "TestSmoke_overwrite",
        "hidden_name": "TestDetail_overwrite",
        "perturbed_fns": ("Store.Put",),
        "trap_bodies": {
            "Store.Put": (
                "func (s *Store) Put(k uint64, tag string, size int) bool {\n"
                "\tif !validateKey(k) {\n\t\treturn false\n\t}\n"
                "\tt := canon(tag)\n"
                '\tif t == "" {\n\t\treturn false\n\t}\n'
                "\tif overLimit(size, s.limit) {\n\t\treturn false\n\t}\n"
                "\tb := s.BucketOf(k)\n"
                "\ts.buckets[b] = append(s.buckets[b], entry{key: k, size: size, tag: t})\n"
                "\treturn true\n"
                "}\n"
            )
        },
        "gen": _detail_store_overwrite,
    },
    "prune_boundary": {
        "requires": ("NewStore", "Store.Put", "Store.Count", "Store.Prune"),
        "description": "prune removes exactly the records with size > max, keeps size == max, returns the count",
        "prose": "pruning removes exactly the records whose size is larger than the given maximum and reports how many were removed; records exactly at the maximum are kept",
        "smoke_name": "TestSmoke_prune",
        "hidden_name": "TestDetail_prune",
        "perturbed_fns": ("Store.Prune",),
        "trap_bodies": {
            "Store.Prune": (
                "func (s *Store) Prune(maxSize int) int {\n"
                "\tremoved := 0\n"
                "\tfor b := 0; b < nbuckets; b++ {\n"
                "\t\tkept := s.buckets[b][:0]\n"
                "\t\tfor _, e := range s.buckets[b] {\n"
                "\t\t\tif e.size >= maxSize {\n"
                "\t\t\t\tremoved++\n"
                "\t\t\t\tcontinue\n"
                "\t\t\t}\n"
                "\t\t\tkept = append(kept, e)\n"
                "\t\t}\n"
                "\t\ts.buckets[b] = kept\n"
                "\t}\n"
                "\treturn removed\n"
                "}\n"
            )
        },
        "gen": _detail_store_prune,
    },
}

SCHED_DETAILS: dict[str, dict[str, object]] = {
    "band_mapping": {
        "requires": ("NewScheduler", "Sched.Slot"),
        "description": "band index is the seeded hash of the point reduced modulo 8, with out-of-range points clamped into [0, capacity) first (not a power-of-two mask)",
        "prose": "a point is assigned to one of 8 bands by a seeded hash: points outside the capacity are clamped into range first — negative points become 0 and points at or beyond the capacity become the last in-range point — then the clamped point is hashed and reduced to a band",
        "smoke_name": "TestSmoke_band_mapping",
        "hidden_name": "TestDetail_band_mapping",
        "perturbed_fns": ("band",),
        "trap_bodies": {
            "band": (
                "func band(x, cap int) int {\n"
                "\tif cap <= 0 {\n\t\treturn 0\n\t}\n"
                "\tif x < 0 {\n\t\tx = 0\n\t}\n"
                "\tif x >= cap {\n\t\tx = cap - 1\n\t}\n"
                "\treturn int((mixPipe(uint64(x)) >> 61) & (nbuckets - 1))\n"
                "}\n"
            )
        },
        "gen": _detail_sched_band_mapping,
    },
    "endpoint_swap": {
        "requires": ("NewScheduler", "Sched.Add", "Sched.At", "Sched.Len"),
        "description": "reversed interval endpoints are swapped before use",
        "prose": "a range given with its endpoints reversed is normalized so the smaller endpoint becomes the start",
        "smoke_name": "TestSmoke_endpoint_swap",
        "hidden_name": "TestDetail_endpoint_swap",
        "perturbed_fns": ("swapOrder",),
        "trap_bodies": {
            "swapOrder": "func swapOrder(iv Interval) Interval {\n\treturn iv\n}\n"
        },
        "gen": _detail_sched_endpoint_swap,
    },
    "clamp_capacity": {
        "requires": ("NewScheduler", "Sched.Add", "Sched.At", "Sched.Len"),
        "description": "interval endpoints clamp into [0, capacity); partially out-of-range intervals are accepted",
        "prose": "ranges are clamped into the capacity: values below 0 become 0 and values above the capacity become the capacity",
        "smoke_name": "TestSmoke_clamp_capacity",
        "hidden_name": "TestDetail_clamp_capacity",
        "perturbed_fns": ("clampBoth",),
        "trap_bodies": {
            "clampBoth": (
                "func clampBoth(iv Interval, cap int) Interval {\n"
                "\tif iv.Start < 0 || iv.End > cap {\n"
                "\t\tiv.Start, iv.End = 0, 0\n"
                "\t}\n"
                "\treturn iv\n"
                "}\n"
            )
        },
        "gen": _detail_sched_clamp_capacity,
    },
    "empty_interval_reject": {
        "requires": ("NewScheduler", "Sched.Add"),
        "description": "an interval that is empty after normalization (start == end) is rejected",
        "prose": "an interval that is empty after normalization — start equals end — is rejected",
        "smoke_name": "TestSmoke_empty_interval",
        "hidden_name": "TestDetail_empty_interval",
        "perturbed_fns": ("Sched.Add",),
        "trap_bodies": {
            "Sched.Add": (
                "func (sc *Scheduler) Add(iv Interval) bool {\n"
                "\tn := norm(iv, sc.cap)\n"
                "\tif !canPlace(sc, n) {\n\t\treturn false\n\t}\n"
                "\tinsertAt(sc, n)\n"
                "\treturn true\n"
                "}\n"
            )
        },
        "gen": _detail_sched_empty_interval,
    },
    "overlap_semantics": {
        "requires": ("NewScheduler", "Sched.Add", "Sched.Merge", "Sched.Len"),
        "description": "interval overlap is half-open: touching intervals do not overlap, so Add accepts touching intervals and Merge coalesces touching or overlapping runs",
        "prose": "schedules never contain overlapping intervals; an interval that only touches a scheduled interval is allowed, and merging coalesces intervals that touch or overlap into one",
        "smoke_name": "TestSmoke_overlap_semantics",
        "hidden_name": "TestDetail_overlap_semantics",
        "perturbed_fns": ("overlaps",),
        "trap_bodies": {
            "overlaps": "func overlaps(a, b Interval) bool {\n\treturn a.Start <= b.End && b.Start <= a.End\n}\n"
        },
        "gen": _detail_sched_overlap_semantics,
    },
    "insertion_order": {
        "requires": ("NewScheduler", "Sched.Add", "Sched.At", "Sched.Len"),
        "description": "scheduled intervals are kept sorted by start time, not insertion order",
        "prose": "scheduled intervals are kept sorted by their start time",
        "smoke_name": "TestSmoke_insertion_order",
        "hidden_name": "TestDetail_insertion_order",
        "perturbed_fns": ("insertAt",),
        "trap_bodies": {
            "insertAt": (
                "func insertAt(sc *Scheduler, iv Interval) {\n"
                "\tsc.slots = append(sc.slots, iv)\n"
                "}\n"
            )
        },
        "gen": _detail_sched_insertion_order,
    },
    "gap_definition": {
        "requires": ("NewScheduler", "Sched.Add", "Sched.Gap"),
        "description": "the gap is the largest uncovered span inside the capacity, counting the leading and trailing spans",
        "prose": "the gap is the largest uncovered span inside the capacity, counting the span before the first interval and after the last",
        "smoke_name": "TestSmoke_gap",
        "hidden_name": "TestDetail_gap",
        "perturbed_fns": ("Sched.Gap",),
        "trap_bodies": {
            "Sched.Gap": (
                "func (sc *Scheduler) Gap() int {\n"
                "\tbest := 0\n"
                "\tprev := 0\n"
                "\tfor _, s := range sc.slots {\n"
                "\t\tif s.Start > prev && length(Interval{prev, s.Start}) > best {\n"
                "\t\t\tbest = length(Interval{prev, s.Start})\n"
                "\t\t}\n"
                "\t\tif s.End > prev {\n"
                "\t\t\tprev = s.End\n"
                "\t\t}\n"
                "\t}\n"
                "\treturn best\n"
                "}\n"
            )
        },
        "gen": _detail_sched_gap,
    },
    "at_coverage": {
        "requires": ("NewScheduler", "Sched.Add", "Sched.At"),
        "description": "point coverage is half-open [start, end): the start is covered, the end is not",
        "prose": "a point is covered when it lies inside a scheduled interval: the start is covered, the end is not",
        "smoke_name": "TestSmoke_at_coverage",
        "hidden_name": "TestDetail_at_coverage",
        "perturbed_fns": ("findSlot",),
        "trap_bodies": {
            "findSlot": (
                "func findSlot(sc *Scheduler, x int) int {\n"
                "\tlo, hi := 0, len(sc.slots)\n"
                "\tfor lo < hi {\n"
                "\t\tmid := (lo + hi) / 2\n"
                "\t\tif sc.slots[mid].End < x {\n"
                "\t\t\tlo = mid + 1\n"
                "\t\t} else {\n"
                "\t\t\thi = mid\n"
                "\t\t}\n"
                "\t}\n"
                "\tif lo < len(sc.slots) && sc.slots[lo].Start <= x && x <= sc.slots[lo].End {\n"
                "\t\treturn lo\n"
                "\t}\n"
                "\treturn -1\n"
                "}\n"
            )
        },
        "gen": _detail_sched_at_coverage,
    },
}

DETAILS: dict[str, dict[str, dict[str, object]]] = {"store": STORE_DETAILS, "sched": SCHED_DETAILS}

# The constant hidden guard: coarse put/get behaviour shared by every unit.
# It is not a detail: it never moves with d or v, so it is a fixed baseline in
# the reward (unit pass = guard AND all drawn details).
STORE_GUARD_TEST = (
    "func TestDetail_guard(t *testing.T) {\n"
    "\tpass := true\n"
    '\tdefer func() { recordDetail("guard", pass) }()\n'
    "\ts := NewStore()\n"
    '\tif !s.Put(1, "x", 10) {\n\t\tpass = false\n\t\tt.Fatal("put rejected")\n\t}\n'
    "\trec, ok := s.Get(1)\n"
    '\tif !ok {\n\t\tpass = false\n\t\tt.Fatal("record missing")\n\t}\n'
    '\tif rec.Key != 1 || rec.Size != 10 || rec.Tag != "x" {\n'
    "\t\tpass = false\n"
    '\t\tt.Fatalf("Get(1) = %+v, want key 1 size 10 tag x", rec)\n\t}\n'
    '\tif _, ok := s.Get(999); ok {\n\t\tpass = false\n\t\tt.Fatal("phantom record")\n\t}\n'
    '\tif n := s.Count(); n != 1 {\n\t\tpass = false\n\t\tt.Fatalf("Count = %d, want 1", n)\n\t}\n'
    "}\n"
)

SCHED_GUARD_TEST = (
    "func TestDetail_guard(t *testing.T) {\n"
    "\tpass := true\n"
    '\tdefer func() { recordDetail("guard", pass) }()\n'
    "\tsc := NewScheduler()\n"
    "\tif !sc.Add(Interval{Start: 10, End: 20}) {\n\t\tpass = false\n\t\tt.Fatal(\"add rejected\")\n\t}\n"
    '\tif n := sc.Len(); n != 1 {\n\t\tpass = false\n\t\tt.Fatalf("Len = %d, want 1", n)\n\t}\n'
    "\tif !sc.At(15) {\n\t\tpass = false\n\t\tt.Fatal(\"covered point reported uncovered\")\n\t}\n"
    "}\n"
)

RECORD_DETAIL_HELPER = (
    "func recordDetail(id string, pass bool) {\n"
    "\tpath := os.Getenv(\"FAB_DETAILS_LOG\")\n"
    '\tif path == "" {\n'
    '\t\tpath = "/logs/verifier/details.json"\n'
    "\t}\n"
    '\tif _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {\n'
    "\t\treturn\n"
    "\t}\n"
    '\tf, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)\n'
    "\tif err != nil {\n"
    "\t\treturn\n"
    "\t}\n"
    '\tfmt.Fprintf(f, "{\\"id\\": %q, \\"pass\\": %v}\\n", id, pass)\n'
    "\tf.Close()\n"
    "}\n"
)


def draw_details(domain: str, seed: int, d: int, present: set[str] | None = None) -> list[str]:
    """Deterministically draw exactly ``d`` detail ids for a unit.

    ``d == 0`` means every detail the config can host (the closure-I default
    behaviour).  ``present`` restricts the draw to details whose required
    entry points exist in the config.  The draw is a fresh splitmix shuffle
    per (domain, seed, d), so each cell of the grid is an independent sample
    of d details — no ordering or nesting artefacts, and the expected
    per-detail pass rate is the same at every d.
    """
    ids = [did for did in DETAILS[domain] if present is None or set(DETAILS[domain][did]["requires"]).issubset(present)]  # type: ignore[arg-type]
    if d == 0 or d >= len(ids):
        return ids
    if d < 0:
        raise ValueError(f"d must be >= 0, got {d}")
    rng = SplitMix(seed * 1000003 + d * 1009 + sum(ord(ch) for ch in domain))
    return sorted(ids, key=lambda _i: rng.next())[:d]


def detail_smoke_source(domain: str, cfg: Config, detail_id: str) -> tuple[str, set[str]]:
    entry = DETAILS[domain][detail_id]
    gen = entry["gen"]
    out = gen(cfg)
    src = str(out["smoke"])
    imports: set[str] = set()
    if "slices.Contains" in src:
        imports.add("slices")
    if "reflect.DeepEqual" in src:
        imports.add("reflect")
    if "strconv.Itoa" in src:
        imports.add("strconv")
    return src, imports


def detail_hidden_source(domain: str, cfg: Config, detail_id: str) -> tuple[str, set[str]]:
    entry = DETAILS[domain][detail_id]
    out = entry["gen"](cfg)
    src, imports = out["hidden"]  # type: ignore[misc]
    return str(src), set(imports)


def detail_shown(domain: str, cfg: Config, detail_id: str) -> list[str]:
    out = DETAILS[domain][detail_id]["gen"](cfg)
    return list(out["shown"])  # type: ignore[arg-type]


def domain_canon_size(domain: str, cfg: Config) -> int:
    if domain == "store":
        have_tags = "Store.Tags" in STORE_ENTRIES[: cfg.E]
        return 1 + (1 if have_tags else 0) + (3 if cfg.canon_mode == "pipe" else 0)
    return 1 + (2 if cfg.canon_mode == "pipe" else 0)


def domain_misc_size(domain: str, cfg: Config) -> int:
    if domain == "store":
        have_bucket_agg = any(k in STORE_ENTRIES[: cfg.E] for k in ("Store.Total", "Store.Count", "Store.Balance"))
        return 2 + (2 if have_bucket_agg else 0)
    present = set(SCHED_ENTRIES[: cfg.E])
    n = 1  # overlaps
    if "Sched.Total" in present or "Sched.Gap" in present:
        n += 1  # length
    if "Sched.Slot" in present:
        n += 1  # band
    if "Sched.Add" in present:
        n += 1  # insertAt
    if "Sched.At" in present:
        n += 1  # findSlot
    return n


def domain_place_size(domain: str, cfg: Config) -> int:
    return 1 if domain == "sched" else 0


# --------------------------------------------------------------------------- #
# graph stats
# --------------------------------------------------------------------------- #
def compute_stats(domain: str, cfg: Config) -> UnitStats:
    fns = DOMAINS[domain]["functions"](cfg)
    edges = DOMAINS[domain]["edges"](cfg)
    groups_s = DOMAINS[domain]["groups_in_S"](cfg)
    present_keys = set(fns)
    edges = [(a, b) for a, b in edges if a in present_keys and b in present_keys]
    S = {k for k, f in fns.items() if f.group in groups_s}
    internal = sum(1 for a, b in edges if a in S and b in S)
    in_edges = sum(1 for a, b in edges if a not in S and b in S)
    out_edges = sum(1 for a, b in edges if a in S and b not in S)
    ratio = internal / max(1, in_edges + out_edges)

    # depth = longest directed path (#edges); fan-out over the graph
    adj: dict[str, list[str]] = {k: [] for k in fns}
    for a, b in edges:
        adj[a].append(b)
    memo: dict[str, int] = {}

    def longest(key: str) -> int:
        if key in memo:
            return memo[key]
        best = 0
        for nxt in adj[key]:
            best = max(best, 1 + longest(nxt))
        memo[key] = best
        return best

    depth = max((longest(k) for k in fns), default=0)
    out_deg = [len(adj[k]) for k in fns]
    fanout_max = max(out_deg, default=0)
    fanout_mean = sum(out_deg) / max(1, len(out_deg))

    return UnitStats(
        internal=internal,
        in_edges=in_edges,
        out_edges=out_edges,
        ratio=ratio,
        depth=depth,
        fanout_max=fanout_max,
        fanout_mean=fanout_mean,
        S=tuple(sorted(S)),
        n_funcs=len(fns),
        loc=0,
        sizes={},
    )


# --------------------------------------------------------------------------- #
# config search
# --------------------------------------------------------------------------- #
def _candidate_configs(domain: str, n_funcs: int, seed: int) -> list[Config]:
    consts = DOMAINS[domain]["consts"](seed)
    entries = DOMAINS[domain]["entries"]
    consumers = DOMAINS[domain]["consumers"]
    out: list[Config] = []
    for E in range(4, len(entries) + 1):
        present = set(entries[:E])
        available = [c for c in consumers if set(consumers[c]).issubset(present)]
        for K in range(1, 6):
            for canon_mode in ("simple", "pipe"):
                canon_n = domain_canon_size(domain, Config(domain, seed, n_funcs, 0.0, E=E, K=K, canon_mode=canon_mode))
                misc_n = domain_misc_size(domain, Config(domain, seed, n_funcs, 0.0, E=E, K=K, canon_mode=canon_mode))
                place_n = domain_place_size(domain, Config(domain, seed, n_funcs, 0.0, E=E, K=K, canon_mode=canon_mode))
                fixed = E + (K + 1) + canon_n + misc_n + place_n + 1
                for canon_in_S in (False, True):
                    for misc_in_S in (False, True):
                        for place_in_S in (False, True):
                            for mask in range(1 << len(available)):
                                chosen = tuple(c for i, c in enumerate(available) if mask >> i & 1)
                                fill = n_funcs - fixed - len(chosen)
                                if fill < 0 or fill > 8:
                                    continue
                                out.append(
                                    Config(
                                        domain=domain,
                                        seed=seed,
                                        n_funcs=n_funcs,
                                        target_ratio=0.0,
                                        E=E,
                                        K=K,
                                        canon_mode=canon_mode,
                                        canon_in_S=canon_in_S,
                                        misc_in_S=misc_in_S,
                                        place_in_S=place_in_S,
                                        consumers=chosen,
                                        fill=fill,
                                        consts=consts,
                                    )
                                )
    return out


def search_config(
    domain: str,
    n_funcs: int,
    target_ratio: float,
    seed: int,
    *,
    target_depth: int = 0,
    target_fanout: int = 0,
    s_band: tuple[int, int] | None = None,
) -> Config:
    best: Config | None = None
    best_stats: UnitStats | None = None
    best_score = float("inf")
    for cfg in _candidate_configs(domain, n_funcs, seed):
        cfg.target_ratio = target_ratio
        cfg.target_depth = target_depth
        cfg.target_fanout = target_fanout
        stats = compute_stats(domain, cfg)
        if s_band is not None and not (s_band[0] <= len(stats.S) <= s_band[1]):
            continue
        score = abs(stats.ratio - target_ratio)
        if target_depth:
            score += 0.02 * abs(stats.depth - target_depth)
        if target_fanout:
            score += 0.01 * abs(stats.fanout_max - target_fanout)
        score += 0.0001 * (9 - cfg.E)  # prefer the fuller API on ties
        if score < best_score:
            best_score = score
            best = cfg
            best_stats = stats
    if best is None or best_stats is None:
        raise ValueError(
            f"no config fits {domain} with n_funcs={n_funcs} s_band={s_band}; "
            "raise n_funcs or loosen the target"
        )
    return best


# --------------------------------------------------------------------------- #
# rendering: module / excised / cheat trees
# --------------------------------------------------------------------------- #
def _render_store_go(
    domain: str,
    cfg: Config,
    fns: dict[str, FnSpec],
    mode: str,
    trap_bodies: dict[str, str] | None = None,
) -> str:
    spec = DOMAINS[domain]
    pkg = spec["pkg"]
    S = set(compute_S(domain, cfg))
    parts: list[str] = []
    imports: set[str] = set()
    order: list[str] = []
    group_order = ("entry", "mixer", "canon", "place", "misc", "leaf", "consumer", "filler")
    for group in group_order:
        order.extend(sorted((k for k, f in fns.items() if f.group == group), key=lambda k: (k not in spec["entries"], k)))
    for key in order:
        f = fns[key]
        if mode == "trap" and key in (trap_bodies or {}):
            trap_body = trap_bodies[key]  # type: ignore[index]
            parts.append(trap_body)
            imports |= f.gold_imports
            if "strings." in trap_body:
                imports.add("strings")
            if "sort." in trap_body:
                imports.add("sort")
            if "math/bits" in trap_body:
                imports.add("math/bits")
        elif mode == "excised" and key in S:
            parts.append(f"{f.sig} {{\n\t{_stub_body(key)}\n}}\n")
        elif mode == "cheat" and key in S:
            if f.cheat:
                parts.append(f.cheat)
                imports |= f.cheat_imports
            else:
                parts.append(f.gold)
                imports |= f.gold_imports
        else:
            parts.append(f.gold)
            imports |= f.gold_imports
    body = spec["types"] + "\n" + "\n".join(parts)
    return render_go_file(pkg, imports, body, spec["pkg_doc"])


def compute_S(domain: str, cfg: Config) -> list[str]:
    fns = DOMAINS[domain]["functions"](cfg)
    groups_s = DOMAINS[domain]["groups_in_S"](cfg)
    return sorted(k for k, f in fns.items() if f.group in groups_s)


def render_params_file(domain: str, cfg: Config) -> str:
    """``params.go`` — every seeded constant, never excised, never patched.

    Identical in module/, excised_tree/ and the cheat tree, so it carries no
    patch hunks; the L0 solver reads all the numbers here.
    """
    spec = DOMAINS[domain]
    c = cfg.consts
    mul: list[int] = c["mixMul"][: cfg.K]  # type: ignore[index]
    adds: list[int] = c["mixAdd"][: cfg.K]  # type: ignore[index]
    lines = [f"const nbuckets = {int(c['nbuckets'])}", ""]
    for j in range(cfg.K):
        lines.append(f"const paramStageMul{j + 1} uint64 = {_hex(mul[j])}")
        lines.append(f"const paramStageAdd{j + 1} uint64 = {_hex(adds[j])}")
    lines.append(f"const paramMixRot = {int(c['mixRot'])}")
    imports: set[str] = set()
    if domain == "store":
        lines.append(f"const paramFoldBase uint64 = {_hex(int(c['foldBase']))}")
        lines.append(f"const paramStoreLimit = {int(c['limit'])}")
        lines.append("const paramReservedKey = math.MaxUint64")
        imports.add("math")
    else:
        lines.append(f"const paramCap = {int(c['cap'])}")
    return render_go_file(
        spec["pkg"],
        imports,
        "\n".join(lines),
        "Seeded configuration constants — part of the specification, never excised.",
    )


def ref_slot(x: int, cap: int, mul: list[int], add: list[int], rot: int, nbuckets: int = 8) -> int:
    """Python mirror of the sched ``band`` helper (clamp then mix mod nbuckets)."""
    if cap <= 0:
        return 0
    x = min(max(x, 0), cap - 1)
    return ref_mix(x, mul, add, rot) % nbuckets


def _fold_tag(tag: str, base: int) -> int:
    h = 0
    for ch in tag.encode():
        h = (h * base + ch) & M64
    return h


def _smoke_agg_keys(mul: list[int], adds: list[int], rot: int, nb: int) -> list[int]:
    """Four keys 0..63 with bucket occupancy {2,1,1} — the balance worked
    example demonstrates max-occupancy, not just 'at least one'."""
    by_bucket: dict[int, list[int]] = {}
    for k in range(64):
        by_bucket.setdefault(ref_bucket(k, mul, adds, rot, nb), []).append(k)
    pair = next(ks[:2] for ks in by_bucket.values() if len(ks) >= 2)
    rest = [ks[0] for ks in by_bucket.values() if ks[0] not in pair]
    return sorted(pair + rest[:2])


def _smoke_baseline(domain: str, cfg: Config) -> tuple[list[str], set[str]]:
    """Coarse in-tree tests that never depend on any detail's value.

    They are reduction- and policy-agnostic on purpose: e.g. the aggregates
    example checks Count and Total (which any correct store satisfies) and not
    Balance (whose example value would pin the bucket reduction)."""
    body: list[str] = []
    imports = {"testing"}
    present = set(DOMAINS[domain]["entries"][: cfg.E])
    if domain == "store":
        if {"NewStore", "Store.Put", "Store.Get"}.issubset(present):
            body.append(
                "func TestSmokeRoundtrip(t *testing.T) {\n"
                "\ts := NewStore()\n"
                '\tif !s.Put(1, "x", 10) {\n\t\tt.Fatal("put rejected")\n\t}\n'
                "\trec, ok := s.Get(1)\n"
                '\tif !ok {\n\t\tt.Fatal("stored record missing")\n\t}\n'
                '\tif rec.Key != 1 || rec.Size != 10 || rec.Tag != "x" {\n'
                '\t\tt.Fatalf("Get(1) = %+v, want key 1 size 10 tag x", rec)\n\t}\n'
                '\tif _, ok := s.Get(999); ok {\n\t\tt.Fatal("phantom record")\n\t}\n'
                "}\n"
            )
        if {"NewStore", "Store.Put", "Store.Count", "Store.Total"}.issubset(present):
            mul: list[int] = [int(x) for x in cfg.consts["mixMul"][: cfg.K]]  # type: ignore[index]
            adds: list[int] = [int(x) for x in cfg.consts["mixAdd"][: cfg.K]]  # type: ignore[index]
            rot: int = int(cfg.consts["mixRot"])
            nb: int = int(cfg.consts["nbuckets"])
            keys = _smoke_agg_keys(mul, adds, rot, nb)
            puts = "".join(f'\ts.Put({k}, "t{k}", {s})\n' for k, s in zip(keys, (10, 20, 30, 40), strict=True))
            body.append(
                "func TestSmokeAggregates(t *testing.T) {\n"
                "\ts := NewStore()\n"
                f"{puts}"
                '\tif got := s.Count(); got != 4 {\n\t\tt.Fatalf("Count = %d, want 4", got)\n\t}\n'
                '\tif got := s.Total(); got != 100 {\n\t\tt.Fatalf("Total = %d, want 100", got)\n\t}\n'
                "}\n"
            )
    else:
        if {"NewScheduler", "Sched.Add", "Sched.At", "Sched.Len"}.issubset(present):
            body.append(
                "func TestSmokeSchedule(t *testing.T) {\n"
                "\tsc := NewScheduler()\n"
                "\tif !sc.Add(Interval{Start: 10, End: 20}) {\n\t\tt.Fatal(\"add rejected\")\n\t}\n"
                "\tif !sc.At(15) || sc.At(5) {\n\t\tt.Fatal(\"coverage wrong\")\n\t}\n"
                '\tif n := sc.Len(); n != 1 {\n\t\tt.Fatalf("Len = %d, want 1", n)\n\t}\n'
                "}\n"
            )
        if {"NewScheduler", "Sched.Add", "Sched.Total"}.issubset(present):
            body.append(
                "func TestSmokeTotals(t *testing.T) {\n"
                "\tsc := NewScheduler()\n"
                "\tsc.Add(Interval{Start: 0, End: 10})\n"
                "\tsc.Add(Interval{Start: 20, End: 30})\n"
                '\tif got := sc.Total(); got != 20 {\n\t\tt.Fatalf("Total = %d, want 20", got)\n\t}\n'
                "}\n"
            )
    return body, imports


def _in_tree_tests(domain: str, cfg: Config, drawn: set[str], v: str) -> str:
    """In-tree smoke suite — part of the L0 information set.

    Baseline tests (coarse round-trip / aggregates) always run.  Each detail
    has a worked-example test (``TestSmoke_<id>``) asserting the gold value on
    a concrete, discriminating input.  A detail's worked example is included
    iff the detail is NOT drawn (its value is forced — visible in repro
    output) or v == "high" (the affordance: a wrong choice shows up as
    ``got X, want Y`` in repro output).  When a detail is drawn at v=low its
    example is dropped, so the L0 set never reveals its value and
    ``./repro.sh`` shows the failing suite only.

    It is not the verifier (B3/B5): the hidden suite is property-based over
    seeded random inputs, so the examples here never pin the property."""
    pkg = DOMAINS[domain]["pkg"]
    body, imports = _smoke_baseline(domain, cfg)
    present = set(DOMAINS[domain]["entries"][: cfg.E])
    for detail_id in DETAILS[domain]:
        if not set(DETAILS[domain][detail_id]["requires"]).issubset(present):  # type: ignore[arg-type]
            continue
        include = detail_id not in drawn or v == "high"
        if not include:
            continue
        src, extra = detail_smoke_source(domain, cfg, detail_id)
        body.append(src)
        imports |= extra
    fns = DOMAINS[domain]["functions"](cfg)
    filler_names = [f.name for f in fns.values() if f.group == "filler"]
    if filler_names:
        filler_checks = {
            "clampInt": "if clampInt(5, 0, 3) != 3 {\n\t\tt.Fatal(\"clampInt\")\n\t}",
            "absDiff": "if absDiff(3, 8) != 5 {\n\t\tt.Fatal(\"absDiff\")\n\t}",
            "asciiSum": "if asciiSum(\"abc\") != 294 {\n\t\tt.Fatal(\"asciiSum\")\n\t}",
            "minU64": "if minU64(3, 9) != 3 {\n\t\tt.Fatal(\"minU64\")\n\t}",
            "parity": "if parity(6) != 0 {\n\t\tt.Fatal(\"parity\")\n\t}",
            "digitSum": "if digitSum(1234) != 10 {\n\t\tt.Fatal(\"digitSum\")\n\t}",
            "revStr": "if revStr(\"abc\") != \"cba\" {\n\t\tt.Fatal(\"revStr\")\n\t}",
            "pow2ceil": "if pow2ceil(9) != 16 {\n\t\tt.Fatal(\"pow2ceil\")\n\t}",
        }
        checks = "\n\t".join(filler_checks[n] for n in filler_names)
        body.append(f"func TestSmokeFillers(t *testing.T) {{\n\t{checks}\n}}\n")
    if not body:
        body.append("func TestSmokeNoop(t *testing.T) {}\n")
    return render_go_file(pkg, imports, "\n".join(body), "In-tree smoke tests (worked examples; not the verifier).")


def render_module_files(domain: str, cfg: Config, drawn: set[str], v: str) -> dict[str, str]:
    fns = DOMAINS[domain]["functions"](cfg)
    return {
        "go.mod": f"module {cfg.module_name()}\n\ngo 1.23\n",
        "params.go": render_params_file(domain, cfg),
        f"{DOMAINS[domain]['pkg']}.go": _render_store_go(domain, cfg, fns, "gold"),
        f"{DOMAINS[domain]['pkg']}_test.go": _in_tree_tests(domain, cfg, drawn, v),
    }


def render_excised_files(
    domain: str,
    cfg: Config,
    drawn: set[str],
    v: str,
    trap: str | None = None,
) -> dict[str, str]:
    fns = DOMAINS[domain]["functions"](cfg)
    mode = "trap" if trap is not None else "excised"
    trap_bodies = DETAILS[domain][trap]["trap_bodies"] if trap is not None else None
    return {
        "go.mod": f"module {cfg.module_name()}\n\ngo 1.23\n",
        "params.go": render_params_file(domain, cfg),
        f"{DOMAINS[domain]['pkg']}.go": _render_store_go(domain, cfg, fns, mode, trap_bodies),
        f"{DOMAINS[domain]['pkg']}_test.go": _in_tree_tests(domain, cfg, drawn, v),
    }


def render_cheat_files(domain: str, cfg: Config, drawn: set[str], v: str) -> dict[str, str]:
    fns = DOMAINS[domain]["functions"](cfg)
    return {
        "go.mod": f"module {cfg.module_name()}\n\ngo 1.23\n",
        "params.go": render_params_file(domain, cfg),
        f"{DOMAINS[domain]['pkg']}.go": _render_store_go(domain, cfg, fns, "cheat"),
        f"{DOMAINS[domain]['pkg']}_test.go": _in_tree_tests(domain, cfg, drawn, v),
    }


# --------------------------------------------------------------------------- #
# hidden suites
# --------------------------------------------------------------------------- #
def _hidden_seed_helper() -> str:
    return (
        "func hiddenSeed() int64 {\n"
        '\tif s := os.Getenv("HIDDEN_SEED"); s != "" {\n'
        "\t\tif n, err := strconv.ParseInt(s, 10, 64); err == nil {\n"
        "\t\t\treturn n\n"
        "\t\t}\n"
        "\t}\n"
        "\treturn 20260919\n"
        "}\n"
    )


def store_hidden_tests(cfg: Config, drawn: set[str]) -> list[tuple[str, str]]:
    """Hidden verifier for the fabricated store unit: the constant guard plus
    one property test per drawn detail (B3: seeded-random inputs beyond the L0
    examples; B4: only the exported API).  Each test records its own pass/fail
    line to ``/logs/verifier/details.json`` so a trial yields a per-detail
    vector, not just an aggregate reward."""
    c = cfg.consts
    mul: list[int] = [int(x) for x in c["mixMul"][: cfg.K]]  # type: ignore[index]
    adds: list[int] = [int(x) for x in c["mixAdd"][: cfg.K]]  # type: ignore[index]
    rot: int = int(c["mixRot"])
    limit: int = int(c["limit"])
    fold_base: int = int(c["foldBase"])
    refs = [
        _hidden_seed_helper(),
        RECORD_DETAIL_HELPER,
        "const refBuckets = 8",
        f"var refLimit = {limit}",
        f"var refFoldBase uint64 = 0x{fold_base:X}",
        f"var refMul = []uint64{{{', '.join(f'0x{m:X}' for m in mul)}}}",
        f"var refAdd = []uint64{{{', '.join(f'0x{a:X}' for a in adds)}}}",
        f"var refRot = {rot}",
        (
            "func refMix(x uint64) uint64 {\n"
            "\tv := x\n"
            "\tfor i := range refMul {\n"
            "\t\tv = v*refMul[i] + refAdd[i]\n"
            "\t}\n"
            "\treturn bits.RotateLeft64(v, refRot)\n"
            "}\n"
        ),
        "func refBucket(k uint64) int {\n\treturn int(refMix(k) % refBuckets)\n}\n",
        (
            "func refCanon(s string) string {\n"
            "\ts = strings.TrimSpace(s)\n"
            "\ts = strings.ToLower(s)\n"
            "\tvar b strings.Builder\n"
            "\tdash := false\n"
            "\tfor i := 0; i < len(s); i++ {\n"
            "\t\tch := s[i]\n"
            "\t\tif (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {\n"
            "\t\t\tif dash && b.Len() > 0 {\n"
            "\t\t\t\tb.WriteByte('-')\n"
            "\t\t\t}\n"
            "\t\t\tdash = false\n"
            "\t\t\tb.WriteByte(ch)\n"
            "\t\t} else {\n"
            "\t\t\tdash = true\n"
            "\t\t}\n"
            "\t}\n"
            "\treturn strings.Trim(b.String(), \"-\")\n"
            "}\n"
        ),
        (
            "func refFold(tag string) uint64 {\n"
            "\th := uint64(0)\n"
            "\tfor i := 0; i < len(tag); i++ {\n"
            "\t\th = h*refFoldBase + uint64(tag[i])\n"
            "\t}\n"
            "\treturn h\n"
            "}\n"
        ),
    ]
    present = set(STORE_ENTRIES[: cfg.E])
    tests: list[str] = []
    if {"NewStore", "Store.Put", "Store.Get", "Store.Count"}.issubset(present):
        tests.append(STORE_GUARD_TEST)
    for detail_id in DETAILS["store"]:
        if detail_id not in drawn or not set(DETAILS["store"][detail_id]["requires"]).issubset(present):  # type: ignore[arg-type]
            continue
        src, _extra = detail_hidden_source("store", cfg, detail_id)
        tests.append(src)
    body = "\n".join(refs + tests)
    imports = _hidden_imports(body)
    imports.add(cfg.module_name())
    content = render_go_file("store_test", imports, body, "Hidden black-box suite for the fabricated store unit (generated).")
    content = content.replace("NewStore(", "store.NewStore(")
    return [("store_bb_test.go", gofmt_text(content))]


def _hidden_imports(body: str) -> set[str]:
    """Imports required by the rendered hidden body (refs + tests)."""
    imports = {"testing"}
    if "rand.New" in body:
        imports.add("math/rand")
    if "bits.RotateLeft64" in body:
        imports.add("math/bits")
    if "strings." in body:
        imports.add("strings")
    if "reflect.DeepEqual" in body:
        imports.add("reflect")
    if "slices.Contains" in body:
        imports.add("slices")
    if "strconv." in body:
        imports.add("strconv")
    if "os." in body:
        imports.add("os")
    if "fmt." in body:
        imports.add("fmt")
    if "filepath.Dir" in body:
        imports.add("path/filepath")
    return imports


def cfg_canon(s: str) -> str:
    s = s.strip().lower()
    out: list[str] = []
    dash = False
    for ch in s:
        if ("a" <= ch <= "z") or ("0" <= ch <= "9"):
            if dash and out:
                out.append("-")
            dash = False
            out.append(ch)
        else:
            dash = True
    return "".join(out).strip("-")


def sched_hidden_tests(cfg: Config, drawn: set[str]) -> list[tuple[str, str]]:
    """Hidden verifier for the fabricated sched unit (see store_hidden_tests)."""
    c = cfg.consts
    mul: list[int] = [int(x) for x in c["mixMul"][: cfg.K]]  # type: ignore[index]
    adds: list[int] = [int(x) for x in c["mixAdd"][: cfg.K]]  # type: ignore[index]
    rot: int = int(c["mixRot"])
    cap: int = int(c["cap"])
    refs = [
        _hidden_seed_helper(),
        RECORD_DETAIL_HELPER,
        "const refBuckets = 8",
        f"var refCap = {cap}",
        f"var refMul = []uint64{{{', '.join(f'0x{m:X}' for m in mul)}}}",
        f"var refAdd = []uint64{{{', '.join(f'0x{a:X}' for a in adds)}}}",
        f"var refRot = {rot}",
        (
            "func refMix(x uint64) uint64 {\n"
            "\tv := x\n"
            "\tfor i := range refMul {\n"
            "\t\tv = v*refMul[i] + refAdd[i]\n"
            "\t}\n"
            "\treturn bits.RotateLeft64(v, refRot)\n"
            "}\n"
        ),
        (
            "func refSlot(x, cap int) int {\n"
            "\tif cap <= 0 {\n\t\treturn 0\n\t}\n"
            "\tif x < 0 {\n\t\tx = 0\n\t}\n"
            "\tif x >= cap {\n\t\tx = cap - 1\n\t}\n"
            "\treturn int(refMix(uint64(x)) % refBuckets)\n"
            "}\n"
        ),
    ]
    present = set(SCHED_ENTRIES[: cfg.E])
    tests: list[str] = []
    if {"NewScheduler", "Sched.Add", "Sched.At", "Sched.Len"}.issubset(present):
        tests.append(SCHED_GUARD_TEST)
    for detail_id in DETAILS["sched"]:
        if detail_id not in drawn or not set(DETAILS["sched"][detail_id]["requires"]).issubset(present):  # type: ignore[arg-type]
            continue
        src, _extra = detail_hidden_source("sched", cfg, detail_id)
        tests.append(src)
    body = "\n".join(refs + tests)
    imports = _hidden_imports(body)
    imports.add(cfg.module_name())
    content = render_go_file("sched_test", imports, body, "Hidden black-box suite for the fabricated sched unit (generated).")
    content = content.replace("NewScheduler(", "sched.NewScheduler(").replace("Interval", "sched.Interval")
    return [("sched_bb_test.go", gofmt_text(content))]


def hidden_tests(domain: str, cfg: Config, drawn: set[str]) -> list[tuple[str, str]]:
    if domain == "store":
        return store_hidden_tests(cfg, drawn)
    return sched_hidden_tests(cfg, drawn)


# --------------------------------------------------------------------------- #
# author artifacts (docs + patches)
# --------------------------------------------------------------------------- #
def _fmt_constants(cfg: Config) -> list[str]:
    c = cfg.consts
    rows = [
        ("buckets", str(c["nbuckets"])),
        ("mixer stages", str(cfg.K)),
    ]
    for j in range(cfg.K):
        rows.append((f"stage {j + 1} multiplier", _hex(int(c["mixMul"][j]))))  # type: ignore[index]
        rows.append((f"stage {j + 1} addend", _hex(int(c["mixAdd"][j]))))  # type: ignore[index]
    rows.append(("final rotate", str(c["mixRot"])))
    if cfg.domain == "store":
        rows.append(("store size limit", str(c["limit"])))
        rows.append(("tag hash base", _hex(int(c["foldBase"]))))
        rows.append(("reserved key", "18446744073709551615 (2^64-1)"))
    else:
        rows.append(("scheduler capacity", str(c["cap"])))
    return rows


def render_contract(domain: str, cfg: Config, stats: UnitStats, drawn: set[str]) -> str:
    """L2 spec: full prose + configuration + worked examples + a coverage
    table mapping every hidden test (the guard plus one row per drawn
    detail) to its contract sentence."""
    c = cfg.consts
    rows = _fmt_constants(cfg)
    table = "\n".join(f"| {k} | `{v}` |" for k, v in rows)
    hidden = hidden_tests(domain, cfg, drawn)
    names = [n for _, t in hidden for n in re.findall(r"^func (Test\w+)", t, re.MULTILINE)]

    if domain == "store":
        prose = (
            "The unit is a small in-memory record store. A record is a key, a size and a tag. "
            "Records are distributed across 8 buckets by a seeded mixer: bucket(k) = mix(k) mod 8 "
            "where mix applies the unit's K affine stages (v = v*M_j + A_j for j = 1..K) followed "
            "by a left rotation by R. The reserved key 2^64-1 is never stored. Tags are "
            "canonicalized: trimmed, lowercased, then every run of non-alphanumeric characters is "
            "collapsed to a single dash and leading/trailing dashes are removed; a tag that "
            "canonicalizes to the empty string is rejected. A record whose size exceeds the "
            "store's limit is rejected. Storing a key that is already present replaces the "
            "previous record (last write wins).\n"
            "\n"
            "- Fetching a key returns the stored record (the original key, the stored size, the "
            "canonical tag) or not-found for keys never stored.\n"
            "- The count is the number of stored records; the total is the sum of their sizes; "
            "the balance is the number of records in the most populated bucket.\n"
            "- The tag hash of a canonical tag is computed over its bytes: h starts at 0 and "
            "for each byte c, h = h*B + c (unsigned 64-bit wraparound), with B the tag hash "
            "base below.\n"
            "- Enumerating tags returns the distinct canonical tags of stored records, in "
            "ascending tag-hash order (ties broken lexicographically), optionally filtered "
            "to tags beginning with a prefix.\n"
            "- Pruning removes every record whose size exceeds the given maximum and returns the "
            "number removed; counts, totals and balance update accordingly.\n"
        )
    else:
        prose = (
            "The unit is a small interval scheduler. An interval is a half-open range [Start, End). "
            "Intervals are normalized before use: reversed endpoints are swapped, then both are "
            "clamped into [0, capacity); a normalized interval with Start == End is empty and is "
            "rejected. Schedules never contain overlapping intervals. Points are assigned to one "
            "of 8 bands by the seeded mixer: band(x) = mix(x) mod 8 (mix as in the configuration "
            "below), after clamping x into [0, capacity).\n"
            "\n"
            "- Adding an interval succeeds only when it is non-empty after normalization and does "
            "not overlap any scheduled interval; it is inserted in start order.\n"
            "- A point is covered when it lies inside a scheduled interval.\n"
            "- Overlap checks report whether an interval would conflict with the schedule.\n"
            "- The length is the total covered span; the gap is the largest uncovered span inside "
            "[0, capacity); merging coalesces intervals that touch or overlap and returns the "
            "number of coalescences.\n"
        )
    coverage = [
        ("TestDetail_guard", "The basic flow works: a valid record/interval is accepted, retrievable and counted exactly once; absent keys are not found."),
    ]
    for detail_id in DETAILS[domain]:
        if detail_id not in drawn:
            continue
        coverage.append((DETAILS[domain][detail_id]["hidden_name"], DETAILS[domain][detail_id]["description"]))
    present_names = set(names)
    coverage_rows = "\n".join(f"| {name} | {sentence} |" for name, sentence in coverage if name in present_names)
    mul = [int(x) for x in c["mixMul"][: cfg.K]]  # type: ignore[index]
    adds = [int(x) for x in c["mixAdd"][: cfg.K]]  # type: ignore[index]
    rot = int(c["mixRot"])
    if domain == "store":
        examples = (
            f"- key `7` maps to bucket `{ref_bucket(7, mul, adds, rot)}`.\n"
            f"- key `42` maps to bucket `{ref_bucket(42, mul, adds, rot)}`.\n"
            "- tag `\"  Alpha beta \"` canonicalizes to `\"alpha-beta\"`; tag `\"!!!\"` canonicalizes to the empty string and is rejected.\n"
            f"- a record of size `{int(c['limit'])}` is accepted and one of size `{int(c['limit']) + 1}` is rejected.\n"
        )
    else:
        cap = int(c["cap"])
        examples = (
            f"- point `7` maps to band `{ref_slot(7, cap, mul, adds, rot)}`; point `0` maps to band `{ref_slot(0, cap, mul, adds, rot)}`.\n"
            f"- intervals are clamped into `[0, {cap})`.\n"
            "- intervals `[0,10)` and `[10,20)` are adjacent and merge into one.\n"
        )
    return (
        f"# Contract (L2) — {cfg.unit_name()}\n\n"
        f"{prose}\n"
        "## Configuration (this unit)\n\n"
        "| constant | value |\n"
        "|---|---|\n"
        f"{table}\n\n"
        "## Worked examples\n\n"
        f"{examples}"
        "\n## Coverage of hidden assertions\n\n"
        "| hidden test | contract sentence |\n"
        "|---|---|\n"
        f"{coverage_rows}\n"
    )


def render_bugreport(domain: str, cfg: Config, drawn: set[str], v: str) -> str:
    """L0 instruction (B6/B7): symptom + expected-vs-got + reproduction
    command, plus one prose sentence per drawn detail.

    The prose is deliberately not a spec: it describes the behaviour without
    pinning the exact value (e.g. "spread across all 8 buckets" never says
    modulo vs mask), so a detail can be silently wrong.  The text is identical
    across the v levels — only ``./repro.sh`` differs (at v=high the in-tree
    suite asserts a discriminating worked example per drawn detail, so a wrong
    choice prints ``got X, want Y``; at v=low it does not)."""
    details_block = "\n".join(f"- {DETAILS[domain][did]['prose']}." for did in drawn)
    if domain == "store":
        body = (
            "The store is unusable: the first operation panics, and when the failure is masked "
            "records are not placed correctly and the size aggregates read zero.\n\n"
            "Expected: valid records are accepted and retrieved exactly as stored, and the "
            "behavioural rules below hold.\n"
            "Got: panic on first use; records not placed correctly; zero totals and counts.\n"
        )
    else:
        body = (
            "The scheduler is unusable: the first operation panics, and when the failure is "
            "masked intervals are never scheduled, every point is reported uncovered, and the "
            "coverage totals read zero.\n\n"
            "Expected: valid intervals are accepted and reported covered afterward, and the "
            "behavioural rules below hold.\n"
            "Got: panic on first use; empty schedule; nothing covered; zero totals.\n"
        )
    return (
        f"# Bug report\n\n{body}\n"
        "## Behavioural details\n\n"
        f"{details_block}\n\n"
        "Reproduce with:\n\n```\n./repro.sh\n```\n\n"
        "Run it from the unit root directory (the parent of `_author/`). "
        "The script copies the tree into a scratch dir and runs the in-tree test suite.\n\n"
        "Work in the repository tree. Keep unrelated tests passing. "
        f"{NO_WEB_CLAUSE}\n"
    )


def render_api(domain: str, cfg: Config, stats: UnitStats) -> str:
    spec = DOMAINS[domain]
    present = spec["entries"][: cfg.E]
    pkg = spec["pkg"]
    lines = [f"# Exported API — {cfg.unit_name()}", "", f"Package `{pkg}`."]
    sigs = {
        "store": {
            "NewStore": "`NewStore() *Store`",
            "Store.Put": "`(s *Store) Put(k uint64, tag string, size int) bool`",
            "Store.Get": "`(s *Store) Get(k uint64) (Record, bool)`",
            "Store.BucketOf": "`(s *Store) BucketOf(k uint64) int`",
            "Store.Total": "`(s *Store) Total() int`",
            "Store.Count": "`(s *Store) Count() int`",
            "Store.Balance": "`(s *Store) Balance() int`",
            "Store.Tags": "`(s *Store) Tags(prefix string) []string`",
            "Store.Prune": "`(s *Store) Prune(maxSize int) int`",
        },
        "sched": {
            "NewScheduler": "`NewScheduler() *Scheduler`",
            "Sched.Slot": "`(sc *Scheduler) Slot(x int) int`",
            "Sched.Add": "`(sc *Scheduler) Add(iv Interval) bool`",
            "Sched.Overlaps": "`(sc *Scheduler) Overlaps(iv Interval) bool`",
            "Sched.At": "`(sc *Scheduler) At(x int) bool`",
            "Sched.Len": "`(sc *Scheduler) Len() int`",
            "Sched.Total": "`(sc *Scheduler) Total() int`",
            "Sched.Gap": "`(sc *Scheduler) Gap() int`",
            "Sched.Merge": "`(sc *Scheduler) Merge() int`",
        },
    }[domain]
    for k in present:
        lines.append(f"- {sigs[k]}")
    lines.append("")
    lines.append("## Pre-existing callers")
    lines.append(f"Package-internal reporting helpers drive the exported API: {', '.join(cfg.consumers) if cfg.consumers else 'none'}.")
    lines.append("")
    lines.append(f"Excised set S ({len(stats.S)} functions): `{', '.join(stats.S)}`.")
    return "\n".join(lines) + "\n"


def render_closure(domain: str, cfg: Config, stats: UnitStats) -> str:
    spec = DOMAINS[domain]
    exported = [k for k in stats.S if k.split(".")[0] in ("Store", "Sched") or k in ("NewStore", "NewScheduler")]
    return (
        f"# Closure — {cfg.unit_name()}\n\n"
        f"Package: {spec['pkg']}.\n\n"
        "Removed functions (bodies stubbed): "
        + ", ".join(stats.S)
        + ".\n\n"
        f"Exported entry point(s): {', '.join(exported) or 'none'}.\n"
    )


def render_difficulty(domain: str, cfg: Config) -> str:
    return (
        "# Difficulty\n\n"
        "predicted_flip: L2\n\n"
        "L0 is fair but hard: every seeded constant is given in params.go (never excised) and "
        "the in-tree smoke suite exemplifies each checked behavior, so the excised bodies are "
        "inferable — but the whole excised subsystem (mixer pipeline, canonicalizer, "
        "aggregates/prune) must be reconstructed, and the hidden suite is property-based "
        "(seeded random inputs with a reference implementation), so a special-cased or guessed "
        "body fails. At L2 the full contract pins the behavior down, so the unit is expected "
        "to be solved there.\n"
    )


# --------------------------------------------------------------------------- #
# unit assembly + proofs
# --------------------------------------------------------------------------- #
def _gofmt_tree(files: dict[str, str]) -> dict[str, str]:
    out = dict(files)
    for rel, text in files.items():
        if rel.endswith(".go"):
            out[rel] = gofmt_text(text)
    return out


def _write_tree(root: Path, files: dict[str, str]) -> None:
    for rel, text in files.items():
        dest = root / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(text, encoding="utf-8")


def prove_unit(
    unit_dir: Path, cfg: Config, scratch_root: Path, drawn: set[str], v: str
) -> dict[str, object]:
    domain = cfg.domain
    module = unit_dir / "module"
    hidden_dir = unit_dir / "hidden"
    proof_root = scratch_root / cfg.unit_name()
    if proof_root.exists():
        shutil.rmtree(proof_root)
    proof_root.mkdir(parents=True)

    result: dict[str, object] = {}

    build = run_go(["go", "build", "./..."], module, timeout=240)
    result["module_build"] = build.returncode == 0
    result["module_build_log"] = (build.stdout + build.stderr)[-2000:]

    test_mod = run_go(["go", "test", "-count=1", "-timeout", "3m", "./..."], module, timeout=300)
    result["module_test"] = test_mod.returncode == 0
    result["module_test_log"] = (test_mod.stdout + test_mod.stderr)[-2000:]

    vet = run_go(["go", "vet", "./..."], module, timeout=240)
    result["module_vet"] = vet.returncode == 0
    result["module_vet_log"] = (vet.stdout + vet.stderr)[-2000:]

    gofmt_bad = run_go(["gofmt", "-l", "."], module, timeout=60)
    result["gofmt_clean"] = gofmt_bad.stdout.strip() == ""

    hidden_src = hidden_dir / f"{DOMAINS[domain]['pkg']}_bb_test.go"

    def mount_hidden(tree: Path) -> None:
        shutil.copy2(hidden_src, tree / hidden_src.name)

    def fresh(mode: str) -> Path:
        tree = proof_root / mode
        _write_tree(tree, render_excised_files(domain, cfg, drawn, v))
        if mode == "gold":
            apply_patch_file(tree, unit_dir / "_author" / "gold.patch")
        elif mode == "cheat":
            apply_patch_file(tree, unit_dir / "_author" / "cheat.patch")
        mount_hidden(tree)
        return tree

    buggy = fresh("buggy")
    bug_run = run_go(["go", "test", "-count=1", "-timeout", "3m", "./..."], buggy, timeout=300)
    result["buggy_fails"] = bug_run.returncode != 0
    result["buggy_log"] = (bug_run.stdout + bug_run.stderr)[-2000:]

    gold = fresh("gold")
    gold_run = run_go(["go", "test", "-count=1", "-timeout", "3m", "./..."], gold, timeout=300)
    result["gold_passes"] = gold_run.returncode == 0
    result["gold_log"] = (gold_run.stdout + gold_run.stderr)[-2000:]
    gold2_run = run_go(
        ["go", "test", "-count=1", "-timeout", "3m", "./..."],
        gold,
        timeout=300,
        extra_env={"HIDDEN_SEED": AUDIT_HIDDEN_SEED},
    )
    result["gold_passes_seed2"] = gold2_run.returncode == 0
    result["gold_seed2_log"] = (gold2_run.stdout + gold2_run.stderr)[-2000:]
    gold_vet = run_go(["go", "vet", "./..."], gold, timeout=240)
    result["gold_vet"] = gold_vet.returncode == 0

    cheat = fresh("cheat")
    cheat_run = run_go(["go", "test", "-count=1", "-timeout", "3m", "./..."], cheat, timeout=300)
    result["cheat_fails"] = cheat_run.returncode != 0
    result["cheat_log"] = (cheat_run.stdout + cheat_run.stderr)[-2000:]

    result["ok"] = bool(
        result["module_build"]
        and result["module_test"]
        and result["module_vet"]
        and result["gofmt_clean"]
        and result["buggy_fails"]
        and result["gold_passes"]
        and result["gold_passes_seed2"]
        and result["gold_vet"]
        and result["cheat_fails"]
    )
    return result


def _failed_test_names(proc: subprocess.CompletedProcess[str]) -> list[str]:
    return sorted(set(re.findall(r"^--- FAIL:\s+(Test\w+)", proc.stdout, re.MULTILINE)))


def audit_unit_details(
    unit_dir: Path, cfg: Config, drawn: set[str], v: str, scratch_root: Path
) -> dict[str, object]:
    """Per-detail trap audit (the independence + affordance + B3 proof).

    For every drawn detail the unit is rebuilt with exactly that detail set to
    its trap value and everything else gold, then three Go-level checks run:

    - v=low smoke suite must PASS: the wrong value is invisible to the L0 set
      (the prose does not pin it and no worked example asserts it) — the trap
      is consistent with everything the solver sees at v=low.
    - v=high smoke suite must FAIL, and only the detail's own worked-example
      test: the wrong value IS visible at v=high (the affordance).
    - the hidden suite must FAIL, and only the detail's own test: the
      per-detail verifier has teeth, and no other detail's test or the guard
      depends on this detail's value (independence by construction).
    """
    domain = cfg.domain
    pkg = DOMAINS[domain]["pkg"]
    hidden_src = unit_dir / "hidden" / f"{pkg}_bb_test.go"
    root = scratch_root / (unit_dir.name + "-audit")
    if root.exists():
        shutil.rmtree(root)
    root.mkdir(parents=True)
    results: dict[str, object] = {}
    for did in sorted(drawn):
        entry = DETAILS[domain][did]
        smoke_name = entry["smoke_name"]
        hidden_name = entry["hidden_name"]

        tree_low = root / f"{did}-low"
        _write_tree(tree_low, render_excised_files(domain, cfg, drawn, "low", trap=did))
        r_low = run_go(
            ["go", "test", "-count=1", "-timeout", "3m", "-run", "^TestSmoke_", "./..."],
            tree_low,
            timeout=300,
        )

        tree_high = root / f"{did}-high"
        _write_tree(tree_high, render_excised_files(domain, cfg, drawn, "high", trap=did))
        r_high = run_go(
            ["go", "test", "-count=1", "-timeout", "3m", "-v", "-run", "^TestSmoke_", "./..."],
            tree_high,
            timeout=300,
        )
        high_failed = _failed_test_names(r_high)

        shutil.copy2(hidden_src, tree_high / hidden_src.name)
        r_hidden = run_go(
            ["go", "test", "-count=1", "-timeout", "3m", "-v", "-run", "^TestDetail_", "./..."],
            tree_high,
            timeout=300,
        )
        hidden_failed = _failed_test_names(r_hidden)

        results[did] = {
            "trap": did,
            "smoke_low_passes": r_low.returncode == 0,
            "smoke_low_log": (r_low.stdout + r_low.stderr)[-600:],
            "smoke_high_fails": r_high.returncode != 0,
            "smoke_high_failed_tests": high_failed,
            "smoke_high_fails_own": smoke_name in high_failed,
            "smoke_high_extra_failures": [n for n in high_failed if n != smoke_name],
            "hidden_fails": r_hidden.returncode != 0,
            "hidden_failed_tests": hidden_failed,
            "hidden_fails_own": hidden_name in hidden_failed,
            "hidden_extra_failures": [n for n in hidden_failed if n != hidden_name],
        }
    results["ok"] = all(
        bool(
            r["smoke_low_passes"]
            and r["smoke_high_fails"]
            and r["smoke_high_fails_own"]
            and not r["smoke_high_extra_failures"]
            and r["hidden_fails"]
            and r["hidden_fails_own"]
            and not r["hidden_extra_failures"]
        )
        for r in results.values()
        if isinstance(r, dict) and "trap" in r
    )
    return results


def generate_unit(
    *,
    seed: int,
    n_funcs: int,
    ratio: float,
    out: Path,
    domain: str = "store",
    depth: int = 0,
    fanout: int = 0,
    proof: bool = True,
    scratch_root: Path | None = None,
    s_band: tuple[int, int] | None = None,
    d: int = 0,
    v: str = "high",
    cfg: Config | None = None,
) -> UnitResult:
    """Generate one fabricated unit.

    ``d`` is the number of drawn behavioural details (0 = all registry
    details), ``v`` the verification-affordance level ("low": the bug report
    states the drawn details in prose and repro.sh shows the failing suite
    only; "high": repro.sh additionally prints a discriminating expected-vs-got
    line per drawn detail).  ``cfg`` overrides the config search (used by the
    dial grid to hold |S| and lines constant).
    """
    if domain not in DOMAINS:
        raise ValueError(f"unknown domain {domain!r}; choose from {sorted(DOMAINS)}")
    if v not in ("low", "high"):
        raise ValueError(f"v must be 'low' or 'high', got {v!r}")
    out = Path(out)
    if cfg is None:
        cfg = search_config(
            domain, n_funcs, ratio, seed, target_depth=depth, target_fanout=fanout, s_band=s_band
        )
    if not cfg.consts:
        cfg.consts = DOMAINS[domain]["consts"](seed)
    if d != 0 or v != "high":
        cfg.dial_suffix = f"-d{d}-v{v}"
    present = set(DOMAINS[domain]["entries"][: cfg.E])
    drawn_ids = draw_details(domain, seed, d, present=present)
    drawn = set(drawn_ids)
    stats = compute_stats(domain, cfg)
    module_files = _gofmt_tree(render_module_files(domain, cfg, drawn, v))
    excised_files = _gofmt_tree(render_excised_files(domain, cfg, drawn, v))
    cheat_files = _gofmt_tree(render_cheat_files(domain, cfg, drawn, v))
    pkg = DOMAINS[domain]["pkg"]
    rel_go = f"{pkg}.go"

    if out.exists():
        shutil.rmtree(out)
    out.mkdir(parents=True)
    _write_tree(out / "module", module_files)
    _write_tree(out / "excised_tree", excised_files)
    hidden = hidden_tests(domain, cfg, drawn)
    for rel, content in hidden:
        dest = out / "hidden" / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(content, encoding="utf-8")

    author = out / "_author"
    (author / "excised").mkdir(parents=True)
    gold_patch = render_patch(excised_files[rel_go], module_files[rel_go], rel_go)
    (author / "gold.patch").write_text(gold_patch, encoding="utf-8")
    excision_patch = render_patch(module_files[rel_go], excised_files[rel_go], rel_go)
    (author / "excised" / "excision.patch").write_text(excision_patch, encoding="utf-8")
    cheat_patch = render_patch(excised_files[rel_go], cheat_files[rel_go], rel_go)
    (author / "cheat.patch").write_text(cheat_patch, encoding="utf-8")
    (author / "api.md").write_text(render_api(domain, cfg, stats), encoding="utf-8")
    (author / "bugreport.md").write_text(render_bugreport(domain, cfg, drawn, v), encoding="utf-8")
    (author / "closure.md").write_text(render_closure(domain, cfg, stats), encoding="utf-8")
    (author / "contract.md").write_text(render_contract(domain, cfg, stats, drawn), encoding="utf-8")
    (author / "difficulty.md").write_text(render_difficulty(domain, cfg), encoding="utf-8")

    repro = (
        "#!/usr/bin/env bash\n"
        "set -uo pipefail\n"
        'cd "$(dirname "$0")"\n'
        'work="$(mktemp -d /tmp/fab.XXXXXX)"\n'
        'trap \'rm -rf "$work"\' EXIT\n'
        'cp -r excised_tree/. "$work"/\n'
        'cd "$work"\n'
        "go test -count=1 -timeout 5m -v ./...\n"
    )
    (out / "repro.sh").write_text(repro, encoding="utf-8")
    (out / "repro.sh").chmod(0o755)

    proof_result: dict[str, object] = {}
    if proof:
        scratch = scratch_root or (Path(os.environ.get("OUTPUTS_DIR", "outputs")) / "fabricated_proofs")
        proof_result = prove_unit(out, cfg, scratch, drawn, v)

    loc = module_files[rel_go].count("\n")
    sizes = {
        "module_go_loc": loc,
        "smoke_test_loc": excised_files[f"{pkg}_test.go"].count("\n"),
        "hidden_loc": sum(t.count("\n") for _, t in hidden),
        "bugreport_lines": len((author / "bugreport.md").read_text(encoding="utf-8").splitlines()),
        "gold_patch_bytes": len(gold_patch.encode()),
        "excision_patch_bytes": len(excision_patch.encode()),
        "cheat_patch_bytes": len(cheat_patch.encode()),
        "hidden_bytes": sum(len(t.encode()) for _, t in hidden),
        "removed_lines": (module_files[rel_go].count("\n") - excised_files[rel_go].count("\n")),
    }
    stats.loc = loc
    stats.sizes = sizes

    details_manifest = []
    for did in drawn_ids:
        entry = DETAILS[domain][did]
        shown = detail_shown(domain, cfg, did)
        details_manifest.append(
            {
                "id": did,
                "description": entry["description"],
                "prose": entry["prose"],
                "hidden_tests_covering": [entry["hidden_name"]],
                "stated_in_bugreport": True,
                "checkable_from_repro": v == "high",
                "examples_shown": shown,
            }
        )
    manifest = {
        "unit": cfg.unit_name(),
        "domain": domain,
        "module_path": cfg.module_name(),
        "seed": seed,
        "n_funcs_target": n_funcs,
        "n_funcs": stats.n_funcs,
        "ratio_requested": ratio,
        "ratio_achieved": round(stats.ratio, 6),
        "edges": {
            "internal": stats.internal,
            "boundary_in": stats.in_edges,
            "boundary_out": stats.out_edges,
            "total": stats.internal + stats.in_edges + stats.out_edges,
        },
        "graph": {"depth": stats.depth, "fanout_max": stats.fanout_max, "fanout_mean": round(stats.fanout_mean, 3)},
        "excision": {"S": list(stats.S), "S_size": len(stats.S)},
        "config": {
            "E": cfg.E,
            "K": cfg.K,
            "canon_mode": cfg.canon_mode,
            "canon_in_S": cfg.canon_in_S,
            "misc_in_S": cfg.misc_in_S,
            "place_in_S": cfg.place_in_S,
            "consumers": list(cfg.consumers),
            "fill": cfg.fill,
        },
        "dial": {
            "d_requested": d,
            "d_achieved": len(drawn_ids),
            "v": v,
            "registry_size": len(DETAILS[domain]),
            "draw": "splitmix shuffle of the registry per (domain, seed, d), take first d",
            "constant_across_grid": {
                "S_size": len(stats.S),
                "n_funcs": stats.n_funcs,
                "module_loc": loc,
                "removed_lines": sizes["removed_lines"],
            },
        },
        "details": details_manifest,
        "files": sizes,
        "proof": proof_result,
        "generated_at": datetime.now(UTC).isoformat(timespec="seconds"),
        "command": (
            f"uv run python scripts/fabricate_repo.py --seed {seed} --n-funcs {n_funcs} "
            f"--ratio {ratio} --domain {domain} --d {d} --v {v} --out {out}"
        ),
    }
    (out / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    return UnitResult(cfg=cfg, stats=stats, unit_dir=out, proof=proof_result)


def main_cli(argv: list[str] | None = None) -> int:
    """CLI entry used by scripts/fabricate_repo.py (kept thin)."""
    import argparse

    parser = argparse.ArgumentParser(description="Generate a fabricated Go repo unit with a target closure ratio.")
    parser.add_argument("--seed", type=int, required=True)
    parser.add_argument("--n-funcs", type=int, default=24)
    parser.add_argument("--ratio", type=float, required=True)
    parser.add_argument("--domain", choices=sorted(DOMAINS), default="store")
    parser.add_argument("--depth", type=int, default=0, help="soft target for longest call chain (0 = unconstrained)")
    parser.add_argument("--fan-out", type=int, default=0, help="soft target for max out-degree (0 = unconstrained)")
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--s-min", type=int, default=None, help="min |S| for the config search (dose-response band)")
    parser.add_argument("--s-max", type=int, default=None, help="max |S| for the config search (dose-response band)")
    parser.add_argument("--d", type=int, default=0, help="number of drawn behavioural details (0 = all)")
    parser.add_argument("--v", choices=("low", "high"), default="high", help="verification-affordance level")
    parser.add_argument("--no-proof", action="store_true")
    parser.add_argument("--log", type=Path, default=Path("outputs/closure_R.log"), help="append a JSONL row per unit")
    args = parser.parse_args(argv)

    s_band = None
    if args.s_min is not None or args.s_max is not None:
        s_band = (args.s_min or 0, args.s_max or 10**9)
    result = generate_unit(
        seed=args.seed,
        n_funcs=args.n_funcs,
        ratio=args.ratio,
        out=args.out,
        domain=args.domain,
        depth=args.depth,
        fanout=args.fan_out,
        proof=not args.no_proof,
        s_band=s_band,
        d=args.d,
        v=args.v,
    )
    s = result.stats
    print(f"unit: {result.unit_dir}")
    print(f"requested ratio: {args.ratio:.3f}   achieved: {s.ratio:.3f} "
          f"(internal {s.internal}, boundary_in {s.in_edges}, boundary_out {s.out_edges})")
    print(f"excised set S: {len(s.S)} of {s.n_funcs} functions; depth {s.depth}; fan-out max {s.fanout_max}")
    pkg_go = f"{DOMAINS[args.domain]['pkg']}.go"
    print(f"module {pkg_go} {s.loc} LOC; gold.patch {s.sizes['gold_patch_bytes']} B; "
          f"excision.patch {s.sizes['excision_patch_bytes']} B; cheat.patch {s.sizes['cheat_patch_bytes']} B")
    if result.proof:
        p = result.proof
        verdict = "OK" if p.get("ok") else "FAILED"
        print(f"proof: module build/test/vet {p['module_build']}/{p['module_test']}/{p['module_vet']}; "
              f"buggy fails {p['buggy_fails']}; gold passes {p['gold_passes']}; cheat fails {p['cheat_fails']} -> {verdict}")
        if not p.get("ok"):
            for key in ("module_build_log", "module_test_log", "module_vet_log", "buggy_log", "gold_log", "cheat_log"):
                log = str(p.get(key, "")).strip()
                if log:
                    print(f"--- {key} ---")
                    print(log[-1500:])
            return 1

    row = {
        "ts": datetime.now(UTC).isoformat(timespec="seconds"),
        "event": "fabricate_unit",
        "unit": result.unit_dir.name,
        "domain": args.domain,
        "seed": args.seed,
        "d": args.d,
        "v": args.v,
        "n_funcs": s.n_funcs,
        "ratio_requested": args.ratio,
        "ratio_achieved": round(s.ratio, 6),
        "internal": s.internal,
        "boundary_in": s.in_edges,
        "boundary_out": s.out_edges,
        "S_size": len(s.S),
        "proof_ok": bool(result.proof.get("ok")) if result.proof else None,
        "module_loc": s.loc,
    }
    args.log.parent.mkdir(parents=True, exist_ok=True)
    with args.log.open("a", encoding="utf-8") as fh:
        fh.write(json.dumps(row) + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main_cli())
