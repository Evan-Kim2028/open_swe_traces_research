# Bug report

Author and committer fields come out mangled. Names containing '<' break
the email extraction, timestamps land in the wrong timezone (half-hour
zones in particular are off by sign), and signatures with no timestamp
show a bogus date instead of none. Re-encoding a commit loses its
signature's original timezone — everything comes back UTC or the local
zone. Negative-timezone stamps like -0530 are interpreted as -5h plus
+30m.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
