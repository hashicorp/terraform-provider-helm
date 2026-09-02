// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"crypto/sha3"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"helm.sh/helm/v3/pkg/releaseutil"
)

type resourceMeta struct {
	metav1.TypeMeta
	Metadata metav1.ObjectMeta
}

func convertYAMLManifestToJSON(manifest string, keyLists bool) (string, error) {
	m := map[string]json.RawMessage{}

	resources := releaseutil.SplitManifests(manifest)
	for _, resource := range resources {
		jsonbytes, err := yaml.YAMLToJSON([]byte(resource))
		if err != nil {
			return "", fmt.Errorf("could not convert manifest to JSON: %v", err)
		}

		resourceMeta := resourceMeta{}
		err = yaml.Unmarshal([]byte(resource), &resourceMeta)
		if err != nil {
			return "", err
		}

		gvk := resourceMeta.GetObjectKind().GroupVersionKind()

		// Helm hands back a document for every template it rendered, including
		// the ones a conditional emptied out and the ones that are only
		// comments. Those carry no kind, and storing them produced a single
		// bogus "//" entry in the manifest that tracked nothing in the cluster
		// and moved in and out of the diff as templates toggled.
		if gvk.Kind == "" {
			continue
		}

		key := fmt.Sprintf("%s/%s/%s", strings.ToLower(gvk.GroupKind().String()),
			resourceMeta.APIVersion,
			resourceMeta.Metadata.Name)

		if namespace := resourceMeta.Metadata.Namespace; namespace != "" {
			key = fmt.Sprintf("%s/%s", namespace, key)
		}

		if gvk.Kind == "Secret" {
			secret := corev1.Secret{}
			err = yaml.Unmarshal([]byte(resource), &secret)
			if err != nil {
				return "", err
			}

			for k, v := range secret.Data {
				h := hashSensitiveValue(string(v))
				secret.Data[k] = []byte(h)
			}

			jsonbytes, err = json.Marshal(secret)
			if err != nil {
				return "", err
			}
		}

		if keyLists {
			keyed, err := keyManifestLists(jsonbytes)
			if err != nil {
				return "", fmt.Errorf("could not key list-maps in %s: %v", key, err)
			}
			jsonbytes = keyed
		}

		m[key] = jsonbytes
	}

	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// listMapKeys names the manifest fields that the Kubernetes API declares as
// `x-kubernetes-list-type: map`, together with the field that identifies an
// element within them. Server-side apply already treats these as unordered
// collections keyed by that field rather than as ordered arrays.
//
// Terraform has no such notion: it renders a JSON array positionally, so
// inserting a single element shifts every element after it and the plan reports
// the whole list as changed. A chart that adds one environment variable at the
// top of a container's `env` therefore reads as though the entire pod spec were
// being rewritten. Re-keying these lists by their identifying field lets
// Terraform diff them entry by entry, so an inserted element shows up as one
// addition and nothing else moves.
//
// Only fields on this list are re-keyed. Anything absent from it - `args`,
// `command`, `rules`, `finalizers` - is genuinely ordered and is left as an
// array.
var listMapKeys = map[string]string{
	"containers":          "name",
	"initContainers":      "name",
	"ephemeralContainers": "name",
	"env":                 "name",
	"ports":               "name",
	"volumeMounts":        "name",
	"volumeDevices":       "name",
	"volumes":             "name",
	"imagePullSecrets":    "name",
	"secrets":             "name",
	"sysctls":             "name",
	"hostAliases":         "ip",
}

// keyManifestLists rewrites the list-map fields of a single rendered resource
// into JSON objects keyed by their identifying field. It is a presentation
// change only: the result is stored in the `manifest` attribute, which exists
// to be diffed and is never applied to the cluster.
func keyManifestLists(resource []byte) ([]byte, error) {
	var decoded any
	if err := json.Unmarshal(resource, &decoded); err != nil {
		return nil, err
	}

	return json.Marshal(keyListMapsIn(decoded))
}

func keyListMapsIn(node any) any {
	switch typed := node.(type) {
	case map[string]any:
		for field, child := range typed {
			child = keyListMapsIn(child)

			if elements, isList := child.([]any); isList {
				if key, keyed := listMapKeys[field]; keyed {
					if byKey, ok := elementsByKey(elements, key); ok {
						child = byKey
					}
				}
			}

			typed[field] = child
		}
		return typed
	case []any:
		for i, element := range typed {
			typed[i] = keyListMapsIn(element)
		}
		return typed
	default:
		return node
	}
}

// elementsByKey re-keys elements by the value of key, reporting false if the
// list does not actually behave like a list-map. Every element has to be an
// object carrying a unique, non-empty string under key; a Service with unnamed
// ports or a container port list that omits `name` stays an array, because
// keying it would either lose an element or invent an identity for it.
func elementsByKey(elements []any, key string) (map[string]any, bool) {
	if len(elements) == 0 {
		return nil, false
	}

	byKey := make(map[string]any, len(elements))
	for _, element := range elements {
		object, isObject := element.(map[string]any)
		if !isObject {
			return nil, false
		}

		identity, isString := object[key].(string)
		if !isString || identity == "" {
			return nil, false
		}

		if _, duplicate := byKey[identity]; duplicate {
			return nil, false
		}

		byKey[identity] = object
	}

	return byKey, true
}

func hashSensitiveValue(v string) string {
	hash := sha3.SumSHAKE256([]byte(v), 8)
	return fmt.Sprintf("(sensitive value %x)", hash)
}

// redactSensitiveValues removes values that appear in `set_sensitive` blocks from the manifest JSON
func redactSensitiveValues(text string, sensitiveValues map[string]string) string {
	masked := text

	for originalValue := range sensitiveValues {
		hashedValue := hashSensitiveValue(originalValue)
		masked = strings.ReplaceAll(masked, originalValue, hashedValue)
	}

	return masked
}

func redactSecretData(secret *corev1.Secret) {
	for k, v := range secret.Data {
		h := hashSensitiveValue(string(v))
		secret.Data[k] = []byte(h)
	}
}
