# Difficulty — updreq

predicted_flip: L2
details: 12

Missed edges will be: the shallow-only flush being a *valid* empty request while a bare flush is
malformed (two empty outcomes), capabilities living only on the first command line behind a NUL,
the space written before the capability string on encode, the ref name keeping interior spaces
rather than truncating at the second space, both-ids-zero being `invalid` rather than create, and
40-or-64-hex acceptance on every id field.

Hardness driver: several independent empty-input and first-line special cases on one codec.
