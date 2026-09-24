# Bug report

Writing the index file produces output the reader cannot parse back.
Entries come out in caller order instead of sorted, the per-entry flags
lose the stage bits and overflow on long names, and the padding between
entries is wrong so name boundaries desynchronize. Version-4 prefix
compression is not applied — full names are written uncompressed, or the
strip count is computed against the wrong entry. Timestamps before the
epoch crash the writer instead of being rejected, and the trailing
checksum covers the wrong byte range — or is written even when the
no-checksum option is set.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
