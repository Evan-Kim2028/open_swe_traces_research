# Exported API — createcommit

`GitService.CreateCommit(ctx, owner, repo, commit, opts)` POSTs a commit
object. The `opts` (`*CreateCommitOptions`) may carry a `Signer` that
produces the commit signature; `commit.Verification.Signature` may
carry a pre-computed one. Callers rely on:

- `opts == nil` being legal.
- `body.Signature` populated from `commit.Verification.Signature` when
  present, else from `opts.Signer` when present.
