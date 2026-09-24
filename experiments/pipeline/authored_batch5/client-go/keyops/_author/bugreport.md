# Bug report

Key-range helpers in `kv` panic: callers cannot compute a successor key, a
prefix successor, compare keys, or render keys for logs, and replica-read
types panic on inspection.

Expected: `NextKey("ab")` returns `61 62 00`; `PrefixNextKey("ab")` returns
`"ac"`, `PrefixNextKey("a\xff")` returns `"b"`, `PrefixNextKey("\xff\xff")`
returns an empty slice, and `PrefixNextKey("")` returns empty.
`StrKey([]byte{0xde,0xad})` returns `"dead"`. `CmpKey("a","b")` is -1.
`ReplicaReadType(1).String()` is `"follower"`,
`ReplicaReadFollower.IsFollowerRead()` is true while
`ReplicaReadLeader.IsFollowerRead()` is false, and `ReplicaReadType(99)`
prints `"unknown-99"`.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
