# Exported API — flagbuilder

`BuildFlags(options interface{}) (string, error)` — space-separated `--flag=value` string; quotes values that contain `"`.

`BuildFlagsList(options interface{}) ([]string, error)` — argv form, never quoted.

Struct fields with `flag:"name"` become flags; `flag:"-"` skips a subtree; `flag:"name,repeat"` repeats a string slice as multiple flags instead of comma-join. `flag-empty` suppresses the zero/empty encoding. `flag-include-empty` on `*string` emits even empty strings. Output is sorted. Maps `map[string]string` become `k=v` joined by commas (keys sorted). Durations that stringify as `0` are emitted as `0s`. resource.Quantity uses the decimal form (not millivalues).

Callers: component flag assembly (kubelet, apiserver, KCM). In-tree tests: `TestBuildKCMFlags`, `TestKubeletConfigSpec`, `TestBuildAPIServerFlags`, `TestBuildFlagsQuoting`.
