// This file converts authored names into identifiers that Apikit can safely write
// to protobuf files. Transport generators and external plugins use the same
// functions so identical design names produce identical protobuf names.
package codegen

import (
	"regexp"
	_ "strings"
)

var (
	protobufDigits = regexp.MustCompile("[0-9]+")

	protobufKeywords = map[string]struct{}{
		"bool": {}, "bytes": {}, "double": {}, "fixed32": {}, "fixed64": {},
		"float": {}, "int32": {}, "int64": {}, "sfixed32": {}, "sfixed64": {},
		"sint32": {}, "sint64": {}, "string": {}, "uint32": {}, "uint64": {},
		"enum": {}, "import": {}, "map": {}, "message": {}, "oneof": {},
		"option": {}, "package": {}, "public": {}, "repeated": {}, "reserved": {},
		"returns": {}, "rpc": {}, "service": {}, "syntax": {},
	}
)

// ProtobufName returns the identifier written for a protobuf message, service,
// or method. It keeps common acronyms uppercase and makes the first character
// legal for protobuf source.
func ProtobufName(name string) string {
	panic("excised: ProtobufName")
}

// ProtobufFieldName returns the snake-case identifier written for a protobuf
// field or oneof. It makes the first character legal for protobuf source.
func ProtobufFieldName(name string) string {
	panic("excised: ProtobufFieldName")
}

// protobufIdentifier removes characters protobuf identifiers cannot contain
// and separates digits so the generated Go name matches protoc-gen-go.
func protobufIdentifier(name string, firstUpper, acronym bool) string {
	panic("excised: protobufIdentifier")
}
