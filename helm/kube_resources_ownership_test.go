// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TestSetDryRunOwnershipMetadata pins the exact behaviour of Helm's own
// setMetadataVisitor (helm.sh/helm/v3/pkg/action/validate.go), which every
// real Install, Upgrade and Rollback runs with force=true and which the
// experiments.manifest dry run must replicate to avoid "Provider produced
// inconsistent result after apply" on the resources[...] map.
func TestSetDryRunOwnershipMetadata(t *testing.T) {
	t.Run("object with no metadata at all", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "ConfigMap",
			"metadata": map[string]any{"name": "app"},
		}}

		require.NoError(t, setDryRunOwnershipMetadata(obj, "litellm", "litellm"))

		assert.Equal(t, "Helm", obj.GetLabels()["app.kubernetes.io/managed-by"])
		assert.Equal(t, "litellm", obj.GetAnnotations()["meta.helm.sh/release-name"])
		assert.Equal(t, "litellm", obj.GetAnnotations()["meta.helm.sh/release-namespace"])
	})

	t.Run("existing labels and annotations are preserved, not replaced", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "ConfigMap",
			"metadata": map[string]any{
				"name":        "app",
				"labels":      map[string]any{"app.kubernetes.io/name": "litellm"},
				"annotations": map[string]any{"checksum/config": "abc123"},
			},
		}}

		require.NoError(t, setDryRunOwnershipMetadata(obj, "litellm", "litellm"))

		assert.Equal(t, "litellm", obj.GetLabels()["app.kubernetes.io/name"], "unrelated label must survive")
		assert.Equal(t, "Helm", obj.GetLabels()["app.kubernetes.io/managed-by"])
		assert.Equal(t, "abc123", obj.GetAnnotations()["checksum/config"], "unrelated annotation must survive")
	})

	t.Run("a conflicting prior value is force-overwritten, matching force=true", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "ConfigMap",
			"metadata": map[string]any{
				"name": "app",
				"labels": map[string]any{
					"app.kubernetes.io/managed-by": "something-else",
				},
				"annotations": map[string]any{
					"meta.helm.sh/release-name": "wrong-release",
				},
			},
		}}

		require.NoError(t, setDryRunOwnershipMetadata(obj, "litellm", "litellm-ns"))

		assert.Equal(t, "Helm", obj.GetLabels()["app.kubernetes.io/managed-by"])
		assert.Equal(t, "litellm", obj.GetAnnotations()["meta.helm.sh/release-name"])
		assert.Equal(t, "litellm-ns", obj.GetAnnotations()["meta.helm.sh/release-namespace"])
	})

	t.Run("namespace differs from release name", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "Secret",
			"metadata": map[string]any{"name": "app-secret"},
		}}

		require.NoError(t, setDryRunOwnershipMetadata(obj, "my-release", "some-other-namespace"))

		assert.Equal(t, "my-release", obj.GetAnnotations()["meta.helm.sh/release-name"])
		assert.Equal(t, "some-other-namespace", obj.GetAnnotations()["meta.helm.sh/release-namespace"])
	})
}
