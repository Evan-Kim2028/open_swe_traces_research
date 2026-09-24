# Bug report

The mockstore MVCC read path panics: `mvccLock.check`/`lockErr`,
`mvccEntry.Get`/`Less`, and `regionContains` are stubbed, so every
snapshot-isolation read dies.

Expected (in-package probes): a lock with `startTS > ts` or `op` Lock/
PessimisticLock is invisible (`check` returns `(ts, nil)`); a lock whose
startTS is in `resolvedLocks` is skipped; otherwise `check` returns an
`*ErrLocked`. `Get` at `IsolationLevel_SI` over values
`[{commitTS:5,type:typePut,value:"v5"},{commitTS:3,type:typePut,
value:"v3"}]` returns `"v5"` for `ts=7` and `"v3"` for `ts=4`.
`regionContains("a","c","b")` is true, `("a","c","c")` false,
`("a","","z")` true.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
