# Verifier authoring — kOps batch-2 (VFkops)

Hidden black-box suites for the 14 authored units under
`/home/evan/Documents/oswt-AUkops/experiments/pipeline/authored_batch2/kops/*/`.
Worktree: `oswt-VFkops`. Driver: `src/openswe_traces/synth/verifier_batch2.py`.
Suites: `src/openswe_traces/synth/testdata/batch2_kops/<unit>_bb_prop_test.go`.

Rules enforced: B4 (only API listed in `api.md`; in-package test files only where
`api.md` names unexported helpers reachable from in-package tests), B5 (seeded-random
property tests via `HIDDEN_SEED`, default 20260919 — verified passing under both
20260919 and 20260920), B1 (sha256 checksum guard in `tests/test.sh`), A3 (cheat patch
fails real assertions), A10 (`[verifier] network_mode = "no-network"`).

Overlap gate: `scripts/check_unit_overlap.py --extra /home/evan/Documents/oswt-AUkops`
=> CLEAN, 14 new vs 10 existing, 0 overlaps. No units excluded.

## Results

| unit | details | tests | L0 bare/gold/cheat | L2 bare/gold/cheat | sec |
|---|---|---|---|---|---|
| cidrsubnet | 6 | 6 | fail/pass/fail | fail/pass/fail | 72 |
| difftext | 11 | 11 | fail/pass/fail | fail/pass/fail | 73 |
| distroident | 9 | 9 | fail/pass/fail | fail/pass/fail | 87 |
| featureflags | 7 | 7 | fail/pass/fail | fail/pass/fail | 69 |
| fiutils | 7 | 7 | fail/pass/fail | fail/pass/fail | 68 |
| fspath | 13 | 13 | fail/pass/fail | fail/pass/fail | 359 |
| hashparse | 7 | 7 | fail/pass/fail | fail/pass/fail | 75 |
| hostsguard | 12 | 12 | fail/pass/fail | fail/pass/fail | 83 |
| jsontransform | 11 | 11 | fail/pass/fail | fail/pass/fail | 75 |
| kubeversion | 8 | 8 | fail/pass/fail | fail/pass/fail | 94 |
| stringorset | 8 | 8 | fail/pass/fail | fail/pass/fail | 74 |
| subnet | 9 | 9 | fail/pass/fail | fail/pass/fail | 75 |
| systemdunit | 7 | 7 | fail/pass/fail | fail/pass/fail | 74 |
| taintparse | 6 | 6 | fail/pass/fail | fail/pass/fail | 88 |

Total: 121 property tests across 14 units, 28 packaged task dirs,
preflight wall time 22.7 min. **28/28 PASS — zero exclusions.**

## Per-unit notes

- **cidrsubnet**: DETAILS#6 claims `newSize<oldSize` errors; observed upstream behaviour
  tolerates shrink (delegates to go-cidr `SubnetBig`). Suite asserts the
  extend/number/bounds contract and only shape-checks tolerated shrink; divergence
  noted, gold-green.
- **difftext**: Detail03 verified: the insert/delete order flip is keyed on the edit
  reaching end-of-input bytes (no trailing newline), not doc position — tested both
  ways. Detail04 elision: context 2, any skipped run -> one `...`. Identical
  non-empty input -> `...\n`; identical empty -> `""` (tested as observed).
- **distroident**: Exact-key and prefix tables covered incl. literal trailing-dot
  rules and the `-` joined-key error format.
- **featureflags**: In-package suite (api.md lists `new` as in-package reachable).
  Covers syntax, trim/skip, unknown-flag ignore, precedence, re-registration, Get
  error text, persistence.
- **fiutils**: Detail02 dup-multiplicity corner randomized
  (`["a","a","b"] == ["a","b","b"]` membership-not-multiset). Sanitize
  length/substitution + last-200 truncation verified.
- **fspath**: Covers ErrExist/non-ENOTDIR propagation, create serialization,
  temp+rename residue, ENOENT normalize asymmetry (ReadFile yes, Remove raw),
  ReadDir/ReadTree dir inclusion asymmetry, RemoveAll skeleton, SHA256 preferred.
- **hashparse**: Length-before-hex error ordering, unknown-algo/prefix fall-through,
  HashFile raw-vs-wrapped error, algorithm-sensitive Equal.
- **hostsguard**: In-package suite (HostMap.records + pseudoAtomicWrite are in-package
  reachable). Marker lines verified as consumed into the managed set (mutator
  receives them); stray END markers dropped; duplicate blocks collapse; deterministic
  grouped render; mtime-skip on identical.
- **jsontransform**: Verified: slice callback path is `<path>[]` (both callback and
  elements); non-float64 Go int is an unhandled type; SortSlice order is raw
  JSON-marshal text lexicographic.
- **kubeversion**: Tolerant parse vs URL fallback vs strict ParseVersion asymmetry;
  reversed GTE comparison; current-only Pre/Build strip; panic-on-bad-candidate.
- **stringorset**: Force-array vs cardinality marshal, dedup, `[]` empty,
  malformed-array swallow vs malformed-string error asymmetry, sorted Value, Equal
  ignoring encoding flag.
- **subnet**: Overlap nil-safety + base-containment semantics; BelongsTo
  family/ones/masked-base; SplitInto shape + IPv4-only error; Allocate
  skip-base/overlap-membership/exhaustion error; IPv6 allocate.
- **systemdunit**: Allowed-punctuation passthrough, space-only quoting, `\xNN`
  fallback, Set append/dup-keep, raw content before entries, single-blank
  separators, extension suffix list.
- **taintparse**: Three-key map shape, colon-count error, `=`-needs-colon asymmetry,
  role precedence master>control-plane>node>api-server>legacy-value; api-server role
  renders `apiserver` (verified).

## Cheat-patch fix

`taintparse/_author/cheat.patch` imported `"strings"` but never used it — the cheat
failed to BUILD (infra-classed, not an assertion failure). Restored the blank import
(`_ "strings"`) so the cheat compiles and fails on assertions, per A3.

## Artifacts

- Tasks: `experiments/pipeline/tasks_batch2/kops/<unit>-L{0,2}` (28 dirs)
- Preflight JSON: `outputs/vfkops_preflight.json`; stderr table appended to
  `outputs/VFkops.log`
- Driver: `src/openswe_traces/synth/verifier_batch2.py`
