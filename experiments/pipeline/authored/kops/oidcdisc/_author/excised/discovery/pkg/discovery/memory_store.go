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
	"context"
	"sync"
	_ "time"

	api "example.internal/clustkit/discovery/apis/discovery.kops.k8s.io/v1alpha1"
)

type MemoryStore struct {
	universes map[string]*Universe
	mu        sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	panic("excised: NewMemoryStore")
}

func (s *MemoryStore) getOrCreateUniverse(id string) *Universe {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.universes[id]; ok {
		return u
	}
	u := &Universe{
		ID:                 id,
		DiscoveryEndpoints: make(map[NamespacedName]*api.DiscoveryEndpoint),
	}
	s.universes[id] = u
	return u
}

func (s *MemoryStore) UpsertDiscoveryEndpoint(ctx context.Context, universeID string, ep *api.DiscoveryEndpoint) error {
	panic("excised: MemoryStore.UpsertDiscoveryEndpoint")
}

func (s *MemoryStore) ListDiscoveryEndpoints(ctx context.Context, universeID string) ([]*api.DiscoveryEndpoint, error) {
	panic("excised: MemoryStore.ListDiscoveryEndpoints")
}

func (s *MemoryStore) GetDiscoveryEndpoint(ctx context.Context, universeID string, ns, name string) (*api.DiscoveryEndpoint, error) {
	panic("excised: MemoryStore.GetDiscoveryEndpoint")
}
