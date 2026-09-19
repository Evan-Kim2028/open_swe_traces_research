# Contract (L2) — ignorerules

A ruleset decides which files under a chart directory are excluded from packaging. Rules are parsed one per line; blank lines and `#` comments are skipped, and a malformed pattern (bad character class, bad escape) aborts parsing with an error. AddDefaults adds the built-in ignores (version-control dirs, OS cruft, editor backups, .proj/.idea/.vscode, ownership files). Matching is name-based against the path relative to the chart root: a pattern without a slash matches the basename at any depth; a pattern with a slash anchors to the root; `*` does not cross `/` while `**` does; a trailing `/` marks a directory-only pattern that matches only directories; patterns match directories themselves, not just files under them. There is no negation: a `!` is a literal character, not an un-ignore. Matching considers whether the path is a directory from the file info.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestParse` | rule file parsing |
| `TestParseFail` | malformed patterns error |
| `TestParseFile` | load rules from file |
| `TestIgnore` | match semantics incl. dir-only and ** |
| `TestAddDefaults` | default rules populated |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/ignore/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestIgnoreContractTableProperty`, `TestIgnoreParseAndDefaultsProperty`, `TestIgnoreOracleAgreementProperty`, `TestIgnoreUnseenRandomProperty`: TestIgnoreContractTableProperty

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
