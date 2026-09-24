# authored_batch3 — go-github: 21 new units

Date: 2026-09-21. Branch: au2gogithub (oswt-AU2gogithub worktree).
Base image: `ladder-base:go-github`. Source: `experiments/pipeline/work/go-github/scratch`
(github.com/google/go-github/v92). Tooling: `experiments/pipeline/authored_batch3/_tools/`
(mkunit.py spec → excise → gold; verifyunit.sh = excision green → gold passes closure
tests → cheat fails closure tests).

The brief asked for 20; 21 candidates were authored and all 21 verify clean, so the
extra unit ships too — it costs nothing and adds a lottery ticket.

## Validation summary

- **Excision**: all 21 units apply, build, vet, and keep the full `./github` test
  suite green under stubs.
- **Gold**: all 21 gold patches restore behavior and pass their restored closure
  tests (`-run <testre>` against original test files).
- **Cheat**: all 21 cheat patches build and FAIL closure tests. Size audit
  (cheat_validity.py logic, adapted to `_author/` layout): max ratio 0.56
  (runidre), all < 0.6 floor. No gold or cheat patch touches `*_test.go` (A12).
- **Overlap**: `check_unit_overlap.py` symbol-level comparison — batch3 vs the
  14-unit batch2 go-github cohort (sibling worktree oswt-AUgogithub): **0
  overlaps**; intra-batch3: **0 overlaps**. Two INFO notes (shared file,
  disjoint symbols): gethdr↔eventdispatch (`repos_hooks_deliveries.go`),
  propvalues↔customprop (`orgs_properties.go`).
  - Tool fix: `EXCISED_RE` captured only up to the first `.`, so method markers
    (`excised: Client.checkBodyDestination`) collapsed to the receiver name
    (`Client`) and produced two false-positive OVERLAPs (apiversion↔urlpolicy,
    errfmt↔erris — all genuinely disjoint methods). Regex widened to
    `[A-Za-z_][A-Za-z0-9_.]*`; identical qualified symbols still intersect.

## Per-unit table

| unit | closure (excised symbols) | surface | DETAILS n | yes/doc/part/no | overlap | cheat:gold | ratio |
|---|---|---|---|---|---|---|---|
| apiversion | `Client.checkRequestAPIVersionBeforeDo` (github.go) | predicate: is this request's API version allowed | 5 | 2/1/1/1 | clean | 3:11 | 0.27 |
| auditstream | 8 audit-stream cfg constructors (orgs_audit_log.go) | serialisation: config struct → request shape, vendor literal each | 5 | 1/0/2/2 | clean | 3:8 | 0.38 |
| boolresp | `parseBoolResponse` (github.go) + 8 indirect predicate callers | predicate: 204-vs-404 → bool | 4 | 2/0/1/1 | clean | 1:9 | 0.11 |
| brnotprot | `isBranchNotProtected` (github.go) | predicate: does this 400 body mean "not protected" | 4 | 2/0/2/0 | clean | 1:2 | 0.50 |
| comfortfade | `PullRequestReviewRequest.isComfortFadePreview` (pulls_reviewers.go) | predicate: is this review-request shape the preview form | 6 | 3/0/1/2 | clean | 5:23 | 0.22 |
| credhdrs | `setCredentialsAsHeaders` (github.go) | serialisation: client creds → request headers on the clone | 4 | 2/0/2/0 | clean | 1:13 | 0.08 |
| enturls | `parseURL`, `WithEnterpriseURLs` (url.go, github.go) | parsing+normalisation: enterprise base/upload URL conventions | 6 | 1/2/1/2 | clean | 3:35 | 0.09 |
| errfmt | `Error()` of 6 error types, `Error.UnmarshalJSON`, `sanitizeURL`, `formatRateReset` (github.go) | serialisation: error → human string; URL scrub; duration render | 9 | 0/0/2/7 | clean | 2:66 | 0.03 |
| erris | `Is()` of 5 error types, `compareHTTPResponse`, `equalDurationPtr` (github.go) | predicate: field-wise error equality behind errors.Is | 8 | 2/1/4/1 | clean | 11:81 | 0.14 |
| gethdr | `getHeader` (repos_hooks_deliveries.go) | predicate: case-insensitive header lookup | 3 | 2/0/1/0 | clean (INFO: file shared w/ eventdispatch, disjoint) | 2:7 | 0.29 |
| hostrunvalid | `validateCreateHostedRunnerRequest` (actions_hosted_runners.go) | predicate: required-field validation | 6 | 1/4/0/1 | clean | 5:14 | 0.36 |
| pagevalues | `Response.populatePageValues` (github.go) | parsing: Link header → pagination cursors | 6 | 0/0/4/2 | clean | 6:27 | 0.22 |
| patopts | `addListFineGrainedPATOptions` (orgs_personal_access_tokens.go) | serialisation: opts → repeated `owner[]`/`token_id[]` params | 5 | 2/0/2/1 | clean | 19:42 | 0.45 |
| propvalues | `CustomProperty.DefaultValue{String,Strings,Bool}` (orgs_properties.go) | predicate+coercion: ValueType-switched payload accessors | 4 | 1/0/3/0 | clean (INFO: file shared w/ customprop, disjoint) | 7:39 | 0.18 |
| ratecat | `GetRateLimitCategory` (github.go) | predicate: (method, path) → rate-limit bucket | 7 | 2/0/2/3 | clean | 8:51 | 0.16 |
| ratehdrs | `parseRate`, `parseTokenExpiration` (github.go) | parsing: rate-limit/token-expiry headers → Rate/Timestamp | 4 | 2/0/1/1 | clean | 6:15 | 0.40 |
| respclassify | `CheckResponse`, `parseSecondaryRate` (github.go) | predicate+classification: status/headers/body → typed error | 8 | 1/3/4/0 | clean | 9:74 | 0.12 |
| runidre | `DeploymentProtectionRuleEvent.GetRunID` (github.go) | parsing: regex capture → int64 run ID | 4 | 2/1/0/1 | clean | 5:9 | 0.56 |
| signmsg | `createSignature`, `createSignatureMessage` (git_commits.go) | serialisation: commit → canonical signable payload | 5 | 2/1/2/0 | clean | 8:44 | 0.18 |
| timestamp | `Timestamp.UnmarshalJSON` (timestamp.go) | parsing: unix-seconds vs milliseconds vs RFC3339 | 7 | 0/2/2/3 | clean | 1:2 | 0.50 |
| urlpolicy | `checkURLPathTraversal`, `sameOrigin`, `normalizedPort`, `isAllowedOrigin`, `shouldAuthorizeRequest`, `checkBodyDestination` (github.go) | predicate: URL safety + credential-scope policy | 5 | 0/3/2/0 | clean | 9:37 | 0.24 |

