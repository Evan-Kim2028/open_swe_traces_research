# Exported API — brnotprot

Branch-protection helpers translate GitHub's "branch not protected" API error
into a package sentinel so callers can `errors.Is` it.

- `ErrBranchNotProtected` — sentinel returned by `GetBranchProtection`,
  `GetRequiredStatusChecks`, `ListRequiredStatusChecksContexts`, and
  `GetSignaturesProtectedBranch` when the branch has no protection.
- `githubBranchNotProtected` (unexported const) — the upstream message string.
- `isBranchNotProtected` (excised body) is consulted by the four callers above
  on every API error.
