# Bug report

`util` byte/GC-time formatting helpers panic: `FormatBytes`,
`BytesToString`, `CompatibleParseGCTime`, the hex/uppercase key helpers and
`String` are stubbed.

Expected: `FormatBytes(1024)` = `"1024 Bytes"`, `FormatBytes(1025)` =
`"1.00 KB"`, `FormatBytes(2048)` = `"2 KB"`, `FormatBytes(5242880)` =
`"5 MB"`, `FormatBytes(20971521)` = `"20.0 MB"`, `FormatBytes(2147483648)` =
`"2 GB"`. `BytesToString(1024)` = `"1024 Bytes"`, `BytesToString(2097152)` =
`"2 MB"`. `HexRegionKeyStr([]byte{0xde,0xad})` = `"DEAD"`.
`CompatibleParseGCTime("20181218-19:53:37 +0800 CST")` parses to
2018-12-18T11:53:37Z; `CompatibleParseGCTime("foo")` errors.
`ToUpperASCIIInplace([]byte("abc1"))` = `"ABC1"`.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
