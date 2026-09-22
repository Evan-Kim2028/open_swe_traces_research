# Closure — fsnode

Package: `utils/merkletrie/filesystem`. File: `node.go`.

Removed (18 functions stubbed): `NewRootNode`, `NewRootNodeWithOptions`,
`node.Hash`, `node.Name`, `node.IsDir`, `node.Skip`, `node.Children`,
`node.NumChildren`, `node.calculateChildren`, `node.resolveScope`,
`node.pathComponents`, `node.shouldSkipIgnored`, `node.newChildNode`,
`node.calculateHash`, `node.metadataMatches`,
`node.doCalculateHashForRegular`, `node.doCalculateHashForSymlink`,
`node.String`.

Kept: `Options`/`node` types and their doc comments, `ignore` map, the
`gitignore`/`index`/`noder` packages it consumes (ignorepattern,
ignorescope, indexops own those). Disjoint from `merklediff` (the trie
diff machinery) — this is the filesystem noder feeding it.

Tests deleted: `node_test.go`, `node_ignore_test.go` (2 — the package's
only tests).
