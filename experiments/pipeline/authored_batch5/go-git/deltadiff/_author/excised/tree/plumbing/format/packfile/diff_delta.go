package packfile

import (
	"bytes"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/format/packfile/util"
	_ "example.internal/gitkit/v6/utils/ioutil"
	_ "example.internal/gitkit/v6/utils/sync"
)

// See https://github.com/jelmer/dulwich/blob/master/dulwich/pack.py and
// https://github.com/tarruda/node-git-core/blob/master/src/js/delta.js
// for more info

const (
	// Standard chunk size used to generate fingerprints
	s = 16

	// https://github.com/git/git/blob/f7466e94375b3be27f229c78873f0acf8301c0a5/diff-delta.c#L428
	// Max size of a copy operation (64KB).
	maxCopySize = 64 * 1024
)

// GetDelta returns an EncodedObject of type OFSDeltaObject. Base and Target object,
// will be loaded into memory to be able to create the delta object.
// To generate target again, you will need the obtained object and "base" one.
// Error will be returned if base or target object cannot be read.
func GetDelta(base, target plumbing.EncodedObject) (plumbing.EncodedObject, error) {
	panic("excised: GetDelta")
}

func getDelta(index *deltaIndex, base, target plumbing.EncodedObject) (o plumbing.EncodedObject, err error) {
	panic("excised: getDelta")
}

// DiffDelta returns the delta that transforms src into tgt.
func DiffDelta(src, tgt []byte) []byte {
	panic("excised: DiffDelta")
}

func diffDelta(index *deltaIndex, src, tgt []byte) []byte {
	panic("excised: diffDelta")
}

func encodeInsertOperation(ibuf, buf *bytes.Buffer) {
	panic("excised: encodeInsertOperation")
}

func encodeCopyOperation(offset, length int) []byte {
	panic("excised: encodeCopyOperation")
}
