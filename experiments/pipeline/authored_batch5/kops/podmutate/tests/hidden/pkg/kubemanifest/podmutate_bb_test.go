package kubemanifest

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"example.internal/clustkit/pkg/apis/kops"
	"example.internal/clustkit/pkg/apis/nodeup"
)

// TestDetail01 — RemapImages only applies the mapper to strings whose last path element is
// "image" AND whose grandparent element is "containers" or "initContainers".
func TestDetail01(t *testing.T) {
	o := NewObject(map[string]interface{}{
		"image": "top-level",
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{"name": "c0", "image": "i0"},
			},
			"initContainers": []interface{}{
				map[string]interface{}{"name": "init", "image": "ii"},
			},
			"sidecars": []interface{}{
				map[string]interface{}{"image": "side"},
			},
			"nested": map[string]interface{}{
				"image": "wrong-grandparent",
			},
		},
	})

	var seen []string
	if err := o.RemapImages(func(image string) string {
		seen = append(seen, image)
		return "R:" + image
	}); err != nil {
		t.Fatalf("RemapImages: %v", err)
	}

	spec := o.data["spec"].(map[string]interface{})
	c0 := spec["containers"].([]interface{})[0].(map[string]interface{})
	if c0["image"] != "R:i0" {
		t.Errorf("containers[0].image = %v, want R:i0", c0["image"])
	}
	init := spec["initContainers"].([]interface{})[0].(map[string]interface{})
	if init["image"] != "R:ii" {
		t.Errorf("initContainers[0].image = %v, want R:ii", init["image"])
	}

	if o.data["image"] != "top-level" {
		t.Errorf("top-level image key was remapped (depth < 3): %v", o.data["image"])
	}
	side := spec["sidecars"].([]interface{})[0].(map[string]interface{})
	if side["image"] != "side" {
		t.Errorf("image under non-container grandparent was remapped: %v", side["image"])
	}
	if spec["nested"].(map[string]interface{})["image"] != "wrong-grandparent" {
		t.Errorf("image with wrong grandparent was remapped: %v", spec["nested"])
	}
	if len(seen) != 2 {
		t.Errorf("mapper invoked on %d fields %v, want exactly the 2 qualifying image fields", len(seen), seen)
	}
}

// TestDetail02 — the mapper's result is written back; identical results leave the value
// unchanged (the write-through-mutator-only-on-diff is exercised via an identity mapper).
func TestDetail02(t *testing.T) {
	o := NewObject(map[string]interface{}{
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{"name": "c0", "image": "same"},
			},
		},
	})
	calls := 0
	if err := o.RemapImages(func(image string) string {
		calls++
		return image
	}); err != nil {
		t.Fatalf("RemapImages: %v", err)
	}
	if calls != 1 {
		t.Errorf("mapper called %d times, want 1 (only the qualifying field)", calls)
	}
	c0 := o.data["spec"].(map[string]interface{})["containers"].([]interface{})[0].(map[string]interface{})
	if c0["image"] != "same" {
		t.Errorf("identity remap changed value: %v", c0["image"])
	}
}

