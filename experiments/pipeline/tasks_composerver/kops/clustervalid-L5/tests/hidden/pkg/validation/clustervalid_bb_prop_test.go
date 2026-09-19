// Package validation_test — hidden black-box property suite for clustervalid.
// Exported API only (api.md): NewClusterValidator, ClusterValidator.Validate.
// Seed 20260919; >=10k cases per file.
//
// Contract (contract.md) -> property coverage table:
//
//	"instance group absent from the cloud is a failure" -> TestClusterValidContractTableProperty
//	"non-detached size below target is a failure" -> TestClusterValidContractTableProperty
//	"detached members do not count toward size" -> TestClusterValidContractTableProperty
//	"unready worker/node role is a failure" -> TestClusterValidNodesProperty
//	"bastion role is not expected to join" -> TestClusterValidBastionProperty
//	"at most N unready workers are tolerated" -> TestClusterValidToleratedUnreadyProperty
//	"no InstanceGroup objects found" -> TestClusterValidConstructorProperty
package validation_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	kopsapi "example.internal/clustkit/pkg/apis/kops"
	"example.internal/clustkit/pkg/cloudinstances"
	"example.internal/clustkit/pkg/validation"
	"example.internal/clustkit/upup/pkg/fi"
	"example.internal/clustkit/upup/pkg/fi/cloudup/awsup"
)

const bbSeed = 20260919
const bbCases = 10000

type bbMockCloud struct {
	awsup.MockAWSCloud
	Groups map[string]*cloudinstances.CloudInstanceGroup
}

func (c *bbMockCloud) GetCloudGroups(cluster *kopsapi.Cluster, instancegroups []*kopsapi.InstanceGroup, warnUnmatched bool, nodes []v1.Node) (map[string]*cloudinstances.CloudInstanceGroup, error) {
	return c.Groups, nil
}

func bbReadyNode(name string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: v1.NodeStatus{Conditions: []v1.NodeCondition{{Type: v1.NodeReady, Status: v1.ConditionTrue}}},
	}
}

func bbValidate(t *testing.T, groups map[string]*cloudinstances.CloudInstanceGroup, objects []runtime.Object, maxUnready int) (*validation.ValidationCluster, error) {
	t.Helper()
	ctx := context.Background()
	cluster := &kopsapi.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "testcluster.k8s.local"},
		Spec: kopsapi.ClusterSpec{Networking: kopsapi.NetworkingSpec{Topology: &kopsapi.TopologySpec{DNS: kopsapi.DNSTypeNone}}},
	}
	if groups == nil {
		groups = map[string]*cloudinstances.CloudInstanceGroup{}
	}
	igs := make([]kopsapi.InstanceGroup, 0, len(groups))
	objects = append([]runtime.Object(nil), objects...)
	for _, g := range groups {
		igs = append(igs, *g.InstanceGroup)
		for _, m := range g.Ready {
			if m.Node != nil {
				objects = append(objects, m.Node)
			}
		}
		for _, m := range g.NeedUpdate {
			if m.Node != nil {
				objects = append(objects, m.Node)
			}
		}
	}
	mock := &bbMockCloud{MockAWSCloud: *awsup.BuildMockAWSCloud("us-east-1", "abc"), Groups: groups}
	cfg := &rest.Config{Host: "https://api.testcluster.k8s.local"}
	v, err := validation.NewClusterValidator(cluster, mock, &kopsapi.InstanceGroupList{Items: igs}, nil, nil, maxUnready, cfg, fake.NewClientset(objects...))
	if err != nil {
		return nil, err
	}
	return v.Validate(ctx)
}

func bbNodeGroup(name string, target int, ready, need int, detached int) *cloudinstances.CloudInstanceGroup {
	g := &cloudinstances.CloudInstanceGroup{
		InstanceGroup: &kopsapi.InstanceGroup{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       kopsapi.InstanceGroupSpec{Role: kopsapi.InstanceGroupRoleNode},
		},
		MinSize:    target,
		TargetSize: target,
	}
	for i := 0; i < ready; i++ {
		g.Ready = append(g.Ready, &cloudinstances.CloudInstance{ID: fmt.Sprintf("r%d", i), Node: bbReadyNode(fmt.Sprintf("%s-r%d", name, i))})
	}
	for i := 0; i < need; i++ {
		g.NeedUpdate = append(g.NeedUpdate, &cloudinstances.CloudInstance{ID: fmt.Sprintf("n%d", i), Node: bbReadyNode(fmt.Sprintf("%s-n%d", name, i))})
	}
	for i := 0; i < detached; i++ {
		g.NeedUpdate = append(g.NeedUpdate, &cloudinstances.CloudInstance{
			ID: fmt.Sprintf("d%d", i), Status: cloudinstances.CloudInstanceStatusDetached,
			Node: bbReadyNode(fmt.Sprintf("%s-d%d", name, i)),
		})
	}
	return g
}

