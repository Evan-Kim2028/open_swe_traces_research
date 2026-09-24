package packfile

import (
	"io"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/idxfile"
)

type objectIter struct {
	p    *Packfile
	typ  plumbing.ObjectType
	iter idxfile.EntryIter
}

func (i *objectIter) Next() (plumbing.EncodedObject, error) {
	if err := i.p.init(); err != nil {
		return nil, err
	}
	i.p.m.Lock()
	defer i.p.m.Unlock()
	return i.next()
}

func (i *objectIter) next() (plumbing.EncodedObject, error) {
	for {
		e, err := i.iter.Next()
		if err != nil {
			return nil, err
		}
		oh, err := i.p.headerFromOffset(int64(e.Offset))
		if err != nil {
			return nil, err
		}
		if i.typ == plumbing.AnyObject || oh.Type == i.typ {
			return i.p.objectFromHeader(oh)
		}
	}
}

func (i *objectIter) ForEach(f func(plumbing.EncodedObject) error) error {
	if err := i.p.init(); err != nil {
		return err
	}
	i.p.m.Lock()
	defer i.p.m.Unlock()
	for {
		o, err := i.next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := f(o); err != nil {
			return err
		}
	}
}

func (i *objectIter) Close() {
	i.p.m.Lock()
	defer i.p.m.Unlock()
	_ = i.iter.Close()
}
