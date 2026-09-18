# Missing behavior

A blank (or whitespace-only) pattern matches no path. The patterns `*` and `~`
match every path.

A pattern with no `*` and no leading `~` matches that path and only that path.
Backslashes in the candidate are treated as slashes.

A pattern starting with `~` is the rest of the string as a regular expression.
Compile failure is an error.

The pattern `TEST` matches names ending in `_test.go` and does not match names
like `_test_no.go`.

`*` in a glob matches any run of non-slash characters. `a/b/*.pb.go` matches
files in `a/b` whose names end in `.pb.go`, not `*.nopb.go`.

`**` matches across slashes, including zero directories (`a/**/*.pb.go` matches
`a/xxx.pb.go` and `a/x/y/z/yyy.pb.go`). A `**` glued to non-slash characters on
both sides is invalid.

After a rule's exclude list is compiled, a file is skipped for that rule iff any
pattern matches its path. An empty exclude list never skips. Loading a
configuration document compiles each rule's exclude list so later membership
tests use those compiled patterns. An invalid exclude is a configuration error
rather than a silent match.

Worked cases the hidden tests assert:

- Exact path `a/b/c.go` matches that path and not `a/b/d.go`.
- `~b/[cd].go$` matches `a/b/c.go` and `b/d.go`, not `b/x.go`.
- `TEST` matches `a/b/c_test.go` and not `a/b/c_test_no.go`.
- `a/b/*.pb.go` matches `a/b/xxx.pb.go` and not `a/b/xxx.nopb.go`.
- `a/**/*.pb.go` matches `a/xxx.pb.go` and `a/x/y/z/yyy.pb.go`.
- Empty pattern matches none of those paths; `*` and `~` match all of them.
- A rule with no excludes still runs on a fixture file; a rule whose exclude is
  that fixture path does not.
- Loading a document with a rule-level exclude compiles it so `some/file.go`
  matches and `some/any-other.go` does not.
- A bad exclude string is a configuration error.

A matcher that always returns false is wrong: expected `a/b/c.go` to match
itself, actual no match.

Coverage the hidden checks enforce:

- Exact, regexp, TEST, `*`, `**`, empty, and match-all patterns behave as specified.
- A rule is applied unless a compiled exclude matches the file under test.
- Loading a document with a rule-level exclude compiles it so one path matches
  and a sibling path does not.

Reproduce with:

```
go test -count=1 -timeout 15m ./lint/ ./test/ ./config/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
