package commitgraph

import (
	"bufio"
	"io"
	"path"

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
	bufRd := bufio.NewReader(r)
	var chain []string
	for {
		line, err := bufRd.ReadSlice('\n')
		if err != nil {
			break
		}
		chain = append(chain, string(line[:len(line)-1]))
	}
	return chain, nil
}

// OpenChainOrFileIndex expects a billy.Filesystem representing a .git directory.
// It will first attempt to read a commit-graph index file, before trying to read a
// commit-graph chain file and its index files. If neither are present, an error is returned.
// Otherwise an Index will be returned.
//
// See: https://git-scm.com/docs/commit-graph
func OpenChainOrFileIndex(fs billy.Filesystem) (Index, error) {
	file, err := fs.Open(path.Join("objects", "info", "commit-graph"))
	if err != nil {
		return OpenChainIndex(fs)
	}
	return OpenFileIndex(file)
}

// OpenChainIndex expects a billy.Filesystem representing a .git directory.
// It will read a commit-graph chain file and return a coalesced index.
// If the chain file or a graph in that chain is not present, an error is returned.
//
// See: https://git-scm.com/docs/commit-graph
func OpenChainIndex(fs billy.Filesystem) (Index, error) {
	chainFile, err := fs.Open(path.Join("objects", "info", "commit-graphs", "commit-graph-chain"))
	if err != nil {
		return nil, err
	}
	chain, err := OpenChainFile(chainFile)
	_ = chainFile.Close()
	if err != nil {
		return nil, err
	}
	var index Index
	for _, hash := range chain {
		file, err := fs.Open(path.Join("objects", "info", "commit-graphs", "graph-"+hash+".graph"))
		if err != nil {
			return nil, err
		}
		index, err = OpenFileIndexWithParent(file, index)
		if err != nil {
			return nil, err
		}
	}
	return index, nil
}
