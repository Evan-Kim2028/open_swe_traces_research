# Missing behavior

Constructing a runner from a configuration, an optional "treat all findings as
failing exit", a max-open-files cap, and extra rules yields an object that can
scan path patterns and render the resulting findings.

Extra rules whose names are not already in the configuration are inserted with
their default configuration. Scanning accepts include and exclude patterns; if
the caller passes no excludes, the configuration's excludes are used; if those
are also empty, `vendor/...` is excluded. Includes default to the working
directory when none remain after trimming. The result is a channel of findings.

Rendering named after a known formatter drains the channel, drops findings below
the configured confidence, and computes an exit code: 0 if nothing passed the
filter; the warning code on the first kept finding; the error code if any kept
finding is configured as error. With "set exit status" construction, both codes
are 1, so any kept finding exits 1. The formatter runs concurrently with the
drain.

Worked cases the hidden tests assert:

- Scanning the if-return sample under default rules plus if-return produces five
  findings.
- Stylish rendering of those five findings contains each file/line/rule/message
  and exits 1.
- Construction with set-exit-status and an extra named rule loads configuration,
  that extra rule, file-cap 2048, and codes 1/1.

A constructor that returns a nil runner is wrong: expected five findings from
the if-return sample, actual a nil channel or error.

Coverage the hidden checks enforce:

- Scanning the if-return sample under default rules plus if-return produces five findings.
- Stylish rendering of those five findings contains each file/line/rule/message and exits 1.
- Construction with set-exit-status and an extra named rule loads configuration, that extra rule, file-cap 2048, and codes 1/1.

Reproduce with:

```
go test -count=1 -timeout 15m ./revivelib/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
