# Bug report

File mode handling is broken: mode strings are not parsed or rendered in octal, files whose permissions already match get chmod'ed anyway (reporting a change that did not happen), and hash comparisons on missing files surface errors instead of treating the file as absent.

Expected: an empty mode string yields the default; `"644"` parses as octal; a mode renders back with a leading `0`; a file already at the right mode reports no change; a missing file reports a hash mismatch, not an error.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