Totals: 21 units, all parsing/predicate/serialisation (no orchestration
closures). 111 DETAILS lines; Inferable mix: 33 yes / 20 doc / 41 partially /
17 no.

## Cheat shapes (what each small patch does)

- **apiversion**: accepts only `version == apiVersionMin` — fails max + out-of-range.
- **auditstream**: returns correct `StreamType` literal for 3 of 8 constructors.
- **boolresp**: swallows all errors as (false,nil), not just 404 — fails the 400 case.
- **brnotprot**: `HasSuffix` on the formatted error (which ends with `]`) — never matches.
- **comfortfade**: returns true on first comment with `Side` — fails mixed-style cases.
- **credhdrs**: sets auth on the original request, not the clone — never reaches the wire.
- **enturls**: appends `api/v3/`/`api/uploads/` unconditionally — fails already-formed URLs.
- **errfmt**: restores only AcceptedError + Error literals — other 7 formats stay stubbed.
- **erris**: restores `compareHTTPResponse` + `equalDurationPtr` — Is methods stay stubbed.
- **gethdr**: `headers[strings.ToLower(key)]` — fails mixed-case stored keys.
- **hostrunvalid**: validates Name only — fails image/size/groupID cases.
- **pagevalues**: parses first/last rels only — prev/next cursors stay empty.
- **patopts**: emits `owner[]` but drops `token_id[]` handling.
- **propvalues**: restores DefaultValueString only — Strings/Bool stay stubbed.
- **ratecat**: classifies `/graphql` + `/search/` only — other 8 categories fall to Core.
- **ratehdrs**: parses `MST` layout only — fails numeric-offset (`+0200`) headers.
- **respclassify**: restores primary rate-limit case only — 202/2FA/secondary/redirect unhandled.
- **runidre**: parses `match[0]` (whole match) instead of `match[1]` — submatch off-by-one.
- **signmsg**: validates params then returns raw message as the "signature".
- **timestamp**: drops the `*1e6` — millisecond inputs land near epoch.
- **urlpolicy**: restores `checkURLPathTraversal` only — all origin/port/destination checks stay stubbed.

## Notes for the next batch

- Closure sizing is the hard part: `parseBoolResponse`, `CheckResponse`, and
  `GetRateLimitCategory` each had wide indirect blast radii (service-level
  predicate callers, `withRateLimits` subtests). Iterate `deltests` until the
  stubbed tree is green — the failing tests ARE the closure boundary.
- Duplicate names in `delfunc`/`deltests` lists corrupt the splice output
  (overlapping byte ranges). Dedupe spec lists before running mkunit.
- Deleting test funcs can orphan imports — the `unimport` spec key exists for
  exactly this; check `go build` + `go vet` after every deltests change.
- The verify cheat stage runs only the spec `testre`; tests the cheat breaks
  outside the closure don't show. Keep cheats narrow anyway — blast radius in
  the full suite wastes review time later.
- `Timestamp.UnmarshalJSON`'s stub vs gold diverged only in a year-2286–3000
  band no test hits — a "correct-looking" cheat passed everything. A cheat
  must demonstrably fail a *restored* test, not just differ from gold.
