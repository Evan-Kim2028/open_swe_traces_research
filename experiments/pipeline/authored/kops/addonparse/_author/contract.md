# Contract (L2) — addonparse

Parse trims the buffer and loads YAML objects. Empty/whitespace/comment-only input yields an empty addons list (not an error). If the first object is kind Addons with empty group/version, it is reparsed as the channel document. Any other first object is treated as a raw manifest: the addon name is `manifest-` plus the first 12 hex chars of a hash of the location string, the manifest URL is the location, and ManifestHash is a hash of the trimmed content. Invalid YAML is an error.

GetCurrent wraps each spec into an addon (name from spec.Name if set, else the document’s metadata name), drops entries whose KubernetesVersion range does not contain the given version (unparsable range → skip), and for duplicate names keeps the candidate that replaces the existing one.

Replace is true if Id differs, or ManifestHash differs, or the existing SystemGeneration is older than the candidate’s. If the existing generation is newer, do not replace. Equal id+hash+generation is not a replacement.

Required-updates: NeedsPKI consults whether the channel CA secret exists; if PKI is required and missing, InstallPKI is set. If an existing version is present and the new version does not replace it, the new version is cleared. If PKI is already installed and there is no new version, the result is nil (no work).

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestParseAddons` | kind Addons document parses |
| `TestParseAddonsMultiDocIndex` | first-object Addons wins in a multi-doc stream |
| `TestParseAddonsWrapsDirectManifest` | non-Addons YAML is wrapped as a synthetic addon |
| `TestParseAddonsDirectManifestNameUsesLocationHash` | synthetic name is manifest- plus 12 hex of location hash |
| `TestParseAddonsRejectsInvalidYAML` | invalid YAML is an error |
| `TestParseAddonsEmpty` | empty input is not an error |
| `Test_Filtering` | kubernetes version range selects the matching addon |
| `Test_Replacement` | id/hash/generation decide which duplicate wins |
| `Test_GetRequiredUpdates` | nil vs non-nil update from existing version + replace |
| `Test_NeedsRollingUpdate` | (uses GetCurrent/version) rolling when replace is true |
| `Test_InstallPKI` | NeedsPKI missing CA → InstallPKI |
