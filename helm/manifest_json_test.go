// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertYAMLManifestToJSON(t *testing.T) {
	yamlManifest := readTestFile(t, "testdata/manifest_json/rendered_manifest.yaml")
	expectedJSON := readTestFile(t, "testdata/manifest_json/rendered_manifest.json")

	json, err := convertYAMLManifestToJSON(yamlManifest, false)

	assert.NoError(t, err)
	assert.JSONEq(t, expectedJSON, json)
}

func readTestFile(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err.Error())
	}
	return string(b)
}

// deployment renders a Deployment whose container carries the given env vars,
// in the order given. Charts commonly prepend a newly introduced variable
// rather than slotting it in alphabetically, which is what makes the positional
// diff so noisy.
func deployment(env ...string) string {
	var entries string
	for _, name := range env {
		entries += fmt.Sprintf("        - name: %s\n          value: %q\n", name, strings.ToLower(name))
	}
	return `apiVersion: apps/v1
kind: Deployment
metadata:
  name: litellm
  namespace: litellm
spec:
  template:
    spec:
      containers:
      - name: litellm
        image: ghcr.io/berriai/litellm-database:v1.98.0
        args:
        - --port
        - "4000"
        env:
` + entries
}

func containerEnv(t *testing.T, manifestJSON string) any {
	t.Helper()
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(manifestJSON), &decoded))

	for _, resource := range decoded {
		object := resource.(map[string]any)
		spec := object["spec"].(map[string]any)
		template := spec["template"].(map[string]any)
		podSpec := template["spec"].(map[string]any)

		switch containers := podSpec["containers"].(type) {
		case map[string]any: // keyed
			return containers["litellm"].(map[string]any)["env"]
		case []any: // positional
			return containers[0].(map[string]any)["env"]
		}
	}

	t.Fatal("no deployment found in manifest")
	return nil
}

// TestKeyedLists_InsertionDoesNotChurn is the property the feature exists for.
// Inserting one environment variable at the front of the list must read as a
// single addition, not as a rewrite of every entry after it.
func TestKeyedLists_InsertionDoesNotChurn(t *testing.T) {
	existing := []string{"AWS_DEFAULT_REGION", "AWS_REGION_NAME", "BEDROCK_REGION_NAME", "EMAIL_LOGO_URL", "JSON_LOGS"}
	withNew := append([]string{"LITELLM_LOG"}, existing...)

	t.Run("positional lists churn", func(t *testing.T) {
		before, err := convertYAMLManifestToJSON(deployment(existing...), false)
		require.NoError(t, err)
		after, err := convertYAMLManifestToJSON(deployment(withNew...), false)
		require.NoError(t, err)

		oldEnv := containerEnv(t, before).([]any)
		newEnv := containerEnv(t, after).([]any)

		// Terraform pairs array elements by index, so count how many indices
		// hold a different entry than they did before.
		moved := 0
		for i := range newEnv {
			if i >= len(oldEnv) || !reflect.DeepEqual(oldEnv[i], newEnv[i]) {
				moved++
			}
		}

		assert.Equal(t, len(newEnv), moved,
			"every index should differ, which is exactly the noise being fixed")
	})

	t.Run("keyed lists do not churn", func(t *testing.T) {
		before, err := convertYAMLManifestToJSON(deployment(existing...), true)
		require.NoError(t, err)
		after, err := convertYAMLManifestToJSON(deployment(withNew...), true)
		require.NoError(t, err)

		oldEnv := containerEnv(t, before).(map[string]any)
		newEnv := containerEnv(t, after).(map[string]any)

		added := []string{}
		for name, entry := range newEnv {
			previous, existed := oldEnv[name]
			if !existed {
				added = append(added, name)
				continue
			}
			assert.True(t, reflect.DeepEqual(previous, entry),
				"%s should be untouched by the insertion", name)
		}

		assert.Equal(t, []string{"LITELLM_LOG"}, added, "only the new variable should appear")
		assert.Len(t, oldEnv, len(existing))
		assert.Len(t, newEnv, len(withNew))
	})
}

