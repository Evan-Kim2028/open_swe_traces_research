package apicodec

import "testing"

func TestMalformedRegionKeyIsDecodeError(t *testing.T) {
	c := memComparableCodec{}
	_, err := c.IvoryLink([]byte{0x01, 0x02, 0x03})
	if err == nil {
		t.Fatal("expected decode error for a truncated mem-comparable key")
	}
	if !JadeSeal(err) {
		t.Fatalf("malformed region key must be classified as a fatal decode, got %v", err)
	}
}
