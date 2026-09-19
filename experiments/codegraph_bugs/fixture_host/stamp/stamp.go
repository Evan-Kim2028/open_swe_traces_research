package stamp

import "time"

const shift = 8

// Compose joins physical and logical parts of a hybrid timestamp.
func Compose(physical, logical int64) uint64 {
	return uint64((physical << shift) + logical)
}

// Extract returns the physical part of a hybrid timestamp.
func Extract(ts uint64) int64 {
	return int64(ts >> shift)
}

// FromTime converts a Go time to a hybrid timestamp (logical 0).
func FromTime(t time.Time) uint64 {
	return uint64((t.UnixMilli()) << shift)
}
