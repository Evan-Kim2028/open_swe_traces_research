# Closure — signmsg

Package: github (root). File: github/git_commits.go.
Removed bodies: createSignature and createSignatureMessage — stubbed to
return the invalid-parameters error unconditionally.
Kept: createCommit/Commit types, MessageSigner interface, CreateCommit
service method.
Tests removed: 6 funcs in github/git_commits_test.go —
CreateSignedCommit, CreateSignedCommitWithInvalidParams,
CreateCommit_WithSigner, createSignature_nilSigner,
createSignatureMessage_withoutCommitter,
createSignatureMessage_withoutTree.
