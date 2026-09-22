# Bug report

`config`'s path parsing and latch/scope validation panic: `ParsePath`,
`TxnLocalLatches.Valid`, and `GetTxnScopeFromConfig` are stubbed, so a
client cannot parse its PD endpoints.

Expected: `ParsePath("tikv://node1:2379,node2:2379")` returns
`["node1:2379","node2:2379"], false, "", nil`;
`ParsePath("tikv://node1:2379?disableGC=true&keyspaceName=DEFAULT")`
returns `["node1:2379"], true, "DEFAULT", nil`; `ParsePath("etcd://x:2379")`
errors. `(&TxnLocalLatches{Enabled: true}).Valid()` errors while
`{Enabled: true, Capacity: 100}` and `{Enabled: false}` pass. With a fresh
default config, `GetTxnScopeFromConfig()` returns `"global"`.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
