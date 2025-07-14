/*
Copyright 2025 The Flux authors

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

package generic_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

type mockImplementation struct {
	t *testing.T

	b []byte
}

func (m *mockImplementation) ReadFile(name string) ([]byte, error) {
	m.t.Helper()
	g := NewWithT(m.t)
	g.Expect(name).To(Equal("/var/run/secrets/kubernetes.io/serviceaccount/token"))
	return m.b, nil
}
