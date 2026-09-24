# authored_au5kops — 20 new kops units

Job: author 20 NEW units in kops (`experiments/pipeline/authored_au5kops/kops/<unit>/_author/`),
no solver trials, no commits. Harness: `outputs/au5kops_work/{spec.json,driver.py,cheatgen.py,
verify_cheat.sh}` over a pristine obfuscated tree (`outputs/au5kops_work/pristine`, module
`example.internal/clustkit`), verified in docker image `ladder-base:kops`.

All 20 units: excised build green, kept tests green, gold restores green, cheat builds green,
cheat/gold added-line ratio < 0.6 (`scripts/ops/cheat_validity.py`), zero overlap vs the bank
(`scripts/check_unit_overlap.py`: 20 new vs 30 existing — CLEAN, no shared files at all).

## Rejected candidates (before/during authoring)

Overlap rejects — files the bank already owns, dropped at survey:

| candidate surface | collides with |
|---|---|
| `pkg/apis/kops/parse.go` (`ParseInstanceGroupRole` etc.) | `igrole` |
| `util/pkg/vfs/context.go` (VFSContext, path build) | `vfspaths` |
| `util/pkg/vfs/memfs.go` | `memfs` |
| `upup/pkg/fi/http.go` | `fidownload` |
| `upup/pkg/fi/values.go` | `fivalues` |
| `upup/pkg/fi/files.go` | `filemodes` |
| `pkg/model/iam/subject.go` | `iamsubj` |
| `pkg/jsonutils/streamwriter.go` | `jsonstream` |
| `util/pkg/reflectutils/{walk,field_path}.go` | `reflectfmt`, `fieldpath` |
| `channels/pkg/channels/*.go` | `addonparse` |

Suitability rejects (orchestration glue — clients, schedulers, config trees): cloudup
`apply_cluster`/`perftests` paths, `fi.Context`-dependent executors, cloud SDK call sites,
nodeup renderers with heavy wiring. Not individually logged; filtered before spec.

Quota reserves — verified clean but not authored (quota of 20 reached): `iampolicy`
(`pkg/model/iam/iam_builder.go`, 13 fns), `nodeidlinode` (3 fns), `nodeiddo` (3 fns). Unit
dirs removed; they remain in `spec.json` for a future batch.

One *internal* collision was designed around, not rejected: `igroles` needs
`ParseInstanceGroupRole` (bank `igrole` owns `parse.go`) — so igroles excises only
`instancegroup.go` and deletes `parse_test.go` (kept parser test transitively hits an excised
predicate). Documented in igroles' closure.md.

## Units in authoring order

