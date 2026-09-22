# Bug report

The config lexer mis-tokenizes values. Booleans, `$` variables, escapes and
quoted strings all come back wrong: `yes`/`on` are not recognised as bools,
`$VAR` keeps its `$`, `"a\tb"` does not unescape, and `'raw'` strings are
treated like double-quoted ones. Numbers are worse — `1k` loses its suffix,
`-5` is rejected, `2021-01-01T00:00:00Z` never produces a datetime token,
`1.2.3.4:4222` dies mid-scan, and trailing `b`/`i` on `1KiB` breaks it.
Arrays demand commas (a bare newline between items errors) yet accept `[,]`
silently; maps reject `;` separators. `#`/`//` comments after values are
misparsed, and `(…)` block strings never terminate on `)` at line start.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
