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

package channels

import (
	"fmt"
	"net/url"
	_ "strings"

	"example.internal/clustkit/channels/pkg/api"
	_ "example.internal/clustkit/pkg/kubemanifest"
	_ "example.internal/clustkit/upup/pkg/fi/utils"
	"example.internal/clustkit/util/pkg/vfs"
	_ "github.com/blang/semver/v4"
	_ "k8s.io/klog/v2"
)

type Addons struct {
	ChannelName     string
	ChannelLocation url.URL
	APIObject       *api.Addons
}

func LoadAddons(vfsContext *vfs.VFSContext, name string, location *url.URL) (*Addons, error) {
	klog.V(2).Infof("Loading addons channel from %q", location)
	data, err := vfsContext.ReadFile(location.String())
	if err != nil {
		return nil, fmt.Errorf("error reading addons from %q: %v", location, err)
	}

	return ParseAddons(name, location, data)
}

func ParseAddons(name string, location *url.URL, data []byte) (*Addons, error) {
	panic("excised: ParseAddons")
}

func (a *Addons) GetCurrent(kubernetesVersion semver.Version) (*AddonMenu, error) {
	panic("excised: Addons.GetCurrent")
}

func (a *Addons) wrapInAddons() ([]*Addon, error) {
	panic("excised: Addons.wrapInAddons")
}

func (s *Addon) matches(kubernetesVersion semver.Version) bool {
	panic("excised: Addon.matches")
}
