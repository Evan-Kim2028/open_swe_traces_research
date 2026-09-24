# Difficulty — filemodes

predicted_flip: L4
details: 4
Commitments: octal parse/format with asymmetric spellings (parse has no leading-zero requirement, format adds one), perm-bit masking with Lstat, honest changed-flag reporting, and missing-file-as-mismatch hash semantics.

Hardness driver: small closure but every line carries a choice — default-on-parse-error, `ModePerm` masking vs full mode, and the not-exist-isn't-an-error rule.
