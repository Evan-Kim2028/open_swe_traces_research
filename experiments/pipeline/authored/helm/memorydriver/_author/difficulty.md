# Difficulty — memorydriver

predicted_flip: L3
The surface is small but the state contract is dense: version-sorted record sets, newest-per-name semantics for List/Query, exact-key vs newest semantics, replace-in-place ordering, and read/write locking. A mid model will get CRUD right but botch the sorted-insert invariants or List-vs-Query selection; test names at L3 pin each one.

Hardness driver: concurrent state store with ordering invariants.
