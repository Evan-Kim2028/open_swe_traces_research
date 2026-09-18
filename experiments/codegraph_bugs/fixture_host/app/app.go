package app

import "fixturehost/mathx"

// SumClamped adds a and b then clamps to [0, 100].
func SumClamped(a, b int) int {
	return mathx.Clamp(mathx.Add(a, b), 0, 100)
}
