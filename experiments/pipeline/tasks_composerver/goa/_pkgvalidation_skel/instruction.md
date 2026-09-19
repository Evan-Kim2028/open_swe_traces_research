# Bug report

Generated input checks panic, or they accept malformed dates, hostnames, IPv4 written as IPv6, and non-RFC UUID strings. A value that does not match a declared regular expression is reported as valid. Callers that rely on these checks to reject bad payloads no longer see an error.

Reproduce with:

```
go test -count=1 ./pkg/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
