# Contract (L2) — formmapping

Binding request-supplied key/value data into a destination struct walks fields recursively. The tag name (form, uri, header, or a caller-supplied tag) selects the key; an empty tag falls back to the field name and "-" skips the field. Unexported fields are skipped unless they are anonymous embedded fields. Pointer fields are allocated lazily and only when some inner field was actually set (this also makes self-referential structs safe). A `default=` option supplies a value when the key is absent or the first value is empty; for slice/array fields a `collection_format` tag (csv, ssv, tsv, pipes; multi/default = no split) re-splits the value list, and an unknown collection format is an error. Slices are rebuilt to the value count; fixed arrays must match the value count exactly or it is an error. Scalar conversion: strings keep the raw value, every other kind is space-trimmed first; empty numeric/bool/float strings coerce to the zero value; time.Duration parses duration syntax; time.Time honors `time_format` (default RFC3339), `time_utc`, `time_location`, and unix/unixmilli/unixmicro/unixnano formats. A field of struct or map type whose value is not otherwise handled is decoded as a JSON object/array from the single string. A field whose type implements UnmarshalParam (or a `parser="encoding.TextUnmarshaler"` option whose type implements it) takes the first value verbatim, ahead of built-in handling. Mapping into map[string]string takes the last value per key; map[string][]string copies all values; other map types return the exported conversion errors. Keys absent with no default leave the field untouched and are not errors.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestMappingBaseTypes` | every scalar kind (ints, uints, floats, bool, string) is set from a string value |
| `TestMappingDefault/TestMappingFormWithEmptyToDefault` | absent key or empty first value falls back to the `default=` option |
| `TestMappingSkipField/TestMappingIgnoreField/TestMappingUnexportedField` | `-` tags and unexported fields are skipped |
| `TestMappingSlice/TestMappingArray` | slices resize to values; arrays must match length exactly |
| `TestMappingCollectionFormat(+Invalid)` | csv/ssv/tsv/pipes separators split values; unknown format errors |
| `TestMappingMultipleDefaultWithCollectionFormat` | semicolon defaults split per collection format |
| `TestMappingTime/TestMappingTimeUnixNano/TestMappingTimeDuration` | time_format/utc/location/unix variants and duration parsing |
| `TestMappingStructField/TestMappingMapField` | struct and map fields decode from JSON text |
| `TestMappingPtrField/TestMappingIgnoredCircularRef` | pointers allocated only when set; circular types don't recurse forever |
| `TestMappingCustom*/TestMapping*UnmarshalText*` | UnmarshalParam and parser=encoding.TextUnmarshaler custom types win over built-ins |
| `TestMappingForm/TestMappingURI/TestMapFormWithTag` | tag name selects which key namespace is read |
| `TestMappingFormFieldNotSent/TestMappingEmptyValues` | absent keys leave fields untouched without error |
| `TestMappingUnknownFieldType` | unsupported kinds return the unknown-type error |
