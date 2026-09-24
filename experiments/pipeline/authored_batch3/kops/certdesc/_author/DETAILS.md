# Details — certdesc

1. `PkixNameToString` flattens the RDN sequence into comma-joined `key=value` pairs; the 2.5.4.x OIDs map to lowercase short names (3→cn, 5→serial, 6→c, 7→l, 10→o, 11→ou) and anything else falls back to the numeric OID string. Inferable: no — the short-name map is arbitrary.
2. `keyUsageToString` emits the `KeyUsage*` identifier for every SET bit — the Go constant name, not an X.509 name (e.g. `KeyUsageCertSign`, not `Cert Sign`). Inferable: partially — the naming convention is visible in the table.
3. `parseKeyUsage`/`parseExtKeyUsage` exact-match against the same tables and return `(0,false)` on miss — no error, just ok=false. Inferable: partially.
4. `extKeyUsageToString` renders known usages by name and unknowns as `ExtKeyUsage:<n>` with a warning. Inferable: no — the fallback spelling.
5. `BuildTypeDescription` collects `CA` (when IsCA) + usage names + ext-usage names, SORTS them, comma-joins, then reverse-looks-up `wellKnownCertificateTypes` — `CA,KeyUsageCRLSign,KeyUsageCertSign` → `ca` etc. Inferable: no — the canonical table is arbitrary.
6. The comma join has no spaces; the type expansion in `IssueCert` parses the same comma-separated form back. Inferable: partially.
