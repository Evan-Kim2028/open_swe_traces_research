# Bug report — subjvalid

**Title:** Subject validation accepts embedded wildcards as literal;
collision check misses partial-token semantics.

**Symptoms:**
- `subjectIsLiteral("foo.*bar")` returns false, so a publish to the
  literal subject `foo.*bar` is rejected as wildcard.
- `isValidSubject` treats a `>` mid-subject (`foo.>.bar`) as valid and
  accepts empty tokens (`foo..bar`), letting malformed subscriptions
  into the sublist.
- `tokenAt` treats the index as 0-based, so API helpers
  `streamNameFromSubject`/`consumerNameFromSubject` read the WRONG
  token of `$JS.API.STREAM.INFO.<stream>` — stream lookups land on the
  consumer token or fail.
- `SubjectsCollide` declares `foo.*` and `foo.*.bar.>` colliding (it
  compares tokens pairwise without the length/`>` rules), producing
  false-positive interest conflicts.

**Reproduction:** `IsValidPublishSubject("foo.*bar")` should succeed —
the `*` is inside a token — but is rejected; and `tokenAt(subject, 5)`
returns token 4.

**Expected:** whole-token wildcard semantics, strict `>`-final
placement, 1-based token indexing, and the collide length rules.
