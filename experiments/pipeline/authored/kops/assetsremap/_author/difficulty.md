# Difficulty — assetsremap

predicted_flip: L4
The proxy vs registry rewrite lattice is easy to under-specify (hub vs host detection, slash-to-dash flattening, idempotent second pass), and hash lookup has a getAssets URL switch plus success-only cache. Concurrent snapshot getters need the mutex+copy+sort. Mid models get docker-hub prepend and miss convergence or comma escaping; L4 test names separate those cases.

Hardness driver: two-layer image rewrite + concurrent collection + hash cache polarity.
