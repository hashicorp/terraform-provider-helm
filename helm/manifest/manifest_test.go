package manifest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
)

const crdYaml = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: firsts.stable.example.com
spec:
  group: stable.example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                cronSpec:
                  type: string
                image:
                  type: string
                replicas:
                  type: integer
  scope: Namespaced
  names:
    plural: firsts
    singular: first
    kind: First
`

func TestCrdToManifest(t *testing.T) {
	tests := []struct {
		name     string
		crds     []chart.CRD
		expected string
	}{
		{
			name: "basic",
			crds: []chart.CRD{
				{
					Name: "foo",
					File: &chart.File{
						Data: []byte(crdYaml),
					},
				},
			},
			expected: fmt.Sprintf("---\n# Source: %s\n%s", "foo", crdYaml),
		},
		{
			name: "white space",
			crds: []chart.CRD{
				{
					Name: "foo",
					File: &chart.File{
						Data: []byte(fmt.Sprintf("\n\n---\n%s", crdYaml)),
					},
				},
			},
			expected: fmt.Sprintf("---\n# Source: %s\n%s", "foo", crdYaml),
		},
		{
			name: "multidoc",
			crds: []chart.CRD{
				{
					Name: "bar",
					File: &chart.File{
						Data: []byte(fmt.Sprintf("---\n%[1]s\n---\n%[1]s", crdYaml)),
					},
				},
			},
			expected: fmt.Sprintf("---\n# Source: %[1]s\n%[2]s---\n# Source: %[1]s\n%[2]s", "bar", crdYaml),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CrdToManifest(tt.crds)
			require.Equal(t, tt.expected, result)
		})
	}
}
