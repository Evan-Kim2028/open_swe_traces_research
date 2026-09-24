# Exported API — signmsg

`GitService.CreateCommit` accepts a `MessageSigner`; when provided, the
commit is serialized to Git's canonical payload and signed, and the
resulting armored signature rides along in the commit object.

- `createSignature` (unexported, excised) — renders the payload and runs
  the signer.
- `createSignatureMessage` (unexported, excised) — serializes tree,
  parents, author, committer and message into the signed text.
- `MessageSigner` interface — kept; the injection point.
