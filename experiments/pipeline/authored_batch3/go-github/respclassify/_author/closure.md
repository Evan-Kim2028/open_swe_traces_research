# Closure — respclassify

Package: github (root). File: github/github.go.
Removed bodies: CheckResponse's classification switch (stub returns the
generic ErrorResponse for everything non-2xx) and parseSecondaryRate.
Kept: 2xx early return, body read/restore prelude, error types, parseRate.
Tests removed: ~40 funcs across 14 test files — the CheckResponse unit
tests plus every service test exercising a redirect, accepted, OTP,
primary/secondary rate-limit, or unexpected-structure path (artifact
downloads, workflow logs, archive links, SBOM fetch, followRedirects and
withRateLimits subtests). See spec respclassify.json for the full list.
