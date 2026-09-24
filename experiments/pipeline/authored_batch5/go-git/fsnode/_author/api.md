# Exported API — fsnode

Package `utils/merkletrie/filesystem` — the `noder.Noder` adapter over a
billy filesystem: `NewRootNode`/`NewRootNodeWithOptions`, `node.Hash`,
`Name`, `IsDir`, `Skip`, `Children`, `NumChildren`, plus internals
`calculateChildren`, `resolveScope`, `pathComponents`,
`shouldSkipIgnored`, `newChildNode`, `calculateHash`, `metadataMatches`,
`doCalculateHashForRegular`, `doCalculateHashForSymlink`, `String`.

Kept visible: `Options`/`node` types, `ignore` map, all doc comments on
racy-git and IgnoreScope semantics.

Callers: `Status`/`DiffTree` filesystem walks. In-tree tests removed: 2
(`node_test.go`, `node_ignore_test.go`).
