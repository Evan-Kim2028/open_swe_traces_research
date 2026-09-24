// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"testing"

	"github.com/google/go-github/v92/github"
)

// auditConfigs builds one AuditLogStreamConfig per constructor, keyed by a
// vendor label. Both Amazon S3 constructors are included separately.
func auditConfigs(enabled bool) map[string]*github.AuditLogStreamConfig {
	return map[string]*github.AuditLogStreamConfig{
		"azureblob": github.NewAzureBlobStreamConfig(enabled, &github.AzureBlobConfig{Container: "c"}),
		"azurehub":  github.NewAzureHubStreamConfig(enabled, &github.AzureHubConfig{Name: "n"}),
		"s3oidc":    github.NewAmazonS3OIDCStreamConfig(enabled, &github.AmazonS3OIDCConfig{Bucket: "b"}),
		"s3keys":    github.NewAmazonS3AccessKeysStreamConfig(enabled, &github.AmazonS3AccessKeysConfig{Bucket: "b"}),
		"splunk":    github.NewSplunkStreamConfig(enabled, &github.SplunkConfig{Domain: "d"}),
		"hec":       github.NewHecStreamConfig(enabled, &github.HecConfig{Domain: "d"}),
		"gcloud":    github.NewGoogleCloudStreamConfig(enabled, &github.GoogleCloudConfig{Bucket: "b"}),
		"datadog":   github.NewDatadogStreamConfig(enabled, &github.DatadogConfig{Site: "US"}),
	}
}

// Detail 1: every constructor copies its enabled argument into
// AuditLogStreamConfig.Enabled. (Inferable: yes)
func TestDetail01(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		for name, cfg := range auditConfigs(enabled) {
			if cfg == nil {
				t.Fatalf("%s constructor returned nil", name)
			}
			if cfg.Enabled != enabled {
				t.Errorf("%s: Enabled = %v, want %v", name, cfg.Enabled, enabled)
			}
		}
	}
}

// Detail 2: every constructor stows its vendor config argument in
// VendorSpecific unchanged. Inferable: partially — asserted that the stored
// value is the very argument passed (same pointer, same type).
func TestDetail02(t *testing.T) {
	blob := &github.AzureBlobConfig{Container: "cont", KeyID: "k"}
	if got := github.NewAzureBlobStreamConfig(true, blob); got.VendorSpecific != blob {
		t.Errorf("AzureBlob: VendorSpecific = %#v, want the passed config", got.VendorSpecific)
	}
	hub := &github.AzureHubConfig{Name: "hub"}
	if got := github.NewAzureHubStreamConfig(true, hub); got.VendorSpecific != hub {
		t.Errorf("AzureHub: VendorSpecific = %#v, want the passed config", got.VendorSpecific)
	}
	oidc := &github.AmazonS3OIDCConfig{Bucket: "b"}
	if got := github.NewAmazonS3OIDCStreamConfig(true, oidc); got.VendorSpecific != oidc {
		t.Errorf("S3OIDC: VendorSpecific = %#v, want the passed config", got.VendorSpecific)
	}
	keys := &github.AmazonS3AccessKeysConfig{Bucket: "b"}
	if got := github.NewAmazonS3AccessKeysStreamConfig(true, keys); got.VendorSpecific != keys {
		t.Errorf("S3AccessKeys: VendorSpecific = %#v, want the passed config", got.VendorSpecific)
	}
	splunk := &github.SplunkConfig{Domain: "d"}
	if got := github.NewSplunkStreamConfig(true, splunk); got.VendorSpecific != splunk {
		t.Errorf("Splunk: VendorSpecific = %#v, want the passed config", got.VendorSpecific)
	}
	hec := &github.HecConfig{Domain: "d"}
	if got := github.NewHecStreamConfig(true, hec); got.VendorSpecific != hec {
		t.Errorf("Hec: VendorSpecific = %#v, want the passed config", got.VendorSpecific)
	}
	gc := &github.GoogleCloudConfig{Bucket: "b"}
	if got := github.NewGoogleCloudStreamConfig(true, gc); got.VendorSpecific != gc {
		t.Errorf("GoogleCloud: VendorSpecific = %#v, want the passed config", got.VendorSpecific)
	}
	dd := &github.DatadogConfig{Site: "US"}
	if got := github.NewDatadogStreamConfig(true, dd); got.VendorSpecific != dd {
		t.Errorf("Datadog: VendorSpecific = %#v, want the passed config", got.VendorSpecific)
	}
}

