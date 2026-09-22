# DETAILS — urlpolicy

1. A URL whose path contains a literal `..` segment is rejected with
   ErrPathForbidden; `..` inside a segment (file..txt) or in the query is
   fine. Inferable: doc — the doc comment spells out the rule.
2. Origins compare equal case-insensitively on scheme and hostname, with
   explicit ports normalized against scheme defaults (http→80, https→443,
   others none). Inferable: partially — normalization rule is documented,
   the port table is conventional.
3. An empty allow-list means the package default origins, not "allow all".
   Inferable: doc — the doc comment states it.
4. Credentials may flow to the configured base or upload origin; anything
   else is out of scope. Inferable: doc — matches the doc comment.
5. Sending a body to an out-of-scope destination is refused with
   ErrUntrustedDestination carrying the redacted URL. Inferable:
   partially — refusal is documented, the wrapped error shape is
   arbitrary.
