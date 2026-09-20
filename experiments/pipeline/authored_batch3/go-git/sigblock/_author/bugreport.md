# Bug report

Signature verification and re-encoding are broken. Payloads whose
signature block sits at the end are split at the FIRST signature-looking
line instead of the last, so message text is mislabeled as signature.
Signature markers mid-line count when they should not. Objects signed
twice are accepted instead of flagged, a second signature header right
after the first survives stripping, and lines inside the body that happen
to look like signature headers are removed. For tags, the trailing
inline signature is left in the verification payload, so verification
fails on legitimately signed tags.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
