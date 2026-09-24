# DETAILS — auditstream

1. Every constructor copies its `enabled` argument into
   `AuditLogStreamConfig.Enabled`. Inferable: yes — kept in the stub.
2. Every constructor stows its vendor config argument in
   `AuditLogStreamConfig.VendorSpecific` unchanged. Inferable: partially —
   the field exists and the arg is otherwise unused, but pass-through is not
   forced.
3. `StreamType` is set to a distinct human-readable vendor string per
   constructor: "Azure Blob Storage", "Azure Event Hubs", "Amazon S3" (both
   S3 constructors share it), "Splunk", "HTTPS Event Collector", "Google
   Cloud Storage", "Datadog". Inferable: no — the exact literal spellings are
   arbitrary.
4. The two Amazon S3 constructors (OIDC and access-keys) emit the same
   `StreamType` and differ only in the `VendorSpecific` payload type.
   Inferable: no — sharing one stream-type tag is arbitrary.
5. No constructor validates or mutates the vendor config; a nil `cfg` is
   stored as a typed-nil `VendorSpecific`. Inferable: partially — pass-through
   is plausible, nil tolerance is an edge choice.
