# Exported API — addonparse

`ParseAddons(name string, location *url.URL, data []byte) (*Addons, error)`

`(*Addons) GetCurrent(kubernetesVersion semver.Version) (*AddonMenu, error)` — version-filter and pick a winner per addon name.

`(*Addon) ChannelVersion() *ChannelVersion`

`(*Addon) GetRequiredUpdates(ctx, k8sClient, cmClient, existingVersion) (*AddonUpdate, error)` — nil update means nothing to do.

Callers: channels apply/get-addons. In-tree tests parse YAML, filter by kubernetes version, assert replacement, and required-updates/PKI.
