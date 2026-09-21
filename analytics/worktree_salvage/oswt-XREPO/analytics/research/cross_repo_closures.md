# Cross-repo closures: can a task span a dependency boundary?

Date: 2026-02-19. Branch `xrepo`. Three units built, packaged at L0 and L2,
preflight-gated in-image. Log: `outputs/XREPO.log`.

## Prior art, and what it did not test

`synth/two_repo.py` + `scripts/two_repo.py` built a "library→consumer" unit
where the consumer was `integration_tests/` inside the *same* prepared tree —
one module, one `go test` invocation, call stitching across a parent index.
The ladder note already flags it: *parent index, not a second repo*. It tested
whether a solver can follow calls from a library's tests into a consumer's
code; it did not test an excision whose surrounding constraint lives in a
different module, nor any packaging where the library is an independent unit
of source.

## The pair

- **C**: `example.internal/httprouter` — the obfuscated gin tree already in the
  bank (`repos2/gin`, gin @5c6a15f).
- **L**: `github.com/go-playground/validator/v10` @v10.30.3 — the real
  third-party dependency gin's `binding` package delegates all
  `binding:"<tag>"` validation to. Staged as a **nested module** at
  `deps/validator/` inside the consumer tree, wired by
  `replace github.com/go-playground/validator/v10 => ./deps/validator` in the
  consumer `go.mod`, dep `go.mod` aligned to the versions already cached in
  `ladder-base:gin` (offline-clean, `GOPROXY=off` builds pass).

Chosen over other gin deps because gin's own tests exercise validator
behaviour through the consumer API (`binding:"required"`, `min`, `max`, `gt`,
`uuid`, custom `notone` via `binding.Validator.Engine()`), while the tag DSL
surface is far wider than what gin's tests pin — real headroom for hidden
commitments.

## The three units

All three excise a family inside `deps/validator/baked_in.go`; all hidden tests
live in `binding/` as `package binding_test` and drive only exported consumer
API (`binding.Validator.ValidateStruct`, `Context.ShouldBindJSON`); all
instructions describe the consumer-visible symptom (`panic:` during binding)
without naming a dep symbol or revealing the fault is in a dependency. Each
unit's cheat patch is a consumer-side fallback in `default_validator.go` that
swallows the panic and applies a deliberately shallow re-check of the obvious
cases.

| unit | excised closure (in L) | hidden suite | contract headroom C doesn't pin |
|---|---|---|---|
| `condreq` | `requireCheckFieldKind/Value` + all 13 `required_*`/`excluded_*`/`skip_unless` impls | 17 tests, 3000-case seeded sweep, panic taxonomy, ShouldBindJSON | multi-pair polarity (`required_unless` ≠ `skip_unless`), kind-coerced sibling compare, `nil` literal vs length on collections, non-nil-empty-slice = present, `*T`→0 = present, missing-sibling defaults, odd/dup-param panics, quoted params |
| `fieldcmp` | `hasLengthOf`, `hasMinOf`, `hasMaxOf`, `isEq/Ne/Lt/Lte/Gt/Gte`, `isEqIgnoreCase`, `isNeIgnoreCase` | 13 tests, 3000-case seeded sweep over str/int/float/duration | rune-vs-byte length, `len` as numeric value on ints, `eq` on bool literals and collection counts, `time.Duration` params (`150ms` or bare ns), `time.Time` vs now ignoring the param, case-folding scope, `Bad field type` panics |
| `oneofuniq` | `isOneOf`, `isOneOfCI`, `isNoneOf`, `isNoneOfCI`, `isUnique` | 16 tests, two 3000-case seeded sweeps | numeric fields rendered as base-10 text (params never parsed: `oneof=03` rejects `3`), quoted values, `unique` ptr-deref + double-nil = dup, `unique=Field` struct projection, map VALUE uniqueness, scalar `unique=Sibling`, `Bad field type` vs `Bad field name` panics |

## Preflight

`closure-K` in-image gate (`scripts/preflight_task.py`), per task: bare excised
tree, `+gold.patch`, `+cheat.patch`, each running the task's own `tests/test.sh`
(network=none):

| unit | level | bare | gold | cheat | verdict |
|---|---|---|---|---|---|
| condreq | L0 | fail (panic through dep) | pass | fail | **pass** |
| condreq | L2 | fail | pass | fail | **pass** |
| fieldcmp | L0 | fail (panic through dep) | pass | fail | **pass** |
| fieldcmp | L2 | fail | pass | fail | **pass** |
| oneofuniq | L0 | fail (panic through dep) | pass | fail | **pass** |
| oneofuniq | L2 | fail | pass | fail | **pass** |

