# Bug report

Submodule and branch configuration handling is unsafe and lossy.
Suspicious submodule names — ones containing parent-directory
components disguised by Unicode or platform filename tricks — are
accepted instead of skipped, while names that merely look odd are
wrongly refused. Round-tripping .gitmodules loses unknown options and
submodule names. Branch sections write empty remote and merge keys
instead of removing them, multi-line descriptions come back corrupted,
and a rebase value that real git accepts is rejected. Boolean config
values such as "yes", "on", or a number are misread — "on" becomes
unset and an empty value becomes false instead of unset.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
