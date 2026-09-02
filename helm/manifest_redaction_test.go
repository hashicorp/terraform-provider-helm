// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var setResourceObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"name":  types.StringType,
	"type":  types.StringType,
	"value": types.StringType,
}}

func setSensitiveList(t *testing.T, entries ...setResourceModel) types.List {
	t.Helper()
	l, diags := types.ListValueFrom(context.Background(), setResourceObjectType, entries)
	require.False(t, diags.HasError(), "%v", diags)
	return l
}

func sensitiveEntry(name, value string) setResourceModel {
	return setResourceModel{Name: types.StringValue(name), Type: types.StringValue("string"), Value: types.StringValue(value)}
}

// TestSensitiveSetValues pins the fix for a real bug: redactSensitiveValues
// used to be handed a map keyed by attribute NAME with the secret value
// discarded or replaced by a placeholder, so it searched a stored manifest
// for literal strings like "dbPassword" and never touched the actual secret
// text - set_sensitive values were never redacted from state or plan output.
// sensitiveSetValues now returns the real values directly, so getting the
// orientation backwards again is not possible.
func TestSensitiveSetValues(t *testing.T) {
	t.Run("returns the real values, not the attribute names", func(t *testing.T) {
		list := setSensitiveList(t,
			sensitiveEntry("dbPassword", "correct-horse-battery-staple"),
			sensitiveEntry("apiKey", "sk-canary-12345"),
		)
		values := sensitiveSetValues(context.Background(), list)

		assert.ElementsMatch(t, []string{"correct-horse-battery-staple", "sk-canary-12345"}, values)
		assert.NotContains(t, values, "dbPassword", "must never return the attribute name in place of its value")
		assert.NotContains(t, values, "apiKey")
	})

	t.Run("null list", func(t *testing.T) {
		assert.Empty(t, sensitiveSetValues(context.Background(), types.ListNull(setResourceObjectType)))
	})

	t.Run("unknown list", func(t *testing.T) {
		assert.Empty(t, sensitiveSetValues(context.Background(), types.ListUnknown(setResourceObjectType)))
	})

	t.Run("empty list", func(t *testing.T) {
		assert.Empty(t, sensitiveSetValues(context.Background(), setSensitiveList(t)))
	})

	t.Run("unknown value is skipped, not converted to an empty string", func(t *testing.T) {
		list := setSensitiveList(t, setResourceModel{
			Name: types.StringValue("pending"), Type: types.StringValue("string"), Value: types.StringUnknown(),
		})
		assert.Empty(t, sensitiveSetValues(context.Background(), list))
	})

	t.Run("null value is skipped", func(t *testing.T) {
		list := setSensitiveList(t, setResourceModel{
			Name: types.StringValue("absent"), Type: types.StringValue("string"), Value: types.StringNull(),
		})
		assert.Empty(t, sensitiveSetValues(context.Background(), list))
	})

	t.Run("empty string value is skipped", func(t *testing.T) {
		// Guards the catastrophic case: strings.ReplaceAll(text, "", marker)
		// matches every position and would corrupt the whole manifest.
		list := setSensitiveList(t, sensitiveEntry("blank", ""))
		assert.Empty(t, sensitiveSetValues(context.Background(), list))
	})

	t.Run("mix of known, unknown, null and empty entries", func(t *testing.T) {
		list := setSensitiveList(t,
			sensitiveEntry("real", "keep-me"),
			setResourceModel{Name: types.StringValue("u"), Type: types.StringValue("string"), Value: types.StringUnknown()},
			setResourceModel{Name: types.StringValue("n"), Type: types.StringValue("string"), Value: types.StringNull()},
			sensitiveEntry("empty", ""),
			sensitiveEntry("real2", "keep-me-too"),
		)
		assert.ElementsMatch(t, []string{"keep-me", "keep-me-too"}, sensitiveSetValues(context.Background(), list))
	})
}

// TestRedactSensitiveValues_EndToEnd exercises the exact call shape the three
// real call sites use: sensitiveSetValues feeding directly into
// redactSensitiveValues, against a manifest containing several resources -
// the case the map-orientation bug affected in production.
func TestRedactSensitiveValues_EndToEnd(t *testing.T) {
	manifest := `apiVersion: v1
kind: Secret
metadata:
  name: litellm-masterkey
type: Opaque
data:
  masterkey: c3VwZXItc2VjcmV0
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: litellm
spec:
  template:
    spec:
      containers:
      - name: litellm
        env:
        - name: DATABASE_PASSWORD
          value: "correct-horse-battery-staple"
        - name: API_KEY
          value: "sk-canary-12345"
`
	list := setSensitiveList(t,
		sensitiveEntry("env.DATABASE_PASSWORD", "correct-horse-battery-staple"),
		sensitiveEntry("env.API_KEY", "sk-canary-12345"),
	)

	for _, keyed := range []bool{false, true} {
		jsonManifest, err := convertYAMLManifestToJSON(manifest, keyed)
		require.NoError(t, err)

		redacted := redactSensitiveValues(jsonManifest, sensitiveSetValues(context.Background(), list))

		assert.NotContains(t, redacted, "correct-horse-battery-staple", "keyed=%v: real secret value must not survive redaction", keyed)
		assert.NotContains(t, redacted, "sk-canary-12345", "keyed=%v: real secret value must not survive redaction", keyed)
		assert.NotContains(t, redacted, "env.DATABASE_PASSWORD", "keyed=%v: attribute name must never appear as a redaction target", keyed)
		assert.Contains(t, redacted, "(sensitive value", "keyed=%v: hash marker must be present", keyed)
	}
}
