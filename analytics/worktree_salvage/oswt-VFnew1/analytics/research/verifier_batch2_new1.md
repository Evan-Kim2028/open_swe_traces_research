# verifier_batch2 — go-git, 25 hidden black-box suites

Date: 2026-09-20. Repo `go-git` (obfuscated module `example.internal/gitkit/v6`,
upstream commit `0f3a0a2`). Verifier-author role only: suites were written
against `api.md` + `DETAILS.md` + the excised tree. `bugreport.md`,
`contract.md` and `gold.patch` were never read; packaging copies them
mechanically.

## Method

- One `TestDetailNN_*` per numbered `DETAILS.md` line (1:1, verified: 308
  details = 308 tests). Every suite is an external `package <pkg>_test`
  driving only the exported API documented in `api.md` — no
  `blackbox_hygiene` exemptions needed; B4 passes cleanly on all 25.
- Seeded-random inputs throughout: `rand.New(rand.NewSource(hiddenSeed()))`,
  `HIDDEN_SEED` env override, default 20260919. Suites verified host-side
  under seeds {1, 42, 20260919, 20260920, 7301989}; the gate's audit seed is
  7301989 and is exercised as the second in-image run per suite.
- Each unit run host-side against three trees (`scripts/vf_gogit_verify.sh`):
  excised (must panic/assert — never setup errors), gold (must pass), cheat
  (must fail). Full sweep repeated after every fix; 25/25 clean.
- Packaging: `scripts/vf_gogit_package.py` builds the skeleton
  (`environment/src` = excised tree, CURSOR-allowlist `task.toml`,
  `tests/hidden/`, `gold.patch`/`cheat.patch` in `tests/` + `patches/`) then
  `build_affordance_levels(levels=(-2, 0), name_scheme="L")` → `-L0` (bug
  report) and `-L2` (contract) under
  `experiments/pipeline/tasks_batch2/go-git/`. `tests/test.sh` runs
  `go test -v -count=1 -run '^(TestDetailNN…)$' ./<pkg>/…` with sha256
  checksum guards on every hidden file — the `-v` makes each `--- PASS`
  visible in the verifier log.
- Preflight: consolidated gate `src/openswe_traces/gate/` (job O,
  `scripts/gate_tasks.py`) — bare×2 (seeds 20260919 + 7301989) must FAIL,
  gold×2 must PASS, cheat×1 must FAIL; A10 replays gold under
  `--network=none`. All inside the task image built `FROM ladder-base:go-git`
  (rebuilt locally; `RUN rm -rf /app` before `COPY src/` kills stale-tree
  overlay, same fix as gin).

## Fixes along the way

Hidden-suite corrections made while probing gold behaviour (assertions now
match observable contract, not guesses): packscan declared-size is an upper
bound and WriteObject errors translate to `ErrReferenceDeltaNotFound`;
idxdecode `PackfileChecksum` init via `plumbing.NewHash(hex)`; gitattrs
root-vs-subdir macro policy (`ErrMacroNotAllowed` + accumulated patterns);
ignorepattern unclosed `[` → NoMatch; ignorescope global config resolves via
`$HOME/.gitconfig` and mid-walk errors come from `ReadDir`; revparse colon
edge forms (`:7:`→stage-0 path, `:{`→ColonPath, NUL→second-token error);
commitobj multi-signature trigger is repeated `gpgsig` headers. `indexdec`
Detail11 (REUC) asserted a fixed stage→hash map; gold's assignment is
map-iteration order, so the test now pins the key set + hash multiset
(order-free) plus exact single-stage mapping — re-verified across seeds
{7, 42, 12345, 20260919, 20260920} ×3.

Two B4 false positives fixed in the suites: a comment `structural encode (`
and a local variable named `matcher` (`matcher.Match(`) collided with
unexported excised symbol names (`Commit.encode`, `matcher.Match`) in the
B4 lexical scan — reworded/renamed, semantics unchanged.

Packaging/static fixes: L2 instruction now carries the full L0 bug report
plus the package paths under test (monotone info across levels — nesting
check 0 failures); `validation.json["coverage"]` declares every
`TestDetailNN` row (L0 coverage 25/25 pass).

## Per-unit results

(details = DETAILS.md lines; tests = hidden Test functions; bare/gold/cheat
from in-image `validation.json.gate_executed`; s = summed seconds of the
5 in-image runs on the L2 dir)

