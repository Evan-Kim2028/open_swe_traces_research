# Climb L3→L6 on the four L2 non-flippers — packaging + preflight

2026-09-20. Units: `helm/depresolver`, `helm/ignorerules`, `helm/repindex`, `kops/clustervalid`
(0/3 at L0 and 0/3 at L2, `sweep_L0_2026-09-19.md`). This session packaged and gated only;
Harbor trials are run by the parent session. Driver: `uv run python scripts/climb_l3l6.py`
(body in `src/openswe_traces/synth/climb_l3l6.py`); log `outputs/CLIMB.log`; machine-readable
state `outputs/climb_l3l6_state.json`, `outputs/climb_l3l6_summary.json`.

## Construction

`build_affordance_levels(levels=(1,2,3,4), name_scheme="L")` → Harbor L3/L4/L5/L6, staged at
`open_swe_traces_research/experiments/dose_response/sweep_climb/<family>-L{3,4,5,6}`.
Source for every level is the preflight-PASS L2 dir in
`oswt-closureK/experiments/pipeline/tasks_composerver/<repo>/<unit>-L2`; `tests/test.sh`,
`tests/gold.patch`, `tests/cheat.patch` are copied byte-identical (`cmp` verified). `task.toml`
is the Cursor allowlist shape copied verbatim from `sweep_L0/helm-chartloader-L0/task.toml`
(`[agent] network_mode="allowlist", allowed_hosts=["cursor.com","*.cursor.com","*.cursor.sh","downloads.cursor.com"]`,
verifier `no-network`, 1800 s). Level deltas confirmed on disk:

- L3 = L2 tree + `## Hidden unit tests (names only)` appendix (5 names + one-liner).
- L4 = L3 instruction (byte-identical) + `environment/src/<pkg>/l4_exported_api.go` sidecar
  (package doc + excised signatures; e.g. depresolver lists `Resolve`, `HashReq`, `HashV2Req`,
  `GetLocalPath`). L3 tree ≡ L2 tree (`diff -r`, modulo pre-existing dangling testdata symlinks).
- L5 = L4 + the representative hidden file restored into `environment/src` (byte-identical to
  the `tests/hidden` copy).
- L6 = L4 + all hidden files in src.

**L5 ≡ L6 degeneracy.** Each of the four units has exactly ONE hidden test file, so
"restore one" and "restore all" produce byte-identical trees and tests (L6 preflight reused
the L5 in-image result on a `(tree_sha256, tests_sha256)` key match, flagged
`reused_from_identical_tree_and_tests` in `validation.json`). On this bank the "needed to see
one assertion" and "needed the full spec" readings are indistinguishable: a flip at L5 is the
strongest signal the ladder can emit here.

## Hidden-test checksum equality proof

sha256 of every file under `tests/hidden/`, recomputed on disk 2026-09-20:

| unit | hidden file | L0 | L2 | L3 | L4 | L5 | L6 |
|---|---|---|---|---|---|---|---|
| helm/depresolver | `internal/resolver/depresolver_bb_prop_test.go` | `42ce4856` | `42ce4856` | `42ce4856` | `42ce4856` | `42ce4856` | `42ce4856` |
| helm/ignorerules | `pkg/ignore/ignorerules_bb_prop_test.go` | `80de8d6d` | `80de8d6d` | `80de8d6d` | `80de8d6d` | `80de8d6d` | `80de8d6d` |
| helm/repindex | `pkg/repo/v1/repindex_bb_prop_test.go` | `26681d8c` | `26681d8c` | `26681d8c` | `26681d8c` | `26681d8c` | `26681d8c` |
| kops/clustervalid | `pkg/validation/clustervalid_bb_prop_test.go` | `2f8c6310` | `2f8c6310` | `2f8c6310` | `2f8c6310` | `2f8c6310` | `2f8c6310` |

(Full 64-hex digests in `outputs/climb_l3l6_state.json`.) Equal within each unit across all
six levels — the climb is comparable.

## Preflight (in-image gate, `openswe_traces.pipeline.preflight` on branch closure-K)

Gate semantics: bare (excised tree) must FAIL with assertions, gold.patch must PASS,
cheat.patch must FAIL. All 16 dirs **PASS**; nothing excluded.

| dir | bare | gold | cheat | evidence (bare / cheat) | image | s |
|---|---|---|---|---|---|---|
| helm-depresolver-L3 | fail | pass | fail | `panic: excised: Resolver.Resolve` / `panic: excised: HashReq` | `openswe-audit:…a597a` | 50 |
| helm-depresolver-L4 | fail | pass | fail | same | `…a02` | 48 |
| helm-depresolver-L5 | fail | pass | fail | same | `…ec2412` | 49 |
| helm-depresolver-L6 | fail | pass | fail | same (cached: identical tree+tests) | `…ec2412` | — |
| helm-ignorerules-L3 | fail | pass | fail | `panic: excised: Parse` / same | `…e458a` | 18 |
| helm-ignorerules-L4 | fail | pass | fail | same | `…a02` | 16 |
| helm-ignorerules-L5 | fail | pass | fail | same | `…a9557` | 18 |
| helm-ignorerules-L6 | fail | pass | fail | same (cached) | `…a9557` | — |
| helm-repindex-L3 | fail | pass | fail | `panic: excised: IndexFile.MustAdd` / same | `…981c` | 53 |
| helm-repindex-L4 | fail | pass | fail | same | `…b53e` | 70 |
| helm-repindex-L5 | fail | pass | fail | same | `…cabd7` | 56 |
| helm-repindex-L6 | fail | pass | fail | same (cached) | `…cabd7` | — |
| kops-clustervalid-L3 | fail | pass | fail | `panic: excised: NewClusterValidator` / `panic: nil pointer` | `…fe22` | 99 |
| kops-clustervalid-L4 | fail | pass | fail | same | `…356e` | 96 |
| kops-clustervalid-L5 | fail | pass | fail | same | `…a2b85` | 94 |
| kops-clustervalid-L6 | fail | pass | fail | same (cached) | `…a2b85` | — |

