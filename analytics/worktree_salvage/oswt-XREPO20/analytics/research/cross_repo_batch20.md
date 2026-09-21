# Cross-repo closure batch — 20 units across two dependency boundaries

2026-09-20 · worktree `oswt-XREPO20` (branch `xrepo20`) · staged at
`experiments/dose_response/sweep_xrepo20/` (main checkout) · log `outputs/XREPO20.log`

Follow-up to `cross_repo_closures.md` (3-unit gin→validator pilot). This batch
scales the recipe to 20 units over **two** consumer/dependency pairs:

- **Pair 1 — gin → go-playground/validator v10.30.3** (14 units). Consumer
  module `example.internal/httprouter`, dep staged as a nested module at
  `deps/validator` with `replace github.com/go-playground/validator/v10 => ./deps/validator`.
- **Pair 2 — kOps → k8s.io/apimachinery + github.com/blang/semver/v4** (6 units).
  Consumer module `example.internal/kops`, deps staged at `deps/apimachinery`
  and `deps/semver` with two `replace` directives.

Every unit excises a closure *inside the pinned dependency* and verifies it
only through the consumer's exported API (B4). Per the pilot's verdict — the
boundary adds discovery/navigation realism but not inferential difficulty —
each unit's edge-case surface is authored per `authoring_hard_l0_units.md`;
the "cross-repo" badge is not the difficulty claim.

## Per-unit table

All units: `bare=fail` (panic/assertion), `gold=pass`, `cheat=fail` at both
L0 and L2 — 40/40 packaged tasks pass preflight. Commitments = numbered lines
in `DETAILS.md`; each has one hidden property test in the coverage table.

