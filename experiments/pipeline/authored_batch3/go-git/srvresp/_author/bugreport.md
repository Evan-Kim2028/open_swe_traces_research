# Bug report

The acknowledgement replies exchanged while negotiating common objects
during fetch are misread. A refusal reply is surfaced as an error instead of
ending the exchange, replies carrying several acknowledgements are cut off
after the first or lose their status flags, and an acknowledgement whose
status word is not one of the known ones is rejected instead of read. On the
write side an empty acknowledgement set produces an empty message rather
than a refusal, and a response that carries no status flags still emits one
line per acknowledgement instead of stopping after the first.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
