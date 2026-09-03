// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertYAMLManifestToJSON_Malformed exercises the inputs a chart with a
// template bug, or Helm's own document splitting, can actually produce -
// separately from the well-formed cases the rest of the suite covers.
func TestConvertYAMLManifestToJSON_Malformed(t *testing.T) {
	for name, tc := range map[string]struct {
		manifest string
		wantErr  bool
	}{
		"completely empty":       {manifest: "", wantErr: false},
		"only whitespace":        {manifest: "   \n\t\n  ", wantErr: false},
		"only a comment":         {manifest: "# just a comment\n", wantErr: false},
		"only document markers":  {manifest: "---\n---\n---\n", wantErr: false},
		"invalid YAML syntax":    {manifest: "apiVersion: v1\nkind: [unterminated\n", wantErr: true},
		"tab indentation":        {manifest: "apiVersion: v1\nkind: Pod\nmetadata:\n\tname: p\n", wantErr: true},
		"resource missing kind":  {manifest: "apiVersion: v1\nmetadata:\n  name: p\n", wantErr: false},
		"resource missing name":  {manifest: "apiVersion: v1\nkind: Pod\nmetadata: {}\n", wantErr: false},
		"a bare scalar document": {manifest: "just some text\n", wantErr: true},
		"a bare list document":   {manifest: "- one\n- two\n", wantErr: true},
	} {
		for _, keyed := range []bool{false, true} {
			t.Run(name+fmtKeyed(keyed), func(t *testing.T) {
				_, err := convertYAMLManifestToJSON(tc.manifest, keyed)
				if tc.wantErr {
					assert.Error(t, err, "expected an error for %q", tc.manifest)
				} else {
					assert.NoError(t, err, "expected no error for %q", tc.manifest)
				}
			})
		}
	}
}

func fmtKeyed(k bool) string {
	if k {
		return "/keyed"
	}
	return "/unkeyed"
}

// A resource with a group in its apiVersion (most real resources) versus one
// without (core/v1) must not collide, and neither should keying disturb the
// resource key itself.
func TestConvertYAMLManifestToJSON_ResourceKeyUnaffectedByKeying(t *testing.T) {
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: ns
spec:
  template:
    spec:
      containers:
      - name: c
`
	unkeyed, err := convertYAMLManifestToJSON(manifest, false)
	require.NoError(t, err)
	keyed, err := convertYAMLManifestToJSON(manifest, true)
	require.NoError(t, err)

	assert.Contains(t, unkeyed, `"ns/deployment.apps/apps/v1/app"`)
	assert.Contains(t, keyed, `"ns/deployment.apps/apps/v1/app"`)
}

// redactSensitiveValues is a post-processing string replace over the final
// JSON, so it must still find and mask a set_sensitive value regardless of
// whether the surrounding list was keyed or left as an array.
func TestKeyedLists_SensitiveValuesStillRedacted(t *testing.T) {
	manifest := `apiVersion: v1
kind: Pod
metadata:
  name: p
spec:
  containers:
  - name: app
    env:
    - name: API_KEY
      value: "correct-horse-battery-staple"
`
	for _, keyed := range []bool{false, true} {
		jsonManifest, err := convertYAMLManifestToJSON(manifest, keyed)
		require.NoError(t, err)

		redacted := redactSensitiveValues(jsonManifest, []string{"correct-horse-battery-staple"})
		assert.NotContains(t, redacted, "correct-horse-battery-staple")
		assert.Contains(t, redacted, "(sensitive value")
	}
}
