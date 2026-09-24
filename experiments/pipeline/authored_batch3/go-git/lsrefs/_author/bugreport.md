# Bug report

The v2 reference-listing exchange used when a client asks a server what it
advertises is broken. Outgoing argument lists are emitted in the wrong
shape — flags ride on the wrong lines, and invalid prefixes are written
instead of rejected before anything is sent. Incoming answers are
misread: peeled entries come back as separate references instead of
folding into their base, an entry pointing at a target is returned as a
plain hash entry with its target forgotten, and the server's marker for a
reference that does not exist yet is accepted even when it lacks the
required target field. On the encode side peeled entries get their own
lines and entries without a known hash lose their marker entirely.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
