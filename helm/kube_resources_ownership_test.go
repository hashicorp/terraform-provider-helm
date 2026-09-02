// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TestSetDryRunOwnershipMetadata pins the one thing Helm's own
// setMetadataVisitor (helm.sh/helm/v3/pkg/action/validate.go) does that the
// experiments.manifest dry run must replicate to avoid "Provider produced
// inconsistent result after apply" on the resources[...] map: stamping
// app.kubernetes.io/managed-by=Helm.
//
// setMetadataVisitor also stamps two meta.helm.sh/* annotations, which this
// function deliberately does not replicate - normalizeK8sObject's
// stripHelmMetaAnnotations strips every meta.helm.sh/* annotation from both
// the dry-run and live objects before either is stored, so setting them here
// would only be redacted away before the two are ever compared.
func TestSetDryRunOwnershipMetadata(t *testing.T) {
	t.Run("object with no labels at all", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "ConfigMap",
			"metadata": map[string]any{"name": "app"},
		}}

		require.NoError(t, setDryRunOwnershipMetadata(obj))

		assert.Equal(t, "Helm", obj.GetLabels()["app.kubernetes.io/managed-by"])
	})

	t.Run("existing labels are preserved, not replaced", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "ConfigMap",
			"metadata": map[string]any{
				"name":   "app",
				"labels": map[string]any{"app.kubernetes.io/name": "litellm"},
			},
		}}

		require.NoError(t, setDryRunOwnershipMetadata(obj))

		assert.Equal(t, "litellm", obj.GetLabels()["app.kubernetes.io/name"], "unrelated label must survive")
		assert.Equal(t, "Helm", obj.GetLabels()["app.kubernetes.io/managed-by"])
	})

	t.Run("a conflicting prior value is force-overwritten, matching force=true", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "ConfigMap",
			"metadata": map[string]any{
				"name": "app",
				"labels": map[string]any{
					"app.kubernetes.io/managed-by": "something-else",
				},
			},
		}}

		require.NoError(t, setDryRunOwnershipMetadata(obj))

		assert.Equal(t, "Helm", obj.GetLabels()["app.kubernetes.io/managed-by"])
	})

	t.Run("Secret kind is unaffected by the function itself", func(t *testing.T) {
		// stripSecretManagedByLabel later strips this label specifically for
		// Secrets before comparison, but setDryRunOwnershipMetadata has no
		// kind-specific behaviour of its own - it always sets the label, and
		// normalization decides afterward whether that matters for this kind.
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "Secret",
			"metadata": map[string]any{"name": "app-secret"},
		}}

		require.NoError(t, setDryRunOwnershipMetadata(obj))

		assert.Equal(t, "Helm", obj.GetLabels()["app.kubernetes.io/managed-by"])
	})
}
