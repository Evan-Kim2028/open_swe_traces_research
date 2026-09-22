package transport

import (
	"context"
	_ "errors"
	_ "fmt"
	"io"
	_ "slices"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/format/config"
	_ "example.internal/gitkit/v6/plumbing/format/pktline"
	"example.internal/gitkit/v6/plumbing/protocol/capability"
	"example.internal/gitkit/v6/plumbing/protocol/packp"
	"example.internal/gitkit/v6/storage"
	_ "example.internal/gitkit/v6/utils/ioutil"
	_ "example.internal/gitkit/v6/x/storage"
)

const (
	initialFlush  = 16
	pipeSafeFlush = 32
	largeFlush    = 16384
	maxInVein     = 256
)

func nextFlush(statelessRPC bool, count int) int {
	panic("excised: nextFlush")
}

func applyServerACKs(
	statelessRPC bool,
	acks []packp.ACK,
	common map[plumbing.Hash]struct{},
	statelessCommon *[]plumbing.Hash,
	gotContinue *bool,
	gotReady *bool,
	inVein *int,
) {
	panic("excised: applyServerACKs")
}

// NegotiatePack performs the pack negotiation phase of the fetch operation.
func NegotiatePack(
	ctx context.Context,
	st storage.Storer,
	caps capability.List,
	statelessRPC bool,
	reader io.Reader,
	writer io.WriteCloser,
	req *FetchRequest,
) (shallowInfo *packp.ShallowUpdate, err error) {
	panic("excised: NegotiatePack")
}

func isSubset(needle, haystack []plumbing.Hash) bool {
	panic("excised: isSubset")
}

func readShallows(
	statelessRPC bool,
	r io.Reader,
	req *FetchRequest,
	shallowInfo **packp.ShallowUpdate,
	firstRound bool,
) error {
	panic("excised: readShallows")
}

// ReconcileObjectFormatV2 aligns the storer's object format with the Protocol
// v2 server's advertised object-format before any packfile is requested. On a
// fresh clone the storer's format is unset (HEAD still points at the
// refs/heads/.invalid placeholder) and the server's sha256 is adopted;
// otherwise a mismatch is a hard error, since indexing a sha256 pack as sha1
// (or vice versa) corrupts the store and only surfaces later as a checksum
// failure. It mirrors NegotiatePack's v0/v1 object-format handling and git's
// fetch-pack.c, including the case where the server omits object-format (it
// only speaks sha1) but the client repository uses another algorithm.
func ReconcileObjectFormatV2(st storage.Storer, caps capability.List) error {
	panic("excised: ReconcileObjectFormatV2")
}
