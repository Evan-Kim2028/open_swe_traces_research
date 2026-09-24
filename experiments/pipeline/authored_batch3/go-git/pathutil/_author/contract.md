# Contract — pathutil

Hidden suite: `tests/hidden/internal/pathutil/pathutil_bb_test.go`
(package `pathutil`, in-package). One `TestDetailNN` per DETAILS.md line.
All assertions are pure predicate calls — no I/O.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | `IsDotGitName` true for `.git`, `.GIT`, `.Git`, `git~1`; false for `git~2`, `GIT~9`, `.gitx`, `git`, `.gi`. |
| TestDetail02 | 2 | partially | `IsHFSDotGit` matches `.git`, `.GIT`, `.g<U+200C>it` (ignored codepoint skipped); rejects `.gitx`, `.gi`; `IsHFSDotGitmodules(".gitmodules")` true while `IsHFSDotGitattributes` false — needle-specific. |
| TestDetail03 | 3 | partially | `IsNTFSDotGit` true for `.git`, `.git `, `.git.`, `.git  ..`, `.git::$INDEX_ALLOCATION`, `.git:x`; false for `.gitx`, `.gitx:x`, `.gi`, `.gitmodules`. |
| TestDetail04 | 4 | no | SHAPE: `IsNTFSDotGitmodules("gitmod~N")` for N=1..9 hits a bounded NON-EMPTY digit range — some digits match, some don't; the exact bound is not pinned. |
| TestDetail05 | 5 | partially | `WindowsValidPath` rejects `CON`, `con.txt`, `NUL:x`, `LPT3 `, `com1`; accepts `CONTACT`, `console.log`, `nullex` — base-name comparison, not prefix. |
| TestDetail06 | 6 | no | SHAPE: plain `.git` and `git~1` are VALID to `WindowsValidPath`; `.git::$INDEX_ALLOCATION` and `CON` are not. |
| TestDetail07 | 7 | doc | `ValidTreePath` rejects `""`, `/`, `//`, `a/./b`, `a/../b`, `..`, control byte `a/b\x01c`, and `.git` disguises at any component (`a/.git/b`, `.GIT/x`, `a/git~1`); accepts `a/b/c`, `a/CON/b`, `normal.txt`, `.gitignore`. |
| TestDetail08 | 8 | doc | `HasUnsafeComponent` true for `a/../b`, `a\..\b`, `a/b\x01c`, `nodot\x7f`, `..`; false for `a/b/c`, `a/gitx`, `no-dots-here`, `a/..b`, `a/b..`. |
| TestDetail09 | 9 | doc | `HasUnsafeComponent` true for canonicalised `..` forms: `a/.. /b`, `a/.<U+200C>./b`, `a/..::$I/b`, `a/..:/b`. |
| TestDetail10 | 10 | no | SHAPE: `ReplaceTildeWithHome` returns `~`, `~x`, and ordinary paths unchanged (or with error); a failed `~user` lookup returns the ORIGINAL string alongside the error. |
| TestDetail11 | 11 | doc | `ValidTreePath("a\\.git")` fails and `HasUnsafeComponent("x\\..\\y")` fires — `\` is a separator on Linux too. |
| TestDetail12 | 12 | partially | `IsNTFSDot("gitmod~1", ".gitmodules", "gitmod")` true; `IsNTFSDot("other~1", …)` false — the caller-supplied prefix controls the match. |

Refusals/softening: line 7's volume-name-prefix clause is NOT asserted —
`filepath.VolumeName` is a no-op off Windows so the clause is unobservable on the test
platform; recorded here. Line 4 pins a bounded digit range only (committed: `~1`–`~4`), not
the exact bound. Line 10 asserts value-preservation on failure, not which lookup fails.
