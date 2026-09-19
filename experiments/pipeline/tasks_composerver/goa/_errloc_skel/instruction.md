# Bug report

Invalid designs still fail, but the printed errors have empty file names and line 0, or panic while formatting. Callers cannot point at the line in the design that caused the problem. Errors recorded during evaluation and errors recorded during later checking disagree about whether a location is present.

Reproduce with:

```
go test -count=1 ./eval/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
