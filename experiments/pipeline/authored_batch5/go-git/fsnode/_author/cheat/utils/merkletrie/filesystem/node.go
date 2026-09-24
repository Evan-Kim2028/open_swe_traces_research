// Package filesystem provides a merkletrie noder implementation for billy filesystems.
package filesystem

import (
	"io"
	iofs "io/fs"
	"os"
	"path"
	_ "strings"
	"time"

	"github.com/go-git/go-billy/v6"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	format "example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/plumbing/format/gitignore"
	"example.internal/gitkit/v6/plumbing/format/index"
	_ "example.internal/gitkit/v6/utils/convert"
	_ "example.internal/gitkit/v6/utils/ioutil"
	"example.internal/gitkit/v6/utils/merkletrie/noder"
	_ "example.internal/gitkit/v6/utils/sync"
)

var ignore = map[string]bool{
	".git": true,
}

// Options contains configuration for the filesystem node.
type Options struct {
	// AutoCRLF converts CRLF line endings in text files into LF line endings.
	AutoCRLF bool

	// Index is used to enable the metadata-first comparison optimization while
	// correctly handling the "racy git" condition. If no index is provided,
	// the function works without the optimization.
	Index *index.Index

	// IgnoreScope, if non-nil, is consulted while walking the tree. Untracked
	// entries (files or directories) that it reports as ignored are excluded
	// from the walk, so callers do not have to descend into large gitignored
	// directories like node_modules. Tracked entries are always walked even
	// when ignored, so modifications to them are still reported.
	//
	// It is the scope in effect at the root of the walk, normally
	// gitignore.NewScope of gitignore.RootPatterns plus any patterns the
	// caller supplies. The walk derives each directory's scope from the
	// listing it already takes, so a .gitignore is opened only in directories
	// actually visited and never below an excluded one, and an excluded
	// directory stays authoritative for everything under it.
	//
	// Requires Index to be set: without an index there is no way to identify
	// tracked entries, so the scope is treated as a no-op.
	IgnoreScope *gitignore.Scope
}

// The node represents a file or a directory in a billy.Filesystem. It
// implements the interface noder.Noder of merkletrie package.
//
// This implementation implements a "standard" hash method being able to be
// compared with any other noder.Noder implementation inside of gitkit.
type node struct {
	fs         billy.Filesystem
	submodules map[string]plumbing.Hash
	idx        *index.Index
	idxMap     map[string]*index.Entry
	// trackedDirs holds every directory path that has at least one entry
	// in the index. It is populated only when IgnoreScope is set so the
	// walker can keep tracked entries even if their parent directory
	// matches an ignore rule.
	trackedDirs map[string]struct{}

	options *Options

	// scope is the ignore scope governing this node's entries. On a child it
	// starts as the parent's scope and is replaced by this directory's own on
	// the first calculateChildren, which is when the listing that reveals
	// whether a .gitignore is present becomes available. scopeResolved tracks
	// that transition; the root node is created already resolved.
	scope         *gitignore.Scope
	scopeResolved bool

	path     string
	hash     []byte
	children []noder.Noder
	isDir    bool
	mode     os.FileMode
	size     int64
	modTime  time.Time
}

// NewRootNode returns the root node based on a given billy.Filesystem.
//
// In order to provide the submodule hash status, a map[string]plumbing.Hash
// should be provided where the key is the path of the submodule and the commit
// of the submodule HEAD
//
// Deprecated: Use NewRootNodeWithOptions instead for better performance.
// This function is kept for backward compatibility.
func NewRootNode(
	fs billy.Filesystem,
	submodules map[string]plumbing.Hash,
) noder.Noder {
	return NewRootNodeWithOptions(fs, submodules, Options{})
}

// NewRootNodeWithOptions returns the root node based on a given billy.Filesystem
// with options for CRLF handling and an index. Providing an index enables the
// metadata-first comparison optimization while correctly handling the "racy git"
// condition. If no index is provided, the function works without the optimization.
//
// The index's ModTime field is used to detect the racy git condition. When a file's
// mtime equals or is newer than the index ModTime, we must hash the file content
// even if other metadata matches, because the file may have been modified in the
// same second that the index was written.
//
// Reference: https://git-scm.com/docs/racy-git
func NewRootNodeWithOptions(
	fs billy.Filesystem,
	submodules map[string]plumbing.Hash,
	options Options,
) noder.Noder {
	return &node{
		fs:          fs,
		submodules:  submodules,
		options:     &options,
		isDir:       true,
		scope:       options.IgnoreScope,
		scopeResolved: true,
	}
}

