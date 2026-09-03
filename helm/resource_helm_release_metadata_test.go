// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

func emptyStringList() types.List { return types.ListValueMust(types.StringType, []attr.Value{}) }

func baseModel() HelmReleaseModel {
	return HelmReleaseModel{
		Name:          types.StringValue("litellm"),
		Namespace:     types.StringValue("litellm"),
		Chart:         types.StringValue("litellm-helm"),
		Repository:    types.StringValue("oci://ghcr.io/berriai"),
		Version:       types.StringValue("1.98.0"),
		Values:        emptyStringList(),
		Set:           types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		SetList:       types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		SetSensitive:  types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		SetWORevision: types.Int64Value(0),
		Metadata:      types.ObjectNull(metadataAttrTypes()),
	}
}

func deployedMetadata() types.Object {
	return types.ObjectValueMust(metadataAttrTypes(), map[string]attr.Value{
		"name":           types.StringValue("litellm"),
		"namespace":      types.StringValue("litellm"),
		"chart":          types.StringValue("litellm-helm"),
		"version":        types.StringValue("1.83.3-stable"),
		"app_version":    types.StringValue("v1.83.3-stable"),
		"values":         types.StringValue(`{"image":{"tag":"v1.93.0"}}`),
		"revision":       types.Int64Value(68),
		"first_deployed": types.Int64Value(1757673881),
		"last_deployed":  types.Int64Value(1788380977),
		"notes":          types.StringValue("1. Get the application URL..."),
	})
}

func dryRun(t *testing.T) *release.Release {
	t.Helper()
	return &release.Release{
		Name:      "litellm",
		Namespace: "litellm",
		Chart: &chart.Chart{Metadata: &chart.Metadata{
			Name: "litellm-helm", Version: "1.98.0", AppVersion: "1.98.0",
		}},
		Config: map[string]interface{}{"image": map[string]interface{}{"tag": "v1.98.0"}},
	}
}

// TestRecomputeMetadata covers the TODO left on recomputeMetadata: it must fire
// for the fields metadata is derived from, and stay quiet otherwise.
func TestRecomputeMetadata(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate   func(*HelmReleaseModel)
		nilState bool
		want     bool
	}{
		"no state at all":    {nilState: true, want: true},
		"nothing changed":    {mutate: func(*HelmReleaseModel) {}, want: false},
		"chart changed":      {mutate: func(m *HelmReleaseModel) { m.Chart = types.StringValue("other") }, want: true},
		"repository changed": {mutate: func(m *HelmReleaseModel) { m.Repository = types.StringValue("other") }, want: true},
		"version changed":    {mutate: func(m *HelmReleaseModel) { m.Version = types.StringValue("1.99.0") }, want: true},
		"values changed": {mutate: func(m *HelmReleaseModel) {
			m.Values = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a: b")})
		}, want: true},
		"name changed only":      {mutate: func(m *HelmReleaseModel) { m.Name = types.StringValue("renamed") }, want: false},
		"namespace changed only": {mutate: func(m *HelmReleaseModel) { m.Namespace = types.StringValue("other") }, want: false},
		"set_wo_revision bumped": {mutate: func(m *HelmReleaseModel) { m.SetWORevision = types.Int64Value(1) }, want: true},
	} {
		t.Run(name, func(t *testing.T) {
			state := baseModel()
			plan := baseModel()
			if tc.mutate != nil {
				tc.mutate(&plan)
			}
			if tc.nilState {
				assert.True(t, recomputeMetadata(plan, nil))
				return
			}
			assert.Equal(t, tc.want, recomputeMetadata(plan, &state))
		})
	}
}

func TestPlannedMetadata_WithoutDryRun(t *testing.T) {
	plan := baseModel()
	state := baseModel()
	state.Metadata = deployedMetadata()

	planned := plannedMetadata(&plan, &state, nil)
	require.False(t, planned.IsUnknown(), "the object itself must not be unknown")
	attrs := planned.Attributes()

	// Known from configuration alone.
	assert.Equal(t, types.StringValue("litellm"), attrs["name"])
	assert.Equal(t, types.StringValue("litellm"), attrs["namespace"])

	// first_deployed does not move across an upgrade, so it carries over.
	assert.Equal(t, types.Int64Value(1757673881), attrs["first_deployed"])

	// Everything that needs the dry run stays unknown.
	for _, field := range []string{"chart", "version", "app_version", "values", "revision", "last_deployed", "notes"} {
		assert.True(t, attrs[field].IsUnknown(), "%s should still be unknown", field)
	}
}