## Contract-row analysis (solver-free, step 5)

Each `...ContractTable` property was re-read against the contract row it maps to
(`_author/contract.md` vs `gold.patch` ground truth). **All four units are packaging
defects, not capability limits** — in three cases the contract *contradicts* gold, in the
fourth the assertion has no row at all.

### helm/depresolver — row does not pin the asserted literal (defect)

`TestDRContractTable`/`TestDRHashReqStable` assert `lock.Digest == HashReq(req, lock.Dependencies)`
and the **exact literal** `sha256:fb239e8363…aaaf` for
`HashReq([{alpine 0.1.0 http://localhost:8879/charts}], same)`.
Row `TestHashReq` → "lock hash stability and difference"; prose: "a sha256 over the
canonicalized (sorted) requirement set". Gold is `json.Marshal([2][]*chart.Dependency{req,
lock})` + `provenance.Digest` — it hashes the **(req, lock) pair in declaration order**;
nothing is sorted. The row pins only equality/difference, not the encoding, and "sorted"
affirmatively misdescribes the mechanism. The literal is reachable only from memory of
upstream helm or from the test file itself (an oracle to iterate against).

### helm/ignorerules — contract contradicts ≥3 asserted rows (defect)

`TestIgnoreContractTableProperty` truth table vs prose and gold `pkg/ignore/rules.go`:

| asserted row | contract | gold |
|---|---|---|
| `*` on `"."` → false; `.*` on `"."` → false | silent | `Ignore` returns false for `"."`/`"./"` before matching |
| `!helm.txt`: `helm.txt`→false, `tiller.txt`→**true** | "There is no negation: a `!` is a literal character" | `!` sets `p.negate`; a non-match under a negated rule returns **true** |
| `**`, `**/deep.txt`, `foo/**/bar` → parse error | "`*` does not cross `/` while `**` does" (implies supported) | `strings.Contains(rule,"**")` → error |

`TestIgnoreParseAndDefaultsProperty` adds a fourth: `AddDefaults` must ignore
`templates/.dotfile` (only rule is `templates/.?*`); the contract lists "version-control
dirs, OS cruft, editor backups, .proj/.idea/.vscode, ownership files" — a different set that
does not cover the asserted path.

### helm/repindex — contract contradicts 2 asserted rows (defect)

`TestRIContractTable` vs prose "a filename that is already an absolute URL is used as-is" and
"an existing entry for the same name/version is replaced, not duplicated":

| asserted row | contract | gold `MustAdd` |
|---|---|---|
| `MustAdd(…,"http://cdn.com/a-1.0.0.tgz","http://h",…)` → URL `http://h/a-1.0.0.tgz` | absolute URL "used as-is" | `filepath.Split` → basename → `urlutil.URLJoin(base, file)` whenever base non-empty |
| same name+version added twice → `len(Entries["dup"]) == 2` | "replaced, not duplicated" | unconditional `append` |

### kops/clustervalid — assertion has no coverage row (defect)

`TestClusterValidConstructorProperty` asserts `NewClusterValidator(…, &InstanceGroupList{},
…)` errors (`"no InstanceGroup objects found"`) and succeeds for n≥1. The coverage table has
**no constructor row** (all rows are `Test_Validate*`) and no prose sentence states the
empty-list guard. This is the "capability-or-unfairness candidate" from the L0→L2 note; the
re-read says unfairness — the information exists nowhere at L2.

## Pre-registered flip predictions (before any L3–L6 trial)

Reasoning common to all four: the missing/contradicted information lives only inside the
hidden test file, so neither L3 (names) nor L4 (signatures) can repair it; first rung that
can is L5 (≡ L6 here). If a unit still fails at L5/L6 the residual is capability — the model
saw the assertion and still could not execute or trust it over the contract.

| unit | predicted flip | what L5 discloses that L2 lacks | author `predicted_flip` |
|---|---|---|---|
| helm/depresolver | **L5** | the literal `sha256:fb239e…` oracle; encoding must be iterated to match | L4 |
| helm/ignorerules | **L5** | truth-table rows for `.`/`!`/`**` and `templates/.dotfile` default | L2 (falsified) |
| helm/repindex | **L5** | absolute-URL→basename-join and dup-append assertions | L4 |
| kops/clustervalid | **L5** | the empty-IG-list constructor error | L4 |

Read: all four flip at L5 → the L0–L2 "hard residue" was entirely packaging defects
(contract wording), and the fix is authoring-side (pin the literal/mechanism or drop the
assertion), not solver-side. Any L3/L4 flip would instead mean name/signature affordances
matter — predicted only if the model already half-knows upstream helm/kops and just needs
the cue.

## Reproduce

```
uv run python scripts/climb_l3l6.py            # package + gate all four units
uv run python scripts/climb_l3l6.py --skip-preflight   # repackage only
```

Output lands in `sweep_climb/` (16 dirs), `outputs/CLIMB.log`,
`outputs/climb_l3l6_{state,summary}.json`. Per-dir `validation.json` carries the rule
verdicts and the preflight record.
