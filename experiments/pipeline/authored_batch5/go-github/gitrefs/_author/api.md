# Exported API — gitrefs

Ref mutation on `GitService`:

- `CreateRef(ctx, owner, repo, CreateRef{Ref, SHA})` — POST; both fields
  required; `Ref` is normalized to carry the `refs/` prefix.
- `UpdateRef(ctx, owner, repo, ref, UpdateRef{SHA, Force})` — PATCH;
  `ref` and `SHA` required; `ref` is stripped of any `refs/` prefix and
  URL-escaped into the route.
- `DeleteRef(ctx, owner, repo, ref)` — DELETE; `ref` trimmed of `refs/`
  and escaped the same way.