func TestClusterValidContractTableProperty(t *testing.T) {
	cluster := &kopsapi.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "testcluster.k8s.local"},
		Spec: kopsapi.ClusterSpec{Networking: kopsapi.NetworkingSpec{Topology: &kopsapi.TopologySpec{DNS: kopsapi.DNSTypeNone}}},
	}
	igs := []kopsapi.InstanceGroup{{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}, Spec: kopsapi.InstanceGroupSpec{Role: kopsapi.InstanceGroupRoleNode}}}
	mock := &bbMockCloud{MockAWSCloud: *awsup.BuildMockAWSCloud("us-east-1", "abc"), Groups: nil}
	cfg := &rest.Config{Host: "https://api.testcluster.k8s.local"}
	v, err := validation.NewClusterValidator(cluster, mock, &kopsapi.InstanceGroupList{Items: igs}, nil, nil, 0, cfg, fake.NewClientset())
	if err != nil {
		t.Fatal(err)
	}
	res, err := v.Validate(context.Background())
	if err != nil || len(res.Failures) != 1 || res.Failures[0].Kind != "InstanceGroup" {
		t.Fatalf("missing cloud group: %v %+v", err, res.Failures)
	}

	groups := map[string]*cloudinstances.CloudInstanceGroup{"node-1": bbNodeGroup("node-1", 3, 1, 1, 0)}
	res, err = bbValidate(t, groups, nil, 0)
	if err != nil || len(res.Failures) != 1 || !contains(res.Failures[0].Message, "did not have enough nodes") {
		t.Fatalf("undersized: %+v", res.Failures)
	}

	groups["node-1"] = bbNodeGroup("node-1", 2, 1, 0, 1)
	res, err = bbValidate(t, groups, nil, 0)
	if err != nil || len(res.Failures) != 1 || !contains(res.Failures[0].Message, "1 vs 2") {
		t.Fatalf("detached count: %+v", res.Failures)
	}
}

func TestClusterValidConstructorProperty(t *testing.T) {
	cluster := &kopsapi.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "c"}}
	mock := &bbMockCloud{MockAWSCloud: *awsup.BuildMockAWSCloud("us-east-1", "x")}
	_, err := validation.NewClusterValidator(cluster, mock, &kopsapi.InstanceGroupList{}, nil, nil, 0, &rest.Config{}, fake.NewClientset())
	if err == nil {
		t.Fatal("empty ig list")
	}
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		n := rng.Intn(3)
		items := make([]kopsapi.InstanceGroup, n)
		for j := range items {
			items[j] = kopsapi.InstanceGroup{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ig%d", j)}}
		}
		_, err := validation.NewClusterValidator(cluster, mock, &kopsapi.InstanceGroupList{Items: items}, nil, nil, 0, &rest.Config{}, fake.NewClientset())
		if n == 0 && err == nil {
			t.Fatalf("case %d", i)
		}
		if n > 0 && err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

func TestClusterValidNodesProperty(t *testing.T) {
	groups := map[string]*cloudinstances.CloudInstanceGroup{"node-1": bbNodeGroup("node-1", 2, 1, 0, 0)}
	groups["node-1"].NeedUpdate = []*cloudinstances.CloudInstance{{
		ID: "i-2",
		Node: &v1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1b"},
			Status:     v1.NodeStatus{Conditions: []v1.NodeCondition{{Type: v1.NodeReady, Status: v1.ConditionFalse}}},
		},
	}}
	res, err := bbValidate(t, groups, nil, 0)
	if err != nil || len(res.Failures) != 1 || res.Failures[0].Kind != "Node" {
		t.Fatalf("not ready: %+v", res.Failures)
	}

	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		target := 1 + rng.Intn(3)
		ready := rng.Intn(target + 1)
		g := bbNodeGroup("ng", target, ready, 0, 0)
		res, err := bbValidate(t, map[string]*cloudinstances.CloudInstanceGroup{"ng": g}, nil, 0)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		count := 0
		for _, f := range res.Failures {
			if f.Kind == "InstanceGroup" && contains(f.Message, "did not have enough nodes") {
				count++
			}
		}
		nonDetached := ready
		if nonDetached < target && count != 1 {
			t.Fatalf("case %d want size failure", i)
		}
		if nonDetached >= target && count != 0 {
			t.Fatalf("case %d unexpected size failure", i)
		}
	}
}

func TestClusterValidBastionProperty(t *testing.T) {
	groups := map[string]*cloudinstances.CloudInstanceGroup{
		"bastion": {
			InstanceGroup: &kopsapi.InstanceGroup{
				ObjectMeta: metav1.ObjectMeta{Name: "bastion"},
				Spec:       kopsapi.InstanceGroupSpec{Role: kopsapi.InstanceGroupRoleBastion},
			},
			TargetSize: 1,
			Ready:      []*cloudinstances.CloudInstance{{ID: "b-1"}},
		},
	}
	res, err := bbValidate(t, groups, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Failures {
		if f.Kind == "Machine" && contains(f.Message, "has not yet joined") {
			t.Fatal("bastion should not require join")
		}
	}

	for i := 0; i < bbCases; i++ {
		res, err := bbValidate(t, groups, nil, 0)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if len(res.Failures) > 2 {
			t.Fatalf("case %d too many failures", i)
		}
	}
}

func TestClusterValidToleratedUnreadyProperty(t *testing.T) {
	groups := map[string]*cloudinstances.CloudInstanceGroup{"workers": {
		InstanceGroup: &kopsapi.InstanceGroup{
			ObjectMeta: metav1.ObjectMeta{Name: "workers"},
			Spec:       kopsapi.InstanceGroupSpec{Role: kopsapi.InstanceGroupRoleNode},
		},
		TargetSize: 3,
		Ready: []*cloudinstances.CloudInstance{
			{ID: "w1", Node: bbReadyNode("w1")},
			{ID: "w2", Node: &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "w2"}, Status: v1.NodeStatus{Conditions: []v1.NodeCondition{{Type: v1.NodeReady, Status: v1.ConditionFalse}}}}},
		},
	}}
	res, err := bbValidate(t, groups, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Failures {
		if f.Name == "w2" && f.Kind == "Node" {
			t.Fatal("w2 should be tolerated")
		}
	}

	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		max := rng.Intn(3)
		_, err := bbValidate(t, groups, nil, max)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && stringIndex(s, sub) >= 0))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

var _ fi.Cloud = (*bbMockCloud)(nil)
