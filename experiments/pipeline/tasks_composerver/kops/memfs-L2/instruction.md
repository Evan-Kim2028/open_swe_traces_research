# Contract (L2) — memfs

An in-memory path tree is rooted in a context. Joining a slash-separated relative path creates missing children and returns the final node. Create is exclusive: a node that already has contents returns exists; Write always replaces contents. Read of a node with no contents is not-exist. Directory listing returns the immediate child nodes (files and subdirs). Tree listing recursively yields only leaves (a node is a leaf when it has no children), including nested files. Remove zeros contents of that node; RemoveAll trees then removes each leaf.

Join and directory/tree walks take the node mutex; nested Join locks each child along the path.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestMemFsCreateFile` | exclusive create vs overwrite write; exists on second create |
| `TestMemFsReadDir` | immediate children after joins/writes |
| `TestMemFsReadTree` | recursive leaves only |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./util/pkg/vfs/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
