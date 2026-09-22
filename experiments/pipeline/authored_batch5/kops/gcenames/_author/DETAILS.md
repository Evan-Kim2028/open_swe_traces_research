# Details — gcenames

1. `IsNotFound` uses `errors.As` (unwraps) and checks `Code == 404`; `IsNotReady` does NOT
   unwrap (direct type assert) and scans `Errors[].Reason` for `resourceNotReady`.
   Inferable: partially — the asymmetry is odd but real.
2. All name helpers replace `.` with `-` in the cluster name (GCE rejects dots). Inferable:
   doc — implied by GCE naming rules.
3. `ClusterPrefixedName`/`ClusterSuffixedName` fatal-exit when the fixed part leaves less
   than 10 chars for the cluster portion; truncation uses a 6-char hash suffix when needed.
   Inferable: no — the 10-char floor and hash length are arbitrary.
4. `LabelForCluster` uses a fixed label key constant. Inferable: no — key spelling arbitrary.
5. `ServiceAccountName` caps at 30 chars via `ClusterSuffixedName`. Inferable: no — 30 is
   a GCE constraint but arbitrary from the code alone.
6. `LastComponent` returns the whole string when there's no `/`. Inferable: yes.
7. `SSHUsernameForImage` case-insensitively matches an `ubuntu` prefix on the image's last
   path component; everything else → `admin`. Inferable: partially — the Ubuntu special
   case is a documented quirk.
8. `ZoneToRegion` splits on `-` and takes the first two components; `us-central1` (no zone
   letter) errors. Inferable: yes.
