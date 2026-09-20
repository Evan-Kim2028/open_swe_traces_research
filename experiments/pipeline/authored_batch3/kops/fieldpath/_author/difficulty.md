# Difficulty — fieldpath

predicted_flip: L2
details: 10
Commitments: the four element kinds and their canonical spellings, `/` as a second separator, empty-segment collapsing, strict vs prefix matching, wildcard-on-the-receiver-side matching, and error/fatal boundaries.

Hardness driver: the wildcard direction (`[*]` on the pattern side matching `[0]` on the target side) reads backwards and invites an inverted implementation; `/` separators and `.` collapsing are invisible in any reasonable grammar guess.