| # | unit | closure | surface | rejects before accept | overlap | Inferable y/d/p/n | cheat |
|---|---|---|---|---|---|---|---|
| 1 | igroles | `pkg/apis/kops/instancegroup.go` role predicates (HasControlPlane, HasAPIServer, HasNode, HasBastion, IsMaster, ToLowerString, …) | predicate | 0 | none (designed around `igrole`'s parse.go) | 3/0/4/1 | 0.56 |
| 2 | channelver | `pkg/apis/kops/channel.go` — channel URL resolve, version recommend, image/package select | predicate+versioning | 0 | none | 6/0/2/1 | 0.27 |
| 3 | clusterpreds | `pkg/apis/kops/cluster.go` spec predicates/defaults | predicate | 0 | none | 3/0/9/1 | 0.26 |
| 4 | kapiutil | `pkg/apis/kops/util/*` — node-role lookup, taint parse, semver parse/compare | parse+predicate | 0 | none | 1/0/4/0 | 0.31 |
| 5 | subnetmath | `pkg/util/subnet/*` — CIDR overlap/contain/split/allocate | predicate+math | 0 | none | 4/0/2/0 | 0.41 |
| 6 | strorset | `pkg/util/stringorset/stringorset.go` — string-or-set JSON round-trip | serialization | 0 | none | 2/0/4/0 | 0.32 |
| 7 | hashparse | `util/pkg/hashing/hash.go` — `alg:hex` parse/format, digest stream | parse+serialization | 0 | none | 3/0/4/0 | 0.41 |
| 8 | fiutils | `upup/pkg/fi/utils/*` — slice equality, sha256 hex, IP/CIDR preds, sanitize, yaml | predicates+parse | 0 | none | 3/0/3/2 | 0.20 |
| 9 | firesources | `upup/pkg/fi/resources.go` — Resource impls, streamed byte-compare | serialization | 0 | none | 4/0/2/1 | 0.45 |
| 10 | fichanges | `upup/pkg/fi/changes.go` — BuildChanges reflect-compare | predicate | 0 | none | 4/0/3/0 | 0.11 |
| 11 | fideps | `upup/pkg/fi/topological_sort.go` — dependency inference walk | predicate | 0 | none | 5/0/2/0 | 0.27 |
| 12 | featflags | `pkg/featureflag/featureflag.go` — `+A,-B` parse, enabled precedence | parse | 0 | none | 4/1/1/1 | 0.56 |
| 13 | manifest | `pkg/kubemanifest/{manifest,visitor}.go` — multi-doc YAML load/save, field getters, visitor | parse+serialization | 0 | none | 3/0/4/1 | 0.31 |
| 14 | podmutate | `pkg/kubemanifest/{images,containerargs,volumes,critical,priority,selinux}.go` — pod mutators | predicate+mutation | 0 | none | 5/0/2/1 | 0.42 |
| 15 | gcenames | `upup/pkg/fi/cloudup/gce/utils.go` — name compose/truncate, error classify | predicate+naming | 0 | none | 2/1/2/3 | 0.43 |
| 16 | gceurl | `gce/{gce_url,labels}.go` — compute URL parse/build, `-XY` label escape | parse+serialization | 0 | none | 2/0/3/2 | 0.34 |
| 17 | awstags | `awsup/aws_utils.go` — tag lookup, smithy error unwrap, name truncation | predicate+serialization | 0 | none | 3/0/3/1 | 0.36 |
| 18 | vfsimpl | `vfs/{s3fs,gsfs,azureblob,k8sfs}.go` — Path/Join/Base/String accessors | serialization | 0 | none (vfspaths owns context.go only) | 4/0/3/2 | 0.45 |
| 19 | jsonxform | `pkg/jsonutils/transform.go` — in-place tree transform, SortSlice | serialization | 0 | none (jsonstream owns streamwriter.go) | 3/0/3/2 | 0.32 |
| 20 | nodelabels | `pkg/nodelabels/builder.go` — role-dispatched node label build | predicate | 0 | none | 2/0/4/2 | 0.18 |

"Rejects before accept" is 0 for every unit because the funnel rejected at survey: the spec's
23 candidates all passed mechanical verification (excised build + kept-tests + gold roundtrip
in docker) on the first full pass. Survey rejected ~10 overlap + ~8 suitability candidates
before spec entry.

## Cheat notes

Cheats are plausible-partial implementations (honor the obvious path, miss edge cases), not
lookup tables. Three first-draft cheats tripped the 0.6 implementation alarm and were
regenerated thinner: `strorset` (0.64→0.32: array-input path dropped, singleton-scalar marshal
dropped), `awstags` (0.62→0.36: first-element-only tag lookups, no ARN parse), `vfsimpl`
(0.63→0.45: no region resolution, no sentinel check). `igroles` and `featflags` sit at 0.56 —
under the floor but worth a second look in review since they're closest.

## Process notes (tooling, not candidate rejects)

- `excise_funcs.py` needed three fixes to handle this tree: generic-func declarations
  (`func F[T C](...)`), `interface{}` braces in signatures, and `//`/`{}` inside string
  literals & comments during brace-matching and import-usage checks. Fixed with a single-pass
  lexer-lite scanner; all 23 candidates then verified clean.
- `igroles` kept-test failure: `cluster_test.go` (`WarmPoolSpec.ResolveDefaults` →
  `HasControlPlane`) and `parse_test.go` (kept `ParseInstanceGroupRole` → excised
  `ToLowerString`) were added to the unit's del list.
- `cheatgen.py`'s "bodies swapped" counter under-reports when multiple bodies land in one file
  region — patches were verified by grep (`excised:` marker count) instead.
- gofmt in cheatgen expands one-line bodies — cheat size can't be gamed by whitespace; the
  ratio gate forces genuinely less code.

## Search-cost trend

Search cost per accepted unit did NOT rise. All 23 spec candidates passed mechanical
verification on the first complete pass (failures seen earlier were exciser bugs, not
bad candidates). Post-survey rejects per acceptance: 0. The kops pure-helper surface is not
exhausted — three verified reserves remain (iampolicy, nodeidlinode, nodeiddo) plus un-specced
leaves in `pkg/util`, `util/pkg/vfs`, and `upup/pkg/fi/cloudup/*` naming/tag helpers — but the
best parse/predicate files are now claimed; a next batch trends toward multi-file orchestration
or thinner helpers. Stopped at the 20-unit quota with cost flat, per instructions.
