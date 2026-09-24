# Closure — brnotprot

Package: github (root). File: github/repos.go.
Removed bodies: isBranchNotProtected — stubbed to `return false` (no error is
ever recognized as not-protected, so the raw error propagates).
Kept: githubBranchNotProtected const, ErrBranchNotProtected sentinel, all four
call sites.
Tests removed: 4 funcs in github/repos_test.go
(TestRepositoriesService_{GetBranchProtection,GetRequiredStatusChecks,
ListRequiredStatusChecksContexts,GetSignaturesProtectedBranch}_branchNotProtected).
