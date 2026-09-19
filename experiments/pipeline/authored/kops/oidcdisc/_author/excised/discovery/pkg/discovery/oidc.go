/*
Copyright 2025 The ClusterKit Authors.

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

package discovery

import (
	"encoding/json"
	"fmt"
	"net/http"
	_ "strings"

	_ "example.internal/clustkit/discovery/apis/discovery.kops.k8s.io/v1alpha1"
	_ "k8s.io/klog/v2"
)

func (s *Server) handleOIDCDiscovery(w http.ResponseWriter, r *http.Request) {
	panic("excised: Server.handleOIDCDiscovery")
}

func (s *Server) handleOIDCJWKS(w http.ResponseWriter, r *http.Request) {
	panic("excised: Server.handleOIDCJWKS")
}

type OIDCDiscoveryResponse struct {
	Issuer                           string   `json:"issuer,omitempty"`
	JWKSURI                          string   `json:"jwks_uri,omitempty"`
	ResponseTypesSupported           []string `json:"response_types_supported,omitempty"`
	SubjectTypesSupported            []string `json:"subject_types_supported,omitempty"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported,omitempty"`
}

func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		fmt.Printf("Error encoding response: %v\n", err)
	}
}
