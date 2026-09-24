# Difficulty — codecbytes

predicted_flip: L2
details: 8

Missed edges: the terminating all-pad group after an exact-multiple input;
empty input still emits a group; marker `0xFF - padCount` vs padCount itself;
the three distinct decode rejections; `buf` reuse semantics.

Hardness driver: the grouped-padding format is simple to describe but the
terminator rule and the marker arithmetic are routinely off-by-one; decode
validation order matters for which error fires.
