# Closure — testroot

Package: `utils` (`tests/utils`). Files: `tests/utils/helpers.go`.

Removed (1 function stubbed): `RequiresRoot`.

Kept: `enableRoot` and the `init` that registers `-test.root` (required by
the hidden suite; excising init would poison every import of the package),
both imports.

Tests deleted: `tests/dmflakey/dmflakey_test.go`,
`tests/robustness/main_test.go`, `tests/robustness/powerfailure_test.go` —
each package's `TestMain` calls `RequiresRoot` unconditionally, so the
excised stub would panic the test binaries. Deleting whole root-gated
suites loses no runnable coverage: without `-test.root` they exit 0 before
running a single test.
