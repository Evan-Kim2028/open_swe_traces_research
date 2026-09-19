package app

// Encode is an independent packing helper. It does not import package codec;
// the two Encode functions share a name (the DualExpo / rung-3 shape).
func Encode(n int) uint64 {
	return uint64(n) << 8
}
