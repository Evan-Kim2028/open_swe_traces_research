# Bug report

The mockstore MVCC binary codec panics: `mvccLock`/`mvccValue`
marshal/unmarshal, the `marshalHelper` primitives, and `MvccKey`
encode/decode are stubbed, so MVCC entries cannot round-trip through the
raw store.

Expected (in-package probes): `(&mvccLock{startTS:5, primary:[]byte("p"),
value:[]byte("v"), op:kvrpcpb.Op_Put, ttl:100, forUpdateTS:6, txnSize:1}).
MarshalBinary()` succeeds and unmarshals back to identical fields;
`mvccValue{valueType:typePut, startTS:3, commitTS:5, value:[]byte("v")}`
round-trips through its marshal pair. `NewMvccKey([]byte("abc")).Raw()` is
`"abc"`; `NewMvccKey(nil)` and `MvccKey(nil).Raw()` are nil.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
