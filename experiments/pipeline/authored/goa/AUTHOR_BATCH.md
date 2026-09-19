# AUTHOR_BATCH — goa (identity-obfuscated apikit)

Repo: `experiments/pipeline/repos/goa/src` (module `example.internal/apikit/v3`).
Base image: `ladder-base:goa`.
Offline-passing packages only (16 `ok` lines in `test.jsonl`).
Gold patches restore function bodies; they do not touch `*_test.go` (A12).
Cheat patches return zero values.

Ranked hardest-first. Closure size is excised function count / body lines / files.

| rank | unit | closure | predicted_flip | why |
|---:|---|---|---|---|
| 1 | errloc | 8 funcs / ~117 lines / 1 file | L5 | call-stack location heuristics under inlining |
| 2 | evalrun | 12 funcs / ~152 lines / 1 file | L4 | execute→prepare→validate→finalize + generation caps |
| 3 | httpxray | 11 funcs / ~161 lines / 4 files | L4 | multi-file HTTP trace wrap + status flags + race |
| 4 | grpcxray | 9 funcs / ~159 lines / 1 file | L4 | unary/stream interceptors + EOF-once close |
| 5 | xrayseg | 12 funcs / ~135 lines / 1 file | L4 | mutex segment tree, Once in-progress UDP flush |
| 6 | jsonrpcwire | 12 funcs / ~239 lines / 1 file | L3 | ID identity (empty/null/number) + envelope validate |
| 7 | grpcerr | 12 funcs / ~156 lines / 1 file | L3 | status mapping + merged history + join-aware cancel |
| 8 | evalctx | 7 funcs / ~102 lines / 1 file | L3 | root topo order, cycle detection |
| 9 | grpctrace | 12 funcs / ~101 lines / 1 file | L3 | continue/start/discard/zero-rate + client metadata |
| 10 | pkgvalidation | 3 funcs / ~80 lines / 1 file | L2 | format switch; **control** |

Control unit: `pkgvalidation` (`control: true`).
