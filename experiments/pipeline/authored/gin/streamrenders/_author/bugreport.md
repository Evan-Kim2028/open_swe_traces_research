# Bug report

binary/text responses are malformed: wrong or missing Content-Length, custom headers clobbered or dropped, redirects panic on valid codes, and format strings print literally.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
