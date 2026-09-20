# Difficulty — indexenc

predicted_flip: L2
details: 12

Missed edges: (name,stage) sort discarding caller order; flag saturation at 0xFFF; extended
flags existing only when needed AND counting toward padding; padLen ≥ 1 keeping the name
terminated; v4 strip-length measured from END of previous name; skip-hash writing zero bytes;
negative-nsec check on top of negative-sec.

Hardness driver: layout is partially legible from the intact decoder but the version branches,
padding arithmetic and trailer choices are not.
