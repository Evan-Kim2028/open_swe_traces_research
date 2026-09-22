package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/plumbing/format/pktline"
	"example.internal/gitkit/v6/plumbing/protocol/capability"
	"example.internal/gitkit/v6/plumbing/protocol/packp"
	"example.internal/gitkit/v6/storage"
	"example.internal/gitkit/v6/utils/ioutil"
	_ "example.internal/gitkit/v6/x/storage"
)

const (
	initialFlush  = 16
	pipeSafeFlush = 32
	largeFlush    = 16384
	maxInVein     = 256
)

func nextFlush(statelessRPC bool, count int) int {
	return count << 1
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
	reader = ioutil.NewContextReader(ctx, reader)
	writer = ioutil.NewContextWriteCloser(ctx, writer)

	upreq := &packp.UploadRequest{}
	if caps.Supports(capability.MultiACK) {
		upreq.Capabilities.Set(capability.MultiACK)
	}
	if req.Progress != nil && caps.Supports(capability.Sideband) {
		upreq.Capabilities.Set(capability.Sideband)
	}
	if caps.Supports(capability.OFSDelta) {
		upreq.Capabilities.Set(capability.OFSDelta)
	}
	if req.Filter != "" {
		if caps.Supports(capability.Filter) {
			upreq.Filter = req.Filter
			upreq.Capabilities.Set(capability.Filter)
		} else {
			return nil, ErrFilterNotSupported
		}
	}
	upreq.Wants = req.Wants

	if req.Depth > 0 {
		if !caps.Supports(capability.Shallow) {
			return nil, ErrShallowNotSupported
		}
		upreq.Depth = packp.DepthRequest{Deepen: req.Depth}
		upreq.Shallows, err = st.Shallow()
		if err != nil {
			return nil, err
		}
	}

	if isSubset(req.Wants, req.Haves) && len(upreq.Shallows) == 0 {
		if err := pktline.WriteFlush(writer); err != nil {
			return nil, err
		}
		if err := writer.Close(); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("closing writer: %w", err)
		}
		return nil, ErrNoChange
	}

	if err := upreq.Encode(writer); err != nil {
		return nil, fmt.Errorf("sending upload-request: %w", err)
	}
	uphav := packp.UploadHaves{Haves: req.Haves, Done: true}
	if err := uphav.Encode(writer); err != nil {
		return nil, fmt.Errorf("sending upload-haves: %w", err)
	}
	if statelessRPC {
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("closing writer: %w", err)
		}
	}
	if err := readShallows(statelessRPC, reader, req, &shallowInfo, true); err != nil {
		return nil, err
	}
	var srvrs packp.ServerResponse
	if err := srvrs.Decode(reader); err != nil {
		return nil, fmt.Errorf("decoding server-response: %w", err)
	}
	if !statelessRPC {
		if err := writer.Close(); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("closing writer: %w", err)
		}
	}
	return shallowInfo, nil
}

func isSubset(needle, haystack []plumbing.Hash) bool {
	for _, h := range needle {
		if !slices.Contains(haystack, h) {
			return false
		}
	}
	return true
}

func readShallows(
	statelessRPC bool,
	r io.Reader,
	req *FetchRequest,
	shallowInfo **packp.ShallowUpdate,
	firstRound bool,
) error {
	if (firstRound || statelessRPC) && req.Depth > 0 {
		var shupd packp.ShallowUpdate
		if err := shupd.Decode(r); err != nil {
			return fmt.Errorf("decoding shallow-update: %w", err)
		}
		if *shallowInfo == nil {
			*shallowInfo = &shupd
		}
	}
	return nil
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
	var clientFormat config.ObjectFormat
	if cfg, err := st.Config(); err == nil && cfg != nil {
		clientFormat = cfg.Extensions.ObjectFormat
	}
	advertised := caps.Get(capability.ObjectFormat)
	if len(advertised) == 0 {
		return nil
	}
	serverFormat := config.ObjectFormat(advertised[0])
	if clientFormat == config.UnsetObjectFormat {
		clientFormat = serverFormat
	}
	if serverFormat != clientFormat {
		return fmt.Errorf("mismatched algorithms: client %s; server %s", clientFormat, serverFormat)
	}
	return nil
}
