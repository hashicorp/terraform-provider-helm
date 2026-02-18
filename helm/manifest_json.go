// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"helm.sh/helm/v3/pkg/releaseutil"
)

type resourceMeta struct {
	metav1.TypeMeta
	Metadata metav1.ObjectMeta
}

func convertYAMLManifestToJSON(manifest string) (string, error) {
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

		m[key] = jsonbytes
	}

	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func hashSensitiveValue(v string) string {
	hash := make([]byte, 8)
	sha3.ShakeSum256(hash, []byte(v))
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
