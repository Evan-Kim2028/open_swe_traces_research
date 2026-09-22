# difficulty — consumerstate

Medium-hard. Two record families on opposite sides of the store boundary.
Subtleties a cheat misses: pending/redelivered seqs are deltas against the
ack floors (not absolute), timestamps are second-resolution offsets from a
shared mints (negative deltas legal), v1 has no pending dseq and adjusts
Delivered by floor-1, redelivered seq==0 is skipped not stored, the
deleted-block tail is magic-dispatched per block, and sources sit between
the scalar header and the deleted tail.
