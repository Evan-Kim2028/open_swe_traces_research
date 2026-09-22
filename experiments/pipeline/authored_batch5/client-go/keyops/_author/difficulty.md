# Difficulty — keyops

predicted_flip: L0
details: 8

Missed edges: all-`0xFF` carry → empty (not `{0xFF,0}`); empty input →
empty; `IsFollowerRead` is "anything but leader" including mixed/learner.

Hardness driver: small file, but the carry-out and empty-key corners of
`PrefixNextKey` are routinely gotten wrong; everything else is shallow.
