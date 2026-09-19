# Difficulty — chartloader

predicted_flip: L3
Three independent entry paths converge on one buffered-file loader; the hard parts are the implicit invariants — required Chart.yaml, BOM stripping, path-traversal/backslash rejection, symlink/device skipping, the size budget, and order-sensitive template merge. A mid model can rebuild the skeleton from the contract but will miss at least one edge (budget accounting or traversal rules) without test names; L3 gives those away.

Hardness driver: multi-file closure with security-sensitive path rules and order-sensitive state.
