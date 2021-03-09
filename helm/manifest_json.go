package helm

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var yamlDocumentSeparator = "\n---"

type resourceMeta struct {
	metav1.TypeMeta
	Metadata metav1.ObjectMeta
}

func convertYAMLManifestToJSON(manifest string) (string, error) {
	m := map[string]json.RawMessage{}
	resources := strings.Split(manifest, yamlDocumentSeparator)
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
		key := fmt.Sprintf("%s/%s", strings.ToLower(gvk.GroupKind().String()),
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
				// compute a checksum for the secret field so it appears in the diff
				sha := sha512.New()
				sha.Write(v)
				sum := sha.Sum(nil)
				secret.Data[k] = []byte(fmt.Sprintf("(sensitive value %x)", sum[:8]))
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

// redactSensitiveValues removes values that appear in `set_sensitive` blocks from
// the manifest JSON
func redactSensitiveValues(text string, d resourceGetter) string {
	masked := text

	for _, v := range d.Get("set_sensitive").(*schema.Set).List() {
		vv := v.(map[string]interface{})

		if sensitiveValue, ok := vv["value"].(string); ok {
			masked = strings.ReplaceAll(masked, sensitiveValue, "(sensitive value)")
		}
	}

	return masked
}
