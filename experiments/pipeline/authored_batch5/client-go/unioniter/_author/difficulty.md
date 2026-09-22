# Difficulty — unioniter

predicted_flip: L2
details: 6

Missed edges: empty dirty values are tombstones (a merge that ignores them
emits deleted records); equal keys advance the snapshot WITHOUT emitting it
(dirty wins); the tombstone-ahead-of-snapshot path logs a warn and skips
dirty only; children must already be direction-ordered.

Hardness driver: the merge loop is a state machine where the wrong branch
compiles and passes plain-merge cases — tombstone handling is invisible in
the signatures.
