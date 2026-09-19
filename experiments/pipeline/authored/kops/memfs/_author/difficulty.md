# Difficulty — memfs

predicted_flip: L3
The surface is CRUD on a tree, but exclusive-create vs write, ReadTree’s “leaves only” (HasChildren), and nested mutex Join are easy to get wrong. Three tests name those behaviors; L3 is enough.

Hardness driver: exclusive create + leaf-only tree walk under nested locks.
