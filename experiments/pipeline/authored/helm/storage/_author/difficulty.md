# Difficulty — storage

predicted_flip: L3
Looks like plain CRUD but the invariants are the trap: newest-per-name listing vs per-name history, deployed-status protection during pruning, prune-failure tolerance, skip-corrupt-rows behavior, and the exact key format. A mid model writes the obvious implementation and fails the pruning/status edge tests; L3 names reveal them.

Hardness driver: sequence contract over revision history.
