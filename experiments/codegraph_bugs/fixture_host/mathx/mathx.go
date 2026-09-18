package mathx

// Add returns a+b. Called from this package's tests and from package app.
func Add(a, b int) int {
	return a + b
}

// Clamp returns v limited to [lo, hi].
func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
