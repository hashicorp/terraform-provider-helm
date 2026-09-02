// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func podWithOneEnvEntry(entryYAML string) string {
	return "apiVersion: v1\nkind: Pod\nmetadata:\n  name: p\nspec:\n  containers:\n  - name: app\n    env:\n" + entryYAML
}

// TestKeyedLists_ValueKinds exercises every JSON value kind Kubernetes puts
// next to a list-map's key field, on both the identifying field itself and on
// sibling fields, since elementsByKey type-asserts on the identity but leaves
// everything else untouched.
func TestKeyedLists_ValueKinds(t *testing.T) {
	for name, tc := range map[string]struct {
		manifest string
		keyed    bool // whether env should end up keyed
	}{
		"string value": {
			manifest: podWithOneEnvEntry("    - name: A\n      value: \"plain\"\n"),
			keyed:    true,
		},
		"empty string value": {
			manifest: podWithOneEnvEntry("    - name: A\n      value: \"\"\n"),
			keyed:    true,
		},
		"null value (valueFrom instead)": {
			manifest: podWithOneEnvEntry("    - name: A\n      valueFrom:\n        fieldRef:\n          fieldPath: metadata.name\n"),
			keyed:    true,
		},
		"boolean-looking string value": {
			manifest: podWithOneEnvEntry("    - name: A\n      value: \"true\"\n"),
			keyed:    true,
		},
		"numeric-looking string value": {
			manifest: podWithOneEnvEntry("    - name: A\n      value: \"12345\"\n"),
			keyed:    true,
		},
		"unicode in value": {
			manifest: podWithOneEnvEntry("    - name: A\n      value: \"h\u00e9llo w\u00f6rld \u65e5\u672c\u8a9e \U0001F680\"\n"),
			keyed:    true,
		},
		"value with embedded newline": {
			manifest: podWithOneEnvEntry("    - name: A\n      value: |\n        line one\n        line two\n"),
			keyed:    true,
		},
		"value containing a JSON-looking string": {
			manifest: podWithOneEnvEntry("    - name: A\n      value: '{\"nested\": \"json\", \"n\": 1}'\n"),
			keyed:    true,
		},
		"name value that looks numeric": {
			// Quoted, "123" is a string identity like any other and should
			// still be keyed - unquoted it would parse as a YAML int instead.
			manifest: podWithOneEnvEntry("    - name: \"123\"\n      value: v\n"),
			keyed:    true,
		},
		"unicode identity": {
			manifest: podWithOneEnvEntry("    - name: \"MOD\u00c8LE\"\n      value: v\n"),
			keyed:    true,
		},
		"whitespace-only identity": {
			manifest: podWithOneEnvEntry("    - name: \" \"\n      value: v\n"),
			keyed:    true, // a single space is a non-empty, unique string
		},
	} {
		t.Run(name, func(t *testing.T) {
			resource := onlyResource(t, tc.manifest, true)
			env := at(t, resource, "spec.containers.app.env")
			if tc.keyed {
				assert.IsType(t, map[string]any{}, env)
			} else {
				assert.IsType(t, []any{}, env)
			}
		})
	}
}

// Numeric identity: containerPort has no "name" by default and int keys are
// not what elementsByKey looks for, so a list keyed on a field holding a
// number (not a string) must stay an array.
func TestKeyedLists_NonStringIdentityStaysArray(t *testing.T) {
	manifest := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: p\nspec:\n  containers:\n  - name: app\n    ports:\n    - name: 8080\n      containerPort: 8080\n"
	// "name: 8080" without quotes parses as a YAML integer.
	resource := onlyResource(t, manifest, true)
	ports := at(t, resource, "spec.containers.app.ports")
	assert.IsType(t, []any{}, ports, "a numeric name value is not a string identity")
}

// Boolean and null identities are equally not strings.
func TestKeyedLists_BooleanAndNullIdentityStayArray(t *testing.T) {
	for name, manifest := range map[string]string{
		"boolean name": "apiVersion: v1\nkind: Pod\nmetadata: {name: p}\nspec:\n  containers:\n  - name: true\n",
		"null name":    "apiVersion: v1\nkind: Pod\nmetadata: {name: p}\nspec:\n  containers:\n  - name: null\n    image: app\n",
	} {
		t.Run(name, func(t *testing.T) {
			resource := onlyResource(t, manifest, true)
			containers := at(t, resource, "spec.containers")
			assert.IsType(t, []any{}, containers)
		})
	}
}

// A deeply nested list-map inside a list-map (containers -> env, both keyed)
// must key independently at each level without cross-contamination.
func TestKeyedLists_NestedListMapsAreIndependent(t *testing.T) {
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
      - name: a
        env:
        - name: X
          value: "a-x"
      - name: b
        env:
        - name: X
          value: "b-x"
`
	resource := onlyResource(t, manifest, true)
	containers := at(t, resource, "spec.template.spec.containers").(map[string]any)
	require.Contains(t, containers, "a")
	require.Contains(t, containers, "b")

	envA := containers["a"].(map[string]any)["env"].(map[string]any)
	envB := containers["b"].(map[string]any)["env"].(map[string]any)
	assert.Equal(t, "a-x", envA["X"].(map[string]any)["value"])
	assert.Equal(t, "b-x", envB["X"].(map[string]any)["value"])
}

// Very large lists must not be quadratic or otherwise pathological, and every
// element has to survive.
func TestKeyedLists_LargeList(t *testing.T) {
	manifest := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: p\nspec:\n  containers:\n  - name: app\n    env:\n"
	const n = 500
	for i := 0; i < n; i++ {
		manifest += "    - name: VAR_" + itoa(i) + "\n      value: \"" + itoa(i) + "\"\n"
	}

	resource := onlyResource(t, manifest, true)
	env := at(t, resource, "spec.containers.app.env").(map[string]any)
	assert.Len(t, env, n)
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

// A resource where the SAME list field name appears at different nesting
// depths with different semantics: top-level "ports" (a Service, unnamed) vs
// container "ports" (named). The field-name allowlist has no notion of path,
// so both are decided independently at their own level by whether their
// elements actually carry a unique name - this is the case that would catch a
// naive "match by field name only" implementation silently keying the wrong
// one.
func TestKeyedLists_SameFieldNameDifferentSemanticsAtDifferentDepths(t *testing.T) {
	manifest := `apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: Pod
  metadata: {name: p}
  spec:
    containers:
    - name: app
      ports:
      - name: http
        containerPort: 8080
`
	resource := onlyResource(t, manifest, true)
	ports := at(t, resource, "items.0.spec.containers.app.ports")
	require.IsType(t, map[string]any{}, ports)
	assert.Contains(t, ports, "http")
}
