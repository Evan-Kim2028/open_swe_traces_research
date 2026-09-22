// Package kopscodecs_test is the hidden black-box suite for kopscodecs.
// One TestDetailNN per DETAILS.md commitment. Exported API only:
// ToVersionedYaml/JSON(+WithVersion), ToMediaTypeWithVersion, Decode.
package kopscodecs_test

import (
	"strings"
	"testing"

	kops "example.internal/clustkit/pkg/apis/kops"
	"example.internal/clustkit/pkg/kopscodecs"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var gvV1alpha2 = schema.GroupVersions{{Group: "kops.k8s.io", Version: "v1alpha2"}}

func clusterObj() *kops.Cluster {
	c := &kops.Cluster{}
	c.Name = "c1"
	return c
}

// Detail 1 (Inferable: yes): ToVersionedYaml / ToVersionedJSON encode in the
// default registered version, v1alpha2.
func TestDetail01(t *testing.T) {
	y, err := kopscodecs.ToVersionedYaml(clusterObj())
	if err != nil {
		t.Fatalf("ToVersionedYaml: %v", err)
	}
	if !strings.Contains(string(y), "v1alpha2") || !strings.Contains(string(y), "Cluster") {
		t.Fatalf("yaml missing default version/kind: %q", y)
	}
	j, err := kopscodecs.ToVersionedJSON(clusterObj())
	if err != nil {
		t.Fatalf("ToVersionedJSON: %v", err)
	}
	if !strings.Contains(string(j), "v1alpha2") {
		t.Fatalf("json missing default version: %q", j)
	}
}

// Detail 2 (Inferable: partially): an unsupported media type is an error that
// mentions the serializer (not an empty result).
func TestDetail02(t *testing.T) {
	out, err := kopscodecs.ToMediaTypeWithVersion(clusterObj(), "application/x-bogus", gvV1alpha2)
	if err == nil {
		t.Fatalf("bogus media type produced output %q", out)
	}
	if !strings.Contains(err.Error(), "serializer") {
		t.Fatalf("error does not mention serializer: %v", err)
	}
}

// Detail 3 (Inferable: no): *unstructured.Unstructured input bypasses version
// remapping and encodes via the raw media-type serializer. Asserted shape:
// an unstructured object whose GVK is NOT registered still encodes and its
// apiVersion/kind pass through unchanged.
func TestDetail03(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": "cm1"},
	}}
	y, err := kopscodecs.ToVersionedYaml(u)
	if err != nil {
		t.Fatalf("unstructured encode: %v", err)
	}
	if !strings.Contains(string(y), "ConfigMap") || !strings.Contains(string(y), "cm1") {
		t.Fatalf("unstructured content not passed through: %q", y)
	}
}

// Detail 4 (Inferable: partially): Decode decodes unstructured first for the
// GVK, then re-decodes typed only for clustkit groups. Asserted shape: a
// doc in the registered group comes back as a typed object (not *unstructured.Unstructured).
func TestDetail04(t *testing.T) {
	doc := []byte("apiVersion: kops.k8s.io/v1alpha2\nkind: InstanceGroup\nmetadata:\n  name: ig1\n")
	obj, gvk, err := kopscodecs.Decode(doc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if _, isU := obj.(*unstructured.Unstructured); isU {
		t.Fatal("registered-group doc decoded to unstructured")
	}
	if gvk == nil || gvk.Kind != "InstanceGroup" {
		t.Fatalf("gvk = %v", gvk)
	}
}

// Detail 5 (Inferable: partially): a non-clustkit apiVersion decodes to the
// unstructured object as-is, with its GVK returned.
func TestDetail05(t *testing.T) {
	doc := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm1\n")
	obj, gvk, err := kopscodecs.Decode(doc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if _, isU := obj.(*unstructured.Unstructured); !isU {
		t.Fatalf("foreign doc decoded typed: %T", obj)
	}
	if gvk == nil || gvk.Kind != "ConfigMap" || gvk.Version != "v1" {
		t.Fatalf("gvk = %v", gvk)
	}
}

// Detail 6 (Inferable: no): the legacy short api group is accepted into the
// clustkit decode path; the current group decodes directly. Asserted shape:
// the current group (kops.k8s.io, visible in kept register.go) decodes to a
// typed object, and a short-group doc (kops/v1alpha2 — the short form of the
// registered group) is NOT returned as a bare foreign-unstructured success:
// it either decodes typed or surfaces a typed-path error. (Under the
// reference implementation the legacy spelling reaches typed decode and
// errors rather than decoding — DETAILS/gold divergence, reported.)
func TestDetail06(t *testing.T) {
	doc := []byte("apiVersion: kops.k8s.io/v1alpha2\nkind: InstanceGroup\nmetadata:\n  name: ig1\n")
	obj, _, err := kopscodecs.Decode(doc, nil)
	if err != nil {
		t.Fatalf("current group: %v", err)
	}
	if _, isU := obj.(*unstructured.Unstructured); isU {
		t.Fatal("current group decoded to unstructured")
	}

	legacy := []byte("apiVersion: kops/v1alpha2\nkind: InstanceGroup\nmetadata:\n  name: ig1\n")
	obj, _, err = kopscodecs.Decode(legacy, nil)
	if err == nil {
		if _, isU := obj.(*unstructured.Unstructured); isU {
			t.Fatal("legacy short-group doc silently treated as foreign-unstructured")
		}
	}
}

// Detail 7 (Inferable: no): the rewrite touches only the legacy v1alpha2
// spelling — other versions are left untouched. Asserted shape: a
// kops/v1alpha3 doc does NOT yield a typed InstanceGroup (either an error or
// an unstructured result — never a silent typed success).
func TestDetail07(t *testing.T) {
	doc := []byte("apiVersion: kops/v1alpha3\nkind: InstanceGroup\nmetadata:\n  name: ig1\n")
	obj, _, err := kopscodecs.Decode(doc, nil)
	if err == nil {
		if _, isU := obj.(*unstructured.Unstructured); !isU {
			t.Fatalf("kops/v1alpha3 silently decoded typed: %T", obj)
		}
	}
}

// Detail 8 (Inferable: yes): a first-stage decode failure propagates as
// (object, gvk, error) without attempting the typed decode.
func TestDetail08(t *testing.T) {
	obj, _, err := kopscodecs.Decode([]byte("\t{bad: [unclosed"), nil)
	if err == nil {
		t.Fatalf("garbage decoded: %v", obj)
	}
}

// Detail 9 (Inferable: no): encode errors are wrapped with the object type
// and which encoder was used. Asserted shape: encoding an object the scheme
// does not know fails and the error names the offending object type.
func TestDetail09(t *testing.T) {
	_, err := kopscodecs.ToVersionedYaml(&unstructured.UnstructuredList{})
	if err == nil {
		t.Fatal("encoding unregistered type did not error")
	}
	if !strings.Contains(err.Error(), "UnstructuredList") || !strings.Contains(err.Error(), "encoder") {
		t.Fatalf("error does not name object type/encoder: %v", err)
	}
}
