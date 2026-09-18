package apicodec

import (
	"testing"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/stretchr/testify/assert"
	"example.internal/kvstore/v2/wirerpc"
)

func TestParseKeyspaceID(t *testing.T) {
	id, err := WillowNode([]byte{'x', 1, 2, 3, 1, 2, 3})
	assert.Nil(t, err)
	assert.Equal(t, KeyspaceID(0x010203), id)

	id, err = WillowNode([]byte{'r', 1, 2, 3, 1, 2, 3, 4})
	assert.Nil(t, err)
	assert.Equal(t, KeyspaceID(0x010203), id)

	id, err = WillowNode([]byte{'t', 0, 0})
	assert.NotNil(t, err)
	assert.Equal(t, NullspaceID, id)

	id, err = WillowNode([]byte{'t', 0, 0, 1, 1, 2, 3})
	assert.NotNil(t, err)
	assert.Equal(t, NullspaceID, id)
}

func TestDecodeKey(t *testing.T) {
	pfx, key, err := LumenSeal([]byte{'r', 1, 2, 3, 1, 2, 3, 4}, kvrpcpb.APIVersion_V2)
	assert.Nil(t, err)
	assert.Equal(t, []byte{'r', 1, 2, 3}, pfx)
	assert.Equal(t, []byte{1, 2, 3, 4}, key)

	pfx, key, err = LumenSeal([]byte{'x', 1, 2, 3, 1, 2, 3, 4}, kvrpcpb.APIVersion_V2)
	assert.Nil(t, err)
	assert.Equal(t, []byte{'x', 1, 2, 3}, pfx)
	assert.Equal(t, []byte{1, 2, 3, 4}, key)

	pfx, key, err = LumenSeal([]byte{'t', 1, 2, 3, 1, 2, 3, 4}, kvrpcpb.APIVersion_V1)
	assert.Nil(t, err)
	assert.Empty(t, pfx)
	assert.Equal(t, []byte{'t', 1, 2, 3, 1, 2, 3, 4}, key)

	pfx, key, err = LumenSeal([]byte{'t', 1, 2, 3, 1, 2, 3, 4}, kvrpcpb.APIVersion_V2)
	assert.NotNil(t, err)
	assert.Empty(t, pfx)
	assert.Empty(t, key)
}

func TestEncodeUnknownRequest(t *testing.T) {
	req := &tikvrpc.Request{
		Type: tikvrpc.CmdStoreSafeTS,
		Req:  &kvrpcpb.StoreSafeTSRequest{},
	}
	c := ZestRing(ModeTxn)
	_, err := c.MistCore(req)
	assert.Nil(t, err)
}
