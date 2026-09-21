# REPAIR2 — audit-driven contract repair on 27 double-failures, verified by shadow implementation

Date: 2026-09-20. Code: `src/openswe_traces/gate/repair2.py` (repair),
`src/openswe_traces/gate/shadow.py` (contract-only implementer gate),
`scripts/ops/repair2_shadow.py` (thin shadow CLI, resume-safe, `--results`
selects the output file, `--force` re-gates a recorded unit).
Log: `outputs/REPAIR2.log`. Staged units:
`experiments/dose_response/sweep_repair2/<name>-L2/` — `instruction.md`
edited in place, `environment/src` untouched by the repair pipeline.

Records: `outputs/repair2/gaps.jsonl` (audit), `outputs/repair2/results.jsonl`
(re-audit + lint + preflight; last row per unit is final),
`outputs/shadow_gate/results.jsonl` (staged shadow runs),
`outputs/shadow_gate/results_orig.jsonl` (original-unit shadow baselines).

## The job

For each audited double-failure unit (fails L0, fails L2, contract defect
found by `scripts/ops/contract_gap_read.py`): rewrite `instruction.md` so the
contract states every commitment the hidden suite grades — DERIVABLE and
COUNTER gaps only, never an ARBITRARY literal — then verify by shadow
implementation: an implementer sees the contract plus the package's
declaration surface only (no gold, no hidden tests, no function bodies),
writes the excised functions, and the hidden suite runs against the result.

Every breakage is classified **preexisting** or **repair-caused** by gating
the *original* unit under the same protocol: same failure on both sides is
preexisting; orig-pass/staged-fail is repair-caused.

## Results — all 27 units

| unit | audited gaps | residual | shadow (staged) | shadow (orig) | classification |
|---|---|---|---|---|---|
| client-go-memdbstaging-L2 | 6 | 6 | pass | — | converged; residual gaps don't block a strong implementer |
| client-go-onregionerror-L2 | 11 | 0 | pass | — | converged |
| client-go-replicaselector-L2 | 14 | 0 | pass | — | converged |
| commitobj-L2 | 2 | 1 | fail_build | fail_build | **preexisting** — both ARBITRARY, contract never edited; 48 excised funcs exceed implementer compile ability (`parentHashesIter`, `scratchEncodedObject` undefined) |
| confparse-L2 | 7 | 0 | pass † | pass | converged after artifact fix (see below) |
| cronparse-L2 | 9 | 0 | pass | — | converged |
| fsrefs-L2 | 3 | 0 | fail_build | fail_build | **preexisting** — missing imports (`plumbing`, `billy`) on both sides |
| gin-enginecfg-L2 | 5 | 6 | fail_build | fail_build | **preexisting** — `ginfs` undefined on both; re-audit found more DERIVABLE rows than round-1 closed |
| gin-mailfmt-L2 | 10 | 0 | pass | — | converged |
| gin-negotiate-L2 | 6 | 0 | pass | — | converged |
| helm-chartdl-L2 | 6 | 2 | fail_build | fail_build | **preexisting** — import collision + `fs.CopyFile` on both |
| helm-chartrepo-L2 | 2 | 0 | pass | — | converged |
| helm-depresolver-L2 | 10 | 8 | pass | n/a ‡ | converged |
| helm-httpgetter-L2 | 4 | 0 | fail_build | fail_build | **preexisting** — `go mod` fetch fails in the sandbox (`gold_fails` too); packaging defect, not contract |
| indexdec-L2 | 7 | 0 | pass | — | converged |
| kops-clustervalid-L2 | 9 | 0 | fail_build | fail_build | **preexisting** — `metav1` import missing on both |
| kops-difftext-L2 | 7 | 0 | pass | — | converged |
| kops-taintparse-L2 | 5 | 0 | pass † | fail_assert (Detail03) | **converged — repair improved it**: orig contract fails `EqualsRequiresColon`, repaired passes |
| ldapdn-L2 | 5 | 0 | pass † | pass | converged after artifact fix (see below) |
| merklediff-L2 | 3 | 0 | pass | — | converged |
| mvccread-L2 | 5 | 2 conflicts | pass | — | converged |
| packscan-L2 | 6 | 0 | fail_build | fail_build | **preexisting** — invented identifiers (`gogitsync`, `packutil`) on both; 24 excised funcs |
| refnames-L2 | 5 | 5 | fail_assert (Detail11) | fail_assert (Detail07+11) | **preexisting residual** — `ReferenceType.String` spellings unstated on both sides; repair did fix Detail07 |
| refspec-L2 | 7 | 0 | pass | — | converged |
| revparse-L2 | 6 | 5 | pass | — | converged |
| treeobj-L2 | 4 | 0 | pass † | — | converged (re-gated on verified-clean tree) |
| ulreq-L2 | 2 | 0 | pass | — | converged |

† re-gated after repair-artifact fixes / tree restore (below).
‡ orig depresolver's src tree was polluted mid-session (below) after its
clean preflight baseline was recorded; no meaningful orig-shadow run exists.

## Bottom line

**19 / 27 units converged** — the repaired contract alone is sufficient for
an honest implementer to pass the hidden suite.