Bare runs fail by *panic propagation across the boundary*
(`binding_test.TestXR…` → `binding.defaultValidator.ValidateStruct` →
`validator.(*Validate).Struct` → `traverseField` → stub), not by infra —
the panic backtrace itself is the first pointer into `deps/`.

Instruction self-check: `ok=true` on all six (no test names, no dep symbols,
no dep file names, `panic:` symptom, one-command repro with `go test`). B1–B8
unskipped rules all pass; B4 black-box holds (external `binding_test` package,
exported calls only).

## Does the boundary matter?

Partly — and differently than expected.

**What crosses the boundary for real:**

- *Discovery.* The panic backtrace is the only signal pointing into `deps/`.
  The solver must notice `go.mod`'s `replace`, realise the module under
  `deps/validator` is compiled source they can edit (not a read-only module
  cache), and navigate a 3500-line `baked_in.go`. Nothing in the consumer names
  the failed component. That is genuine extra inference vs a same-repo stub
  that a stack trace lands inside the familiar tree.
- *No cribbing from L's tests.* The staged module ships no `*_test.go`
  (module-cache provenance), so semantics must come from the contract, the
  README tag table (present, but one line per tag — no edge cases), and
  black-box probing through C.
- *Blast radius is real.* `baked_in.go` holds ~200 registered tags; excising 15
  functions leaves the other ~185 working, so "the validator is broken" is not
  the hypothesis space — the solver must isolate exactly which tag family
  panics and why.

**What does not change:**

- *The constraint shape is the same.* Once the solver is inside `baked_in.go`,
  the inference problem is identical to a single-repo closure: reimplement
  functions against a behavioural contract. The module boundary adds a
  *discovery* step, not a deeper *reconstruction* step — C's call sites
  (`v.validate.Struct(obj)` with `SetTagName("binding")`) constrain almost
  nothing about tag semantics, only the plumbing.
- *Unconstrained commitments dominate.* Gin's own tests pin `required`,
  `min`/`max`, `gt`, `uuid` — a thin slice. Everything that makes these units
  hard (kind coercion, presence semantics, panic taxonomy, param grammar)
  is contract-only, same as a hard single-repo L0 unit.

**Verdict:** the boundary adds a real discovery/navigation dimension and much
better realism (this is the shape of most production bugs), but it does not
change the inferential core of the task. Difficulty is mostly orthogonal to
repo-ness: these units are hard because C under-constrains L's tag DSL, not
because L lives in another module. A "cross-repo" badge alone is not a
difficulty claim; the difficulty still has to be authored into the contract
gap.

## What a 20-unit batch would need

1. **More L surfaces per C.** validator alone supports ~6–8 more families
   (format tags: email/uri/uuid; string predicates; `dive`/`keys`/`endkeys`
   collection traversal; struct-level validation; `omitempty` interplay;
   cross-field `eqfield`/`gtfield`/`*csfield`). Gin exercises all of them
   through the same `binding` API.
2. **A second pair** for diversity — e.g. client-go's apimachinery deps, or
   kops→(its dep tree). Same staging recipe: copy pinned module source into
   `deps/`, add `replace`, align dep `go.mod` to the base image's module cache.
   The `xrepo_pairs.py` builder is pair-generic (`PAIR_SRC`/`DEP_REL` are
   parameters); only `UNITS` entries and author dirs are per-family.
3. **Dep-test stripping stays mandatory** — if a staged module shipped its own
   `*_test.go`, L0 difficulty collapses (the solver copies semantics, not
   infers them). Keep module-cache provenance, not VCS checkouts.
4. **Cheat surface is reusable** — one consumer-side fallback pattern
   (recover + shallow re-check) works per family; ~1h each to author.
5. **Watch outs**: `replace` paths make `go.work`/workspace tools and
   `go mod tidy` inside the image footguns — pin dep go.mod versions to the
   image cache exactly, and never run `go mod tidy` on the consumer inside the
   task (it would rewrite `deps/` requirements). Preflight catches both as
   `infra`.

## Limits

Three units, one pair, one language, one library shape (tag dispatch). No
solver trials run — flip predictions (L2) are authored estimates, not
measured. The claim "boundary adds discovery, not reconstruction depth" is
an argument from the artifact, not from solve data.
