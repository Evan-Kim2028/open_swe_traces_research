package right

import "fixturehost/types"

// Unpack recovers a Point packed by left.Pack.
func Unpack(v int) types.Point {
	return types.Point{X: v >> 8, Y: v & 0xff}
}
