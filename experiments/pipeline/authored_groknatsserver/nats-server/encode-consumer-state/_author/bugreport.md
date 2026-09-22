Consumer state snapshots no longer round-trip.

A consumer with empty pending and empty redelivered used to write a short binary blob that another server could decode back to the same ack floor and delivered sequences. After the last change `EncodedState` returns nothing, so a restarted consumer forgets its ack floor and redelivers everything. The on-disk files that used to start with a two-byte header (magic then version) are now empty.
