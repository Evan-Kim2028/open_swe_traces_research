# AUTHOR.md — tasks_bigL0 (task author → verifier author)

Two L0 units, interface removed (ITER_8/9), obfuscated trees. I am the task author; I did not write verifier tests or build images.

| unit | dir | excision |
|---|---|---|
| keyspacecodec-obf | `experiments/harbor_nex/tasks_bigL0/keyspacecodec-obf/_author/` | 5 files / ~906 lines; helpers deleted; exported methods identity stubs |
| batchcmds-obf | `experiments/harbor_nex/tasks_bigL0/batchcmds-obf/_author/` | 4 files / ~221 lines; packer deleted; send path unary; builder/send stubbed |

Each `_author/`: `tree/` (excised src), `api.md`, `contract.md`, `bugreport.md`, `gold.patch` (A12: no `*_test.go`), `cheat.patch` (hardcodes contract examples), `AUTHOR.md`.

Harbor: copy `tree/` → `environment/src/`, instruction = `bugreport.md`, you fill `tests/` (hidden black-box suite + `test.sh`). Base image `ladder-base:client-go-obf`.

**B4** exported API only (see each `api.md`). **B5** cheat must fail — do not make the hidden suite a restatement of the worked examples. **B6/B7/B8** already in `bugreport.md` (`tests/test.sh` + `go test` without `-run`, no-web). **A12** keep patches off tests. **C5** gold-failing proof = harness until `go` on PATH. **C6** ≥ 3 attempts per level; don’t prune running Harbor jobs.

Per-unit detail: the `AUTHOR.md` next to each tree.
