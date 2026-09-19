# Exported API — osmetadata

`GetLocalMetadata() (*InstanceMetadata, error)` — default search order is config drive then the link-local metadata service.

The same package’s tests construct a service with an explicit search order and exercise config-drive vs HTTP fallback. JSON fields: name, meta.KubernetesCluster, project_id, availability_zone, hostname, uuid.

Callers: nodeup OpenStack identity.
