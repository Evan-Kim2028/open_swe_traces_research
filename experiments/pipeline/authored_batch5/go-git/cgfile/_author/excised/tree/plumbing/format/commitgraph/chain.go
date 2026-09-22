package commitgraph

import (
	_ "bufio"
	"io"
	_ "path"

	"github.com/go-git/go-billy/v6"

	_ "example.internal/gitkit/v6/plumbing"
)

// OpenChainFile reads a commit chain file and returns a slice of the hashes within it
//
// Commit-Graph chains are described at https://git-scm.com/docs/commit-graph
// and are new line separated list of graph file hashes, oldest to newest.
//
// This function simply reads the file and returns the hashes as a slice.
func OpenChainFile(r io.Reader) ([]string, error) {
	panic("excised: OpenChainFile")
}

// OpenChainOrFileIndex expects a billy.Filesystem representing a .git directory.
// It will first attempt to read a commit-graph index file, before trying to read a
// commit-graph chain file and its index files. If neither are present, an error is returned.
// Otherwise an Index will be returned.
//
// See: https://git-scm.com/docs/commit-graph
func OpenChainOrFileIndex(fs billy.Filesystem) (Index, error) {
	panic("excised: OpenChainOrFileIndex")
}

// OpenChainIndex expects a billy.Filesystem representing a .git directory.
// It will read a commit-graph chain file and return a coalesced index.
// If the chain file or a graph in that chain is not present, an error is returned.
//
// See: https://git-scm.com/docs/commit-graph
func OpenChainIndex(fs billy.Filesystem) (Index, error) {
	panic("excised: OpenChainIndex")
}
