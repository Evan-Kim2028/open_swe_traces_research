// Copyright 2021 KVStore Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE: The code in this file is based on code from the
// SQLEngine project, licensed under the Apache License v 2.0
//
// https://github.com/acme/sqlengine/tree/cc5e161ac06827589c4966674597c137cc9e809c/store/kvstore/util/codec/number.go
//

// Copyright 2015 Acme, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codec

import (
	_ "encoding/binary"
	_ "math"

	"github.com/pkg/errors"
)

const signMask uint64 = 0x8000000000000000

// EncodeIntToCmpUint make int v to comparable uint type
func EncodeIntToCmpUint(v int64) uint64 {
	panic("excised: EncodeIntToCmpUint")
}

// DecodeCmpUintToInt decodes the u that encoded by EncodeIntToCmpUint
func DecodeCmpUintToInt(u uint64) int64 {
	panic("excised: DecodeCmpUintToInt")
}

// EncodeInt appends the encoded value to slice b and returns the appended slice.
// EncodeInt guarantees that the encoded value is in ascending order for comparison.
func EncodeInt(b []byte, v int64) []byte {
	panic("excised: EncodeInt")
}

// EncodeIntDesc appends the encoded value to slice b and returns the appended slice.
// EncodeIntDesc guarantees that the encoded value is in descending order for comparison.
func EncodeIntDesc(b []byte, v int64) []byte {
	panic("excised: EncodeIntDesc")
}

// DecodeInt decodes value encoded by EncodeInt before.
// It returns the leftover un-decoded slice, decoded value if no error.
func DecodeInt(b []byte) ([]byte, int64, error) {
	panic("excised: DecodeInt")
}

// DecodeIntDesc decodes value encoded by EncodeInt before.
// It returns the leftover un-decoded slice, decoded value if no error.
func DecodeIntDesc(b []byte) ([]byte, int64, error) {
	panic("excised: DecodeIntDesc")
}

// EncodeUint appends the encoded value to slice b and returns the appended slice.
// EncodeUint guarantees that the encoded value is in ascending order for comparison.
func EncodeUint(b []byte, v uint64) []byte {
	panic("excised: EncodeUint")
}

// EncodeUintDesc appends the encoded value to slice b and returns the appended slice.
// EncodeUintDesc guarantees that the encoded value is in descending order for comparison.
func EncodeUintDesc(b []byte, v uint64) []byte {
	panic("excised: EncodeUintDesc")
}

// DecodeUint decodes value encoded by EncodeUint before.
// It returns the leftover un-decoded slice, decoded value if no error.
func DecodeUint(b []byte) ([]byte, uint64, error) {
	panic("excised: DecodeUint")
}

// DecodeUintDesc decodes value encoded by EncodeInt before.
// It returns the leftover un-decoded slice, decoded value if no error.
func DecodeUintDesc(b []byte) ([]byte, uint64, error) {
	panic("excised: DecodeUintDesc")
}

// EncodeVarint appends the encoded value to slice b and returns the appended slice.
// Note that the encoded result is not memcomparable.
func EncodeVarint(b []byte, v int64) []byte {
	panic("excised: EncodeVarint")
}

// DecodeVarint decodes value encoded by EncodeVarint before.
// It returns the leftover un-decoded slice, decoded value if no error.
func DecodeVarint(b []byte) ([]byte, int64, error) {
	panic("excised: DecodeVarint")
}

// EncodeUvarint appends the encoded value to slice b and returns the appended slice.
// Note that the encoded result is not memcomparable.
func EncodeUvarint(b []byte, v uint64) []byte {
	panic("excised: EncodeUvarint")
}

// DecodeUvarint decodes value encoded by EncodeUvarint before.
// It returns the leftover un-decoded slice, decoded value if no error.
func DecodeUvarint(b []byte) ([]byte, uint64, error) {
	panic("excised: DecodeUvarint")
}

const (
	negativeTagEnd   = 8        // negative tag is (negativeTagEnd - length).
	positiveTagStart = 0xff - 8 // Positive tag is (positiveTagStart + length).
)

// EncodeComparableVarint encodes an int64 to a mem-comparable bytes.
func EncodeComparableVarint(b []byte, v int64) []byte {
	panic("excised: EncodeComparableVarint")
}

// EncodeComparableUvarint encodes uint64 into mem-comparable bytes.
func EncodeComparableUvarint(b []byte, v uint64) []byte {
	panic("excised: EncodeComparableUvarint")
}

var (
	errDecodeInsufficient = errors.New("insufficient bytes to decode value")
	errDecodeInvalid      = errors.New("invalid bytes to decode value")
)

// DecodeComparableUvarint decodes mem-comparable uvarint.
func DecodeComparableUvarint(b []byte) ([]byte, uint64, error) {
	panic("excised: DecodeComparableUvarint")
}

// DecodeComparableVarint decodes mem-comparable varint.
func DecodeComparableVarint(b []byte) ([]byte, int64, error) {
	panic("excised: DecodeComparableVarint")
}