| unit | details | tests | bare | gold | cheat | s |
|---|---|---|---|---|---|---|
| advrefs | 15 | 15 | fail | pass | fail | 51 |
| cfgdecode | 10 | 10 | fail | pass | fail | 12 |
| cfgencode | 7 | 7 | fail | pass | fail | 14 |
| cfgsection | 11 | 11 | fail | pass | fail | 11 |
| cfgurl | 8 | 8 | fail | pass | fail | 11 |
| commitobj | 16 | 16 | fail | pass | fail | 14 |
| endpoints | 8 | 8 | fail | pass | fail | 12 |
| fsrefs | 13 | 13 | fail | pass | fail | 15 |
| gitattrs | 14 | 14 | fail | pass | fail | 10 |
| idxdecode | 10 | 10 | fail | pass | fail | 13 |
| ignorepattern | 12 | 12 | fail | pass | fail | 12 |
| ignorescope | 12 | 12 | fail | pass | fail | 11 |
| indexdec | 14 | 14 | fail | pass | fail | 12 |
| indexops | 10 | 10 | fail | pass | fail | 10 |
| merklediff | 15 | 15 | fail | pass | fail | 15 |
| objectid | 11 | 11 | fail | pass | fail | 43 |
| packdelta | 15 | 15 | fail | pass | fail | 14 |
| packscan | 17 | 17 | fail | pass | fail | 13 |
| pktline | 14 | 14 | fail | pass | fail | 11 |
| refnames | 12 | 12 | fail | pass | fail | 38 |
| refspec | 10 | 10 | fail | pass | fail | 10 |
| revparse | 14 | 14 | fail | pass | fail | 9 |
| sideband | 9 | 9 | fail | pass | fail | 11 |
| treeobj | 16 | 16 | fail | pass | fail | 14 |
| ulreq | 15 | 15 | fail | pass | fail | 14 |

Totals: 25 units, 308 details, 308 tests. Bare fails everywhere by panic
(`panic: excised: <symbol>`) or assertion — never setup/build errors. Gold
passes under both seeds in-image and five seeds host-side. Cheat fails
everywhere (e.g. `--- FAIL: TestDetail04_ZeroIdMarker`). Determinism (A5)
and no-network gold (A10) clean on all 50 dirs.

Excluded units: none. Overlap check `check_unit_overlap.py --extra
oswt-AUnew1` ran before authoring: CLEAN, no source-file collisions with
existing or sibling units.

## Advisory static-check findings (job-O gate additions)

Not preflight verdicts — flagged for the record, all in authored artifacts:

- **coverage (23 × -L2 dirs):** the authored `contract.md` coverage table
  cites upstream test names (`TestAll`, `TestDecode`, …) which the check
  counts as declared rows with no hidden counterpart. Our own
  `TestDetailNN` rows are declared via `validation.json["coverage"]`; all
  25 -L0 dirs pass.
- **B7 excised_leaks (26 dirs, mostly -L2):** `contract.md`/`bugreport.md`
  name excised exported API symbols (`Decode`, `Encode`, `Head`, …) — the
  contract documents the API surface by name, which the lexical leak scan
  counts against it. One -L0 (`commitobj`) mentions unexported `indent`.
  Both are author-side wording; fixing them is contract re-authoring, not
  packaging.

## Artifacts

- Hidden suites: `experiments/pipeline/work/vf_hidden_gogit/<unit>/…`
  (external `_test` packages, mirrored to package relpaths).
- Packaged tasks: `experiments/pipeline/tasks_batch2/go-git/<unit>-L{0,2}`
  (50 dirs), each with `tests/test.sh` (`-v`, checksum-guarded),
  `tests/hidden/`, `gold.patch`, `cheat.patch`, `task.toml` (CURSOR
  allowlist, verifier `no-network`), `environment/` (excised tree +
  Dockerfile over `ladder-base:go-git`).
- Gate matrix: `outputs/gate_gogit.json`; run log with full per-detail
  `--- PASS` vector: `outputs/VFgo-git.log`.
- Scripts: `scripts/vf_gogit_package.py`, `scripts/vf_gogit_verify.sh`.
- `src/openswe_traces/synth/affordance.py`: `render_hidden_test_sh` now
  emits `go test -v` so `--- PASS` lines land in the verifier log.

No commits. No solver trials.
