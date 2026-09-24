package config

import (
	_ "strconv"
	_ "strings"
)

// OptBool is a tri-state boolean: unset, explicitly false, or explicitly true.
// Its zero value (OptBoolUnset) means the setting was not specified, which
// allows merge logic based on reflect.Value.IsZero to skip unset fields while
// still letting an explicit "false" override a previously set "true".
type OptBool byte

const (
	// OptBoolUnset indicates the setting was not specified.
	OptBoolUnset OptBool = iota
	// OptBoolFalse indicates the setting was explicitly set to false.
	OptBoolFalse
	// OptBoolTrue indicates the setting was explicitly set to true.
	OptBoolTrue
)

// NewOptBool converts a plain bool into an OptBool.
func NewOptBool(v bool) OptBool {
	panic("excised: NewOptBool")
}

// IsTrue returns whether the value is explicitly true.
func (o OptBool) IsTrue() bool { return o == OptBoolTrue }

// IsSet returns whether the value was explicitly specified (true or false).
func (o OptBool) IsSet() bool { return o != OptBoolUnset }

func (o OptBool) String() string {
	panic("excised: OptBool.String")
}

// FormatBool returns the strconv-formatted value. Only meaningful when IsSet.
func (o OptBool) FormatBool() string {
	panic("excised: OptBool.FormatBool")
}

// parseConfigBool mirrors upstream Git's git_parse_maybe_bool: it
// accepts true/yes/on (→ OptBoolTrue) and false/no/off (→
// OptBoolFalse) case-insensitively, plus any decimal integer (zero
// → OptBoolFalse, non-zero → OptBoolTrue). Empty or otherwise
// unrecognised values return OptBoolUnset, leaving the caller's
// platform default in place. The empty-string handling is the only
// intentional divergence from upstream, which returns false for
// empty: in our unmarshalCore caller, an empty value means the key
// is unset and the platform default should apply.
//
// Reference: upstream Git git_parse_maybe_bool_text at parse.c
// L157-L173 and git_parse_maybe_bool at parse.c L174-L182 in tag
// v2.54.0[1].
//
// [1]: https://github.com/git/git/blob/v2.54.0/parse.c#L157-L182
func parseConfigBool(v string) OptBool {
	panic("excised: parseConfigBool")
}
