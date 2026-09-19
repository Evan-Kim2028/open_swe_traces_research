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

package flagbuilder

import (
	_ "fmt"
	_ "reflect"
	_ "sort"
	_ "strconv"
	_ "strings"

	_ "example.internal/clustkit/util/pkg/reflectutils"
	_ "k8s.io/apimachinery/pkg/api/resource"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/klog/v2"
)

// BuildFlags returns a space-separated list of arguments.
// @deprecated: please use BuildFlagsList
func BuildFlags(options interface{}) (string, error) {
	panic("excised: BuildFlags")
}

// BuildFlagsList reflects the options interface and extracts the flags from struct tags.
func BuildFlagsList(options interface{}) ([]string, error) {
	panic("excised: BuildFlagsList")
}

func buildFlagsList(options interface{}, quote func(string) string) ([]string, error) {
	panic("excised: buildFlagsList")
}

// maybeQuote quotes s when it contains a double quote, so values survive the space-separated
// string form that a shell or systemd may parse. argv values use neverQuote instead.
func maybeQuote(s string) string {
	panic("excised: maybeQuote")
}

// neverQuote returns s unchanged, for values used directly as exec argv.
func neverQuote(s string) string {
	panic("excised: neverQuote")
}
