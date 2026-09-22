# Exported API — apiversion

Requests can pin a GitHub API version; the client rejects versions outside its
supported window before any network call.

- `ErrUnsupportedAPIVersion` — sentinel error returned for an out-of-range
  request version (`errors.Is`-compatible).
- `WithVersion(version)` request option — sets the `X-GitHub-Api-Version`
  header on a single request.
- `Client` fields `apiVersionMin` / `apiVersionMax` / `apiVersionDefault`
  (unexported) — supported window, defaults `"2022-11-28"`..`"2026-03-10"`,
  overridable via client options.
- `checkRequestAPIVersionBeforeDo` (excised body) is invoked by `bareDo` for
  every outgoing request.
