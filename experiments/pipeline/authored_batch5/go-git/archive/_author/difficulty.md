# Difficulty — archive

predicted_flip: L2
details: 12

Missed edges: refname-only restriction; first-colon path split and
dir-only subtree; heads-before-tags resolution; tag-chain unwrapping;
tree-vs-commit timestamp rule; PAX global header with commit comment;
umask math per file kind and 0777 symlinks; explicit prefix dir entry;
64KiB symlink bound; filter match taxonomy + no-match error; prefix
traversal rejection; zip comment + symlink mode bit.

Hardness driver: plain tar-writing code produces a valid archive that
extracts — everything here is metadata correctness (modes, headers,
commit id) that only fails on inspection or round-trip.
