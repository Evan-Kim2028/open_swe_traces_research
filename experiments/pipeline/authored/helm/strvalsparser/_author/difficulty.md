# Difficulty — strvalsparser

predicted_flip: L4
A hand-rolled recursive-descent parser with a dozen interacting rules: escape semantics differ between normal/literal/file modes, list-index growth, the duplicate-index error, the nested-level cap, and type inference only in non-string modes. The fuzz test means any grammar deviation can panic. A mid model writes a plausible parser but fails edge cases; it typically needs test names and signatures (L4) to enumerate the grammar.

Hardness driver: hand-rolled grammar with mode-dependent semantics.
