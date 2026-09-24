# Difficulty — objfile

predicted_flip: L2
details: 12

Missed edges: shared 32-byte budget over both fields; reader accepting negative size; overflow
truncating but still committing bytes + reporting error; exhausted-pending write committing 0
bytes with overflow; writer panic on pre-header write vs reader's dedicated error; zero hash
sized by object format; cached close error.

Hardness driver: several guard asymmetries between reader and writer plus hash semantics that
depend on NewHasher seeding.
