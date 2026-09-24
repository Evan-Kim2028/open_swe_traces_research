# Bug report

Delta generation produces streams that either fail to apply or apply to
the wrong bytes. Long identical regions are re-emitted as literals instead
of copies (packs balloon), copy instructions over the opcode size limit
are silently dropped, and literal runs longer than a single opcode's
capacity are written with a corrupt length field. Targets sharing only a
short tail with the base lose the tail entirely. Worst case: hashing the
same repeated block over and over makes generation take quadratic time and
hang on large repetitive files.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
