package codec

// Encode packs n into the high 8 bits. Decode is the inverse.
func Encode(n int) uint64 {
	return uint64(n) << 8
}

// Decode unpacks a value produced by Encode.
func Decode(v uint64) int {
	return int(v >> 8)
}