| unit | pair | dep file(s) | excised functions | commits | L0 bare/gold/cheat | L2 bare/gold/cheat |
|---|---|---|---|---|---|---|
| affix | gin→validator | `deps/validator/baked_in.go` | `startsWith`, `endsWith`, `startsNotWith`, `endsNotWith` | 6 | fail/pass/fail | fail/pass/fail |
| colorfmt | gin→validator | `baked_in.go` | `isHEXColor`, `isRGB`, `isRGBA`, `isHSL`, `isHSLA`, `isCMYK` | 6 | fail/pass/fail | fail/pass/fail |
| contain | gin→validator | `baked_in.go` | `containsRune`, `containsAny`, `contains`, `fieldContains` | 6 | fail/pass/fail | fail/pass/fail |
| crossfld | gin→validator | `baked_in.go` | `isNeField`, `isEqField`, `isLtField`, `isLteField`, `isGtField`, `isGteField` | 6 | fail/pass/fail | fail/pass/fail |
| cryptocoin | gin→validator | `baked_in.go` | `isEthereumAddress`, `isEthereumAddressChecksum`, `isBitcoinAddress`, `isBitcoinBech32Address` | 6 | fail/pass/fail | fail/pass/fail |
| exclude | gin→validator | `baked_in.go` | `excludesRune`, `excludesAll`, `excludes`, `fieldExcludes` | 6 | fail/pass/fail | fail/pass/fail |
| hashfmt | gin→validator | `baked_in.go` | `isMD4`, `isMD5`, `isSHA256`, `isSHA384`, `isSHA512`, `isRIPEMD128`, `isRIPEMD160`, `isTIGER128`, `isTIGER160`, `isTIGER192` | 6 | fail/pass/fail | fail/pass/fail |
| hostport | gin→validator | `baked_in.go` | `isHostnameRFC952`, `isHostnameRFC1123`, `isFQDN`, `isDnsRFC1035LabelFormat`, `isHostnamePort`, `isPort` | 6 | fail/pass/fail | fail/pass/fail |
| ipcidr | gin→validator | `baked_in.go` | `isIPv4`, `isIPv6`, `isIP`, `isCIDRv4`, `isCIDRv6`, `isCIDR`, `isMAC` | 6 | fail/pass/fail | fail/pass/fail |
| mailfmt | gin→validator | `baked_in.go` | `isEmail` | 7 | fail/pass/fail | fail/pass/fail |
| structlvl | gin→validator | `struct_level.go`, `validator_instance.go` | `wrapStructLevelFunc`, `ReportError`, `ReportValidationErrors`, `RegisterStructValidation`, `RegisterStructValidationCtx`, `RegisterStructValidationMapRules` | 5 | fail/pass/fail | fail/pass/fail |
| urifmt | gin→validator | `baked_in.go` | `isURI`, `isURL`, `isHttpURL`, `isHttpsURL`, `isUrnRFC2141`, `isDataURI` | 6 | fail/pass/fail | fail/pass/fail |
| uuidfmt | gin→validator | `baked_in.go` | `isUUID`, `isUUID3`, `isUUID4`, `isUUID5`, `isUUIDRFC4122`, `isUUID3RFC4122`, `isUUID4RFC4122`, `isUUID5RFC4122`, `isULID` | 6 | fail/pass/fail | fail/pass/fail |
| xstruct | gin→validator | `baked_in.go` | `isNeCrossStructField`, `isEqCrossStructField`, `isLtCrossStructField`, `isLteCrossStructField`, `isGtCrossStructField`, `isGteCrossStructField` | 6 | fail/pass/fail | fail/pass/fail |
| clusternames | kops→apimachinery | `deps/apimachinery/pkg/util/validation/validation.go` | `IsDNS1123Subdomain`, `IsDNS1123Label`, `IsDNS1035Label`, `IsWildcardDNS1123Subdomain`, `IsDNS1123SubdomainWithUnderscore` | 4 | fail/pass/fail | fail/pass/fail |
| ignames | kops→apimachinery | `deps/apimachinery/pkg/api/validation/generic.go` | `NameIsDNSSubdomain`, `NameIsDNSLabel`, `NameIsDNS1035Label`, `maskTrailingDash` | 5 | fail/pass/fail | fail/pass/fail |
| intstrpct | kops→apimachinery | `deps/apimachinery/pkg/util/intstr/intstr.go` | `GetScaledValueFromIntOrPercent`, `GetValueFromIntOrPercent`, `getIntOrPercentValue`, `getIntOrPercentValueSafely` | 6 | fail/pass/fail | fail/pass/fail |
| labelvals | kops→apimachinery | `deps/apimachinery/pkg/api/validate/content/kube.go` | `IsLabelValue`, `IsLabelKey`, `IsPrefixedLabelKey`, `prefixEach` | 5 | fail/pass/fail | fail/pass/fail |
| portrange | kops→apimachinery | `deps/apimachinery/pkg/util/net/port_range.go` | `Set`, `ParsePortRange`, `ParsePortRangeOrDie`, `Contains`, `String`, `Type` | 4 | fail/pass/fail | fail/pass/fail |
| semververs | kops→semver | `deps/semver/semver.go`, `deps/semver/range.go` | `Parse`, `ParseTolerant`, `ParseRange` | 6 | fail/pass/fail | fail/pass/fail |

Preflight evidence (bare): every unit fails with `panic: excised: <fn>` inside
the excised dep closure, surfaced through the consumer call path — e.g.
`excised: isEmail` (gin), `excised: IsDNS1123Subdomain` (kops). Full evidence
strings in each task's `validation.json` (`preflight.evidence`).

## Uniqueness

`uv run python scripts/check_unit_overlap.py --extra /home/evan/Documents/oswt-XREPO20`
from the main checkout:

```
checked gin: 17 new units vs 10 existing — CLEAN
checked kops: 6 new units vs 10 existing — CLEAN
overlaps: 0
```

