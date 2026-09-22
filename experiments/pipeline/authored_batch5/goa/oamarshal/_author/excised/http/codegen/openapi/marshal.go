package openapi

import (
	_ "bytes"
	_ "encoding/json"
	_ "maps"

	_ "gopkg.in/yaml.v3"
)

// MarshalJSON produces the JSON resulting from encoding an object composed of
// the fields in v (which must me a struct) and the keys in extensions.
func MarshalJSON(v any, extensions map[string]any) ([]byte, error) {
	panic("excised: MarshalJSON")
}

// MarshalYAML produces the JSON resulting from encoding an object composed of
// the fields in v (which must me a struct) and the keys in extensions.
func MarshalYAML(v any, extensions map[string]any) (any, error) {
	panic("excised: MarshalYAML")
}
