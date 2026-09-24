# Bug report

MQTT topic handling is broken. Published topics and subscribe filters
convert to wrong Msgkit subjects: `foo/bar` comes through as `foo/bar`
instead of `foo.bar` (or similar mis-mappings), topics containing `.`
collide with level separators, leading and doubled `/` lose their empty
levels (`/a` and `a//b` come out wrong), and `+`/`#` wildcards in filters
are either rejected outright or mis-converted. Whitespace and DEL bytes in
topics are no longer refused. Reverse conversion for outbound delivery
mangles `//` and `/.` sequences so a published message's topic does not
round-trip. Subscriptions to `#`/`*.` still receive `$SYS`-reserved
subjects. Sparkplug B birth/death topics are no longer recognised, and
NDEATH timestamp rewriting is gone or corrupts the protobuf payload.
Topic validation no longer rejects invalid UTF-8 or embedded NULs.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
