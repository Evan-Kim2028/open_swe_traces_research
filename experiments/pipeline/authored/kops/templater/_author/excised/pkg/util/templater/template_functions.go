/*
Copyright 2020 The ClusterKit Authors.

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

package templater

import (
	_ "example.internal/clustkit"
	_ "example.internal/clustkit/pkg/apis/kops"
	_ "example.internal/clustkit/pkg/apis/kops/util"
	"example.internal/clustkit/third_party/forked/text/template"
	_ "example.internal/clustkit/util/pkg/architectures"
	_ "github.com/Masterminds/sprig/v3"
	_ "github.com/blang/semver/v4"
)

// templateFuncsMap returns a map if the template functions for this template
func (r *Templater) templateFuncsMap(tm *template.Template) template.FuncMap {
	panic("excised: Templater.templateFuncsMap")
}
