# Hidden test verification — authored_batch3/goa (VFbatch3goac)

2026-09-21 — third pass over the same 13-dir cohort. `_author/` inputs are
byte-identical to the VFbatch3goa/VFbatch3goab runs (verified with `diff -r`),
but this run **wrote every suite fresh** per the job brief ("you write tests,
you never solve"), then Docker-proved each unit end to end in this worktree.
The prior worktrees' suites were never opened for assertions; a post-hoc
cross-check confirmed no behavioural disagreement.

**Barrier:** `gold.patch` was never opened. Every assertion derives from
`_author/DETAILS.md`, `_author/api.md`, `_author/bugreport.md`, and the
solver-visible excised tree (`/tmp/excised_vfb3c/<unit>/`). `Inferable: no`
lines assert shape only — pointer identity, membership, ordering, accept/
reject, marker tokens inside errors — never literal prose.

## Harness

- `_tools/verify_goa.sh` (ported to this worktree) materialises three states
  in `ladder-base:goa`: excised+hidden → FAIL, +gold → PASS, +cheat → FAIL;
  reports A12 (gold touches no test file).
- `_tools/gen_testsh.sh` emits `tests/test.sh` per unit: sha256-guarded
  install of the hidden file into `/app/expr/`, `go test -run '^TestDetail'`,
  `/logs/verifier/reward.txt`. Checksums re-synced after the final edits —
  all 12 verified consistent with shipped bytes.

## Result summary

| unit | details | yes | doc | partially | no | tests | excised | gold | cheat | A12 |
|---|---:|---:|---:|---:|---:|---:|---|---|---|---|
| attachsvc | 9 | 0 | 0 | 5 | 4 | 9 | FAIL | PASS | FAIL | clean |
| defval | 12 | 0 | 0 | 4 | 8 | 12 | FAIL | PASS | FAIL | clean |
| grpcend | 14 | 0 | 0 | 5 | 9 | 14 | FAIL | PASS | FAIL(build) | clean |
| httperrexpr | 10 | 0 | 2 | 6 | 2 | 10 | FAIL | PASS | FAIL | clean |
| httpresp | 13 | 0 | 1 | 5 | 7 | 13 | FAIL | PASS | FAIL | clean |
| httpsvc | 12 | 2 | 2 | 4 | 4 | 12 | FAIL | PASS | FAIL | clean |
| methodval | 13 | 1 | 0 | 5 | 7 | 13 | FAIL | PASS | FAIL | clean |
| resultview | 15 | 0 | 0 | 5 | 10 | 15 | FAIL | PASS | FAIL | clean |
| rootval | 14 | 1 | 1 | 6 | 6 | 14 | FAIL | PASS | FAIL | clean |
| secschemes | 12 | 0 | 2 | 3 | 7 | 12 | FAIL | PASS | FAIL | clean |
| svcerrors | 8 | 0 | 0 | 3 | 5 | 8 | FAIL | PASS | FAIL | clean |
| svcexpr | 12 | 0 | 1 | 4 | 7 | 12 | FAIL | PASS | FAIL | clean |

Totals: **144 commitments → 144 `TestDetailNN` → 144 contract rows**, 12/12
units satisfy all four Docker gates (final audit sweep re-ran all twelve in
sequence; every gate re-greened). Raw log: `outputs/VFbatch3goac.log`.
Decision trail: `.audit/vfbatch3goac.tsv`.

Cheat-failure mechanics: most cheats die on behavioural `TestDetailNN`
failures; `grpcend`'s cheat fails at build (`stringintconv` vet error);
`httpsvc`'s cheat nil-panics at `TestDetail01`; `resultview`'s cheat panics
mid-DSL on an unknown view — all count as FAIL because the suite cannot pass.

## Per-unit notes

### svcerrors (8 tests; 5 `no`, 3 `partially`)

DSL-built error designs plus hand-mutated expressions. `no` lines assert
marker presence on the error type's root attribute meta (error names as
values — shape probed behaviourally, not pinned as a layout), wrapped-type
naming by error name, and single-vs-multi report counts. `partially` lines
assert the derivable half of origin/copy rules.

### attachsvc (9 tests; 4 `no`, 5 `partially`)

Graph-corruption suite: DSL-builds a valid design, then detaches/re-points
services across transports (wrong root, foreign member lists, missing
back-pointers) to hit the attach-integrity commitments. `no` lines assert
which corruptions are rejected vs silently ignored; binding-fallback shape
asserted via the transport error surface.

### httperrexpr (10 tests; 2 `doc`, 2 `no`, 6 `partially`)

Error-expression shape and response mapping. Five initial assertions were
over-pinned and corrected to committed shape: unmapped-attribute → headers
fires only for `ErrorResult`-origin types (the solver-visible mapping rule),
identifier comparisons go against `rt.Identifier`, and empty-type messaging
asserts presence of a report, not its prose.

### defval (12 tests; 8 `no`, 4 `partially`)

Default-value validation across primitive/union/user types. Probed
derivations: `OneOf` members accept a `Default` inside the union DSL;
numeric defaults fit by platform-`int` range with exact-kind checks for
wider types (`int32`/`int64`/`uint32`/`uint64` typed literals rejected);
the DSL boundary rejects nil defaults outright. `no` lines assert
accept/reject plus the offending field's name in the diagnostic.

### httpsvc (12 tests; 2 `yes`, 2 `doc`, 4 `no`, 4 `partially`)

Service-level HTTP expression: canonical-endpoint detection requires a
`show`-named method; JSON-RPC endpoints appear only for methods declaring
method-level `JSONRPC()`; no-`Paths` `FullPaths` yields the bare-slash
shape. The attribute-validation stage produces no DSL-reachable error at
service scope, so ordering is asserted on the two observable pairs
(canonical-endpoint < errors, errors < transports) — the committed order,
without inventing an unreachable third probe.