func TestPlannedMetadata_WithDryRun(t *testing.T) {
	plan := baseModel()
	state := baseModel()
	state.Metadata = deployedMetadata()

	planned := plannedMetadata(&plan, &state, dryRun(t))
	attrs := planned.Attributes()

	assert.Equal(t, types.StringValue("litellm"), attrs["name"])
	assert.Equal(t, types.StringValue("litellm"), attrs["namespace"])
	assert.Equal(t, types.StringValue("litellm-helm"), attrs["chart"])
	assert.Equal(t, types.StringValue("1.98.0"), attrs["version"])
	assert.Equal(t, types.StringValue("1.98.0"), attrs["app_version"])
	assert.Equal(t, types.StringValue(`{"image":{"tag":"v1.98.0"}}`), attrs["values"],
		"values is the blob that used to render as a wholesale deletion")
	assert.Equal(t, types.Int64Value(1757673881), attrs["first_deployed"])

	// Only these genuinely have to wait for the apply.
	for _, field := range []string{"revision", "last_deployed", "notes"} {
		assert.True(t, attrs[field].IsUnknown(), "%s must remain unknown", field)
	}
}

// Write-only values are ephemeral and the read path stores "{}" once one is in
// play, so planning a concrete value would guarantee an inconsistent-result
// error after apply.
func TestPlannedMetadata_WriteOnlyValuesStayUnknown(t *testing.T) {
	plan := baseModel()
	plan.SetWORevision = types.Int64Value(1)

	planned := plannedMetadata(&plan, nil, dryRun(t))
	assert.True(t, planned.Attributes()["values"].IsUnknown())

	plan.SetWORevision = types.Int64Unknown()
	planned = plannedMetadata(&plan, nil, dryRun(t))
	assert.True(t, planned.Attributes()["values"].IsUnknown())
}

func TestPlannedMetadata_NilStateAndUnknownConfig(t *testing.T) {
	plan := baseModel()
	plan.Name = types.StringUnknown()
	plan.Namespace = types.StringNull()

	planned := plannedMetadata(&plan, nil, nil)
	attrs := planned.Attributes()

	for _, field := range []string{"name", "namespace", "first_deployed", "values"} {
		assert.True(t, attrs[field].IsUnknown(), "%s should be unknown", field)
	}
}

func TestReleaseValuesJSON(t *testing.T) {
	model := baseModel()

	empty, err := releaseValuesJSON(nil, &model)
	require.NoError(t, err)
	assert.Equal(t, "{}", empty)

	noConfig, err := releaseValuesJSON(&release.Release{}, &model)
	require.NoError(t, err)
	assert.Equal(t, "{}", noConfig)

	withConfig, err := releaseValuesJSON(dryRun(t), &model)
	require.NoError(t, err)
	assert.JSONEq(t, `{"image":{"tag":"v1.98.0"}}`, withConfig)
}

func TestKnownMetadataAttr(t *testing.T) {
	_, ok := knownMetadataAttr(types.ObjectNull(metadataAttrTypes()), "first_deployed")
	assert.False(t, ok)

	_, ok = knownMetadataAttr(types.ObjectUnknown(metadataAttrTypes()), "first_deployed")
	assert.False(t, ok)

	value, ok := knownMetadataAttr(deployedMetadata(), "first_deployed")
	assert.True(t, ok)
	assert.Equal(t, types.Int64Value(1757673881), value)

	_, ok = knownMetadataAttr(deployedMetadata(), "nope")
	assert.False(t, ok)
}

// Renaming a release (or moving its namespace) forces replacement: name and
// namespace both carry RequiresReplace(), so ModifyPlan's "state" is the
// release about to be destroyed, not the one this plan is building. Its
// first_deployed timestamp belongs to that old release and must not leak into
// the plan for the new one - the new release has not been installed yet.
func TestPlannedMetadata_ReplacementDoesNotInheritFirstDeployed(t *testing.T) {
	state := baseModel()
	state.Metadata = deployedMetadata() // first_deployed = 1757673881

	t.Run("name changes", func(t *testing.T) {
		plan := baseModel()
		plan.Name = types.StringValue("renamed")

		planned := plannedMetadata(&plan, &state, nil)
		assert.True(t, planned.Attributes()["first_deployed"].IsUnknown(),
			"first_deployed must not carry over when the release is being replaced")
	})

	t.Run("namespace changes", func(t *testing.T) {
		plan := baseModel()
		plan.Namespace = types.StringValue("other-namespace")

		planned := plannedMetadata(&plan, &state, nil)
		assert.True(t, planned.Attributes()["first_deployed"].IsUnknown(),
			"first_deployed must not carry over when the release is being replaced")
	})

	t.Run("same identity still inherits it", func(t *testing.T) {
		plan := baseModel()

		planned := plannedMetadata(&plan, &state, nil)
		assert.Equal(t, types.Int64Value(1757673881), planned.Attributes()["first_deployed"],
			"an upgrade of the same release should still carry it over")
	})
}
