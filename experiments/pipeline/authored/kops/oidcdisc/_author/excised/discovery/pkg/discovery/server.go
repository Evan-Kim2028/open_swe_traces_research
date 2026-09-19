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
	_ "encoding/json"
	_ "fmt"
	"net/http"

	_ "k8s.io/klog/v2"

	_ "example.internal/clustkit/discovery/apis/discovery.kops.k8s.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Server struct {
	Store Store
	mux   *http.ServeMux
}

func NewServer(store Store) *Server {
	panic("excised: NewServer")
}

func (s *Server) registerRoutes() {
	// Public OIDC Discovery
	s.mux.HandleFunc("GET /{universe}/.well-known/openid-configuration", s.handleOIDCDiscovery)
	s.mux.HandleFunc("GET /{universe}/openid/v1/jwks", s.handleOIDCJWKS)

	// Authenticated Routes
	// 1. Root Discovery (/apis)
	s.mux.HandleFunc("GET /{universe}/apis", s.withAuth(s.handleAPIGroupList))

	// Discovering resources in group
	s.mux.HandleFunc("GET /{universe}/apis/discovery.clustkit.k8s.io/v1alpha1", s.withAuth(s.handleAPIResourceList))

	// Listing DiscoveryEndpoints (All namespaces)
	s.mux.HandleFunc("GET /{universe}/apis/discovery.clustkit.k8s.io/v1alpha1/discoveryendpoints", s.withAuth(s.handleListDiscoveryEndpoints))

	// Listing DiscoveryEndpoints (Specific namespace)
	s.mux.HandleFunc("GET /{universe}/apis/discovery.clustkit.k8s.io/v1alpha1/namespaces/{namespace}/discoveryendpoints", s.withAuth(s.handleListDiscoveryEndpoints))

	// Create DiscoveryEndpoint
	s.mux.HandleFunc("POST /{universe}/apis/discovery.clustkit.k8s.io/v1alpha1/namespaces/{namespace}/discoveryendpoints", s.withAuth(s.handleCreateDiscoveryEndpoint))

	// Get DiscoveryEndpoint
	s.mux.HandleFunc("GET /{universe}/apis/discovery.clustkit.k8s.io/v1alpha1/namespaces/{namespace}/discoveryendpoints/{name}", s.withAuth(s.handleGetDiscoveryEndpoint))

	// Apply (Patch) DiscoveryEndpoint
	s.mux.HandleFunc("PATCH /{universe}/apis/discovery.clustkit.k8s.io/v1alpha1/namespaces/{namespace}/discoveryendpoints/{name}", s.withAuth(s.handleApplyDiscoveryEndpoint))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("excised: Server.ServeHTTP")
}

func (s *Server) withAuth(next func(http.ResponseWriter, *http.Request, *UserInfo)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		universeID := r.PathValue("universe")
		userInfo, err := AuthenticateClientToUniverse(r, universeID)
		if err != nil {
			klog.Warningf("Unauthorized access attempt to universe %s: %v", universeID, err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r, userInfo)
	}
}

// Handlers

func (s *Server) handleAPIGroupList(w http.ResponseWriter, r *http.Request, _ *UserInfo) {
	resp := metav1.APIGroupList{
		TypeMeta: metav1.TypeMeta{Kind: "APIGroupList", APIVersion: "v1"},
		Groups: []metav1.APIGroup{
			{
				Name: "discovery.clustkit.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{GroupVersion: "discovery.kops.k8s.io/v1alpha1", Version: "v1alpha1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{
					GroupVersion: "discovery.kops.k8s.io/v1alpha1",
					Version:      "v1alpha1",
				},
			},
		},
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAPIResourceList(w http.ResponseWriter, r *http.Request, _ *UserInfo) {
	resp := metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
		GroupVersion: "discovery.kops.k8s.io/v1alpha1",
		APIResources: []metav1.APIResource{
			{
				Name:         "discoveryendpoints",
				SingularName: "discoveryendpoint",
				Namespaced:   true,
				Kind:         "DiscoveryEndpoint",
				Verbs:        []string{"get", "list", "create", "update", "patch"},
			},
		},
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleListDiscoveryEndpoints(w http.ResponseWriter, r *http.Request, _ *UserInfo) {
	panic("excised: Server.handleListDiscoveryEndpoints")
}

func (s *Server) handleCreateDiscoveryEndpoint(w http.ResponseWriter, r *http.Request, userInfo *UserInfo) {
	panic("excised: Server.handleCreateDiscoveryEndpoint")
}

func (s *Server) handleApplyDiscoveryEndpoint(w http.ResponseWriter, r *http.Request, userInfo *UserInfo) {
	panic("excised: Server.handleApplyDiscoveryEndpoint")
}

func (s *Server) handleGetDiscoveryEndpoint(w http.ResponseWriter, r *http.Request, _ *UserInfo) {
	panic("excised: Server.handleGetDiscoveryEndpoint")
}
