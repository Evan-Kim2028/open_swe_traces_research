# Difficulty — coalesce

predicted_flip: L4
The merge direction rules are genuinely subtle: user-nil erases but subchart-nil must not shadow globals, alias-aware dependency traversal, coalesce-vs-merge modes, and cleanup ordering. Four of the in-tree tests exist only for the nil/global corners. A mid model writes a straightforward deep-merge and fails those corners; L4 is where the surviving corners get named.

Hardness driver: deep merge with asymmetric nil semantics.
