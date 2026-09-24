# Difficulty — fetchmsg

predicted_flip: L2
details: 16

Missed edges: hash-sort on encode; empty-wants early error; fixed flag
sequence; decode terminating on delim/EOF/blank as well as flush; silent
skip of unknown lines; lenient deepen-relative-with-arg; section rank
ordering; ready→delim pairing; NAK suppression when ACKs exist;
packfile-uris newline-only trim.

Hardness driver: a grammar whose doc comment shows the happy path while the
strict ordering, terminator pairing, and error-vs-leniency choices live in
the excised bodies.
