"""Fabricated self-contained Go repos with a controllable closure ratio.

A unit is a tiny single-package Go module (``example.internal/<unit>``) over a
small typed domain (record store or interval scheduler).  The generator emits
the full module with real, deterministic semantics for every function, plus an
in-tree smoke test suite that passes on the full module.  It then excises a
chosen set S of functions (bodies -> ``panic("excised: ...")`` stubs, decls
kept, imports pruned) and produces the four author artifacts plus a black-box
hidden suite (rule B4: only exported entry points of S; rule B5: seeded-random
property checks with a reference implementation in the test).

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

    def module_name(self) -> str:
        return f"example.internal/{self.unit_name()}"

    def unit_name(self) -> str:
        return f"{self.domain}-s{self.seed}-r{round(self.target_ratio * 100):03d}"


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


def run_go(args: list[str], cwd: Path, timeout: int = 300) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
        env=GO_ENV,
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
    limit: int = int(c["limit"])
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
                "\t\t\tif overLimit(e.size, maxSize) {\n"
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
            f"\treturn bits.RotateLeft64(v, {rot})\n"
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
                f"\tv := x*{_hex(mul[j])} + {_hex(adds[j])}\n"
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
                f"\t\th = h*{_hex(int(c['foldBase']))} + uint64(tag[i])\n"
                "\t}\n"
                "\treturn h\n"
                "}\n"
            ),
            set(),
            "func foldTag(tag string) uint64 {\n\treturn 0\n}\n",
        )

    # --- misc --------------------------------------------------------------
    emit("validateKey", "misc", "func validateKey(k uint64) bool", "func validateKey(k uint64) bool {\n\treturn k != math.MaxUint64\n}\n", {"math"}, "func validateKey(k uint64) bool {\n\treturn true\n}\n")
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
        f"func defaultLimit() int {{\n\treturn {limit}\n}}\n",
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
        if not set(STORE_CONSUMERS[cname]).intersection(present):
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
    c = cfg.consts
    mul: list[int] = c["mixMul"][: cfg.K]  # type: ignore[index]
    adds: list[int] = c["mixAdd"][: cfg.K]  # type: ignore[index]
    rot: int = int(c["mixRot"])
    cap: int = int(c["cap"])
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
            f"\treturn bits.RotateLeft64(v, {rot})\n"
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
                f"\tv := x*{_hex(mul[j])} + {_hex(adds[j])}\n"
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
        f"func defaultCap() int {{\n\treturn {cap}\n}}\n",
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
        if not set(SCHED_CONSUMERS[cname]).intersection(present):
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
        available = [c for c in consumers if set(consumers[c]).intersection(present)]
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


def search_config(domain: str, n_funcs: int, target_ratio: float, seed: int, *, target_depth: int = 0, target_fanout: int = 0) -> Config:
    best: Config | None = None
    best_stats: UnitStats | None = None
    best_score = float("inf")
    for cfg in _candidate_configs(domain, n_funcs, seed):
        cfg.target_ratio = target_ratio
        cfg.target_depth = target_depth
        cfg.target_fanout = target_fanout
        stats = compute_stats(domain, cfg)
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
        raise ValueError(f"no config fits {domain} with n_funcs={n_funcs}; raise n_funcs or loosen the target")
    return best


# --------------------------------------------------------------------------- #
# rendering: module / excised / cheat trees
# --------------------------------------------------------------------------- #
def _render_store_go(domain: str, cfg: Config, fns: dict[str, FnSpec], mode: str) -> str:
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
        if mode == "excised" and key in S:
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
    header = (
        f"const nbuckets = {int(cfg.consts['nbuckets'])}\n\n"
        f"{spec['types']}"
    )
    body = header + "\n" + "\n".join(parts)
    return render_go_file(pkg, imports, body, spec["pkg_doc"])


def compute_S(domain: str, cfg: Config) -> list[str]:
    fns = DOMAINS[domain]["functions"](cfg)
    groups_s = DOMAINS[domain]["groups_in_S"](cfg)
    return sorted(k for k, f in fns.items() if f.group in groups_s)


def _in_tree_tests(domain: str, cfg: Config) -> str:
    pkg = DOMAINS[domain]["pkg"]
    present = set(DOMAINS[domain]["entries"][: cfg.E])
    body: list[str] = []
    if {"NewStore", "Store.Put", "Store.Get", "Store.Count", "Store.Balance"}.issubset(present):
        body.append(
            
                "func TestSmokeRoundtrip(t *testing.T) {\n"
                "\ts := NewStore()\n"
                "\tif !s.Put(1, \"alpha\", 10) {\n\t\tt.Fatal(\"put failed\")\n\t}\n"
                "\trec, ok := s.Get(1)\n"
                "\tif !ok || rec.Size != 10 || rec.Tag != \"alpha\" {\n\t\tt.Fatalf(\"bad get: %+v %v\", rec, ok)\n\t}\n"
                "\tif s.Count() != 1 {\n\t\tt.Fatal(\"count\")\n\t}\n"
                "\tif s.Balance() < 1 {\n\t\tt.Fatal(\"balance\")\n\t}\n"
                "}\n"
            
        )
    if {"Store.Put", "Store.Tags", "Store.Total", "Store.Count", "Store.Prune"}.issubset(present):
        body.append(
            
                "func TestSmokeTagsPrune(t *testing.T) {\n"
                "\ts := NewStore()\n"
                "\ts.Put(1, \"x\", 5)\n"
                "\ts.Put(2, \"y\", 500)\n"
                "\tif len(s.Tags(\"\")) != 2 {\n\t\tt.Fatal(\"tags\")\n\t}\n"
                "\tif s.Total() != 505 {\n\t\tt.Fatal(\"total\")\n\t}\n"
                "\tif n := s.Prune(100); n != 1 {\n\t\tt.Fatalf(\"prune: %d\", n)\n\t}\n"
                "\tif s.Count() != 1 {\n\t\tt.Fatal(\"count after prune\")\n\t}\n"
                "}\n"
            
        )
    if {"NewScheduler", "Sched.Add", "Sched.At", "Sched.Len"}.issubset(present):
        body.append(
            
                "func TestSmokeSchedule(t *testing.T) {\n"
                "\tsc := NewScheduler()\n"
                "\tif !sc.Add(Interval{Start: 10, End: 20}) {\n\t\tt.Fatal(\"add failed\")\n\t}\n"
                "\tif !sc.At(15) || sc.At(5) {\n\t\tt.Fatal(\"at\")\n\t}\n"
                "\tif sc.Len() != 1 {\n\t\tt.Fatal(\"len\")\n\t}\n"
                "\tif sc.Add(Interval{Start: 15, End: 25}) {\n\t\tt.Fatal(\"overlap accepted\")\n\t}\n"
                "}\n"
            
        )
    if {"Sched.Total", "Sched.Gap"}.issubset(present):
        body.append(
            
                "func TestSmokeTotals(t *testing.T) {\n"
                "\tsc := NewScheduler()\n"
                "\tsc.Add(Interval{Start: 0, End: 10})\n"
                "\tsc.Add(Interval{Start: 20, End: 30})\n"
                "\tif sc.Total() != 20 {\n\t\tt.Fatalf(\"total: %d\", sc.Total())\n\t}\n"
                "\tif sc.Gap() < 10 {\n\t\tt.Fatalf(\"gap: %d\", sc.Gap())\n\t}\n"
                "}\n"
            
        )
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
    fns = DOMAINS[domain]["functions"](cfg)
    filler_names = [f.name for f in fns.values() if f.group == "filler"]
    if filler_names:
        checks = "\n\t".join(filler_checks[n] for n in filler_names)
        body.append(f"func TestSmokeFillers(t *testing.T) {{\n\t{checks}\n}}\n")
    if not body:
        body.append("func TestSmokeNoop(t *testing.T) {}\n")
    return render_go_file(pkg, {"testing"}, "\n".join(body), "In-tree smoke tests (not the verifier; see hidden/).")


def render_module_files(domain: str, cfg: Config) -> dict[str, str]:
    fns = DOMAINS[domain]["functions"](cfg)
    return {
        "go.mod": f"module {cfg.module_name()}\n\ngo 1.23\n",
        f"{DOMAINS[domain]['pkg']}.go": _render_store_go(domain, cfg, fns, "gold"),
        f"{DOMAINS[domain]['pkg']}_test.go": _in_tree_tests(domain, cfg),
    }


def render_excised_files(domain: str, cfg: Config) -> dict[str, str]:
    fns = DOMAINS[domain]["functions"](cfg)
    return {
        "go.mod": f"module {cfg.module_name()}\n\ngo 1.23\n",
        f"{DOMAINS[domain]['pkg']}.go": _render_store_go(domain, cfg, fns, "excised"),
        f"{DOMAINS[domain]['pkg']}_test.go": _in_tree_tests(domain, cfg),
    }


def render_cheat_files(domain: str, cfg: Config) -> dict[str, str]:
    fns = DOMAINS[domain]["functions"](cfg)
    return {
        "go.mod": f"module {cfg.module_name()}\n\ngo 1.23\n",
        f"{DOMAINS[domain]['pkg']}.go": _render_store_go(domain, cfg, fns, "cheat"),
        f"{DOMAINS[domain]['pkg']}_test.go": _in_tree_tests(domain, cfg),
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


def store_hidden_tests(cfg: Config) -> list[tuple[str, str]]:
    c = cfg.consts
    mul: list[int] = c["mixMul"][: cfg.K]  # type: ignore[index]
    adds: list[int] = c["mixAdd"][: cfg.K]  # type: ignore[index]
    rot: int = int(c["mixRot"])
    limit: int = int(c["limit"])
    fold_base: int = int(c["foldBase"])
    present = set(STORE_ENTRIES[: cfg.E])
    imports = {"math/bits", "math/rand", "os", "strconv", "strings", "testing"}

    mul_src = ", ".join(f"0x{m:X}" for m in mul)
    add_src = ", ".join(f"0x{a:X}" for a in adds)

    refs = [
        _hidden_seed_helper(),
        "const refBuckets = 8",
        f"var refLimit = {limit}",
        f"var refMul = []uint64{{{mul_src}}}",
        f"var refAdd = []uint64{{{add_src}}}",
        f"var refRot = {rot}",
        f"var refFoldBase uint64 = 0x{fold_base:X}",
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
    ]
    if "Store.Tags" in present:
        refs.append(
            "func refFold(tag string) uint64 {\n"
            "\th := uint64(0)\n"
            "\tfor i := 0; i < len(tag); i++ {\n"
            "\t\th = h*refFoldBase + uint64(tag[i])\n"
            "\t}\n"
            "\treturn h\n"
            "}\n"
        )
        imports.add("reflect")

    tests: list[str] = []
    tests.append(
        
            "func TestBucketOfProperty(t *testing.T) {\n"
            "\ts := NewStore()\n"
            "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
            "\tkeys := []uint64{0, 1, 2, 7, 42, ^uint64(0), ^uint64(0) - 1, 1 << 40}\n"
            "\tfor i := 0; i < 400; i++ {\n"
            "\t\tkeys = append(keys, rng.Uint64())\n"
            "\t}\n"
            "\tfor _, k := range keys {\n"
            "\t\tif got := s.BucketOf(k); got != refBucket(k) {\n"
            "\t\t\tt.Fatalf(\"BucketOf(%d) = %d, want %d\", k, got, refBucket(k))\n"
            "\t\t}\n"
            "\t}\n"
            "}\n"
        
    )
    if {"NewStore", "Store.Put", "Store.Get"}.issubset(present):
        tests.append(
            
                "func TestPutGetRoundtrip(t *testing.T) {\n"
                "\ts := NewStore()\n"
                "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
                "\ttype want struct {\n"
                "\t\tsize int\n"
                "\t\ttag  string\n"
                "\t}\n"
                "\tstate := map[uint64]want{}\n"
                "\tkeyPool := []uint64{0, 1, 42, 999, 1 << 33, ^uint64(0) - 1}\n"
                '\ttags := []string{"alpha", " Alpha ", "a--b", "under_score", "UPPER", "x1", "!!!", ""}\n'
                "\tfor i := 0; i < 250; i++ {\n"
                "\t\tk := keyPool[rng.Intn(len(keyPool))]\n"
                "\t\tif rng.Intn(10) == 0 {\n"
                "\t\t\tk = rng.Uint64()\n"
                "\t\t}\n"
                "\t\ttag := tags[rng.Intn(len(tags))]\n"
                "\t\tsize := rng.Intn(3000)\n"
                "\t\tw := want{size: size, tag: refCanon(tag)}\n"
                "\t\tif k == ^uint64(0) || w.tag == \"\" || size > refLimit {\n"
                "\t\t\tcontinue\n"
                "\t\t}\n"
                "\t\tstate[k] = w\n"
                "\t\ts.Put(k, tag, size)\n"
                "\t}\n"
                "\tfor k, w := range state {\n"
                "\t\trec, ok := s.Get(k)\n"
                "\t\tif !ok {\n"
                "\t\t\tt.Fatalf(\"Get(%d) missing\", k)\n"
                "\t\t}\n"
                "\t\tif rec.Key != k || rec.Size != w.size || rec.Tag != w.tag {\n"
                "\t\t\tt.Fatalf(\"Get(%d) = %+v, want size %d tag %q\", k, rec, w.size, w.tag)\n"
                "\t\t}\n"
                "\t}\n"
                "\tfor i := 0; i < 100; i++ {\n"
                "\t\tk := rng.Uint64()\n"
                "\t\tif _, ok := state[k]; ok {\n"
                "\t\t\tcontinue\n"
                "\t\t}\n"
                "\t\tif _, ok := s.Get(k); ok {\n"
                "\t\t\tt.Fatalf(\"Get(%d) present but never stored\", k)\n"
                "\t\t}\n"
                "\t}\n"
                "}\n"
            
        )
        tests.append(
            
                "func TestPutRejects(t *testing.T) {\n"
                "\ts := NewStore()\n"
                "\tif s.Put(^uint64(0), \"x\", 1) {\n"
                "\t\tt.Fatal(\"reserved key accepted\")\n"
                "\t}\n"
                "\tif s.Put(1, \"!!!\", 1) {\n"
                "\t\tt.Fatal(\"empty canonical tag accepted\")\n"
                "\t}\n"
                "\tif s.Put(1, \"ok\", refLimit+1) {\n"
                "\t\tt.Fatal(\"oversize accepted\")\n"
                "\t}\n"
                "\tif !s.Put(1, \"ok\", refLimit) {\n"
                "\t\tt.Fatal(\"at-limit rejected\")\n"
                "\t}\n"
                "\tif !s.Put(2, \"ok\", 0) {\n"
                "\t\tt.Fatal(\"zero-size rejected\")\n"
                "\t}\n"
                "}\n"
            
        )
    if {"NewStore", "Store.Put", "Store.Count", "Store.Total", "Store.Balance", "Store.BucketOf"}.issubset(present):
        tests.append(
            
                "func TestAggregates(t *testing.T) {\n"
                "\ts := NewStore()\n"
                "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
                "\ttype rec struct {\n"
                "\t\tsize int\n"
                "\t}\n"
                "\tstate := map[uint64]rec{}\n"
                "\tvar buckets [refBuckets]int\n"
                "\tfor i := 0; i < 200; i++ {\n"
                "\t\tk := rng.Uint64() &^ 1\n"
                "\t\tsize := rng.Intn(2000)\n"
                "\t\tif size > refLimit {\n"
                "\t\t\tsize = refLimit\n"
                "\t\t}\n"
                "\t\tif _, ok := state[k]; ok {\n"
                "\t\t\tbuckets[refBucket(k)]--\n"
                "\t\t}\n"
                "\t\tstate[k] = rec{size: size}\n"
                "\t\tbuckets[refBucket(k)]++\n"
                "\t\ts.Put(k, \"t\"+strconv.Itoa(i%50), size)\n"
                "\t}\n"
                "\tif n := s.Count(); n != len(state) {\n"
                "\t\tt.Fatalf(\"Count = %d, want %d\", n, len(state))\n"
                "\t}\n"
                "\ttotal := 0\n"
                "\tfor _, r := range state {\n"
                "\t\ttotal += r.size\n"
                "\t}\n"
                "\tif got := s.Total(); got != total {\n"
                "\t\tt.Fatalf(\"Total = %d, want %d\", got, total)\n"
                "\t}\n"
                "\tmaxb := 0\n"
                "\tfor _, n := range buckets {\n"
                "\t\tif n > maxb {\n"
                "\t\t\tmaxb = n\n"
                "\t\t}\n"
                "\t}\n"
                "\tif got := s.Balance(); got != maxb {\n"
                "\t\tt.Fatalf(\"Balance = %d, want %d\", got, maxb)\n"
                "\t}\n"
                "}\n"
            
        )
    if {"NewStore", "Store.Put", "Store.Tags"}.issubset(present):
        tests.append(_store_tags_test(cfg))
    if {"NewStore", "Store.Put", "Store.Get", "Store.Count", "Store.Total", "Store.Prune"}.issubset(present):
        tests.append(
            
                "func TestPrune(t *testing.T) {\n"
                "\ts := NewStore()\n"
                "\tfor i := 0; i < 100; i++ {\n"
                "\t\ts.Put(uint64(i), \"k\"+strconv.Itoa(i), i)\n"
                "\t}\n"
                "\tif got := s.Prune(50); got != 49 {\n"
                "\t\tt.Fatalf(\"Prune = %d, want 49\", got)\n"
                "\t}\n"
                "\tif got := s.Count(); got != 51 {\n"
                "\t\tt.Fatalf(\"Count after Prune = %d, want 51\", got)\n"
                "\t}\n"
                "\tif _, ok := s.Get(60); ok {\n"
                "\t\tt.Fatal(\"pruned record still present\")\n"
                "\t}\n"
                "\tif _, ok := s.Get(40); !ok {\n"
                "\t\tt.Fatal(\"kept record missing\")\n"
                "\t}\n"
                "\ttotal := 0\n"
                "\tfor i := 0; i <= 50; i++ {\n"
                "\t\ttotal += i\n"
                "\t}\n"
                "\tif got := s.Total(); got != total {\n"
                "\t\tt.Fatalf(\"Total after Prune = %d, want %d\", got, total)\n"
                "\t}\n"
                "}\n"
            
        )
    if {"NewStore", "Store.Put", "Store.Get", "Store.BucketOf"}.issubset(present):
        tests.append(
            
                "func TestAdversarialEdges(t *testing.T) {\n"
                "\ts := NewStore()\n"
                "\tif !s.Put(0, \"zero\", 0) {\n"
                "\t\tt.Fatal(\"key 0 rejected\")\n"
                "\t}\n"
                "\tif !s.Put(^uint64(0)-1, \"max\", 1) {\n"
                "\t\tt.Fatal(\"key max-1 rejected\")\n"
                "\t}\n"
                "\tif s.Put(^uint64(0), \"reserved\", 1) {\n"
                "\t\tt.Fatal(\"reserved key accepted\")\n"
                "\t}\n"
                "\tif s.Put(1, \"\", 1) {\n"
                "\t\tt.Fatal(\"empty tag accepted\")\n"
                "\t}\n"
                "\tif s.Put(1, \"  ---  \", 1) {\n"
                "\t\tt.Fatal(\"dash-only tag accepted\")\n"
                "\t}\n"
                "\tif !s.Put(2, \"  ok tag  \", 1) {\n"
                "\t\tt.Fatal(\"valid tag rejected\")\n"
                "\t}\n"
                "\tif rec, ok := s.Get(2); !ok || rec.Tag != \"ok-tag\" {\n"
                "\t\tt.Fatalf(\"canonical tag wrong: %+v %v\", rec, ok)\n"
                "\t}\n"
                "\tif got := s.BucketOf(0); got < 0 || got >= refBuckets {\n"
                "\t\tt.Fatalf(\"BucketOf(0) out of range: %d\", got)\n"
                "\t}\n"
                "}\n"
            
        )

    body = "\n".join(refs + tests)
    imports.add(cfg.module_name())
    content = render_go_file("store_test", imports, body, "Hidden black-box suite for the fabricated store unit (generated).")
    content = content.replace("NewStore(", "store.NewStore(")
    return [("store_bb_test.go", gofmt_text(content))]


def _store_tags_test(cfg: Config) -> str:
    c = cfg.consts
    fold_base: int = int(c["foldBase"])

    def fold(s: str) -> int:
        h = 0
        for ch in s.encode():
            h = (h * fold_base + ch) & M64
        return h

    # Pick a stored-tag set whose (tag-hash, lex) order differs from pure lex
    # order, so a fold-hash cheat (returning 0) deterministically fails this
    # test; always include "alpha" / "al pha" so the prefix assertion is live.
    pool = ["zebra", "Zebra", "beta", "x", "x1", "gamma", "under_score", "UPPER", "a b c", "q-1"]
    rng = SplitMix(cfg.seed ^ 0x5EED)
    stored: list[str] = []
    for _ in range(500):
        extras = sorted(pool, key=lambda _t: rng.next())[:5]
        stored = ["alpha", "al pha"] + extras
        canon = sorted({cfg_canon(t) for t in stored})
        if len(canon) >= 5 and sorted(canon, key=lambda t: (fold(t), t)) != sorted(canon):
            break
    if not stored:
        raise RuntimeError("could not build a tags-order test set")
    canon_tags = sorted({cfg_canon(t) for t in stored})
    want = sorted(canon_tags, key=lambda t: (fold(t), t))
    puts = "\n".join(f"\t\ts.Put({i + 1}, {_go_str(stored[i])}, 1)" for i in range(len(stored)))
    want_src = ", ".join(_go_str(t) for t in want)
    prefix_want = ", ".join(_go_str(t) for t in want if t.startswith("al"))
    return (
        "func TestTagsSort(t *testing.T) {\n"
        "\ts := NewStore()\n"
        f"{puts}\n"
        "\twant := []string{" + want_src + "}\n"
        "\tgot := s.Tags(\"\")\n"
        "\tif !reflect.DeepEqual(got, want) {\n"
        "\t\tt.Fatalf(\"Tags() = %v, want %v\", got, want)\n"
        "\t}\n"
        "\tgot = s.Tags(\"al\")\n"
        "\tif !reflect.DeepEqual(got, []string{" + prefix_want + "}) {\n"
        "\t\tt.Fatalf(\"Tags(al) = %v\", got)\n"
        "\t}\n"
        "}\n"
    )


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


def sched_hidden_tests(cfg: Config) -> list[tuple[str, str]]:
    c = cfg.consts
    mul: list[int] = c["mixMul"][: cfg.K]  # type: ignore[index]
    adds: list[int] = c["mixAdd"][: cfg.K]  # type: ignore[index]
    rot: int = int(c["mixRot"])
    cap: int = int(c["cap"])
    present = set(SCHED_ENTRIES[: cfg.E])
    mul_src = ", ".join(f"0x{m:X}" for m in mul)
    add_src = ", ".join(f"0x{a:X}" for a in adds)
    imports = {"math/bits", "math/rand", "os", "strconv", "testing"}

    refs = [
        _hidden_seed_helper(),
        "const refBuckets = 8",
        f"var refCap = {cap}",
        f"var refMul = []uint64{{{mul_src}}}",
        f"var refAdd = []uint64{{{add_src}}}",
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
        (
            "func refNorm(iv Interval, cap int) Interval {\n"
            "\tif iv.Start > iv.End {\n"
            "\t\tiv.Start, iv.End = iv.End, iv.Start\n"
            "\t}\n"
            "\tif iv.Start < 0 {\n\t\tiv.Start = 0\n\t}\n"
            "\tif iv.End > cap {\n\t\tiv.End = cap\n\t}\n"
            "\tif iv.End < iv.Start {\n\t\tiv.End = iv.Start\n\t}\n"
            "\treturn iv\n"
            "}\n"
        ),
        "func refOverlaps(a, b Interval) bool {\n\treturn a.Start < b.End && b.Start < a.End\n}\n",
        (
            "type refSched struct {\n"
            "\tcap   int\n"
            "\tslots []Interval\n"
            "}\n"
        ),
        (
            "func (r *refSched) add(iv Interval) bool {\n"
            "\tn := refNorm(iv, r.cap)\n"
            "\tif n.Start >= n.End {\n\t\treturn false\n\t}\n"
            "\tfor _, s := range r.slots {\n"
            "\t\tif refOverlaps(s, n) {\n\t\t\treturn false\n\t\t}\n"
            "\t}\n"
            "\ti := 0\n"
            "\tfor i < len(r.slots) && r.slots[i].Start < n.Start {\n"
            "\t\ti++\n"
            "\t}\n"
            "\tr.slots = append(r.slots, Interval{})\n"
            "\tcopy(r.slots[i+1:], r.slots[i:])\n"
            "\tr.slots[i] = n\n"
            "\treturn true\n"
            "}\n"
        ),
        (
            "func (r *refSched) at(x int) bool {\n"
            "\tfor _, s := range r.slots {\n"
            "\t\tif s.Start <= x && x < s.End {\n\t\t\treturn true\n\t\t}\n"
            "\t}\n"
            "\treturn false\n"
            "}\n"
        ),
    ]
    if {"Sched.Total", "Sched.Gap"}.issubset(present):
        refs.append(
            
                "func (r *refSched) total() int {\n"
                "\tt := 0\n"
                "\tfor _, s := range r.slots {\n"
                "\t\tt += s.End - s.Start\n"
                "\t}\n"
                "\treturn t\n"
                "}\n"
            
        )
        refs.append(
            
                "func (r *refSched) gap() int {\n"
                "\tbest := 0\n"
                "\tprev := 0\n"
                "\tfor _, s := range r.slots {\n"
                "\t\tif s.Start > prev && s.Start-prev > best {\n"
                "\t\t\tbest = s.Start - prev\n"
                "\t\t}\n"
                "\t\tif s.End > prev {\n"
                "\t\t\tprev = s.End\n"
                "\t\t}\n"
                "\t}\n"
                "\tif r.cap > prev && r.cap-prev > best {\n"
                "\t\tbest = r.cap - prev\n"
                "\t}\n"
                "\treturn best\n"
                "}\n"
            
        )
    if "Sched.Merge" in present:
        refs.append(
            
                "func (r *refSched) merge() int {\n"
                "\tif len(r.slots) < 2 {\n\t\treturn 0\n\t}\n"
                "\tmerged := 0\n"
                "\tout := r.slots[:1]\n"
                "\tfor _, iv := range r.slots[1:] {\n"
                "\t\tlast := &out[len(out)-1]\n"
                "\t\tif refOverlaps(*last, iv) || last.End >= iv.Start {\n"
                "\t\t\tif iv.End > last.End {\n"
                "\t\t\t\tlast.End = iv.End\n"
                "\t\t\t}\n"
                "\t\t\tmerged++\n"
                "\t\t} else {\n"
                "\t\t\tout = append(out, iv)\n"
                "\t\t}\n"
                "\t}\n"
                "\tr.slots = out\n"
                "\treturn merged\n"
                "}\n"
            
        )

    tests = [
        (
            "func TestSlotProperty(t *testing.T) {\n"
            "\tsc := NewScheduler()\n"
            "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
            "\tfor i := 0; i < 400; i++ {\n"
            "\t\tx := rng.Intn(3*refCap) - refCap\n"
            "\t\tif got := sc.Slot(x); got != refSlot(x, refCap) {\n"
            "\t\t\tt.Fatalf(\"Slot(%d) = %d, want %d\", x, got, refSlot(x, refCap))\n"
            "\t\t}\n"
            "\t}\n"
            "}\n"
        )
    ]
    if {"NewScheduler", "Sched.Add", "Sched.At", "Sched.Len", "Sched.Overlaps"}.issubset(present):
        tests.append(
            
                "func TestAddAtCoverage(t *testing.T) {\n"
                "\tsc := NewScheduler()\n"
                "\tref := &refSched{cap: refCap}\n"
                "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
                "\tfor i := 0; i < 200; i++ {\n"
                "\t\tstart := rng.Intn(2*refCap) - refCap/2\n"
                "\t\tlength := rng.Intn(refCap / 3)\n"
                "\t\tiv := Interval{Start: start, End: start + length}\n"
                "\t\tgot := sc.Add(iv)\n"
                "\t\twant := ref.add(iv)\n"
                "\t\tif got != want {\n"
                "\t\t\tt.Fatalf(\"Add(%+v) = %v, want %v\", iv, got, want)\n"
                "\t\t}\n"
                "\t\tif sc.Len() != len(ref.slots) {\n"
                "\t\t\tt.Fatalf(\"Len = %d, want %d\", sc.Len(), len(ref.slots))\n"
                "\t\t}\n"
                "\t}\n"
                "\tfor i := 0; i < 300; i++ {\n"
                "\t\tx := rng.Intn(3*refCap) - refCap\n"
                "\t\tif got := sc.At(x); got != ref.at(x) {\n"
                "\t\t\tt.Fatalf(\"At(%d) = %v, want %v\", x, got, ref.at(x))\n"
                "\t\t}\n"
                "\t}\n"
                "\tfor i := 0; i < 100; i++ {\n"
                "\t\ta := Interval{Start: rng.Intn(refCap), End: rng.Intn(refCap) + rng.Intn(refCap / 4)}\n"
                "\t\tn := refNorm(a, refCap)\n"
                "\t\twant := false\n"
                "\t\tfor _, s := range ref.slots {\n"
                "\t\t\tif refOverlaps(s, n) {\n"
                "\t\t\t\twant = true\n"
                "\t\t\t\tbreak\n"
                "\t\t\t}\n"
                "\t\t}\n"
                "\t\tif got := sc.Overlaps(a); got != want {\n"
                "\t\t\tt.Fatalf(\"Overlaps(%+v) = %v, want %v\", a, got, want)\n"
                "\t\t}\n"
                "\t}\n"
                "}\n"
            
        )
    if {"Sched.Total", "Sched.Gap"}.issubset(present):
        tests.append(
            
                "func TestTotalGap(t *testing.T) {\n"
                "\tsc := NewScheduler()\n"
                "\tref := &refSched{cap: refCap}\n"
                "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
                "\tfor i := 0; i < 120; i++ {\n"
                "\t\tstart := rng.Intn(refCap)\n"
                "\t\tiv := Interval{Start: start, End: start + rng.Intn(refCap / 5)}\n"
                "\t\tsc.Add(iv)\n"
                "\t\tref.add(iv)\n"
                "\t}\n"
                "\tif got := sc.Total(); got != ref.total() {\n"
                "\t\tt.Fatalf(\"Total = %d, want %d\", got, ref.total())\n"
                "\t}\n"
                "\tif got := sc.Gap(); got != ref.gap() {\n"
                "\t\tt.Fatalf(\"Gap = %d, want %d\", got, ref.gap())\n"
                "\t}\n"
                "}\n"
            
        )
    if "Sched.Merge" in present:
        tests.append(
            
                "func TestMergeCount(t *testing.T) {\n"
                "\tsc := NewScheduler()\n"
                "\tref := &refSched{cap: refCap}\n"
                "\trng := rand.New(rand.NewSource(hiddenSeed()))\n"
                "\tfor i := 0; i < 150; i++ {\n"
                "\t\tstart := rng.Intn(refCap)\n"
                "\t\tiv := Interval{Start: start, End: start + 1 + rng.Intn(refCap / 10)}\n"
                "\t\tsc.Add(iv)\n"
                "\t\tref.add(iv)\n"
                "\t}\n"
                "\tif got, want := sc.Merge(), ref.merge(); got != want {\n"
                "\t\tt.Fatalf(\"Merge = %d, want %d\", got, want)\n"
                "\t}\n"
                "\tif sc.Len() != len(ref.slots) {\n"
                "\t\tt.Fatalf(\"Len after Merge = %d, want %d\", sc.Len(), len(ref.slots))\n"
                "\t}\n"
                "\tif got := sc.Total(); got != ref.total() {\n"
                "\t\tt.Fatalf(\"Total after Merge = %d, want %d\", got, ref.total())\n"
                "\t}\n"
                "}\n"
            
        )
    tests.append(
        
            "func TestAdversarialEdges(t *testing.T) {\n"
            "\tsc := NewScheduler()\n"
            "\tif sc.Add(Interval{Start: 10, End: 10}) {\n"
            "\t\tt.Fatal(\"empty interval accepted\")\n"
            "\t}\n"
            "\tif !sc.Add(Interval{Start: 30, End: 20}) {\n"
            "\t\tt.Fatal(\"reversed interval rejected\")\n"
            "\t}\n"
            "\tif sc.Add(Interval{Start: -50, End: -10}) {\n"
            "\t\tt.Fatal(\"negative interval accepted\")\n"
            "\t}\n"
            "\tif sc.Add(Interval{Start: refCap + 10, End: refCap + 20}) {\n"
            "\t\tt.Fatal(\"fully beyond-cap interval accepted\")\n"
            "\t}\n"
            "\tif !sc.Add(Interval{Start: refCap - 5, End: refCap + 10}) {\n"
            "\t\tt.Fatal(\"partially beyond-cap interval rejected\")\n"
            "\t}\n"
            "\tif got := sc.Slot(-1); got < 0 || got >= refBuckets {\n"
            "\t\tt.Fatalf(\"Slot(-1) out of range: %d\", got)\n"
            "\t}\n"
            "\tif got := sc.Slot(refCap * 10); got < 0 || got >= refBuckets {\n"
            "\t\tt.Fatalf(\"Slot(cap*10) out of range: %d\", got)\n"
            "\t}\n"
            "}\n"
        
    )

    body = "\n".join(refs + tests)
    imports.add(cfg.module_name())
    content = render_go_file("sched_test", imports, body, "Hidden black-box suite for the fabricated sched unit (generated).")
    content = content.replace("NewScheduler(", "sched.NewScheduler(").replace("Interval", "sched.Interval")
    return [("sched_bb_test.go", gofmt_text(content))]


def hidden_tests(domain: str, cfg: Config) -> list[tuple[str, str]]:
    if domain == "store":
        return store_hidden_tests(cfg)
    return sched_hidden_tests(cfg)


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
    else:
        rows.append(("scheduler capacity", str(c["cap"])))
    return rows


def render_contract(domain: str, cfg: Config, stats: UnitStats) -> str:
    c = cfg.consts
    rows = _fmt_constants(cfg)
    table = "\n".join(f"| {k} | `{v}` |" for k, v in rows)
    hidden = hidden_tests(domain, cfg)
    names = [re.findall(r"^func (Test\w+)", t, re.MULTILINE)[0] for _, t in hidden]

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
            "- Enumerating tags returns the distinct canonical tags of stored records, in "
            "ascending tag-hash order (hash base below; ties broken lexicographically), "
            "optionally filtered to tags beginning with a prefix.\n"
            "- Pruning removes every record whose size exceeds the given maximum and returns the "
            "number removed; counts, totals and balance update accordingly.\n"
        )
        coverage = [
            ("TestBucketOfProperty", "Bucket assignment equals the reference mixer formula: bucket(k) = rotl(stages(k), R) mod 8 for every key, including 0, 2^64-1 and adversarial values."),
            ("TestPutGetRoundtrip", "Storing a record makes it retrievable with its original key, stored size and canonical tag; absent keys report not-found; overwrites replace."),
            ("TestPutRejects", "A record with the reserved key, a tag that canonicalizes to empty, or a size above the store's limit is rejected; at-limit and zero-size records are accepted."),
            ("TestAggregates", "Count equals the number of distinct stored keys, total equals the sum of stored sizes (last write wins), and balance equals the maximum bucket occupancy under the reference formula."),
            ("TestTagsSort", "Tags() returns distinct canonical tags sorted by the tag hash (ties lexicographically); Tags(prefix) returns only tags beginning with the prefix."),
            ("TestPrune", "Prune(maxSize) removes exactly the records with size > maxSize, returns their count, and leaves count/total consistent."),
            ("TestAdversarialEdges", "Key 0 and key 2^64-2 are storable; the reserved key, empty tags and dash-only tags are rejected; a padded tag canonicalizes to the expected dashed form."),
        ]
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
            ("TestSlotProperty", "Band assignment equals the reference mixer formula for every point, including negative and beyond-capacity inputs (clamped)."),
            ("TestAddAtCoverage", "Adding follows the reference schedule model: same accept/reject verdicts, same interval count, same coverage verdicts for every point, same overlap verdicts."),
            ("TestTotalGap", "Total equals the reference covered span and Gap equals the reference largest uncovered span after identical operation sequences."),
            ("TestMergeCount", "Merge coalesces exactly the touching/overlapping runs the reference model coalesces and returns the same count."),
            ("TestAdversarialEdges", "Empty, reversed, negative and beyond-capacity intervals are handled per the contract; band queries on out-of-range points stay in range."),
        ]
    present_names = set(names)
    coverage_rows = "\n".join(f"| {name} | {sentence} |" for name, sentence in coverage if name in present_names)
    return (
        f"# Contract (L2) — {cfg.unit_name()}\n\n"
        f"{prose}\n"
        "## Configuration (this unit)\n\n"
        "| constant | value |\n"
        "|---|---|\n"
        f"{table}\n\n"
        "## Worked examples\n\n"
        f"- key `7` maps to bucket `{ref_bucket(7, [int(x) for x in c['mixMul'][:cfg.K]], [int(x) for x in c['mixAdd'][:cfg.K]], int(c['mixRot']))}`.\n"
        f"- key `42` maps to bucket `{ref_bucket(42, [int(x) for x in c['mixMul'][:cfg.K]], [int(x) for x in c['mixAdd'][:cfg.K]], int(c['mixRot']))}`.\n"
        "- tag `\"  Alpha beta \"` canonicalizes to `\"alpha-beta\"`; tag `\"!!!\"` canonicalizes to the empty string and is rejected.\n"
        + (f"- a record of size `{int(c['limit'])}` is accepted and one of size `{int(c['limit']) + 1}` is rejected.\n" if domain == "store" else f"- intervals are clamped into `[0, {int(c['cap'])})`.\n")
        + "\n## Coverage of hidden assertions\n\n"
        "| hidden test | contract sentence |\n"
        "|---|---|\n"
        f"{coverage_rows}\n"
    )


def render_bugreport(domain: str, cfg: Config) -> str:
    if domain == "store":
        body = (
            "The store is unusable: the first operation panics, and when the failure is masked "
            "records are not placed correctly — every key lands in the same bucket, tags keep "
            "their raw form instead of being normalized, and the size aggregates read zero.\n\n"
            "Expected: records can be inserted and retrieved, keys spread across all 8 buckets, "
            "tags are normalized (lowercase; whitespace and punctuation collapsed to single "
            "dashes), and size totals and counts are exact.\n"
            "Got: panic on first use; all records in one bucket; raw tags; zero totals and counts.\n"
        )
    else:
        body = (
            "The scheduler is unusable: the first operation panics, and when the failure is "
            "masked intervals are never scheduled (the schedule stays empty), every point is "
            "reported uncovered, and the coverage totals read zero.\n\n"
            "Expected: intervals can be added and are reported covered afterward, overlapping "
            "additions are rejected, and the covered span and largest gap are exact.\n"
            "Got: panic on first use; empty schedule; nothing covered; zero totals.\n"
        )
    return (
        f"# Bug report\n\n{body}\n"
        "Reproduce with:\n\n```\n./repro.sh\n```\n\n"
        "Run it from the unit root directory (the parent of `_author/`). "
        "The script copies the tree and the hidden tests into a scratch dir and runs them.\n\n"
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
        "Every excised body is a real, seeded computation; the constants and the shape of the "
        "mixer/canonicalization pipeline are not recoverable from call sites, and the hidden "
        "suite is property-based (seeded random inputs with a reference implementation), so a "
        "special-cased or guessed body fails. At L2 the full contract pins the behavior down, "
        "so the unit is expected to be solved there; at L0 the solver must reconstruct behavior "
        "from failing test output alone.\n"
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


def prove_unit(unit_dir: Path, cfg: Config, scratch_root: Path) -> dict[str, object]:
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
        _write_tree(tree, render_excised_files(domain, cfg))
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
        and result["gold_vet"]
        and result["cheat_fails"]
    )
    return result


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
) -> UnitResult:
    if domain not in DOMAINS:
        raise ValueError(f"unknown domain {domain!r}; choose from {sorted(DOMAINS)}")
    out = Path(out)
    cfg = search_config(domain, n_funcs, ratio, seed, target_depth=depth, target_fanout=fanout)
    stats = compute_stats(domain, cfg)
    module_files = _gofmt_tree(render_module_files(domain, cfg))
    excised_files = _gofmt_tree(render_excised_files(domain, cfg))
    cheat_files = _gofmt_tree(render_cheat_files(domain, cfg))
    pkg = DOMAINS[domain]["pkg"]
    rel_go = f"{pkg}.go"

    if out.exists():
        shutil.rmtree(out)
    out.mkdir(parents=True)
    _write_tree(out / "module", module_files)
    _write_tree(out / "excised_tree", excised_files)
    hidden = hidden_tests(domain, cfg)
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
    (author / "bugreport.md").write_text(render_bugreport(domain, cfg), encoding="utf-8")
    (author / "closure.md").write_text(render_closure(domain, cfg, stats), encoding="utf-8")
    (author / "contract.md").write_text(render_contract(domain, cfg, stats), encoding="utf-8")
    (author / "difficulty.md").write_text(render_difficulty(domain, cfg), encoding="utf-8")

    repro = (
        "#!/usr/bin/env bash\n"
        "set -uo pipefail\n"
        'cd "$(dirname "$0")"\n'
        'work="$(mktemp -d /tmp/fab.XXXXXX)"\n'
        'trap \'rm -rf "$work"\' EXIT\n'
        'cp -r excised_tree/. "$work"/\n'
        'cp hidden/*.go "$work"/\n'
        'cd "$work"\n'
        "go test -count=1 -timeout 5m ./...\n"
    )
    (out / "repro.sh").write_text(repro, encoding="utf-8")
    (out / "repro.sh").chmod(0o755)

    proof_result: dict[str, object] = {}
    if proof:
        scratch = scratch_root or (Path(os.environ.get("OUTPUTS_DIR", "outputs")) / "fabricated_proofs")
        proof_result = prove_unit(out, cfg, scratch)

    loc = module_files[rel_go].count("\n")
    sizes = {
        "module_go_loc": loc,
        "gold_patch_bytes": len(gold_patch.encode()),
        "excision_patch_bytes": len(excision_patch.encode()),
        "cheat_patch_bytes": len(cheat_patch.encode()),
        "hidden_bytes": sum(len(t.encode()) for _, t in hidden),
    }
    stats.loc = loc
    stats.sizes = sizes

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
        "files": sizes,
        "proof": proof_result,
        "generated_at": datetime.now(UTC).isoformat(timespec="seconds"),
        "command": (
            f"uv run python scripts/fabricate_repo.py --seed {seed} --n-funcs {n_funcs} "
            f"--ratio {ratio} --domain {domain} --out {out}"
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
    parser.add_argument("--no-proof", action="store_true")
    parser.add_argument("--log", type=Path, default=Path("outputs/closure_D.log"), help="append a JSONL row per unit")
    args = parser.parse_args(argv)

    result = generate_unit(
        seed=args.seed,
        n_funcs=args.n_funcs,
        ratio=args.ratio,
        out=args.out,
        domain=args.domain,
        depth=args.depth,
        fanout=args.fan_out,
        proof=not args.no_proof,
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
