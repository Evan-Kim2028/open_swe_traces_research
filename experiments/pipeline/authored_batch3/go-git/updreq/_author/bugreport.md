# Bug report

The message codec used on the push side of the wire protocol is broken.
Decoding mishandles what the client sends: shallow lines that precede the
commands are dropped or rejected, an empty push that carries only shallows
fails, and a command whose name contains spaces is truncated at the second
space. Capabilities the client negotiates are lost, truncated input is
accepted as a complete message, and trailing bytes after the terminator are
ignored. Classifying a command gives the wrong action for zero object ids,
so creates and deletes come back as updates or outright errors.

Encoding emits every command as a bare line — the negotiated capabilities
never make it onto the first line, shallow lines are never written, and the
message sometimes lacks its terminator.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
