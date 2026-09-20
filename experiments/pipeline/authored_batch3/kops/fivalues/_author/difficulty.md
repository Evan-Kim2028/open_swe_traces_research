# Difficulty — fivalues

predicted_flip: L4
details: 8
Commitments: nil→zero-value semantics across six shapes, nil-element skipping, pointer aliasing into the input slice, DebugPrint's five-way type dispatch with sentinel strings, error-swallowing conversions, and error-as-string JSON fallbacks.

Hardness driver: many tiny functions each with an edge case (typed-nil vs nil-interface; nil-slice vs empty-slice; alias vs copy) — the "obvious" implementation fails the typed-nil and truncation cases.
