# Difficulty — kopscodecs

predicted_flip: L2
details: 9
Commitments: default version selection, unstructured-vs-typed encode split, media-type lookup error, two-stage decode (unstructured GVK sniff then typed decode), legacy-group apiVersion rewriting at the byte level, and non-clustkit docs falling back to unstructured.

Hardness driver: the group→decode routing has three outcomes (typed, rewritten-then-typed, unstructured) and the rewrite is a surgical line-level string operation with a dead-looking code path; the unstructured encode bypass looks wrong until you understand version remapping is meaningless for it.