// Hash the hash of a filesystem is the result of concatenating the computed
// plumbing.Hash of the file as a Blob and its plumbing.FileMode; that way the
// difftree algorithm will detect changes in the contents of files and also in
// their mode.
//
// Please note that the hash is calculated on first invocation of Hash(),
// meaning that it will not update when the underlying file changes
// between invocations.
//
// The hash of a directory is always a 24-bytes slice of zero values
func (n *node) Hash() []byte {
	if n.hash == nil {
		n.calculateHash()
	}
	return n.hash
}

func (n *node) Name() string {
	return path.Base(n.path)
}

func (n *node) IsDir() bool {
	return n.isDir
}

func (n *node) Skip() bool {
	return false
}

func (n *node) Children() ([]noder.Noder, error) {
	if err := n.calculateChildren(); err != nil {
		return nil, err
	}
	return n.children, nil
}

func (n *node) NumChildren() (int, error) {
	if err := n.calculateChildren(); err != nil {
		return -1, err
	}
	return len(n.children), nil
}

func (n *node) calculateChildren() error {
	if !n.IsDir() {
		return nil
	}
	if len(n.children) != 0 {
		return nil
	}
	files, err := n.fs.ReadDir(n.path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if _, ok := ignore[file.Name()]; ok {
			continue
		}
		fi, err := file.Info()
		if err != nil {
			return err
		}
		c, err := n.newChildNode(fi)
		if err != nil {
			return err
		}
		n.children = append(n.children, c)
	}
	return nil
}

// resolveScope derives this directory's ignore scope from the listing just
// taken for it. Deferring to this point is the whole benefit of the scoped
// walk: whether a .gitignore exists is read off a listing the walk needed
// anyway, the file is opened only in directories actually visited, and
// Scope.Descend declines to open it at all below an excluded directory.
func (n *node) resolveScope(files []iofs.DirEntry) error {
	panic("excised: node.resolveScope")
}

func (n *node) pathComponents() []string {
	panic("excised: node.pathComponents")
}

// shouldSkipIgnored reports whether the child entry of n with the given
// name should be skipped because it matches the ignore scope in effect
// AND has no entry in the index. Tracked entries are never skipped so
// modifications to them are still reported.
func (n *node) shouldSkipIgnored(name string, isDir bool) bool {
	panic("excised: node.shouldSkipIgnored")
}

func (n *node) newChildNode(file os.FileInfo) (*node, error) {
	return &node{
		fs:         n.fs,
		submodules: n.submodules,
		options:    n.options,
		scope:      n.scope,
		path:       path.Join(n.path, file.Name()),
		isDir:      file.IsDir(),
		size:       file.Size(),
		mode:       file.Mode(),
		modTime:    file.ModTime(),
	}, nil
}

func (n *node) calculateHash() {
	if n.isDir {
		n.hash = make([]byte, 24)
		return
	}
	mode, err := filemode.NewFromOSFileMode(n.mode)
	if err != nil {
		n.hash = plumbing.ZeroHash.Bytes()
		return
	}
	var hash plumbing.Hash
	if n.mode&os.ModeSymlink != 0 {
		hash = n.doCalculateHashForSymlink()
	} else {
		hash = n.doCalculateHashForRegular()
	}
	n.hash = append(hash.Bytes(), mode.Bytes()...)
}

func (n *node) metadataMatches(entry *index.Entry) bool {
	panic("excised: node.metadataMatches")
}

func (n *node) doCalculateHashForRegular() plumbing.Hash {
	f, err := n.fs.Open(n.path)
	if err != nil {
		return plumbing.ZeroHash
	}
	defer func() { _ = f.Close() }()
	h := plumbing.NewHasher(format.SHA1, plumbing.BlobObject, n.size)
	if _, err := io.Copy(h, f); err != nil {
		return plumbing.ZeroHash
	}
	return h.Sum()
}

func (n *node) doCalculateHashForSymlink() plumbing.Hash {
	target, err := n.fs.Readlink(n.path)
	if err != nil {
		return plumbing.ZeroHash
	}
	h := plumbing.NewHasher(format.SHA1, plumbing.BlobObject, n.size)
	if _, err := h.Write([]byte(target)); err != nil {
		return plumbing.ZeroHash
	}
	return h.Sum()
}

func (n *node) String() string {
	return n.path
}
