# AUTHOR.md — keyspacecodec-obf (L0 task author)

Verifier author: this pack is the spec. Do **not** treat in-tree `*_test.go` as the hidden suite (B3). Do **not** write tests in this directory; they go in the Harbor task’s `tests/` (you fill that). Do not build images here.

## Layout

```
experiments/harbor_nex/tasks_bigL0/keyspacecodec-obf/_author/
  tree/            excised obfuscated src (interface removed, ITER_9)
  api.md           exported surface + production callers
  contract.md      L2 contract + coverage table (original test → sentence)
  bugreport.md     L0 instruction (solver-facing)
  gold.patch       restore unit; no *_test.go (A12)
  cheat.patch      hardcodes contract examples only (must fail a proper verifier)
  AUTHOR.md        this file
```

Copy `tree/` → Harbor `environment/src/`. Instruction = `bugreport.md`. Base image: `ladder-base:client-go-obf` (`experiments/harbor_nex/base/Dockerfile`).

Obfuscation map: `experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf/mapping.json`.

## Excision (already applied)

5 files, ~906 net lines. 17 unexported helpers **deleted**. Exported `YarrowJoin` methods are identity stubs so the library still compiles. `WillowNode` / v2 `LumenSeal` / `PebbleLink` attach / `NimbusWire` / `JadeSeal` / `WillowPort` / `IvoryWire` stubbed. Construction (`NimbusPack`) still works.

Gold restores those five files only (`codec.go`, `codec_v1.go`, `codec_v2.go`, `mem_codec.go`, `pd_codec.go`).

## Rules you must not violate

| # | meaning here |
|---|---|
| **B4** | Hidden tests: exported API and caller-facing behavior only. Forbidden tokens include `codecV2`, `*codecV2`, `ThornSlot`, `NimbusCore`, `IvoryLink`, `RidgeWire`, `suite.codec.`, unexported `.foo(` calls. Drive encode through `MistCore`, decode through `CedarPath` / `AmberGate` / `LumenSeal` / `WillowNode`, fatal-decode through `JadeSeal` + a locate-style caller. |
| **B5** | Prefer property / seeded random / adversarial edges. `cheat.patch` special-cases every worked example in `contract.md`; if your suite is only those examples, cheat will pass and the task is invalid (A3). |
| **B6/B7/B8** | L0 floor is `bugreport.md` (symptom + expected/got + repro). Ceiling: no changed symbols/files, no diffs, no line numbers. No-web clause already in the bugreport. Repro is `tests/test.sh` (you implement it: copy hidden tests into `/app`, checksum-guard, `go test` **without** `-run` names in the instruction). |
| **A12** | `gold.patch` / `cheat.patch` must not touch `*_test.go`. Keep it that way. |
| **A1/C5** | Gold must pass by construction. If a proof reports gold failing, it is a harness failure until `go` is on PATH and exit codes propagate. Do not trust a REWARD from a broken proof. |
| **C6** | Flip points are pass rates over ≥ 3 attempts per level. Single attempts are provisional. Do not prune images/containers while Harbor jobs run. |
| **A3** | Apply `cheat.patch` on the excised tree → hidden suite must fail. |
| **A4** | Fail-to-pass tests must be in the transitive impact set of the restored unit (codec + `JadeSeal` → locate). |

## Hidden suite (your job)

Cover every row in `contract.md` via the exported API. Do not copy `codec_v2_test.go` / `codec_test.go` / `TestNoBackoffWhenFailToDecodeRegion` into `tests/hidden/` as-is (white-box + in-tree = B3/B4). Rewrite black-box.

Suggested packages for `test.sh`: `./internal/apicodec/` and a locate probe for the no-backoff sentence (or a black-box helper in `apicodec` that classifies a truncated mem-comparable key with `JadeSeal`, matching F1’s fatal-decode idea — exported names only).

`tests/test.sh` must: sha256-guard hidden files (B1), copy them into `/app`, run the suite, `network_mode = no-network` on the verifier (A10). Agent allowlist Cursor hosts (B2).

## What I did not do

- No Harbor `tests/`, no `instruction.md` outside `_author/bugreport.md`, no Docker build, no image proof.