**8 / 27 fail, all preexisting** — seven `fail_build` reproduce identically
on the original contract (missing imports, invented helper identifiers, or
sandbox module-fetch failure on httpgetter): the gap is implementer compile
ability on large excised surfaces (19–48 functions), not contract content.
`refnames` fails `TestDetail11_ReferenceTypeString` on both sides — a
genuinely unstated commitment the repair left open (5 residual DERIVABLE
gaps remain unaudited-closed there).

**0 net repair-caused shadow regressions.** Two transient repair-*artifact*
defects were found by the gate and corrected without touching commitments:

- `ldapdn`: the round-2 repair left `instruction.md` containing the contract
  **twice**, plus the repair agent's own working notes ("Wait — I changed
  the certificate conversion paragraph…", literal-budget reasoning)
  interleaved between the copies. The doubled 18 KB file broke shadow
  codegen (`undefined: enchex`). Fix: keep the agent's final clean copy
  (lines 162–277, which also carries the orig reproduce boilerplate).
  Re-gate: **pass** (2 repair passes).
- `confparse`: the repair systematically mangled `parses` → `parsing` in
  ~10 places ("fails to parsing", "JSON-like files parsing", "all parsing
  to true"). Two independent shadow samples failed
  `TestDetail12_Includes` (pedantic include token `SourceFile`), while the
  orig contract passed — the include bullet itself is verbatim unchanged,
  so the mechanism is degraded prose, not a removed commitment. Fix:
  grammar-only restoration (no commitment touched). Re-gate: **pass**,
  first try.

**Repair helped where it should**: `kops-taintparse` orig contract fails
`TestDetail03_EqualsRequiresColon` under shadow; the repaired contract
passes. `refnames` fixed `TestDetail07_Short` (orig fails it, staged
doesn't) even though `Detail11` stays open.

## Cheat axis (preflight discrimination) — secondary

Preflight cheat=1 means a fresh gold-sighted implementation passes the
hidden suite — the contract leaks enough for a cheater. Comparing orig
preflight to staged:

- **repair-caused `cheat_passes` (orig `ok` → staged `cheat_passes`)**:
  `helm-depresolver`, `mvccread`, `kops-taintparse` — the repairs added
  enough detail to guide a cheat. For taintparse there is a caveat: its
  staged tree was polluted by a wandering generation agent (below), and
  the pollution window overlaps the preflight window; depresolver and
  mvccread staged trees are verified clean, so their cheat_passes stands.
- **preexisting `cheat_passes`**: `confparse` (later repaired to `ok`),
  `cronparse`, `memdbstaging`, `merklediff`, `gin-negotiate`,
  `replicaselector`, `treeobj`.
- **preexisting `gold_fails`**: `helm-httpgetter` (packaging).

## Contamination incident — `cursor-agent` escapes

`cursor-agent -p` is a tool-using agent, not a completion API (the
REPAIRLOOP writeup documented this; this worktree's `composer.py` still
invokes it with `--trust` and no TreeGuard). During tonight's generation
calls agents wrote into experiment source trees:

- `sweep_repair2/kops-taintparse-L2`: `ParseTaint` + `GetNodeRole`
  implemented in-place, stray hidden-test file copied into src. The staged
  gate errored on "no excised stubs". Restored from the clean orig, re-gated
  → pass.
- `sweep_repair2/treeobj-L2`: `tree.go`/`treenoder.go` implemented
  post-gate. The recorded pass was already sound (`apply_shadow` rebuilds
  excised files from the stub baseline captured at context-build), and the
  re-gate on the restored tree confirms it: pass.
- `sweep_L2/helm-depresolver-L2` (**orig**): `resolver.go` implemented and
  a hidden test copied in at 19:33 — *after* its clean 17:39 preflight
  baseline, so the cheat-axis classification stands, but no orig-shadow
  baseline is possible. Staged depresolver is clean and passed.
- `sweep_repair2/ldapdn-L2` contract corruption (above) is the same class
  of agent-escape damage, one level up — in the repair artifact itself.

After restores, a full audit shows every staged `environment/src` is
byte-identical to its orig except depresolver (whose orig is the polluted
side). **Mitigation for any rerun**: hash `environment/src` around every
composer call (TreeGuard), or run `cursor-agent` in a scratch cwd —
`--mode ask`/`--sandbox` as in REPAIRLOOP.

## Caveats

- Shadow gating is a single-sample stochastic check; confparse showed a
  reproducible failure (2/2 identical) while ldapdn needed repair passes.
  Marginal pass/fail calls carry sampling noise.
- Generation cache (`outputs/shadow_gate/gen/`) keys on prompt text —
  orig and staged prompts differ, so baselines are independent samples.
- `commitobj` was never edited (both audited gaps are ARBITRARY); its
  fail_build is the cohort's least-ambiguous preexisting result.
- Units with residual audit gaps can still converge (memdbstaging 6,
  depresolver 8, revparse 5, mvccread 2, refnames 5 → four of the five
  pass shadow anyway): the audit is conservative; the shadow gate is the
  ground truth this job measures.

## Reproduce

```bash
# staged shadow gate (resume-safe, append-only results)
uv run python scripts/ops/repair2_shadow.py experiments/dose_response/sweep_repair2/<unit>
# orig baseline for classification
uv run python scripts/ops/repair2_shadow.py --results outputs/shadow_gate/results_orig.jsonl \
    experiments/dose_response/<orig_sweep>/<unit>
```
