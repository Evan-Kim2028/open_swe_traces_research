# Difficulty — jsonstream

predicted_flip: L2
details: 10
Commitments: push/pop state machine over `{ [ F`, two-space indent tracking, deferred-comma output (rewrite `,\n`→`\n` on close), field-name vs field-value split, path stack maintenance, unescaped string emission, and error states for top-level scalars / unknown tokens.

Hardness driver: the `F` pseudo-state interacting with close-delims (a `}` must pop a pending `F`) is the classic off-by-one-stack bug; deferred commas read backwards at first glance; top-level scalars erroring is easy to "fix" wrongly.
