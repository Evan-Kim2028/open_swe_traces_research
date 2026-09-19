// Package openstackmetadata_test is a hidden black-box property suite for osmetadata.
// Exported API only (api.md): GetLocalMetadata, InstanceMetadata, constants.
// Seed 20260919; >=10k cases.
//
// Contract (contract.md) -> property coverage table:
//
//	"default order config drive then metadata service" -> TestOsmetadataDefaultSearchOrderProperty
//	"no local metadata returns error" -> TestOsmetadataGetLocalProperty
//	"JSON field tags on InstanceMetadata" -> TestOsmetadataJSONRoundTripProperty
//	"random JSON decode stability" -> TestOsmetadataJSONRandomProperty
package openstackmetadata_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	osm "example.internal/clustkit/upup/pkg/fi/cloudup/openstack/openstackmetadata"
)

const bbSeed = 20260919
const bbCases = 10000

func TestOsmetadataDefaultSearchOrderProperty(t *testing.T) {
	want := osm.ConfigDriveID + ", " + osm.MetadataID
	if osm.DefaultMetadataSearchOrder != want {
		t.Fatalf("DefaultMetadataSearchOrder: got %q want %q", osm.DefaultMetadataSearchOrder, want)
	}
	if osm.MetadataLatestPath != "openstack/latest/meta_data.json" {
		t.Fatalf("MetadataLatestPath constant")
	}
}

func TestOsmetadataGetLocalProperty(t *testing.T) {
	var panicked bool
	var meta *osm.InstanceMetadata
	var err error
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		meta, err = osm.GetLocalMetadata()
	}()
	if panicked {
		t.Fatal("GetLocalMetadata must not panic")
	}
	if err != nil {
		if meta != nil {
			t.Fatalf("error path should not return metadata: %v", err)
		}
		return
	}
	if meta == nil {
		t.Fatal("GetLocalMetadata succeeded without InstanceMetadata")
	}
}

func TestOsmetadataJSONRoundTripProperty(t *testing.T) {
	in := osm.InstanceMetadata{
		Name:             "node-1",
		UserMeta:         &osm.Metadata{ClusterName: "k8s.local"},
		ProjectID:        "proj",
		AvailabilityZone: "az1",
		Hostname:         "host1",
		ServerID:         "uuid-1234",
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["name"] != "node-1" || doc["project_id"] != "proj" {
		t.Fatalf("top-level tags: %v", doc)
	}
	meta, ok := doc["meta"].(map[string]interface{})
	if !ok || meta["KubernetesCluster"] != "k8s.local" {
		t.Fatalf("meta.KubernetesCluster tag: %v", doc)
	}
	out := osm.InstanceMetadata{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != in.Name || out.ServerID != in.ServerID || out.UserMeta.ClusterName != in.UserMeta.ClusterName {
		t.Fatalf("round trip: %+v vs %+v", out, in)
	}
}

func TestOsmetadataJSONRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		doc := map[string]interface{}{
			"name":              rngString(rng, 6),
			"project_id":        rngString(rng, 8),
			"availability_zone": rngString(rng, 4),
			"hostname":          rngString(rng, 5),
			"uuid":              rngString(rng, 12),
			"meta": map[string]interface{}{
				"KubernetesCluster": rngString(rng, 7),
			},
		}
		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("case %d marshal: %v", i, err)
		}
		var got osm.InstanceMetadata
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("case %d unmarshal: %v body %s", i, err, raw)
		}
		if got.Name != doc["name"] || got.ProjectID != doc["project_id"] {
			t.Fatalf("case %d field mismatch %+v", i, got)
		}
		if got.UserMeta == nil || got.UserMeta.ClusterName != doc["meta"].(map[string]interface{})["KubernetesCluster"] {
			t.Fatalf("case %d meta mismatch %+v", i, got)
		}
		raw2, err := json.Marshal(&got)
		if err != nil {
			t.Fatalf("case %d remarshal: %v", i, err)
		}
		var again osm.InstanceMetadata
		if err := json.Unmarshal(raw2, &again); err != nil {
			t.Fatalf("case %d second unmarshal: %v", i, err)
		}
		if again.ServerID != got.ServerID || again.Hostname != got.Hostname {
			t.Fatalf("case %d idempotent decode", i)
		}
	}
}

func rngString(rng *rand.Rand, n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789-"
	var b []byte
	for i := 0; i < n; i++ {
		b = append(b, chars[rng.Intn(len(chars))])
	}
	return string(b)
}
