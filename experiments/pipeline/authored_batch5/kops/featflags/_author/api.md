# Exported API — featflags

Package `pkg/featureflag` (importable as `example.internal/clustkit/pkg/featureflag`).

Process-global feature flags driven by the `KOPS_FEATURE_FLAGS` env var.

- `const Name = "KOPS_FEATURE_FLAGS"` — env var read once at package init.
- `func new(key string, defaultValue *bool) *FeatureFlag` — registry insert; re-registration
  of a key returns the SAME flag; the first non-nil default wins.
- `func (f *FeatureFlag) Enabled() bool` — explicit set > default > false.
- `func Bool(b bool) *bool` — pointer helper.
- `func ParseFlags(f string)` — comma-separated list; each item is `Name`, `+Name`, or
  `-Name` (enable / disable); whitespace trimmed; unknown names are logged and skipped;
  empty items skipped.
- `func Get(flagName string) (*FeatureFlag, error)` — registry lookup; error when unknown.

Example: `ParseFlags("+Spotinst,-Azure")` enables Spotinst, disables Azure (until then it
used its registered default).
