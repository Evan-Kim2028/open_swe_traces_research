// Copyright 2021 KVStore Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE: The code in this file is based on code from the
// SQLEngine project, licensed under the Apache License v 2.0
//
// https://github.com/acme/sqlengine/tree/cc5e161ac06827589c4966674597c137cc9e809c/store/kvstore/unionstore/union_iter.go
//

// Copyright 2015 Acme, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unionstore

import (
	_ "example.internal/kvstore/v2/internal/logutil"
	"example.internal/kvstore/v2/kv"
	_ "go.uber.org/zap"
)

// UnionIter is the iterator on an UnionStore.
type UnionIter struct {
	dirtyIt    Iterator
	snapshotIt Iterator

	dirtyValid    bool
	snapshotValid bool

	curIsDirty bool
	isValid    bool
	reverse    bool
}

// NewUnionIter returns a union iterator for BufferStore.
func NewUnionIter(dirtyIt Iterator, snapshotIt Iterator, reverse bool) (*UnionIter, error) {
	it := &UnionIter{
		dirtyIt:       dirtyIt,
		snapshotIt:    snapshotIt,
		dirtyValid:    dirtyIt.Valid(),
		snapshotValid: snapshotIt.Valid(),
		reverse:       reverse,
	}
	err := it.updateCur()
	if err != nil {
		return nil, err
	}
	return it, nil
}

// dirtyNext makes iter.dirtyIt go and update valid status.
func (iter *UnionIter) dirtyNext() error {
	err := iter.dirtyIt.Next()
	iter.dirtyValid = iter.dirtyIt.Valid()
	return err
}

// snapshotNext makes iter.snapshotIt go and update valid status.
func (iter *UnionIter) snapshotNext() error {
	err := iter.snapshotIt.Next()
	iter.snapshotValid = iter.snapshotIt.Valid()
	return err
}

func (iter *UnionIter) updateCur() error {
	iter.isValid = iter.dirtyValid || iter.snapshotValid
	if !iter.isValid {
		return nil
	}
	if !iter.dirtyValid {
		iter.curIsDirty = false
		return nil
	}
	if !iter.snapshotValid {
		iter.curIsDirty = true
		return nil
	}
	cmp := kv.CmpKey(iter.dirtyIt.Key(), iter.snapshotIt.Key())
	if iter.reverse {
		cmp = -cmp
	}
	if cmp == 0 {
		if err := iter.snapshotNext(); err != nil {
			return err
		}
		iter.curIsDirty = true
	} else {
		iter.curIsDirty = cmp < 0
	}
	return nil
}

// Next implements the Iterator Next interface.
func (iter *UnionIter) Next() error {
	var err error
	if iter.curIsDirty {
		err = iter.dirtyNext()
	} else {
		err = iter.snapshotNext()
	}
	if err != nil {
		return err
	}
	return iter.updateCur()
}

// Value implements the Iterator Value interface.
// Multi columns
func (iter *UnionIter) Value() []byte {
	if !iter.curIsDirty {
		return iter.snapshotIt.Value()
	}
	return iter.dirtyIt.Value()
}

// Key implements the Iterator Key interface.
func (iter *UnionIter) Key() []byte {
	if !iter.curIsDirty {
		return iter.snapshotIt.Key()
	}
	return iter.dirtyIt.Key()
}

// Valid implements the Iterator Valid interface.
func (iter *UnionIter) Valid() bool {
	return iter.isValid
}

// Close implements the Iterator Close interface.
func (iter *UnionIter) Close() {
	panic("excised: UnionIter.Close")
}
