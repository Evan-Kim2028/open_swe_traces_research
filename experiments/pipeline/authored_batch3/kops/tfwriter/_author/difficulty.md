# Difficulty — tfwriter

predicted_flip: L2
details: 10
Commitments: the exact legalizer map (incl. `/`→`--` double dash and `prefix_` for digit-first), collision detection on legalized names at query time, scalar/array output conflict with asymmetric messages, legalized-key dedup in outputs, sorted+deduped arrays, fatal-vs-error provider conflicts, and file staging path conventions.

Hardness driver: collisions happen on the LEGALIZED name, not the raw one (two raw names, one error); the two output-conflict errors are asymmetric in a way a unified implementation gets wrong; `/`→`--` is a spelling nobody guesses.
