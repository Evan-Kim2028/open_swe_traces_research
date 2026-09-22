# DETAILS — gitrefs

1. CreateRef rejects an empty `Ref` and an empty `SHA` before any
   request is built. Inferable: partially — required-field validation
   is derivable; the error literals are arbitrary (assert error, not
   text).
2. CreateRef normalizes `body.Ref` to begin with `refs/` (idempotent —
   an already-prefixed ref is unchanged). Inferable: partially — the
   prefix convention is Git-documented; enforcing it client-side is an
   arbitrary normalization.
3. UpdateRef rejects empty `ref` and empty `body.SHA` before building
   the request. Inferable: partially — same guard rationale.
4. UpdateRef and DeleteRef strip a leading `refs/` from `ref` before
   URL-escaping it into the route (so `refs/heads/x` and `heads/x`
   resolve identically). Inferable: partially — the route embeds the
   ref sans prefix; escape behavior is derivable.
5. The ref is path-escaped so `refs/heads/foo bar` and similar can't
   break the URL. Inferable: yes — refURLEscape exists for this.
