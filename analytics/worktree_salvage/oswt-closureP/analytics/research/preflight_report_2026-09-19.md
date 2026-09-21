# Preflight sweep — helm/kops/gin L0/L2 (2026-09-19)

In-image preflight gate (`pipeline/preflight.py`, `scripts/preflight_task.py`): each
packaged task dir's own `tests/test.sh` runs inside its own task image three times —
bare excised tree (must FAIL with assertions/panic, never `[setup failed]`/build),
with `gold.patch` applied (must PASS), and with `cheat.patch` (must FAIL). Verdicts
are cached by (image id, tests sha256, tree sha256) and every Harbor launch path
(`solve-unit`, `solve-watch`) refuses a task whose latest preflight is not PASS.

| repo | unit | level | bare | gold | cheat | seconds |
|---|---|---|---|---|---|---|
| helm | chartloader | L0 | fail | pass | fail | 29 |
| helm | chartloader | L2 | fail | pass | fail | 29 |
| helm | coalesce | L0 | fail | pass | fail | 25 |
| helm | coalesce | L2 | fail | pass | fail | 21 |
| helm | depresolver | L0 | fail | pass | fail | 46 |
| helm | depresolver | L2 | fail | pass | fail | 49 |
| helm | ignorerules | L0 | fail | pass | fail | 22 |
| helm | ignorerules | L2 | fail | pass | fail | 18 |
| helm | kindsorter | L0 | fail | pass | fail | 37 |
| helm | kindsorter | L2 | fail | pass | fail | 33 |
| helm | memorydriver | L0 | fail | pass | fail | 44 |
| helm | memorydriver | L2 | fail | pass | fail | 39 |
| helm | provenance | L0 | fail | pass | fail | 28 |
| helm | provenance | L2 | fail | pass | fail | 24 |
| helm | repindex | L0 | fail | pass | fail | 48 |
| helm | repindex | L2 | fail | pass | fail | 58 |
| helm | storage | L0 | fail | pass | fail | 27 |
| helm | storage | L2 | fail | pass | fail | 27 |
| helm | strvalsparser | L0 | fail | pass | fail | 14 |
| helm | strvalsparser | L2 | fail | pass | fail | 13 |
| kops | addonparse | L0 | fail | pass | fail | 73 |
| kops | addonparse | L2 | fail | pass | fail | 60 |
| kops | assetsremap | L0 | fail | pass | fail | 40 |
| kops | assetsremap | L2 | fail | pass | fail | 39 |
| kops | clustervalid | L0 | fail | pass | fail | 61 |
| kops | clustervalid | L2 | fail | pass | fail | 62 |
| kops | flagbuilder | L0 | fail | pass | fail | 46 |
| kops | flagbuilder | L2 | fail | pass | fail | 39 |
| kops | issuecert | L0 | fail | pass | fail | 26 |
| kops | issuecert | L2 | fail | pass | fail | 23 |
| kops | memfs | L0 | fail | pass | fail | 81 |
| kops | memfs | L2 | fail | pass | fail | 79 |
| kops | oidcdisc | L0 | fail | pass | fail | 49 |
| kops | oidcdisc | L2 | fail | pass | fail | 34 |
| kops | osmetadata | L0 | fail | pass | fail | 54 |
| kops | osmetadata | L2 | fail | pass | fail | 41 |
| kops | templater | L0 | fail | pass | fail | 114 |
| kops | templater | L2 | fail | pass | fail | 55 |
| kops | tomlwriter | L0 | fail | pass | fail | 40 |
| kops | tomlwriter | L2 | fail | pass | fail | 23 |
| gin | bindingdispatch | L0 | fail | pass | fail | 18 |
| gin | bindingdispatch | L2 | fail | pass | fail | 16 |
| gin | bodydecoders | L0 | fail | pass | fail | 17 |
| gin | bodydecoders | L2 | fail | pass | fail | 17 |
| gin | defaultengine | L0 | fail | pass | fail | 22 |
| gin | defaultengine | L2 | fail | pass | fail | 21 |
| gin | formmapping | L0 | fail | pass | fail | 17 |
| gin | formmapping | L2 | fail | pass | fail | 17 |
| gin | htmlrender | L0 | fail | pass | fail | 18 |
| gin | htmlrender | L2 | fail | pass | fail | 17 |
| gin | jsonrenders | L0 | fail | pass | fail | 16 |
| gin | jsonrenders | L2 | fail | pass | fail | 15 |
| gin | multipartfiles | L0 | fail | pass | fail | 17 |
| gin | multipartfiles | L2 | fail | pass | fail | 16 |
| gin | requestbinders | L0 | fail | pass | fail | 18 |
| gin | requestbinders | L2 | fail | pass | fail | 16 |
| gin | streamrenders | L0 | fail | pass | fail | 14 |
| gin | streamrenders | L2 | fail | pass | fail | 15 |
| gin | validator | L0 | fail | pass | fail | 16 |
| gin | validator | L2 | fail | pass | fail | 15 |

