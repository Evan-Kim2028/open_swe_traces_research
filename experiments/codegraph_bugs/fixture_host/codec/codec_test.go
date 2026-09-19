package codec

import "testing"

func TestEncodeDecode(t *testing.T) {
	if Decode(Encode(7)) != 7 {
		t.Fatalf("roundtrip")
	}
}
