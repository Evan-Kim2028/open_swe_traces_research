# Difficulty — osmetadata

predicted_flip: L3
The sequence contract is small but strict: comma-split order, first success, last error, blkid fallback, iso9660-then-vfat mount. Nine tests already enumerate the matrix; a mid model with the contract plus those names (L3) should pass.

Hardness driver: ordered multi-source fallback with last-error propagation.
