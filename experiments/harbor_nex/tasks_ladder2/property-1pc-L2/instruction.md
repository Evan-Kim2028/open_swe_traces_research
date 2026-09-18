# Missing behavior

A commit may take a one-phase shortcut or an async-commit shortcut only when
all of these hold:

- the transaction's geographical scope is the global default, not a local one
- no commit-timestamp upper-bound callback is installed
- no binlog replica is attached
- the corresponding enable flag is on

Async-commit additionally refuses when the mutation count is above the
configured key-count limit or the sum of key sizes is above the configured
total-size limit. One-phase does not apply those size limits.

Ignoring scope/binlog/bound-check and returning only the enable flag is
wrong: expected refuse on a local scope, actual allow.

Worked cases:

- global + both flags + no binlog + no bound check → both allow
- local scope → both refuse
- binlog attached → both refuse
- bound-check callback set → both refuse

Coverage the hidden checks enforce:

- Seeded combinations of scope, flags, binlog, bound-check, key count, and size.
- The four worked cases above.
- Random key sets besides the contract's single letter still follow the rule.

Reproduce with:

```
go test -count=1 -timeout 15m ./txnkv/transaction/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
