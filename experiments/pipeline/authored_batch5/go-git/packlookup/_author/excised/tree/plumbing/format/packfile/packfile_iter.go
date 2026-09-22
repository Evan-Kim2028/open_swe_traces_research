package packfile

import (
	_ "io"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
)

type objectIter struct {
	p    *Packfile
	typ  plumbing.ObjectType
	iter idxfile.EntryIter
}

func (i *objectIter) Next() (plumbing.EncodedObject, error) {
	panic("excised: objectIter.Next")
}

func (i *objectIter) next() (plumbing.EncodedObject, error) {
	panic("excised: objectIter.next")
}

func (i *objectIter) ForEach(f func(plumbing.EncodedObject) error) error {
	panic("excised: objectIter.ForEach")
}

func (i *objectIter) Close() {
	panic("excised: objectIter.Close")
}
