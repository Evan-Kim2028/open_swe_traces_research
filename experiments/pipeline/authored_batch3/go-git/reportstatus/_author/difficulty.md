# Difficulty — reportstatus

predicted_flip: L2
details: 10

Missed edges: decode accepting any `unpack` status and failing only through the error view;
`ng` requiring its reason field while `ok` forbids extra fields; only the FIRST command error
being reported; the empty-status string counting as a failure on both directions; the required
flush terminator.

Hardness driver: two-stage error surface (decode vs error view) plus strict field-count grammar.
