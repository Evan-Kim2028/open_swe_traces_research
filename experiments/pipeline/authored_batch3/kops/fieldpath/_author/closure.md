# Closure — fieldpath

Package: `util/pkg/reflectutils` (`example.internal/clustkit/util/pkg/reflectutils`).

Files: `util/pkg/reflectutils/field_path.go` (177 lines, 6 funcs).

Removed functions (bodies stubbed): `FieldPath.String`, `ParseFieldPath`, `FieldPath.IsEmpty`, `FieldPath.Matches`, `FieldPath.HasPrefixMatch`. (`FieldPath.Extend` stays — the intact walker in `walk.go` uses it.)

Exported entry point(s): `ParseFieldPath(s)`, `(*FieldPath).String`, `IsEmpty`, `Matches`, `HasPrefixMatch` — the grammar for field paths used by reflective config walking.
