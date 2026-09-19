# Bug report

I attached two wrappers named first then second and invoked the stack.
Only the base call ran. expected first then second then the base call,
actual only the base call.

Reproduce with:

```
go test -count=1 -timeout 15m ./wirerpc/interceptor/
```

Do not skip, delete, or weaken the tests. Do not change test assertions
or testdata just to make them green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
