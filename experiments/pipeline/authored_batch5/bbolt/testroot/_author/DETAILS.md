# Details — testroot

1. With `-test.root` unset or false, `RequiresRoot` does NOT return to its
   caller: it prints a skip notice and terminates the process with exit
   code 0. Inferable: partially — "skip tests that require root" is the
   documented intent; terminating the whole test binary with success is
   the mechanism, not stated.
2. With `-test.root` set and the process running as a non-root uid, it
   prints a failure message and exits with a non-zero code. Inferable:
   partially — a privilege guard that exits non-zero on missing privilege
   is the point; the exact code and text are not stated.
3. With `-test.root` set and uid 0, it returns normally — no output, no
   exit — letting the test suite proceed. Inferable: doc — the flag's
   documented purpose is "enable tests that require root".
4. All diagnostic output goes to stderr, not stdout. Inferable: no.
5. The decision reads the package-level flag value and the live uid each
   call — no caching, no error return, no panic. Inferable: partially —
   the no-error signature is visible; the no-cache property is not.
