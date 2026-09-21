# Fabricated-repo generator with a controllable closure ratio

2026-09-19, revised after the L0 fairness audit. Prototype, zero solver cost:
units are generated and *proved locally* (go build/test/vet, no docker, no LLM
calls) so pass rate at instruction level L0 can be measured as a function of
closure ratio on repos the solver has never seen, with hundreds of cheap
trials.

Code: `src/openswe_traces/synth/fabricate.py` (+ thin CLI
`scripts/fabricate_repo.py`, tests `tests/test_fabricate.py`). L0 fairness
self-audit: `src/openswe_traces/synth/fab_fairness.py` (+ CLI
`scripts/fab_fairness_check.py`). Per-unit JSONL log:
`outputs/closure_L.log`. Pilot units for re-audit are listed in
`experiments/dose_response/fab_pilot_units.txt`.

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
multipliers/addends, rotate, hash base, store limit / scheduler capacity,
reserved key) are drawn from a splitmix64 RNG seeded by `--seed`, so each unit
is unique. **The constants are part of the specification**: they all live in a
`params.go` file that ships in every tree (module, excised, cheat) and is never
touched by the excision/gold/cheat patches. The excised set S contains the
*logic* (mixer pipeline, canonicaliser, prune, merge) — the knob is the call
graph structure of S, not secrecy of the numbers. The full module ships with an
in-tree smoke suite of *worked-example* assertions (exact expected values per
behavior; still weak on purpose — B3: examples are not the verifier).

## L0 fairness (post-audit design)

The first batch was class-(c) unfair at L0: the hidden suite asserted exact
seeded constants and reserved-key behaviour that appeared only in the hidden
oracle and gold — unreachable from the excised tree, bugreport or repro
output. Withholding a constant is a feasibility gate, not difficulty. Two
mechanisms repair this, both implemented:

1. **Constants are never excised** — `params.go` (above).
2. **Repro output is informative** — `repro.sh` copies `excised_tree/` into a
   scratch dir and runs the in-tree suite with `go test -v`. The smoke tests
   assert the exact worked examples that `bugreport.md` states in prose (one
   exemplar per hidden-checked behavior), so expected-vs-got output is
   available on any partially-repaired tree, and the examples are readable in
   the test source regardless.

`contract.md` additionally carries the full constants table, the tag-fold
recurrence (`h = h*B + c`, uint64 wraparound), worked examples, and a coverage
table mapping **every** hidden test to a contract sentence.

`scripts/fab_fairness_check.py` re-checks all of this per unit: it greps the
hidden suite for literals absent from the L0 corpus (excised tree + bugreport +
repro output, the last subsumed by the smoke source), verifies `gold.patch`
introduces no constants absent from L0, verifies the coverage table maps every
hidden test, ties every worked-example value in the bugreport back to a
smoke-suite assertion, and re-runs the B6/B7 mechanical checks. It fails the
old pre-fix units on exactly the Grok-audit findings (missing mixer constants,
1-of-6 coverage).

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

`_author/`: `api.md`, `bugreport.md` (L0: symptom, expected vs got, worked
examples, `./repro.sh`, no symbol names, no-web clause), `contract.md` (L2:
prose + per-unit constant table + worked examples + coverage table mapping
every hidden test to a contract sentence), `closure.md`, `difficulty.md`
(`predicted_flip: L2`), `gold.patch` (excised → full), `cheat.patch`
(special-cases the worked examples, wrong defaults otherwise), and
`excised/excision.patch` (full → excised). The unit also keeps `module/` (full),
`excised_tree/` (panic stubs, decls kept, imports pruned so it still compiles,
`params.go` untouched), `hidden/` (black-box suite), `repro.sh`,
`manifest.json`.

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

## Achieved vs requested (post-fix pilot, N=24, small |S| band 8–10, domain store)

| unit | requested | achieved | internal | in | out | \|S\| | depth | fan-out | proof | self-audit |
|---|---|---:|---:|---:|---:|---:|---:|---:|---|---|
| `store-s11-r015` | 0.15 | **0.190** | 4 | 11 | 10 | 10 | 4 | 4 | OK | PASS |
| `store-s23-r020` | 0.20 | **0.200** | 4 | 10 | 10 | 10 | 4 | 4 | OK | PASS |
| `store-s37-r020` | 0.20 | **0.200** | 4 | 10 | 10 | 10 | 4 | 4 | OK | PASS |
| `store-s42-r250` | 2.50 | **3.000** | 9 | 1 | 2 | 10 | 6 | 4 | OK | PASS |
| `store-s53-r250` | 2.50 | **3.000** | 9 | 1 | 2 | 10 | 6 | 4 | OK | PASS |
| `store-s67-r300` | 3.00 | **3.000** | 9 | 1 | 2 | 10 | 6 | 4 | OK | PASS |

All six prove: module build/test/vet green, buggy fails at runtime, gold
passes (incl. `HIDDEN_SEED=20260920`), cheat compiles and fails. All six pass
`scripts/fab_fairness_check.py`. Note the achievable-ratio set in the small
band is discrete: requests near 0.15 land at ~0.19, and requests in
[2.0, 2.5] land at either 1.6 or 3.0 (integer edge counts); the sweep should
always fit on *achieved* ratio.

## Batch command (dose-response run)

One unit per invocation (resume-safe: rerunning an existing `--out` overwrites
it; the JSONL log appends):

```bash
cd ~/Documents/oswt-closureI
for r in 0.1 0.2 0.4 0.6 0.8 1.0 1.5 2.0 3.0; do
  seed=$((1000 + RANDOM % 9000))
  uv run python scripts/fabricate_repo.py \
    --seed "$seed" --n-funcs 24 --ratio "$r" --domain store \
    --s-min 8 --s-max 10 --log outputs/closure_L.log \
    --out "experiments/dose_response/fab/store-s${seed}-r$(printf '%03d' "$(python3 -c "print(int(round($r*100)))")")"
done
```

Each unit writes `manifest.json` with `ratio_requested`, `ratio_achieved`, the
edge counts, the S list and the proof verdicts, and appends one JSONL row to
`outputs/closure_L.log`; a run summary can be read back with
`uv run python -c "import json,sys;[print(json.loads(l)['ratio_requested'], json.loads(l)['ratio_achieved']) for l in open('outputs/closure_L.log')]"`.
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
- **Discrete achievable ratios**: internal/boundary are integer edge counts, so
  the achievable set is sparse near the extremes of a band (small band:
  ~0.19 at the low end, then 0.2; 1.6 then 3.0 at the high end). Requested
  ratios between achievable points snap to the nearest candidate — always fit
  on the achieved column.
- **Repro output is informative only after the stub panic is repaired**: the
  excised stubs panic, so a bare `repro.sh` run reports the panic; the
  expected-vs-got lines print once `NewStore`/`NewScheduler` is repaired but
  the logic is still wrong. The worked examples are also stated verbatim in
  `bugreport.md` and readable in the smoke-suite source.
- **Ruff**: `uv run ruff check .` on the whole repo still reports pre-existing
  violations in files this work did not touch (confirmed present at HEAD, e.g.
  `src/openswe_traces/data.py`, `experiments/ablation_graph/*.py`). The
  deliverable files (`fabricate.py`, `fab_fairness.py`,
  `fabricate_repo.py`, `fab_fairness_check.py`, `test_fabricate.py`) are
  ruff-clean.
