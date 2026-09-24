package capability

import (
	_ "bytes"
)

// List represents a list of capabilities. The zero value is safe to use;
// the internal map is lazily initialized on first write. List is not safe for
// concurrent use.
type List struct {
	m    map[string]*entry
	sort []string
}

type entry struct {
	Name   string
	Values []string
}

// IsEmpty returns true if the List is empty
func (l *List) IsEmpty() bool {
	if l == nil {
		return true
	}
	return len(l.sort) == 0
}

// DecodeList decodes a v0/v1 space-separated capability string into the
// List. This is the format used in advertise-refs, upload-request, and
// update-request messages.
func DecodeList(raw []byte, l *List) {
	panic("excised: DecodeList")
}

// EncodeList encodes the List into a v0/v1 space-separated capability string.
// This is the format used in advertise-refs, upload-request, and
// update-request messages.
func EncodeList(l *List) []byte {
	panic("excised: EncodeList")
}

// Get returns the values for a capability
func (l *List) Get(capability string) []string {
	if l.m == nil {
		return nil
	}
	if _, ok := l.m[capability]; !ok {
		return nil
	}

	return l.m[capability].Values
}

// Set sets a capability removing the previous values
func (l *List) Set(capability string, values ...string) {
	panic("excised: List.Set")
}

func (l *List) init() {
	panic("excised: List.init")
}

// Add adds a capability, values are optional
func (l *List) Add(c string, values ...string) {
	panic("excised: List.Add")
}

// Supports returns true if capability is present
func (l *List) Supports(capability string) bool {
	if l.m == nil {
		return false
	}
	_, ok := l.m[capability]
	return ok
}

// Delete deletes a capability from the List
func (l *List) Delete(capability string) {
	panic("excised: List.Delete")
}

// All returns a slice with all defined capabilities.
func (l *List) All() []string {
	panic("excised: List.All")
}

// MarshalText implements encoding.TextMarshaler.
func (l *List) MarshalText() ([]byte, error) {
	panic("excised: List.MarshalText")
}

// AppendText implements encoding.TextAppender.
func (l *List) AppendText(b []byte) ([]byte, error) {
	panic("excised: List.AppendText")
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (l *List) UnmarshalText(text []byte) error {
	panic("excised: List.UnmarshalText")
}

// String generates the capabilities strings, the capabilities are sorted in
// insertion order.
func (l *List) String() string {
	panic("excised: List.String")
}
