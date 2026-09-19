# AUTHOR.md — batchcmds-obf (L0 task author)

Verifier author: this pack is the spec. Do **not** treat in-tree `client_test.go` as the hidden suite (B3). Do **not** write tests in this directory; they go in the Harbor task’s `tests/` (you fill that). Do not build images here.

## Layout

```
experiments/harbor_nex/tasks_bigL0/batchcmds-obf/_author/
  tree/            excised obfuscated src (interface removed, ITER_8)
  api.md           exported surface + production callers
  contract.md      L2 contract + coverage table (original test → sentence)
  bugreport.md     L0 instruction (solver-facing)
  gold.patch       restore unit; no *_test.go (A12). CloseAddr/JadeJoin rename spill stripped
  cheat.patch      hardcodes contract examples (cancel/deadline + builder fixtures)
  AUTHOR.md        this file
```

Copy `tree/` → Harbor `environment/src/`. Instruction = `bugreport.md`. Base image: `ladder-base:client-go-obf` (`experiments/harbor_nex/base/Dockerfile`).

Obfuscation map: `experiments/harbor_nex/tasks_obf/client-go-batchcmds-obf/mapping.json`.

## Excision (already applied)

4 files, ~221 net lines. Packer method **deleted**; send path always unary; `LumenJoin` errors `"batch send unavailable"`; builder `ThornPipe` returns 0; `KelpBolt` drains and returns nil; fetch loops no-op. `OchreWire` / `reset` / `cancel` kept.

Gold restores `internal/client/client_batch.go`, `internal/client/client.go` (batch branch + `runtime/trace`), `internal/mockstore/mockkv/rpc.go` (dummy packer call), `wirerpc/tikvrpc.go` (`ToBatchCommandsRequest`). It does **not** rename `JadeJoin`→`CloseAddr` (that spill would break `NimbusCore`).

## Rules you must not violate

| # | meaning here |
|---|---|
| **B4** | Hidden tests: exported API / caller-facing behavior only. Do not name `KelpBolt`, `ThornPipe`, `batchCommandsBuilder`, or unexported `.foo(`. Cover packing through `SendRequest` (batching on, forwarded host, coprocessor stream) and cancel/deadline through exported `LumenJoin`. |
| **B5** | Property / extra hosts / extra sizes / mixed cancel bits. `cheat.patch` special-cases the contract examples (10 unforwarded, the 1/2/3/4 host fixture, the five-entry cancel mask, ctx cancel, timeout 0). Live stream grouping (checker 1,2,3,4 vs 3,6,9,12) is **not** in the cheat — keep a case like that so A3 still fails even if builder examples are hardcoded. Add more sizes so the builder cheat is insufficient. |
| **B6/B7/B8** | L0 = `bugreport.md`. No packer/builder/file names. Repro is `tests/test.sh` (you implement: install hidden tests, checksum-guard, `go test -count=1 -timeout 15m ./internal/client/...` without `-run` in the instruction). No-web clause is in the bugreport. |
| **A12** | Patches must not touch `*_test.go` (the obf gold originally spilled into `client_test.go`; this gold is stripped). |
| **A1/C5** | Gold must pass. A gold-failing proof is a harness failure until toolchain/PATH/exit codes are confirmed. |
| **C6** | Flip points are pass rates over ≥ 3 attempts per level. Do not prune images/containers while Harbor jobs run. |
| **A3** | `cheat.patch` on the excised tree → hidden suite must fail. |
| **A4** | Fail-to-pass tests in the transitive impact of the packer + batch send path (`internal/client`). |

## Hidden suite (your job)

Cover every required row in `contract.md` without copying `TestBatchCommandsBuilder` (it calls unexported-type methods). Prefer: real `SendRequest` against the in-tree mock server for stream grouping; `LumenJoin` for cancel/deadline; seeded extra batch sizes / extra forwarded hosts so the cheat’s three fixtures miss.

Skip failpoint tests (`TestPanicInRecvLoop`, `TestRecvErrorInMultipleRecvLoops`) in the hidden suite.

`tests/test.sh`: sha256-guard (B1), copy hidden into `/app`, no-network verifier (A10), agent allowlist (B2).

## What I did not do

- No Harbor `tests/`, no Docker build, no image proof.
