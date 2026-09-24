# Reconciled contracts — coverage rows derived FROM hidden tests

2026-09-20. Proven on go-github, the cohort that flipped 0/10 at L2.

## Why

Author wrote `contract.md` from a reading of gold. Verifier wrote hidden tests from
gold. Nothing reconciled the two. 40% of the first dataset misdescribed gold; 7/7
audited double-failures were contract defects; go-github burned ten L0 screens
plus thirty L2 trials on units that could not flip.

An inverted coverage row flipped 7 of 8 passing units onto exactly that property.
A wrong row does not confuse — it manufactures a wrong implementation.

## The three-pass flow

```
author     → DETAILS.md (numbered commitments) + bugreport.md (L0, B6/B7-clean)
verifier   → one property test per commitment (TestDetailNN)
reconciler → contract.md: prose invariants + one coverage row per hidden test,
             each row derived from (hidden assertion, gold hunk)
```

The reconciler may read gold and the hidden tests. It must not touch
`bugreport.md` and must not weaken any hidden test. B7 still applies to L2:
no symbol, file, or line names. Worked examples come from literals the tests use.

Every row originates in an assertion, so **"missing" is structurally impossible**
(one row per `Test*` in `tests/hidden/`). "False" is still possible if the LLM
mistranslates an assertion; A13 remains the backstop.

## Command

One unit:

```
uv run python scripts/reconcile_contract.py unit \
  --unit refescape \
  --details path/to/_author/DETAILS.md \
  --hidden path/to/tests/hidden \
  --gold path/to/_author/gold.patch \
  --out path/to/_author/contract.md
```

A batch (resume-safe; cache `outputs/reconcile/<sha256>.json`; ≤300 OpenRouter
free-tier requests):

```
uv run python scripts/reconcile_contract.py batch \
  --author-root /path/authored_batch2/<repo> \
  --hidden-root /path/tasks_batch2/<repo> \
  --out-dir outputs/reconcile/<repo> \
  --stage /path/dose_response/sweep_<repo>_rc
```

Packaging runs the same pass automatically when `DETAILS.md` is present
(`pipeline.package._reconciled_contract`). Cold repos (< 3 units already in
`experiments/pipeline/tasks_composerver/<repo>`) then fail closed on A13
(`enforce_cold_a13`).

## go-github — 14 units

Hidden tests from `oswt-VFgogithub/.../tasks_batch2/go-github/<unit>-L2/tests/hidden/`.
Gold from `oswt-AUgogithub/.../authored_batch2/go-github/<unit>/_author/gold.patch`.
Staged at `experiments/dose_response/sweep_gogithub_rc/` (14 L0 + 14 L2).
Gold patches and hidden tests were not modified. L0 `bugreport.md` / instruction
unchanged. L2 instruction is the new contract plus the original bug-report tail.

LLM: nvidia nemotron-3-super/ultra free tier, 9 unique requests after cache.
Preflight: 28/28 pass (bare fail with assertions, gold pass, cheat fail), cached
from the existing sweep trees.

### Per unit

| unit | hidden tests | old rows | new rows | encoding-shape | A13 old | A13 new | preflight |
|---|---:|---:|---:|---|---|---|---|
| auditentry | 6 | 5 | 6 | no | ok | ok | pass L0+L2 |
| copilotpoly | 8 | 4 | 8 | no | ok | ok | pass L0+L2 |
| customprop | 4 | 5 | 4 | JSON null/absent | ok | ok | pass L0+L2 |
| envreview | 7 | 7 | 7 | JSON key presence | **2 false** | ok | pass L0+L2 |
| eventdispatch | 6 | 15 | 6 | no | **2 false** | 2 missing† | pass L0+L2 |
| ndjsonmetrics | 4 | 5 | 4 | no | ok | ok | pass L0+L2 |
| netconfig | 6 | 1 | 6 | no | ok‡ | 1 false† | pass L0+L2 |
| projectsjson | 11 | 4 | 11 | JSON union | **1 missing** | ok | pass L0+L2 |
| pubkeyjson | 5 | 8 | 5 | JSON null/absent | ok | ok | pass L0+L2 |
| refescape | 4 | 4 | 4 | no | ok | ok | pass L0+L2 |
| rulesetjson | 8 | 11 | 8 | **field order** | **2 missing** | 1 false (sentence patched) | pass L0+L2 |
| teamsupdate | 3 | 4 | 3 | **omitempty / null-vs-absent** | ok | ok | pass L0+L2 |
| treeentryjson | 4 | 6 | 4 | **omitempty / null-vs-absent** | **1 missing** | 1 false (sentence patched) | pass L0+L2 |
| webhooksig | 10 | 6 | 10 | no | ok | ok | pass L0+L2 |
| **defective units** | | | | | **5 / 14** | **2 residual judge flags / 14** | **28/28** |

† Judge false alarms (the documented ~13/30 rate). `eventdispatch` "missing" claims
malformed-JSON errors the tests do not assert. `netconfig` "false" denies a
`validation failed:` prefix that `TestDetail05` itself asserts and that gold
passes in preflight.

‡ `netconfig` old contract had **1 coverage row for 6 hidden tests**. A13 did not
flag missing. That is why row-count alignment, not the judge, is the structural
guarantee.

Two new-side `false` rows (rulesetjson marshal of a parameterless element;
treeentryjson "vice versa") were LLM mistranslations of the test. Both sentences
were rewritten from the assertion before staging the final L2 instruction.

### Unreconcilable?

None. JSON field-order / omitempty / null-vs-absent **can** be stated as wire
behaviour (`"sha":null` vs key absent; `{"type":"creation"}` with no parameters
key; parent identifiers omitted vs encoded as JSON null). Those units are flagged
`encoding_shape` because prose describes that behaviour *badly for a solver*,
not because a row cannot be derived. That is the domain-vs-cold-start finding:
the cohort may still fail L2 after this repair, and that result would then be
about JSON, not about a missing coverage table.

Units flagged encoding-shape: `rulesetjson`, `teamsupdate`, `treeentryjson`.
Also JSON-shaped (null/absent/union) but still derived: `customprop`, `envreview`,
`projectsjson`, `pubkeyjson`.

## A13 as a cold-repo gate

Any repo with fewer than 3 units already in `tasks_composerver/<repo>` must pass
A13 on each newly packaged L2 before screening. Accept the judge's false-alarm
rate there: the prior on defects is worse (this cohort: 5/14 A13-defective old
contracts, 0/10 L2 flips, and several old contracts whose row count did not
match the hidden suite at all). Wired in `pipeline.package.package_levels` via
`openswe_traces.synth.reconcile.enforce_cold_a13`. Documented in
`analytics/research/verifier_rules.md` (A13, C6, C7).
