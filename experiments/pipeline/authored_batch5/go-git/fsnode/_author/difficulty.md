# Difficulty — fsnode

predicted_flip: L3
details: 12

Missed edges: hash = blobHash+modeBytes; 24-zero dir hash; index hash
reuse gated on metadata match; racy-git mtime guard; missing index
ModTime forcing content hash; CRLF size adjustment; symlink target
hashing; submodule commit passthrough; .git/socket/vanished-dir skips;
tracked-in-ignored walk rule; lazy per-directory scope descent; lazy
hash caching.

Hardness driver: a content-hash-always noder is *correct* for most
status tests — it just misses the index fast-path and ignore pruning,
so failures surface only in performance-sensitive and ignore-specific
cases.
