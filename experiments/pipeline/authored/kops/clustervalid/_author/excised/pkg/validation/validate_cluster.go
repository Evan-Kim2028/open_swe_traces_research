/*
Copyright 2019 The ClusterKit Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"context"
	_ "fmt"
	_ "net"
	_ "net/url"
	_ "sort"
	_ "strings"

	"example.internal/clustkit/pkg/apis/kops"
	_ "example.internal/clustkit/pkg/dns"
	"example.internal/clustkit/upup/pkg/fi"
	_ "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	_ "k8s.io/client-go/tools/pager"

	"example.internal/clustkit/pkg/cloudinstances"
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/klog/v2"
)

// ValidationCluster uses a cluster to validate.
type ValidationCluster struct {
	Failures []*ValidationError `json:"failures,omitempty"`

	Nodes []*ValidationNode `json:"nodes,omitempty"`
}

// ValidationError holds a validation failure
type ValidationError struct {
	Kind    string `json:"type,omitempty"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
	// The InstanceGroup field is used to indicate which instance group this validation error is coming from
	InstanceGroup *kops.InstanceGroup `json:"instanceGroup,omitempty"`
}

type ClusterValidator interface {
	// Validate validates a k8s cluster
	Validate(ctx context.Context) (*ValidationCluster, error)
}

type clusterValidatorImpl struct {
	cluster    *kops.Cluster
	cloud      fi.Cloud
	restConfig *rest.Config
	k8sClient  kubernetes.Interface

	// allInstanceGroups is the list of all instance groups in the cluster
	allInstanceGroups []*kops.InstanceGroup

	// filterInstanceGroups is a function that returns true if the instance group should be validated
	filterInstanceGroups func(ig *kops.InstanceGroup) bool

	// filterPodsForValidation is a function that returns true if the pod should be validated
	filterPodsForValidation func(pod *v1.Pod) bool

	maxUnreadyNodes int
}

func (v *ValidationCluster) addError(failure *ValidationError) {
	panic("excised: ValidationCluster.addError")
}

// ValidationNode represents the validation status for a node
type ValidationNode struct {
	Name     string             `json:"name,omitempty"`
	Zone     string             `json:"zone,omitempty"`
	Role     string             `json:"role,omitempty"`
	Hostname string             `json:"hostname,omitempty"`
	Status   v1.ConditionStatus `json:"status,omitempty"`
}

// hasPlaceHolderIP checks if the API DNS has been updated.
func hasPlaceHolderIP(host string) (string, error) {
	panic("excised: hasPlaceHolderIP")
}

func NewClusterValidator(cluster *kops.Cluster, cloud fi.Cloud, instanceGroupList *kops.InstanceGroupList, filterInstanceGroups func(ig *kops.InstanceGroup) bool, filterPodsForValidation func(pod *v1.Pod) bool, maxUnreadyNodes int, restConfig *rest.Config, k8sClient kubernetes.Interface) (ClusterValidator, error) {
	panic("excised: NewClusterValidator")
}

func (v *clusterValidatorImpl) Validate(ctx context.Context) (*ValidationCluster, error) {
	panic("excised: clusterValidatorImpl.Validate")
}

var masterStaticPods = []string{
	"kube-apiserver",
	"kube-controller-manager",
	"kube-scheduler",
}

func (v *ValidationCluster) collectPodFailures(ctx context.Context, client kubernetes.Interface, readyNodes []v1.Node, nodeInstanceGroupMapping map[string]*kops.InstanceGroup, podValidationFilter func(pod *v1.Pod) bool, toleratedNodes map[string]bool) error {
	panic("excised: ValidationCluster.collectPodFailures")
}

func (v *ValidationCluster) validateNodes(cloudGroups map[string]*cloudinstances.CloudInstanceGroup, groups []*kops.InstanceGroup, shouldValidateInstanceGroup func(ig *kops.InstanceGroup) bool, toleratedNodes map[string]bool) ([]v1.Node, map[string]*kops.InstanceGroup) {
	panic("excised: ValidationCluster.validateNodes")
}
