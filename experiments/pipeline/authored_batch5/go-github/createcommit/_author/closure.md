# Closure — createcommit

Package: github (root). File: github/git_commits.go.
Removed bodies: GitService.CreateCommit — stubbed to a variant that
never defaults a nil `opts` and never signs: drops the
`opts == nil → &CreateCommitOptions{}` guard and the signature block
(`commit.Verification.Signature` wins over `opts.Signer`, else the
signer runs `createSignature` over the request body). Parents SHA
extraction, Tree wiring, and the POST keep working; a nil `opts` now
panics and sign options are silently ignored.
Kept: createSignature, createSignatureMessage, all Commit fields.
Tests removed: 2 funcs in github/git_commits_test.go
(TestGitService_CreateCommit_WithSigner,
TestGitService_CreateSignedCommit).
