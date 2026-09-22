# Bug report

`internal/logutil`'s `Hex`/`hexStringer.String`/`prettyPrint` are
stubbed, so every `logutil.Hex(msg)` call panics instead of rendering
the message.

Expected (probes): `Hex(&metapb.Peer{Id:7, StoreId:3})` →
`{Id:7 StoreId:3 Role:Voter IsWitness:false}`; a nil `*metapb.Peer` →
`<nil>`; `Hex(&kvrpcpb.GetRequest{Key: {0xde,0xad,0xbe,0xef}})` →
`{Context:<nil> Key:deadbeef Version:0}`.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
