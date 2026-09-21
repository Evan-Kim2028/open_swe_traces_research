# authored_batch2 — 14 new L0-hard Helm units

Task-author pass. Each unit under `experiments/pipeline/authored_batch2/helm/<unit>/_author/` contains api.md, contract.md (L2 prose + coverage table), bugreport.md (L0), excised/excision.patch (stubs + test removal), gold.patch (impl only, no test files), cheat.patch (worked-example hardcodes), difficulty.md, DETAILS.md.

Local verification (ladder-base:helm, Go 1.26.8): for every unit — excision patch applies and `go build`+`go vet` pass on the remaining tree (incl. remaining test files), gold patch applies and compiles, cheat patch applies and compiles. No solver trials, no hidden tests, no Harbor run.

## Units

| unit | closure (package/files) | files in excision | patch lines | details | predicted_flip | commitments | compiles |
|---|---|---|---|---|---|---|---|
| chartmeta | pkg/chart/v2: metadata.go, dependency.go, errors.go | 6 | 396 | 9 | L3 | sanitize-in-place rune rules; required-field order apiVersion→name→version; reserved `.`/`..`; basename check; lenient semver; type enum; alias charset; alias-vs-name dup key; nil-receiver error; `validation: ` prefix | yes |
| searchindex | pkg/cmd/search: search.go | 2 | 371 | 10 | L3 | `\v` 4-field match line; score=field index; boundary-on-separator; literal case-insensitive vs regexp case-sensitive; exclusive threshold; `$$ver` keys; newest-only vs all; SortEntries mutation; score→name→semver-desc order; unparseable-version-sorts-first | yes |
| chartdl | pkg/downloader: chart_downloader.go | 4 | 1142 | 11 | L4 | OCI short-circuit + tag/digest rules; abs-URL repo scan w/ ErrNoOwnerRepo swallow; repo/chart split + creds; index lookup + URL resolve; `algo:`-prefix strip + 32-byte check; OCI filename rewrite; 4 verify strategies; cache get/put; `.tgz` ext; option accumulation | yes |
| dlmanager | pkg/downloader: manager.go | 2 | 765 | 11 | L3 | repo-field dispatch order (file→OCI→alias→URL→identity→missing); conditional note in missing-repo error; ErrRepoNotFound text differs; helm-manager-sha256 names; version-equality asymmetry; empty-version-first-with-URLs; OCI last-colon split w/ ports; trailing-slash dedupe last-wins; OCI URL synth; creds passthrough | yes |
| getterdispatch | pkg/getter: getter.go, plugingetter.go | 4 | 312 | 8 | L2 | exact-scheme membership; first-match + `scheme %q not supported`; http+https single provider vs OCI; option order (call, default-120s, extras); swallowed discovery errors; getter/v1 gating; wire-field subset; call-beats-global | yes |
| httpgetter | pkg/getter: httpgetter.go | 2 | 516 | 8 | L2 | copy-then-overlay opts; Accept-if-set; UA default+override; credential scope = pass-all OR scheme+host:port (port-sensitive); dual parse errors; non-200 error format; needsCustomTLS formula; explicit>custom>shared-lazy transport | yes |
| urlutil | internal/urlutil: urlutil.go | 2 | 111 | 6 | L2 | path-only join preserving scheme/query; asymmetric unparseable fallback (first→clean-compare, second→false); empty-path→`/`; dot/dotdot collapse; port-stripped hostname; pathish base joins | yes |
| sympath | internal/sympath: walk.go | 2 | 203 | 7 | L2 | lexical order; visit-at-link-path w/ resolved info; eval-symlink error; SkipDir swallowed for dir+link but PROPAGATES on plain files; root-stat-error callback; readdir-failure callback; mode-bit detect | yes |
| tlsutil | internal/tlsutil: tls.go | 2 | 139 | 7 | L2 | collect-all-then-join option errors; empty-pair no-op vs single-path error; cert-then-key order + distinct messages; both-blocks gating; CA-append failure; nil RootCAs; insecure passthrough | yes |
| copyst | internal/copystructure: copystructure.go | 2 | 415 | 8 | L2 | nil→empty-map (not nil); per-kind copy rules; nil-interface preservation in maps/slices vs nil collection; len+cap slice; pointer re-wrap; struct field copy; func/chan shared; unsupported-kind error | yes |
| valuesopts | pkg/cli/values: options.go | 2 | 440 | 8 | L2 | 6-layer precedence files→json→set→string→file→literal; `{`-sniff JSON-object vs key=value; two near-identical set-json error formats; `-`+TrimSpace→stdin; unsupported-scheme→local-file fallback; WithURL on remote; per-family wraps | yes |
| helmpath | pkg/helmpath: lazypath.go, home.go | 7 | 244 | 5 | L2 | HELM_* env → no `helm` subdir while XDG/default DO; HELM>XDG>default precedence; lazy env eval; empty-name → no leading dash in `<name>-index.yaml`/`-charts.txt` | yes |
| chartrepo | pkg/repo/v1: chartrepo.go | 3 | 361 | 9 | L3 | URL/scheme-handler error texts; full TLS/auth getter-option bundle; charts.txt + index.yaml dual cache write; random-name throwaway repo + cleanup; ChartNotFoundError version clause; abs-ref passthrough; Path AND RawPath slash-normalize (escaped slashes); query carry-over | yes |
| relsplit | pkg/release/v1/util: filter.go, manifest.go | 4 | 603 | 8 | L2 | Check nil→false vs StatusFilter inner nil→true; Any-empty→false / All-empty→true; order-preserving; `(?m)^---[ \t]*` split incl. fused text; leading-whitespace pre-trim; empty-doc drop; manifest-N keys; numeric-suffix sort | yes |

## Existing closures avoided (all 10 batch-1 units)

`repindex` (index add/get/merge/sort — avoided pkg/repo/v1/index.go), `strvalsparser` (pkg/strvals — avoided), `coalesce` (value coalescing — avoided), `depresolver` (internal/resolver — avoided), `chartloader` (loaders — avoided), `memorydriver` (storage driver), `provenance` (signing — VerifyChart only *calls* it), `storage` (release storage), `kindsorter` (kind_sorter.go, manifest_sorter.go, sorter.go — relsplit deliberately uses the two sibling files filter.go + manifest.go only), `ignorerules` (ignore rules). Downloader was split into two units on disjoint files (chart_downloader.go vs manager.go) with `loadRepoConfig` kept live as the shared boundary; same for pkg/getter (getter.go+plugingetter.go vs httpgetter.go) and pkg/repo/v1 (chartrepo.go vs repindex's index.go).

## Notes for the verifier

- `chartdl`/`dlmanager` share package `pkg/downloader`; shared consts `repoConfig`/`repoCache` are preserved in the stripped chart_downloader_test.go for both units.
- `chartrepo` keeps `startLocalServerForTests`/`CustomGetter` helpers for index_test.go survivors.
- `helmpath` deletes the OS-gated darwin/windows test files too (they encode the same contract).
- cheats are minimal-worked-example stubs; all must fail property suites (they return constants/zero values).