- **Within the batch**: all dep-side excision sets are disjoint (per-file
  symbol sets in the table above never intersect — most families share only
  `baked_in.go` with disjoint func names; `structlvl` alone touches
  `struct_level.go`/`validator_instance.go`). Cheats add family-suffixed
  helper symbols (`xr<Fam>*`, `validateStruct<Fam>`) in the consumer's
  `default_validator.go` / kops `validation.go`, so shared consumer files are
  symbol-disjoint too.
- **vs the 3 pilot units** (`condreq`, `fieldcmp`, `oneofuniq`): dep closures
  disjoint; their cheat helpers were renamed (`cheat*` → `xr<Fam>*`) so the
  shared `binding/default_validator.go` cheat surface is now disjoint as well.
- **vs the 50-unit single-repo bank** (`authored/gin`, `authored/kops`, …):
  no file/symbol overlap — every batch unit's closure lives in a `deps/`
  module path that no bank unit touches.

## What the second pair cost to stage

| | gin→validator | kops→apimachinery+semver |
|---|---|---|
| dep modules staged | 1 (`deps/validator`, 69 files, 1.7 MB) | 2 (`deps/apimachinery`, `deps/semver`; 312 files) |
| `replace` directives added | 1 | 2 |
| consumer tree size | 2.9 MB | 77 MB |
| staging attempts | 1 (worked first try) | 2 (`vendor/` failed, then `deps/`+`replace`) |
| dep `*_test.go` stripped | yes | yes (0 remain under `deps/`) |
| module-cache provenance | copied from image cache | copied from image cache (never VCS) |
| extra builder work | none (pilot pair) | per-file `blank_imports` metadata; `GOFLAGS=-mod=mod` workaround |

The expensive lesson: **`vendor/` staging silently does nothing when the base
image sets `GOFLAGS=-mod=mod`** (`ladder-base:kops` does). Go ignores the
vendor tree, the excision never runs, and preflight shows `bare=pass` — a
false-healthy signal that costs one full preflight cycle to catch. The fix
was the spec's recipe: copy pinned module source into `deps/`, add `replace`,
align dep `go.mod`s to the image's module cache. After re-staging, all 6 kops
units flip to `bare=fail` with the panic inside the dep.

Net answer: pairs scale, but budget **one failed staging attempt per new
consumer** — check `GOFLAGS`/`GONOSUMDB` in the base image before choosing
`vendor/` vs `deps/`. The `replace` recipe generalizes cleanly once learned.

## Repairs this batch needed (all fixed, evidence in log)

- `stub_go_func` mis-braced signatures containing `interface{}` → body brace
  now located at paren depth 0 ending a line (`_body_brace`).
- Unused imports after stubbing → per-file `blank_imports` (gin: `net/mail`,
  `crypto/sha256`, `encoding/hex`, `golang.org/x/crypto/sha3`, `go-urn`;
  kops: `strings`/`fmt`/`strconv`/`errors`/apimachinery pkgs).
- kops cheat helpers moved into the edited consumer file (symbol attribution)
  with newly-unused imports dropped; `intstrpct`'s first cheat was *too
  complete* (`cheat=pass`) — replaced with a version that drops the strconv
  error so non-numeric percents silently scale to 0.
- `clusternames` contract prose named "apimachinery" → reworded (B7).
- `mailfmt` coverage table listed stale test names → regenerated to match the
  hidden suite.

## Reproduce

```bash
uv run python scripts/xr20_author_gin.py            # gin _author artifacts
uv run python scripts/xr20_author_kops.py           # kops _author artifacts
uv run python scripts/build_xrepo_pairs.py          # excision/gold patches + task dirs
uv run python scripts/preflight_task.py experiments/xrepo20/tasks/<fam>-L{0,2}
uv run python scripts/check_unit_overlap.py --extra <this worktree>   # from main checkout
```

Staged tasks: `experiments/dose_response/sweep_xrepo20/{gin,kops}-<fam>-L{0,2}`
(40 dirs, CURSOR network allowlist in each `task.toml`).
