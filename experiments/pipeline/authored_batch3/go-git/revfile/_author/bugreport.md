# Bug report

Pack reverse-index files are read and written incorrectly. Decoding
accepts the wrong magic, skips version and checksum verification, and
leaks the output channel — it stays open after a failure. A file
declaring zero objects hangs instead of failing cleanly, and trailing
garbage after the checksum is silently ignored. Encoding writes the
entries in the wrong order — sorted by object name instead of by pack
offset — picks the wrong hash-function id for SHA-256 packfiles, and
crashes on a typed-nil writer instead of reporting it.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