### secschemes (12 tests; 2 `doc`, 7 `no`, 3 `partially`)

Pure struct-API suite (no DSL): scheme kinds, requirement construction,
`Dup`/`HasNoSecurity` semantics. `no` shapes: disagreement-count == 1 at the
differing kind (injectivity, not spelling), parse order asserted as
token-ordering per the DETAILS listing.

### svcexpr (12 tests; 1 `doc`, 7 `no`, 4 `partially`)

Service expression prepare/validate/finalize. Literal scheme strings are
committed and asserted exactly; malformed-input scheme asserts membership
in the documented codomain; URI/hint literals kept shape-only per their
`no` annotations. First-pass green.

### httpresp (13 tests; 1 `doc`, 7 `no`, 5 `partially`)

Response-expression prepare/finalize. Three DETAILS clauses refused or
weakened (below); all remaining commitments asserted including
header/cookie inheritance through the service-level response attribute.

### methodval (13 tests; 1 `yes`, 7 `no`, 5 `partially`)

Method prepare/validate/finalize ordering and security/interceptor merge.
Stage-order test probes one defect per stage (`Required` on undefined
attrs, `struct:error:name` marker misuse, interceptor payload defect) and
asserts token ordering in the aggregated report — payload → streaming
payload → result → requirements → errors → interceptors. Credential-kind
errors assert the kind name (e.g. "API key"), not attribute literals;
undeclared `ServerInterceptor` is a DSL-time error, so interceptor-stage
defects use a real declaration with an invalid payload.

### grpcend (14 tests; 9 `no`, 5 `partially`)

gRPC endpoint lifecycle: prepare defaults, error-mapping precedence across
endpoint/method/service/API, `grpc:stream:compat` lookup, stream shape
rules, `rpc:tag` validation, security metadata finalization. Probed:
error-type `rpc:tag` checks fire only via a mapped `Response`; non-object
payload `Request.Type` is the primitive itself; inherited mappings require
same-level `Error` declarations. Cheat patch does not build (vet error) —
gate satisfied.

### rootval (14 tests; 1 `yes`, 1 `doc`, 6 `no`, 6 `partially`)

Root walk order, duplicate detection, error-name/requiredness rules,
conversion/creation type-map dedup, relocated-type dependencies, server
hosting, finalize defaults, meta merge. Probed: `WalkSets` emits user types
as their `*AttributeExpr` roots; `ResetDSL` pre-seeds API + default server
(the nil-API path needs `expr.Root.API = nil` inside the DSL); JSON-RPC
routes are service-level; `ErrorName` does not imply `Required`; dedup
keys on the (user, external) pair.

### resultview (15 tests; 10 `no`, 5 `partially`)

Result-type views and projection. Probed derivations: the explicit view is
applied via a bare `View("name")` meta on the result type; projection
memoization is per-call (asserted as shared nested-type pointer identity
inside one `Project`, plus the already-viewed-identifier early return);
hand-built result types lack finalized structure, so all projection
fixtures are DSL-built; view examples land on the projected type's
attribute; a parented view's eval name renders ` of type "<x>"`.

## Refused / weakened assertions

- `httpresp` D12 tag-copy: DETAILS commits `Dup` copying the tag field;
  the observed `Dup` leaves it zero. DETAILS-vs-reference mismatch —
  asserted neither way; contract row scoped to the fields it does copy.
- `httpresp` D10 service-prefix clause and D13 non-object single-key
  branch: not observable/DSL-reachable (endpoint naming and the default
  body fill the paths first); positive branches asserted only.
- `httperrexpr` D9 precedence branch: result-type-identifier content type
  observable only when the result type carries no own content type;
  derivable case asserted.
- `resultview` D7 panic clause: panic-on-invalid-prevalidated-view is not
  DSL-reachable (invalid views are rejected at evaluation); the valid
  in-place projection path asserted.
- `resultview` D11 negative (non-result-type collection element): no
  DSL-reachable trigger; positive name/shape asserted.
- `resultview` D12 reuse direction: asserted only where a generated
  identity is actually reported; the fresh-identity fallback asserted
  unconditionally.
- `resultview` D13 "view attribute meta" selection: exercised via the
  field-level `View` DSL (the reachable spelling); meta-leak prevention
  asserted as fresh attribute pointers.
- `resultview` D15 parent-name form: `of`-marker plus a case-insensitive
  parent mention asserted; the exact rendered form is an unstated
  convention and not pinned.
- `httpsvc` D8 attribute-stage probe: no DSL-reachable error at service
  scope; ordering asserted on the two observable stage pairs.
- `rootval` D5 "generated result types skipped" clause and D3 gRPC-parent
  half: no solver-visible surface; reachable halves asserted.

## apiexpr — incomplete, not authored

`_author/` ships only `cheat.patch` (empty), `excised/`, `gold.patch` — no
`DETAILS.md`, `api.md`, `bugreport.md`. There is nothing to grade against;
no suite or contract was written rather than inventing commitments.
Flagged for a re-author pass before it can enter the bank.

## Notes for staging

- All suites are `package expr_test` — black-box on the exported API only;
  no test reads `gold.patch`, excised internals beyond the solver-visible
  surface, or unexported symbols.
- `Validate()` returns a non-nil zero-length `*ValidationErrors` on
  success — every "no error" assertion checks `len(Errors)`.
- `tests/test.sh` sha256-guards the hidden file both at the source path
  and after install into `/app`; selector `-run '^TestDetail'` covers the
  whole suite (no other `Test*` symbols exist).
- No commits made; all deliverables are untracked files on branch
  `vfbatch3goac`.