## Totals

| metric | n |
|---|---|
| task dirs | 60 |
| preflight PASS (bare=fail, gold=pass, cheat=fail) | 60 |
| preflight FAIL | 0 |
| total wall seconds | 2009 |


## Real packaging failures found by the gate (all fixed, all re-verified)

The first sweep failed 22/60 dirs across 11 units. Every failure was a genuine
packaging defect or a classifier bug — the gate caught exactly the class of
brokenness that used to reach Harbor as a `[setup failed]` reward-0 trial.

| unit (repo) | evidence (first sweep) | root cause | fix |
|---|---|---|---|
| chartloader (helm) | `FAIL …/internal/chart/v3/loader [build failed]` | excision blanked `gzip/errors/fmt/os` but left `archive` (archive.go) and `maps` (load.go) imported-and-unused | excision blanks them; gold gets a restore pair (hunk extended with trailing context — GNU patch rejects hunks ending on a changed line) |
| repindex (helm) | `FAIL …/pkg/repo/v1 [build failed]` | `"path"` left imported-and-unused | same blank/restore pair in excision+gold |
| strvalsparser (helm) | `FAIL …/pkg/strvals [build failed]` (parser.go:229 syntax error) | excision emitted `panic` + `}` but left the original `for`/func braces as context → stray braces; gold was authored against the malformed tree | excision removes the `for`-close as `-`, keeps func-close as context; gold adjusted to match (`+	}` restores for-close) |
| issuecert (kops) | `FAIL …/pkg/pki [build failed]` (undefined: klog ×3 files, then `"time" imported and not used`) | excision blanked `klog` still used in certificate/csr/privatekey, and left `time` unused in csr.go | un-blank klog in all three files; blank `time`; gold mirrors; dead all-context hunks deleted (patch rejects them) |
| addonparse (kops) | `FAIL …/channels/pkg/channels [build failed]` (undefined: klog, semver) | same class across addons.go/addon.go/channel_version.go | un-blank still-used imports; gold mirrors; degenerate hunks removed |
| oidcdisc (kops) | `FAIL …/discovery/pkg/discovery [build failed]` (undefined: klog server.go:76) | klog blanked though server.go still uses it | un-blank in excision; gold mirror |
| osmetadata (kops) | `FAIL …/openstackmetadata [build failed]` (undefined: mount) | `k8s.io/mount-utils` blanked though `mount.` still used in signatures | un-blank in excision; gold mirror |
| clustervalid (kops) | `cheat: PATCH_APPLY_FAILED` (bare=fail, gold=pass) | authored cheat.patch had context lines missing their leading space → malformed patch | restored the two missing leading spaces in authored + task copies |
| storage (helm) | `ok …/pkg/storage/driver [no tests to run]` | **classifier bug**: sibling package's "no tests to run" (soft infra) masked the real `FAIL pkg/storage` panic | split `SOFT_INFRA_SIGNATURES` out of `INFRA_SIGNATURES`; soft only counts when no fail markers exist — in `preflight.classify_run` and `audit.audit_trial_dir` alike |
| assetsremap, memfs (kops) | `ok …/assetcopy [no tests to run]` / `? …/acls [no test files]` | same classifier masking | same fix; re-run passes |

No solver trials and no Harbor runs were launched; every check ran the task's
own `tests/test.sh` inside the task's own image. Authored fixes live in
`experiments/pipeline/authored/<repo>/<unit>/_author/` (excision.patch,
gold.patch, cheat.patch) and were propagated to all materialised level dirs.