// TestDetail03 — VisitContainers visits each map under a "containers" slice index; it does
// not fire on the containers collection itself nor on maps under a non-slice "containers".
func TestDetail03(t *testing.T) {
	o := NewObject(map[string]interface{}{
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{"name": "c0"},
				map[string]interface{}{"name": "c1"},
			},
		},
		"containers": map[string]interface{}{
			"entry": map[string]interface{}{"name": "not-a-container"},
		},
		"deep": map[string]interface{}{
			"inner": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{"name": "deep"},
				},
			},
		},
	})

	var names []string
	err := o.VisitContainers(func(container map[string]interface{}) error {
		if n, ok := container["name"].(string); ok {
			names = append(names, n)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("VisitContainers: %v", err)
	}
	got := map[string]bool{}
	for _, n := range names {
		got[n] = true
	}
	for _, want := range []string{"c0", "c1", "deep"} {
		if !got[want] {
			t.Errorf("container entry %q not visited; visited %v", want, names)
		}
	}
	if got["not-a-container"] {
		t.Errorf("map under non-slice 'containers' key was visited: %v", names)
	}
	if len(names) != 3 {
		t.Errorf("visited %d entries %v, want 3", len(names), names)
	}
}

// TestDetail04 — AddHostPathMapping appends a HostPath volume plus a read-only VolumeMount.
func TestDetail04(t *testing.T) {
	pod := &v1.Pod{}
	pod.Spec.Volumes = []v1.Volume{{Name: "existing"}}
	pod.Spec.Containers = []v1.Container{{Name: "c"}}

	AddHostPathMapping(pod, &pod.Spec.Containers[0], "v1", "/host/p")

	if len(pod.Spec.Volumes) != 2 {
		t.Fatalf("expected appended volume, got %v", pod.Spec.Volumes)
	}
	vol := pod.Spec.Volumes[1]
	if vol.Name != "v1" || vol.HostPath == nil || vol.HostPath.Path != "/host/p" {
		t.Errorf("appended volume wrong: %+v", vol)
	}
	mts := pod.Spec.Containers[0].VolumeMounts
	if len(mts) != 1 {
		t.Fatalf("expected 1 volume mount, got %v", mts)
	}
	if mts[0].Name != "v1" || mts[0].MountPath != "/host/p" || !mts[0].ReadOnly {
		t.Errorf("default mount should be read-only at the given path: %+v", mts[0])
	}
}

// TestDetail05 — the options mutate the just-appended volume/mount.
func TestDetail05(t *testing.T) {
	pod := &v1.Pod{}
	pod.Spec.Containers = []v1.Container{{Name: "c"}}

	AddHostPathMapping(pod, &pod.Spec.Containers[0], "v", "/orig",
		WithReadWrite(),
		WithType(v1.HostPathDirectory),
		WithHostPath("/new-host"),
		WithMountPath("/new-mount"),
	)

	vol := pod.Spec.Volumes[0]
	if vol.HostPath == nil || vol.HostPath.Path != "/new-host" {
		t.Errorf("WithHostPath not applied: %+v", vol.HostPath)
	}
	if vol.HostPath.Type == nil || *vol.HostPath.Type != v1.HostPathDirectory {
		t.Errorf("WithType did not set HostPath.Type pointer: %+v", vol.HostPath)
	}
	mt := pod.Spec.Containers[0].VolumeMounts[0]
	if mt.MountPath != "/new-mount" {
		t.Errorf("WithMountPath not applied: %+v", mt)
	}
	if mt.ReadOnly {
		t.Error("WithReadWrite did not clear ReadOnly")
	}
}

// TestDetail06 — MarkPodAsCritical appends a toleration, preserving existing ones.
func TestDetail06(t *testing.T) {
	pod := &v1.Pod{}
	pod.Spec.Tolerations = []v1.Toleration{{Key: "keep", Operator: v1.TolerationOpEqual}}

	MarkPodAsCritical(pod)

	if len(pod.Spec.Tolerations) != 2 {
		t.Fatalf("expected toleration appended, got %v", pod.Spec.Tolerations)
	}
	if pod.Spec.Tolerations[0].Key != "keep" {
		t.Errorf("existing toleration not preserved: %v", pod.Spec.Tolerations)
	}
	added := pod.Spec.Tolerations[1]
	if added.Operator != v1.TolerationOpExists {
		t.Errorf("appended toleration should have Exists operator (documented): %+v", added)
	}
	if added.Key != "CriticalAddonsOnly" {
		t.Errorf("appended toleration should be CriticalAddonsOnly (documented): %+v", added)
	}
}

// TestDetail07 — priority markers overwrite PriorityClassName unconditionally.
func TestDetail07(t *testing.T) {
	pod := &v1.Pod{Spec: v1.PodSpec{PriorityClassName: "old"}}
	MarkPodAsNodeCritical(pod)
	nodePC := pod.Spec.PriorityClassName
	if nodePC == "" || nodePC == "old" {
		t.Errorf("MarkPodAsNodeCritical did not overwrite PriorityClassName: %q", nodePC)
	}

	pod2 := &v1.Pod{Spec: v1.PodSpec{PriorityClassName: "old"}}
	MarkPodAsClusterCritical(pod2)
	clusterPC := pod2.Spec.PriorityClassName
	if clusterPC == "" || clusterPC == "old" {
		t.Errorf("MarkPodAsClusterCritical did not overwrite PriorityClassName: %q", clusterPC)
	}
	if nodePC == clusterPC {
		t.Errorf("node-critical and cluster-critical priorities should differ, both %q", nodePC)
	}
}

// TestDetail08 — AddHostPathSELinuxContext is gated on cfg.ContainerdConfig.SeLinuxEnabled;
// when enabled it sets the documented pod-level SELinuxOptions (spc_t/s0).
func TestDetail08(t *testing.T) {
	// nil ContainerdConfig → no change
	pod := &v1.Pod{}
	AddHostPathSELinuxContext(pod, &nodeup.Config{})
	if pod.Spec.SecurityContext != nil {
		t.Errorf("nil ContainerdConfig should be a no-op, got %+v", pod.Spec.SecurityContext)
	}

	// disabled → no change
	pod = &v1.Pod{}
	AddHostPathSELinuxContext(pod, &nodeup.Config{
		ContainerdConfig: &kops.ContainerdConfig{SeLinuxEnabled: false},
	})
	if pod.Spec.SecurityContext != nil {
		t.Errorf("disabled SeLinuxEnabled should be a no-op, got %+v", pod.Spec.SecurityContext)
	}

	// enabled → documented SELinux options, creating SecurityContext if needed
	pod = &v1.Pod{}
	AddHostPathSELinuxContext(pod, &nodeup.Config{
		ContainerdConfig: &kops.ContainerdConfig{SeLinuxEnabled: true},
	})
	sc := pod.Spec.SecurityContext
	if sc == nil || sc.SELinuxOptions == nil {
		t.Fatalf("expected pod-level SELinuxOptions, got %+v", sc)
	}
	if sc.SELinuxOptions.Type != "spc_t" || sc.SELinuxOptions.Level != "s0" {
		t.Errorf("SELinuxOptions = %+v, want documented spc_t/s0", sc.SELinuxOptions)
	}
}
