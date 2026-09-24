# Fabricated-repo generator with a controllable closure ratio

2026-09-19. Prototype, zero solver cost: units are generated and *proved locally*
(go build/test/vet, no docker, no LLM calls) so pass rate at instruction level
L0 can be measured as a function of closure ratio on repos the solver has never
seen, with hundreds of cheap trials.

Code: `src/openswe_traces/synth/fabricate.py` (+ thin CLI
`scripts/fabricate_repo.py`, tests `tests/test_fabricate.py`). Per-unit JSONL
log: `outputs/closure_D.log`. Demo units:
`experiments/fabricated/store-s7-r020`, `-r080`, `-r200`.

## Definition (as given)

For the excised set S of functions in a Go module, counting each (caller, callee)
pair once (distinct call edges, not call sites):

- `internal` = calls among S
- `boundary_in` = calls from outside S into S
- `boundary_out` = calls from S to outside
- `ratio = internal / max(1, boundary_in + boundary_out)`

## Design

A unit is a single-package Go module `example.internal/<unit>` over one of two
small typed domains:

- **store** — a record store: 9 exported entry points (`NewStore`, `Put`,
  `Get`, `BucketOf`, `Total`, `Count`, `Balance`, `Tags`, `Prune`) over a
  seeded mixer (K affine stages + final rotate, bucket = mix(k) mod 8), a
  tag-canonicalization pipeline (trim → lowercase → collapse runs of
  non-alphanumerics to `-`), tag-hash-ordered enumeration, and size pruning.
- **sched** — an interval scheduler: 9 exported entry points (`NewScheduler`,
  `Slot`, `Add`, `Overlaps`, `At`, `Len`, `Total`, `Gap`, `Merge`) over the
  same seeded mixer (band = mix(x) mod 8), interval normalization (swap →
  clamp), sorted insertion, and merge.

Every function has real, deterministic semantics; seeded constants (stage
multipliers/addends, rotate, hash bases, store limit / scheduler capacity) are
drawn from a splitmix64 RNG seeded by `--seed`, so each unit is unique and an
excised body is **not recoverable from call sites**. The full module ships with
an in-tree smoke suite (weak on purpose — B3: examples are not the verifier).

### What controls the ratio

The graph is built from a fixed per-domain *edge universe* (entry→helper calls,
helper chains, consumer/reporting leaves calling entries). A parameter grid
decides the concrete module:

| knob | values | effect |
|---|---|---|
| `E` | 4..9 | how many exported entry points exist (full API at 9) |
| `K` | 1..5 | mixer-pipeline depth (internal chain edges) |
| `canon_mode` | simple / pipe | canonicalizer as one function vs a 3-function pipeline |
| `canon_in_S`, `misc_in_S`, `place_in_S` | bool | which helper groups are excised (S membership) |
| consumers | subset of 6/8 reporting leaves | boundary_in edges (outside → S) |
| fill | 0..8 | plain utility functions to reach exactly N (no edges) |

`search_config` enumerates the grid (all consumer subsets included), computes
the *measured* ratio for each candidate, and picks the config closest to the
target (soft penalties for `--depth`/`--fan-out` when given; the fuller API
wins ties). The achieved ratio is always measured on the generated graph, never
assumed. Achievable set at N=24: **store 159 distinct ratios in [0.18, 20]**,
**sched 154 in [0.045, 18]** — a dense coverage for dose-response runs.

### Excise + artifacts (same `_author/` layout as the pipeline)

`_author/`: `api.md`, `bugreport.md` (L0: symptom, expected vs got,
`./repro.sh`, no symbol names, no-web clause), `contract.md` (L2: prose +
per-unit constant table + worked examples + coverage table mapping every hidden
test to a contract sentence), `closure.md`, `difficulty.md`
(`predicted_flip: L2`), `gold.patch` (excised → full), `cheat.patch`
(special-cases the worked examples, wrong defaults otherwise), and
`excised/excision.patch` (full → excised). The unit also keeps `module/` (full),
`excised_tree/` (panic stubs, decls kept, imports pruned so it still compiles),
`hidden/` (black-box suite), `repro.sh`, `manifest.json`.

Hidden suite (B4/B5): external test package (`store_test`/`sched_test`) calling
only the exported API of S, with a reference implementation of the observable
semantics inside the test (the F2-property-test pattern) and seeded-random
inputs from the canonical seed contract (`HIDDEN_SEED` env, else `20260919`);
gold passes under both the default and the audit seed (`20260920`).

