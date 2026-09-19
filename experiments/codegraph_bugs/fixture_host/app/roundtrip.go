package app

import (
	"fixturehost/left"
	"fixturehost/right"
	"fixturehost/types"
)

// RoundTrip packs then unpacks a point through the two packages that
// share types.Point and do not import each other.
func RoundTrip(p types.Point) types.Point {
	return right.Unpack(left.Pack(p))
}
