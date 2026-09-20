# Difficulty — inforefs

predicted_flip: L0
control: true
details: 11

This is the batch's control unit: the `Decode` doc comment spells out nearly every commitment
(reject-whole-body, skip-bad-name, `^{}` preservation, scanner limit, CR handling). If it fails
at L0 anyway, the failure is interesting — it means the solver did not read the doc comment.

Hardness driver: none by design — a documented-spec control against the batch's blind units.
