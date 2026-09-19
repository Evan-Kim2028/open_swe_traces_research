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
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/klog/v2"
)

func getNodeReadyStatus(node *v1.Node) v1.ConditionStatus {
	panic("excised: getNodeReadyStatus")
}

func findNodeCondition(node *v1.Node, conditionType v1.NodeConditionType) *v1.NodeCondition {
	panic("excised: findNodeCondition")
}

// isNodeReady returns if a Node is considered ready.
// It is considered ready if:
// 1) its Ready condition is set to true
// 2) doesn't have NetworkUnavailable condition set to true
func isNodeReady(node *v1.Node) bool {
	panic("excised: isNodeReady")
}