// Detail 3: StreamType is set to a distinct vendor tag per constructor.
// Inferable: no — asserted as shape only: every constructor emits a non-empty
// StreamType and tags differ across vendor families; the literal spellings
// are NOT pinned.
func TestDetail03(t *testing.T) {
	cfgs := auditConfigs(true)
	for name, cfg := range cfgs {
		if cfg.StreamType == "" {
			t.Errorf("%s: StreamType is empty", name)
		}
	}
	// Cross-vendor pairs must carry different tags (the S3 pair is covered
	// separately by Detail 4).
	names := []string{"azureblob", "azurehub", "s3oidc", "splunk", "hec", "gcloud", "datadog"}
	for i, a := range names {
		for _, b := range names[i+1:] {
			if cfgs[a].StreamType == cfgs[b].StreamType {
				t.Errorf("%s and %s share StreamType %q; vendors must be distinct",
					a, b, cfgs[a].StreamType)
			}
		}
	}
}

// Detail 4: the two Amazon S3 constructors emit the same StreamType and
// differ only in the VendorSpecific payload type. Inferable: no — asserted
// as shape only: the tags are equal; the literal is not pinned.
func TestDetail04(t *testing.T) {
	oidc := &github.AmazonS3OIDCConfig{Bucket: "b"}
	keys := &github.AmazonS3AccessKeysConfig{Bucket: "b"}
	a := github.NewAmazonS3OIDCStreamConfig(true, oidc)
	b := github.NewAmazonS3AccessKeysStreamConfig(true, keys)
	if a.StreamType == "" || a.StreamType != b.StreamType {
		t.Errorf("S3 constructors StreamType mismatch: %q vs %q", a.StreamType, b.StreamType)
	}
	if _, ok := a.VendorSpecific.(*github.AmazonS3OIDCConfig); !ok {
		t.Errorf("S3OIDC VendorSpecific type = %T", a.VendorSpecific)
	}
	if _, ok := b.VendorSpecific.(*github.AmazonS3AccessKeysConfig); !ok {
		t.Errorf("S3AccessKeys VendorSpecific type = %T", b.VendorSpecific)
	}
}

// Detail 5: no constructor validates or mutates the vendor config; a nil cfg
// is stored as a typed-nil VendorSpecific. Inferable: partially — asserted
// that nil never panics and VendorSpecific holds a typed nil of the matching
// vendor type.
func TestDetail05(t *testing.T) {
	checks := []struct {
		name string
		cfg  *github.AuditLogStreamConfig
		want any
	}{
		{"azureblob", github.NewAzureBlobStreamConfig(true, nil), (*github.AzureBlobConfig)(nil)},
		{"azurehub", github.NewAzureHubStreamConfig(true, nil), (*github.AzureHubConfig)(nil)},
		{"s3oidc", github.NewAmazonS3OIDCStreamConfig(true, nil), (*github.AmazonS3OIDCConfig)(nil)},
		{"s3keys", github.NewAmazonS3AccessKeysStreamConfig(true, nil), (*github.AmazonS3AccessKeysConfig)(nil)},
		{"splunk", github.NewSplunkStreamConfig(true, nil), (*github.SplunkConfig)(nil)},
		{"hec", github.NewHecStreamConfig(true, nil), (*github.HecConfig)(nil)},
		{"gcloud", github.NewGoogleCloudStreamConfig(true, nil), (*github.GoogleCloudConfig)(nil)},
		{"datadog", github.NewDatadogStreamConfig(true, nil), (*github.DatadogConfig)(nil)},
	}
	for _, c := range checks {
		if c.cfg == nil {
			t.Errorf("%s: nil vendor config -> nil AuditLogStreamConfig", c.name)
			continue
		}
		if c.cfg.VendorSpecific != c.want {
			t.Errorf("%s: nil vendor config -> VendorSpecific %#v (%T), want typed nil",
				c.name, c.cfg.VendorSpecific, c.cfg.VendorSpecific)
		}
	}
}
