# Closure metrics vs measured flip point — retrospective

2026-09-19. Zero-solver-cost analysis of the authored unit bank: does the size of a unit's
*closure* (how much of the removed code is determined only by itself) predict its measured
flip point? Hypothesis: units that flip high have a higher internal/boundary edge ratio than
units that pass at L0.

## Method

- Parse Go with `go/ast` via `tools/goclosure/main.go` (no regex; no `go/types` — the trees are
  obfuscated and have no module cache). Removed functions = decls whose body the excision patch
  deletes or stubs (base-tree body hash differs from post-patch, or the decl is gone).
- `internal_edges`: call references among removed funcs (from removed bodies in the base tree).
- `boundary_in`: call sites in the remaining (post-excision) tree that reference a removed func.
- `boundary_out`: call references from removed bodies to symbols that remain.
- `ratio = internal_edges / max(1, boundary_in + boundary_out)`.
- Method calls resolved by light local type inference (params, receivers, `:=` literals, `new`,
  type assertions, one level of struct fields, import aliases); unresolved receivers fall back to
  name-uniqueness (every func with that name removed ⇒ internal, else external).
- helm/kops excision patches were authored against `pipeline_repos` identity-pass trees
  (`example.internal/chartkit/v4`, `example.internal/clustkit`) that are not on disk. The base
  trees below are the `prepare_repo` trees at the same pinned commits (`example.internal/helm`,
  `example.internal/kops`); each patch is rebranded back by ordered word-boundary rewrites
  (trees.json `patch_rewrites`) before `git apply` — all 20 apply cleanly. Edge counts are
  invariant under a consistent module/brand rename.
- Measured flips from `experiments/pipeline/state.db` (trials) + the main checkout's `HANDOFF.md`
  ("Results so far", 2026-09-19) + `results.md`; per solver, lowest level with a pass;
  `none` = tried with no pass.

## Trees used

| repo | tree | provenance |
|---|---|---|
| client-go | `experiments/pipeline/closure_A/trees/client-go` | authored base tree |
| gin | `experiments/pipeline/repos2/gin/src` | authored base tree |
| go-github | base `experiments/pipeline/closure_A/trees/go-github/base` / excised `experiments/pipeline/closure_A/trees/go-github/excised` | dry-run sandbox; excision derived by diffing the two trees |
| helm | `/home/evan/Documents/open_swe_traces_research/experiments/pipeline/work/helm/tree` | `prepare_repo` tree (main checkout, read-only); authored patches rebranded (example.internal/chartkit/v4→example.internal/helm, example.internal/chartkit→example.internal/helm, …) |
| kops | `/home/evan/Documents/open_swe_traces_research/experiments/pipeline/work/kops/tree` | `prepare_repo` tree (main checkout, read-only); authored patches rebranded (example.internal/clustkit→example.internal/kops, ClusterKit→Kubernetes, …) |

