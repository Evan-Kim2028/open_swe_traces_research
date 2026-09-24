# Bug report

The low-level binary helpers corrupt the formats built on them.
Variable-width integers decode with the plain VLQ formula instead of the
offset variant, so large declared sizes come out small; encoding
produces redundant multi-byte forms that peers reject. The overflow
guard is missing so a hostile length wraps around into a small positive
number. Delimiter reads leak the delimiter into the result or discard a
complete value at end-of-input, and the buffered path keeps the trailing
byte. Binary detection reads the whole stream instead of a bounded
window, misses NULs past the first chunk, and spins forever on a reader
that returns no data and no error.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
