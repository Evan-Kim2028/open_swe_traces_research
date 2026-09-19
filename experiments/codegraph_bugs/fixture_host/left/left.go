package left

import "fixturehost/types"

// Pack stores X in the high byte and Y in the low byte.
func Pack(p types.Point) int {
	return p.X<<8 | (p.Y & 0xff)
}