func TestKeyedLists_OrderedListsAreLeftAlone(t *testing.T) {
	manifestJSON, err := convertYAMLManifestToJSON(deployment("A"), true)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(manifestJSON), &decoded))

	for _, resource := range decoded {
		podSpec := resource.(map[string]any)["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)
		containers := podSpec["containers"].(map[string]any)
		container := containers["litellm"].(map[string]any)

		// containers and env are list-maps, args is genuinely ordered.
		assert.IsType(t, map[string]any{}, containers, "containers should be keyed")
		assert.IsType(t, map[string]any{}, container["env"], "env should be keyed")
		assert.IsType(t, []any{}, container["args"], "args must stay an ordered array")
		assert.Equal(t, []any{"--port", "4000"}, container["args"])
	}
}

func TestElementsByKey(t *testing.T) {
	for name, tc := range map[string]struct {
		elements []any
		key      string
		keyed    bool
	}{
		"named entries": {
			elements: []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}},
			key:      "name", keyed: true,
		},
		"empty list": {elements: []any{}, key: "name", keyed: false},
		"duplicate names": {
			elements: []any{map[string]any{"name": "a"}, map[string]any{"name": "a"}},
			key:      "name", keyed: false,
		},
		"missing key": {
			elements: []any{map[string]any{"name": "a"}, map[string]any{"containerPort": 80.0}},
			key:      "name", keyed: false,
		},
		"empty name": {
			elements: []any{map[string]any{"name": ""}},
			key:      "name", keyed: false,
		},
		"non string key": {
			elements: []any{map[string]any{"name": 3.0}},
			key:      "name", keyed: false,
		},
		"scalar elements": {
			elements: []any{"--port", "4000"},
			key:      "name", keyed: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			byKey, ok := elementsByKey(tc.elements, tc.key)
			assert.Equal(t, tc.keyed, ok)
			if tc.keyed {
				assert.Len(t, byKey, len(tc.elements))
			} else {
				assert.Nil(t, byKey)
			}
		})
	}
}

// A Service with unnamed ports is the classic list that looks like a list-map
// but is not: keying it by name would drop every entry.
func TestKeyedLists_UnnamedServicePortsStayAnArray(t *testing.T) {
	service := `apiVersion: v1
kind: Service
metadata:
  name: litellm
spec:
  ports:
  - port: 4000
    targetPort: 4000
  - port: 4001
    targetPort: 4001
`
	manifestJSON, err := convertYAMLManifestToJSON(service, true)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(manifestJSON), &decoded))

	for _, resource := range decoded {
		ports := resource.(map[string]any)["spec"].(map[string]any)["ports"]
		assert.IsType(t, []any{}, ports, "unnamed ports must stay an array")
		assert.Len(t, ports, 2, "no port may be lost")
	}
}

// Keying must not interfere with the Secret hashing that keeps secret data out
// of the state file.
func TestKeyedLists_SecretsStayRedacted(t *testing.T) {
	secret := `apiVersion: v1
kind: Secret
metadata:
  name: litellm-masterkey
type: Opaque
data:
  masterkey: c3VwZXItc2VjcmV0
`
	for _, keyed := range []bool{false, true} {
		manifestJSON, err := convertYAMLManifestToJSON(secret, keyed)
		require.NoError(t, err)
		assert.NotContains(t, manifestJSON, "c3VwZXItc2VjcmV0",
			"secret data must be hashed regardless of keying")

		// Secret.Data is []byte, so the hashed marker is base64 in the JSON.
		var decoded map[string]map[string]any
		require.NoError(t, json.Unmarshal([]byte(manifestJSON), &decoded))
		for _, resource := range decoded {
			stored := resource["data"].(map[string]any)["masterkey"].(string)
			raw, err := base64.StdEncoding.DecodeString(stored)
			require.NoError(t, err)
			assert.Contains(t, string(raw), "(sensitive value")
		}
	}
}

// The golden manifest must survive a keyed round trip without losing resources.
func TestKeyedLists_PreservesEveryResource(t *testing.T) {
	yamlManifest := readTestFile(t, "testdata/manifest_json/rendered_manifest.yaml")

	plain, err := convertYAMLManifestToJSON(yamlManifest, false)
	require.NoError(t, err)
	keyed, err := convertYAMLManifestToJSON(yamlManifest, true)
	require.NoError(t, err)

	var plainMap, keyedMap map[string]any
	require.NoError(t, json.Unmarshal([]byte(plain), &plainMap))
	require.NoError(t, json.Unmarshal([]byte(keyed), &keyedMap))

	assert.Equal(t, len(plainMap), len(keyedMap))
	for key := range plainMap {
		assert.Contains(t, keyedMap, key)
	}
}
