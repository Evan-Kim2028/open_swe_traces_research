# Details — subjvalid

1. `isValidSubject`: empty subject → false; per `.`-separated token —
  empty token → false; whitespace char in a token → false; `>` allowed
  ONLY as a whole final token (`sfwc` latches and any further token
  fails); a `*` or `>` embedded in a longer token is LITERAL
  (`foo*`/`foo**`/`foo.*bar` are valid). `checkRunes` additionally
  rejects NUL bytes and `utf8.RuneError`. Inferable: partially —
  TestSublistValidLiteralSubjects pins the table, the whole-token rule
  is the surprise.
2. `subjectIsLiteral` is byte-scanning, not token-splitting: `*`/`>`
  make it non-literal ONLY when the char is a complete token — bounded
  by `.` or string edge on BOTH sides (`foo*bar` literal, `foo.*bar`
  literal, `*` non-literal). Inferable: partially — pinned by
  TestSubjectIsLiteral.
3. `isValidLiteralSubject`/`IsValidLiteralSubject`: reject empty
  tokens AND any single-char `*`/`>` token — multi-char tokens pass
  unconditionally (no whitespace check, unlike isValidSubject).
  Inferable: partially — asymmetry with #1 is internal.
4. `tokenAt` is 1-BASED: index 0 and out-of-range → `_EMPTY_`; it
  scans for the index-th `.`-bounded run, returning the tail token
  when the last separator precedes it. Inferable: yes — TestSubjectToken.
5. `ValidateMapping`: empty dest → nil; per-token empty/`>`-position
  rules as #1 but reporting `mappingDestinationErr{token}`; a token
  wrapped as `{{name(args)}}` must match one of the eight
  mapping-function regexes else `ErrUnknownMappingDestinationFunction`;
  finally `NewSubjectTransform(src, dest)` must succeed. Inferable:
  partially — function list is internal.
6. `SubjectsCollide`: equal strings collide; both-literal → equality;
  one-literal → `isSubsetMatchTokenized`; both-wildcard → token-count
  rules (partials-only must be equal length; shorter-without-`>` never
  collides) then pairwise `tokensCanMatch` where a whole-token `*`/`>`
  matches anything else literal equality. Inferable: partially —
  the length rules are the internal contract.
7. `tokensCanMatch`: empty → false; a LEADING `*` or `>` char → true
  (position-only check — `*foo` counts as a wildcard here); else
  string equality. Inferable: partially.
8. `numTokens` counts separators+1 (empty→0). `tokenizeSubjectIntoSlice`
  is intentionally retained — the identical gsl copy is banked.
  Inferable: yes.
