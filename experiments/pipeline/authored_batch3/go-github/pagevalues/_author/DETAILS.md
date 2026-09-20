# DETAILS — pagevalues

1. A link carrying a `cursor=` param sets `Response.Cursor` only when its rel
   is `next`; other rels with a cursor are ignored for Cursor (and the link is
   not consulted for page fields at all). Inferable: no — the
   cursor-implies-next-only rule is arbitrary.
2. A link with a `since=` param but no `page=` is treated as `page=since` —
   `since` is a fallback, so `since=2&rel="prev"` yields PrevPage 2 while
   `since=2021-12-04T10:43:42Z&page=4&rel="next"` still yields NextPage 4.
   Inferable: no — treating since as an alias for page is arbitrary.
3. Links where none of `page`, `before`, `after`, `since` is present are
   skipped entirely. Inferable: partially — skipping empty links is derivable,
   but which params qualify is arbitrary.
4. A `page=` value that fails integer parse populates `NextPageToken` instead
   of `NextPage` (which stays 0). Inferable: partially — a token fallback is a
   reasonable guess; the exact field is not derivable.
5. Malformed links — fewer than two `;` segments, or an href not wrapped in
   `<>` — are skipped, not errors. Inferable: partially — lenient skipping is
   derivable, the exact validity shape is not.
6. `before=`/`after=` params land on `Response.Before`/`Response.After` from
   their respective `prev`/`next` rels regardless of numeric parsing.
   Inferable: partially — presence is derivable, the rel pairing is arbitrary.