### Local proof (A1/A3, no docker)

`prove_unit` runs on scratch copies of the excised tree with the hidden suite
mounted: `go build`/`go test`/`go vet`/`gofmt -l` green on the full module,
`go test` **fails** bare (runtime panic), **passes** with `gold.patch`, and
**fails** with `cheat.patch` (cheat compiles and fails assertions, not by
breaking the build). A12 holds: patches touch only the package source, never
test files.

## Achieved vs requested (demo batch, N=24, seed 7, domain store)

| unit | requested | achieved | internal | in | out | \|S\| | n | depth | fan-out | proof |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| `store-s7-r020` | 0.2 | **0.200** | 4 | 9 | 11 | 11 | 24 | 4 | 4 | OK |
| `store-s7-r080` | 0.8 | **0.800** | 8 | 2 | 8 | 14 | 24 | 4 | 4 | OK |
| `store-s7-r200` | 2.0 | **2.000** | 14 | 6 | 1 | 17 | 24 | 3 | 4 | OK |

All three prove: module build/test/vet green, buggy fails at runtime, gold
passes (incl. `HIDDEN_SEED=20260920`), cheat compiles and fails. The hidden
suite is identical in shape across the three units (same exported API), so the
units differ mainly in the graph property under study. Note `|S|` and the
in/out split co-vary with the ratio (low closure ⇒ thin, boundary-heavy S);
report them per unit (manifest) and stratify if that matters.

## Batch command (dose-response run)

One unit per invocation (resume-safe: rerunning an existing `--out` overwrites
it; the JSONL log appends):

```bash
cd ~/Documents/oswt-closureD
for r in 0.1 0.2 0.4 0.6 0.8 1.0 1.5 2.0 3.0; do
  seed=$((1000 + RANDOM % 9000))
  uv run python scripts/fabricate_repo.py \
    --seed "$seed" --n-funcs 24 --ratio "$r" --domain store \
    --out "experiments/fabricated/store-s${seed}-r$(printf '%03d' "$(python3 -c "print(int(round($r*100)))")")"
done
```

Each unit writes `manifest.json` with `ratio_requested`, `ratio_achieved`, the
edge counts, the S list and the proof verdicts, and appends one JSONL row to
`outputs/closure_D.log`; a run summary can be read back with
`uv run python -c "import json,sys;[print(json.loads(l)['ratio_requested'], json.loads(l)['ratio_achieved']) for l in open('outputs/closure_D.log')]"`.
Sweep the *achieved* ratio (never the requested) when fitting the dose-response
curve. `--no-proof` skips the go runs for bulk generation; re-prove any unit by
regenerating it into the same `--out`.

## Limitations

- **Sizes co-vary**: `|S|`, depth, fan-out and the in/out split are not held
  constant across ratios (all listed in `manifest.json`). Stratification or
  matching is needed before attributing a pass-rate difference to the ratio
  alone; the ratio itself is exact to 3 decimals.
- **Single package, no real repo context**: no cross-package edges, no build
  system, no legacy code — a fabricated unit is a laboratory condition, not a
  proxy for a real repo's difficulty.
- **A2 not checked**: an alternative correct fix (structurally different from
  gold) is not proven per unit; the reference implementations are one valid
  family of solutions, and the contract is behavioral, so alternatives should
  pass, but this is unverified.
- **No docker / A8**: proofs run on the host toolchain, not in a built image;
  verifier-rule A8 (prove in the built image) does not apply to the prototype.
- **Equality properties, not statistical ones**: hidden tests compare module
  output to the reference formula exactly (seeded, deterministic); they do not
  assert distributional properties, so they cannot be fooled by "a hash that
  spreads well" — the exact seeded constants are required.
- **Edge counting**: distinct (caller, callee) pairs, not call sites; a
  function calling the same callee twice contributes one edge.
- **Ruff**: `uv run ruff check .` on the whole repo still reports pre-existing
  violations in files this work did not touch (confirmed present at HEAD, e.g.
  `src/openswe_traces/data.py`, `experiments/ablation_graph/*.py`). The
  deliverable files (`fabricate.py`, `fabricate_repo.py`,
  `test_fabricate.py`) are ruff-clean.
