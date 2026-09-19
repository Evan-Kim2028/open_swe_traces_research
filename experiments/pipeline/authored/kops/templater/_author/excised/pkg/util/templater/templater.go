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

package templater

import (
	_ "bytes"
	_ "fmt"
	_ "strings"

	"example.internal/clustkit/pkg/apis/kops"
	"example.internal/clustkit/third_party/forked/text/template"
)

const (
	templateName = "mainTemplate"
)

// Templater is golang template renders
type Templater struct {
	channel *kops.Channel
}

// NewTemplater returns a new renderer implementation
func NewTemplater(channel *kops.Channel) *Templater {
	panic("excised: NewTemplater")
}

// Render is responsible for actually rendering the template
func (r *Templater) Render(content string, context map[string]interface{}, snippets map[string]string, failOnMissing bool) (rendered string, err error) {
	panic("excised: Templater.Render")
}

// indentContent is responsible for indenting the string content
func indentContent(indent int, content string) string {
	panic("excised: indentContent")
}

// includeSnippet is responsible for including a snippet
func includeSnippet(tm *template.Template, name string, context map[string]interface{}) (string, error) {
	panic("excised: includeSnippet")
}
