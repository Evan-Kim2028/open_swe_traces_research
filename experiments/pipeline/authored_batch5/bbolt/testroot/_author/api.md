# Exported API — testroot

Package `utils` (module `example.internal/boltstore`, `tests/utils/`) — the
privilege gate shared by the root-requiring test suites.

`RequiresRoot()` — called from `TestMain` of the dmflakey and robustness
suites before `m.Run()`. Decides whether the process continues, skips, or
fails, based on the `-test.root` flag (registered by the package's `init`,
which is kept) and the process's effective uid.

Callers: `TestMain` in `tests/dmflakey` and `tests/robustness`. In-tree
tests removed: 3 (`tests/dmflakey/dmflakey_test.go`,
`tests/robustness/main_test.go`, `tests/robustness/powerfailure_test.go`) —
they invoke the excised function directly.