`experiments/pipeline/closure_A/trees.json` points at each tree. Repos with no tree on disk
(goa, nats-server) are skipped — see the skipped table. client-go's tree is a proxy: same
upstream commit and obfuscation scheme as the authored base (`example.internal/kvstore/v2` vs
the base's `example.internal/clientgo`); the excision patches apply cleanly to it for 6/10
units, and the remaining 4 are skipped (patch context drifts off the proxy tree). helm/kops
use the `prepare_repo` trees with rebranded patches (see Method).

## Units and metrics

| repo | unit | family | predicted_flip | control | flip_devin | flip_cursor | n_files | n_funcs_removed | lines_removed | internal_edges | boundary_in | boundary_out | ratio |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| client-go | connarray | single-file |  |  | 2 |  | 1 | 7 | 163 | 2 | 4 | 71 | 0.0267 |
| client-go | doactionbatches | single-file |  |  | none |  | 1 | 5 | 253 | 8 | 9 | 145 | 0.0519 |
| client-go | memdbstaging | single-file |  |  | 0 |  | 1 | 12 | 90 | 0 | 0 | 42 | 0 |
| client-go | onregionerror | single-file |  |  |  |  | 1 | 5 | 595 | 2 | 10 | 343 | 0.0057 |
| client-go | pdoracle | single-file |  |  |  |  | 1 | 10 | 87 | 0 | 1 | 42 | 0 |
| client-go | replicaselector | single-file |  |  |  |  | 1 | 12 | 251 | 2 | 74 | 79 | 0.0131 |
| gin | bindingdispatch | single-file | 2 | True |  | 0 | 1 | 2 | 28 | 0 | 29 | 1 | 0 |
| gin | bodydecoders | cross-file | 3 |  |  |  | 8 | 30 | 125 | 14 | 8 | 40 | 0.2917 |
| gin | defaultengine | single-file | 3 |  |  |  | 1 | 25 | 25 | 0 | 0 | 50 | 0 |
| gin | formmapping | single-file | 5 |  |  | 5 | 1 | 23 | 430 | 45 | 92 | 118 | 0.2143 |
| gin | htmlrender | single-file | 2 |  |  |  | 1 | 5 | 35 | 1 | 9 | 22 | 0.0323 |
| gin | jsonrenders | single-file | 3 |  |  |  | 1 | 13 | 94 | 1 | 0 | 46 | 0.0217 |
| gin | multipartfiles | single-file | 3 |  |  |  | 1 | 3 | 40 | 4 | 0 | 17 | 0.2353 |
| gin | requestbinders | cross-file | 3 |  |  |  | 4 | 14 | 49 | 1 | 0 | 21 | 0.0476 |
| gin | streamrenders | cross-file | 2 |  |  |  | 4 | 10 | 44 | 1 | 0 | 25 | 0.04 |
| gin | validator | single-file | 3 |  |  |  | 1 | 5 | 56 | 6 | 5 | 27 | 0.1875 |
| go-github | redirect-until-found | cross-file | 5 | True |  |  | 2 | 5 | 111 | 5 | 21 | 29 | 0.1 |
| helm | chartloader | cross-file | 3 |  |  |  | 3 | 11 | 291 | 10 | 111 | 93 | 0.049 |
| helm | coalesce | single-file | 4 |  |  | 5 | 1 | 10 | 223 | 14 | 49 | 47 | 0.1458 |
| helm | depresolver | single-file | 4 |  |  | none | 1 | 4 | 199 | 3 | 8 | 50 | 0.0517 |
| helm | ignorerules | single-file | 2 | True |  |  | 1 | 6 | 152 | 2 | 21 | 38 | 0.0339 |
| helm | kindsorter | cross-file | 2 |  |  |  | 3 | 12 | 185 | 9 | 49 | 40 | 0.1011 |
| helm | memorydriver | cross-file | 3 |  |  |  | 2 | 11 | 189 | 5 | 27 | 49 | 0.0658 |
| helm | provenance | single-file | 3 |  |  |  | 1 | 10 | 224 | 4 | 37 | 45 | 0.0488 |
| helm | repindex | single-file | 4 |  |  |  | 1 | 10 | 192 | 5 | 23 | 54 | 0.0649 |
| helm | storage | single-file | 3 |  |  |  | 1 | 12 | 168 | 5 | 29 | 97 | 0.0397 |
| helm | strvalsparser | cross-file | 4 |  |  |  | 2 | 28 | 455 | 53 | 16 | 149 | 0.3212 |
| kops | addonparse | cross-file | 3 |  |  |  | 3 | 8 | 159 | 7 | 19 | 35 | 0.1296 |
| kops | assetsremap | single-file | 4 |  |  | 0 | 1 | 11 | 292 | 5 | 73 | 84 | 0.0318 |
| kops | clustervalid | cross-file | 4 |  |  | 0 | 2 | 9 | 397 | 9 | 5 | 81 | 0.1047 |
| kops | flagbuilder | single-file | 3 |  |  |  | 1 | 5 | 192 | 2 | 15 | 72 | 0.023 |
| kops | issuecert | cross-file | 3 |  |  |  | 4 | 5 | 200 | 2 | 46 | 47 | 0.0215 |
| kops | memfs | single-file | 3 |  |  |  | 1 | 11 | 84 | 0 | 6 | 26 | 0 |
| kops | oidcdisc | cross-file | 4 |  |  |  | 3 | 12 | 274 | 3 | 3 | 95 | 0.0306 |
| kops | osmetadata | single-file | 3 |  |  |  | 1 | 6 | 101 | 5 | 10 | 29 | 0.1282 |
| kops | templater | cross-file | 2 | True |  |  | 2 | 5 | 107 | 2 | 2 | 43 | 0.0444 |
| kops | tomlwriter | single-file | 2 |  |  |  | 1 | 9 | 97 | 10 | 61 | 27 | 0.1136 |
| client-go | connarray-cv | single-file |  |  | 0 |  | 1 | 7 | 163 | 2 | 4 | 71 | 0.0267 |

`flip_devin`/`flip_cursor`: lowest tried level with a pass (`none` = tried, no pass; blank = no
trials yet). `connarray` and `connarray-cv` are the *same authored unit* packaged twice
(results.md lists both); their metric rows are identical. The go-github row is **flagged**: it
has no flip yet (authored in the dry run; `predicted_flip: L5`, control). `fresh_laptop_setup`
describes it as 6 functions; the synced sandbox shows 5 stubbed (bareDoUntilFound,
roundTripWithOptionalFollowRedirect, checkRedirectHost, getArchiveLinkWithoutRateLimit,
getArchiveLinkWithRateLimit) — metrics reflect the sandbox.

## Spearman vs measured flip

n = 10 rows / 9 distinct authored units: client-go/connarray-cv, client-go/memdbstaging, gin/bindingdispatch, kops/assetsremap, kops/clustervalid, client-go/connarray, client-go/doactionbatches, helm/depresolver, gin/formmapping, helm/coalesce.
Encoded flip: L0=0, L2=2, L5=5, `none` = max tried + 1 (depresolver, doactionbatches → 3).
Caveats: (a) `connarray`/`connarray-cv` are the same authored unit — one duplicate row;
(b) `flip` mixes solvers (devin for client-go, cursor elsewhere); (c) helm/kops metrics
come from rebranded patches on `prepare_repo` trees, not the authored trees themselves.

| metric | rho | p |
|---|---:|---:|
| n_files | -0.312 | 0.381 |
| n_funcs_removed | 0.118 | 0.746 |
| lines_removed | 0.405 | 0.246 |
| internal_edges | 0.662 | 0.037 |
| boundary_in | 0.477 | 0.164 |
| boundary_out | 0.261 | 0.466 |
| ratio | 0.720 | 0.019 |

## Spearman vs measured flip — L0/L5 extremes only

n = 7; units: client-go/connarray-cv, client-go/memdbstaging, gin/bindingdispatch, kops/assetsremap, kops/clustervalid, gin/formmapping, helm/coalesce. Restricting to flip ∈ {L0, L5} asks the sharper
question: does closure size separate "solved immediately" from "hard until L5"?

| metric | rho | p |
|---|---:|---:|
| n_files | -0.258 | 0.576 |
| n_funcs_removed | 0.474 | 0.282 |
| lines_removed | 0.474 | 0.282 |
| internal_edges | 0.798 | 0.032 |
| boundary_in | 0.632 | 0.127 |
| boundary_out | 0.316 | 0.490 |
| ratio | 0.798 | 0.032 |

### Raw rows: L5 units vs L0 units

| repo | unit | _solver | _flip | n_files | n_funcs_removed | lines_removed | internal_edges | boundary_in | boundary_out | ratio |
|---|---|---|---|---|---|---|---|---|---|---|
| gin | formmapping | cursor | 5 | 1 | 23 | 430 | 45 | 92 | 118 | 0.2143 |
| helm | coalesce | cursor | 5 | 1 | 10 | 223 | 14 | 49 | 47 | 0.1458 |
| client-go | connarray-cv | devin | 0 | 1 | 7 | 163 | 2 | 4 | 71 | 0.0267 |
| client-go | memdbstaging | devin | 0 | 1 | 12 | 90 | 0 | 0 | 42 | 0 |
| gin | bindingdispatch | cursor | 0 | 1 | 2 | 28 | 0 | 29 | 1 | 0 |
| kops | assetsremap | cursor | 0 | 1 | 11 | 292 | 5 | 73 | 84 | 0.0318 |
| kops | clustervalid | cursor | 0 | 2 | 9 | 397 | 9 | 5 | 81 | 0.1047 |

## Supplementary: Spearman vs predicted flip

All computed units that carry a `predicted_flip` (gin, helm and kops batches + go-github
control). Mostly unmeasured; included for coverage. The go-github unit (`redirect-until-found`,
predicted L5, control) is **flagged**: authored in the dry run, only the verifier sandbox
synced; metadata from `fresh_laptop_setup_2026-09-19.md`.

n = 31 (gin 10, go-github 1, helm 10, kops 10)

| metric | rho | p |
|---|---:|---:|
| n_files | -0.054 | 0.773 |
| n_funcs_removed | 0.235 | 0.203 |
| lines_removed | 0.625 | 0.000 |
| internal_edges | 0.414 | 0.021 |
| boundary_in | 0.116 | 0.534 |
| boundary_out | 0.599 | 0.000 |
| ratio | 0.294 | 0.109 |

## Skipped (no base tree on disk, or patch does not apply)

| repo | unit | reason |
|---|---|---|
| client-go | lockresolver | patch does not apply: command failed (1): git apply --whitespace=nowarn /tmp/tmpte0chjv_.patch |
| client-go | pessimisticlock | patch does not apply: command failed (1): git apply --whitespace=nowarn /tmp/tmpncv0yzyd.patch |
| client-go | rangetask | patch does not apply: command failed (1): git apply --whitespace=nowarn /tmp/tmpc2arzlyw.patch |
| client-go | regionstoresorted | patch does not apply: command failed (1): git apply --whitespace=nowarn /tmp/tmpmhxutwhs.patch |
| goa | errloc | no base tree on disk (measured flip: cursor 0) |
| goa | evalctx | no base tree on disk |
| goa | evalrun | no base tree on disk (measured flip: cursor 2) |
| goa | grpcerr | no base tree on disk |
| goa | grpctrace | no base tree on disk |
| goa | grpcxray | no base tree on disk |
| goa | httpxray | no base tree on disk |
| goa | jsonrpcwire | no base tree on disk |
| goa | pkgvalidation | no base tree on disk |
| goa | xrayseg | no base tree on disk |
| nats-server | seqset | no authored dir / base tree on this machine (results.md lists it; seqset: predicted L4 control; subjecttree: rejected A3) |
| nats-server | subjecttree | no authored dir / base tree on this machine (results.md lists it; seqset: predicted L4 control; subjecttree: rejected A3) |

## Flip source notes

- `analytics/research/task_space_framework.md` reports A-ladder (A0–A4) single-attempt flips on
  older client-go units (dynamic pipeline A3, codec A1, backoff A0, mutations A4). Those are a
  different ladder and pre-date the 3-attempt policy (C6, see the framework's replication note), so
  they are not merged into the measured-flip Spearman above.
- `analytics/research/fresh_laptop_setup_2026-09-19.md`: gin/bindingdispatch = L0 (Composer 2.5),
  consistent with state.db.
- Main-checkout `HANDOFF.md` ("Results so far"): Composer — goa/errloc, kops/assetsremap,
  kops/clustervalid pass L0; gin/formmapping and helm/coalesce flip L2→L5; goa/evalrun passes
  L2; helm/depresolver 0/1 at L2 (flip `none`). Devin — client-go/connarray and memdbstaging
  pass L0; doactionbatches fails L2 twice (flip `none`). These trials are not in this
  worktree's state.db; they enter via `MANUAL_FLIPS` in `scripts/closure_metrics.py`.
- The materialised `tasks_composerver/{helm,kops}/<unit>-L2/environment/src` trees are
  **unexcised** — verified identical to the `prepare_repo` base, although
  `outputs/materialize_hkg.log` lists them as materialised (the chartkit/clustkit-authored
  patches cannot apply to `example.internal/helm`/`kops` trees). They are therefore unusable
  as excision-diff inputs; helm/kops metrics come from the rebranded authored patches.
