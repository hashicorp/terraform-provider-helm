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
			// Decoded into the typed corev1.Secret so redactSecretData can hash
			// Data - a map[string][]byte field, so this also base64-decodes each
			// entry the same way the real API server would. Any field on the
			// resource that isn't part of corev1.Secret's schema (nonstandard,
			// or newer than this pinned k8s.io/api dependency) is dropped by
			// this round-trip, unlike every other Kind, which stays on the
			// generic map path above and keeps unknown fields verbatim. Real
			// Secrets don't carry such fields - the API server itself would
			// reject them - so this is a display-only limitation with no
			// practical impact, not corrected here to avoid the complexity of
			// merging the hashed Data back onto the generic decode.
			secret := corev1.Secret{}
			err = yaml.Unmarshal([]byte(resource), &secret)
			if err != nil {
				return "", err
			}

			redactSecretData(&secret)

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
//
// Most entries use the field Kubernetes itself declares via
// `+listMapKey`/`x-kubernetes-list-map-keys` on the corresponding API type -
// verified against k8s.io/api, not assumed. Two exceptions, both deliberate:
//
//   - `ports` keys on "name" even though neither Container.Ports
//     (`+listMapKey=containerPort,protocol`) nor ServiceSpec.Ports
//     (`+listMapKey=port,protocol`) actually uses name as its merge key, and
//     elementsByKey only supports a single field, not a compound one. name is
//     still a safe choice - elementsByKey requires every element to carry a
//     unique, non-empty value for it, so an unnamed or duplicate-named port
//     list is left as an array rather than keyed incorrectly - and it is
//     useful in the common case of uniquely-named ports, at the cost of not
//     being what Kubernetes would actually merge on.
//   - `sysctls` is deliberately ABSENT even though every element does carry a
//     unique `name`: PodSecurityContext.Sysctls is declared
//     `+listType=atomic`, meaning Kubernetes replaces the entire list on any
//     change rather than merging by element. Presenting it as a keyed map
//     would misleadingly imply a single-entry diff has the same effect as
//     changing one sysctl in isolation, when the applied behavior actually
//     replaces the whole list.
var listMapKeys = map[string]string{
	"containers":          "name",
	"initContainers":      "name",
	"ephemeralContainers": "name",
	"env":                 "name",
	"ports":               "name", // heuristic, not k8s's declared key - see comment above
	"volumeMounts":        "mountPath",
	"volumeDevices":       "devicePath",
	"volumes":             "name",
	"imagePullSecrets":    "name",
	"secrets":             "name",
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

// redactSensitiveValues replaces every occurrence of a set_sensitive value in
// text with a stable hash, so a manifest stored in state or plan output never
// contains a value the caller marked sensitive.
//
// Skips empty strings deliberately: strings.ReplaceAll(text, "", marker)
// matches every position in text and would insert marker between every rune,
// corrupting the whole manifest rather than redacting nothing. sensitiveSetValues
// already filters these out before they reach here; this is a second, cheap
// guard on the one thing that would make the corruption catastrophic instead
// of silent.
func redactSensitiveValues(text string, sensitiveValues []string) string {
	masked := text

	for _, value := range sensitiveValues {
		if value == "" {
			continue
		}

		// text is always JSON (the manifest/resources attributes are always
		// produced by json.Marshal), so a value containing a character JSON
		// escapes - a double quote, a backslash, or a control character such
		// as a newline - never appears in text as its raw bytes. It appears
		// as its JSON-escaped form instead: a multi-line secret like a PEM
		// private key or certificate, or any secret containing a quote or
		// backslash, would otherwise survive "redaction" fully readable,
		// just with backslash-n / backslash-quote / double-backslash in
		// place of the original control characters. jsonEscapedForm is
		// exactly what value looks like once embedded in the JSON string
		// field that holds it, so search for that instead of the raw value.
		//
		// For a value with no JSON-special characters (the common case -
		// plain alphanumeric secrets), the escaped form is byte-identical to
		// the raw value, so this changes nothing for the values every
		// existing test already covers.
		escaped, ok := jsonEscapedForm(value)
		if !ok || escaped == "" {
			continue
		}
		masked = strings.ReplaceAll(masked, escaped, hashSensitiveValue(value))
	}

	return masked
}

// jsonEscapedForm returns value as it appears inside a JSON string field -
// json.Marshal's quoted encoding with the surrounding quotes stripped - or
// false if value cannot be encoded as a JSON string (never true for a Go
// string, which is always valid UTF-8 input to json.Marshal; the check
// exists so a theoretical encoding failure skips that one value instead of
// panicking or silently matching nothing).
func jsonEscapedForm(value string) (string, bool) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", false
	}
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return "", false
	}
	return string(b[1 : len(b)-1]), true
}

func redactSecretData(secret *corev1.Secret) {
	for k, v := range secret.Data {
		h := hashSensitiveValue(string(v))
		secret.Data[k] = []byte(h)
	}
}
