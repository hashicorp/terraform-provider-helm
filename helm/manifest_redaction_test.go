// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"encoding/json"
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

// TestJSONEscapedForm pins jsonEscapedForm's contract directly: it must
// return exactly what json.Marshal would put inside a JSON string field for
// this value, with the surrounding quotes stripped - the plain-value case is
// a byte-identical no-op, everything else is genuinely transformed.
func TestJSONEscapedForm(t *testing.T) {
	for name, tc := range map[string]struct {
		value string
		want  string
	}{
		"plain alphanumeric":               {value: "correct-horse-battery-staple", want: "correct-horse-battery-staple"},
		"empty string":                     {value: "", want: ""},
		"contains a newline":               {value: "line one\nline two", want: `line one\nline two`},
		"contains a quote":                 {value: `say "hello"`, want: `say \"hello\"`},
		"contains a backslash":             {value: `C:\path\to\thing`, want: `C:\\path\\to\\thing`},
		"contains tab and carriage return": {value: "a\tb\rc", want: `a\tb\rc`},
		"contains a null byte":             {value: "a\x00b", want: `a\u0000b`},
		"contains angle brackets and amp":  {value: "<script>&amp;</script>", want: `\u003cscript\u003e\u0026amp;\u003c/script\u003e`},
		"unicode":                          {value: "héllo wörld 日本語 🚀", want: "héllo wörld 日本語 🚀"},
		"multi-line secret with mixed special characters": {
			value: "line-one-canary\nline-two-quo\"te\nline-three-back\\slash",
			want:  `line-one-canary\nline-two-quo\"te\nline-three-back\\slash`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := jsonEscapedForm(tc.value)
			require.True(t, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestRedactSensitiveValues_MultiLineAndSpecialCharacters is a regression
// test for a real gap found during pre-submission review: redactSensitiveValues
// used to search the JSON manifest text for the RAW configured secret value.
// Since that text is always JSON, any value containing a character JSON
// escapes - a quote, a backslash, or a control character such as a newline -
// never appears in the text as its raw bytes, so the raw-value search found
// nothing and the secret (in its readable, merely re-escaped form) shipped
// into state/plan output unredacted. A multi-line secret - a PEM private key
// or certificate being the obvious real-world case - was affected on every
// single apply, silently, regardless of the map-orientation bug fixed
// alongside this one.
func TestRedactSensitiveValues_MultiLineAndSpecialCharacters(t *testing.T) {
	for name, secret := range map[string]string{
		"multi-line with embedded newlines":                                     "line-one-canary\nline-two-indented\nline-three-end",
		"contains a double quote":                                               `value with a "quoted phrase" inside it`,
		"contains a backslash":                                                  `C:\Users\canary\secret.txt`,
		"contains a backslash immediately before a quote":                       `path\"escaped`,
		"contains tab and carriage return":                                      "a\tcanary\rb",
		"contains a null byte":                                                  "before\x00after-canary",
		"contains angle brackets (HTML-unsafe under Go's default json.Marshal)": "<canary>&value</canary>",
	} {
		t.Run(name, func(t *testing.T) {
			manifest := podEnv(secret)
			for _, keyed := range []bool{false, true} {
				jsonManifest, err := convertYAMLManifestToJSON(manifest, keyed)
				require.NoError(t, err)

				redacted := redactSensitiveValues(jsonManifest, []string{secret})

				assert.NotContains(t, redacted, "canary", "keyed=%v: secret content must not survive redaction: %s", keyed, redacted)
				assert.Contains(t, redacted, "(sensitive value", "keyed=%v: hash marker must be present", keyed)
			}
		})
	}
}

// podEnv renders a manifest whose single env value, once through YAML's
// block-scalar handling, round-trips secret byte-for-byte (YAML block
// literals preserve embedded quotes/backslashes/control characters exactly,
// which is what this test needs to actually exercise the escaping gap).
func podEnv(value string) string {
	b, _ := json.Marshal(value) // reuse Go's own JSON string encoding as a safe way to embed an arbitrary string inside YAML too
	return "apiVersion: v1\nkind: Pod\nmetadata:\n  name: p\nspec:\n  containers:\n  - name: app\n    env:\n    - name: TLS_KEY\n      value: " + string(b) + "\n"
}
