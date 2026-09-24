# Bug report

Tag objects round-trip badly. Tags whose headers arrive in a different
order — tagger before tag, extra headers between canonical ones — are
rejected instead of tolerated, while genuinely out-of-order canonical
headers are accepted when they should fail. The sha256 signature header
loses its continuation lines or gains extra leading spaces. Re-encoding a
signed tag drops the trailing signature or merges it into the message,
an unsigned tag emits an empty tagger line, and the signature-excluded
payload no longer matches the originally signed bytes so verification
fails.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
