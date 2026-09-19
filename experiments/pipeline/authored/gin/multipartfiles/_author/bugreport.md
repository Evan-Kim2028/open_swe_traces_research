# Bug report

multipart form binding ignores uploaded files: file fields stay nil/empty (or the bind errors out) even though the request carried files.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
