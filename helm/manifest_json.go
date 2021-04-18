package helm

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

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

// hashSensitiveValue creates a hash of a sensitive value and returns the string
// "(sensitive value xxxxxxxx)". We have to do this because Terraform's sensitive
// value feature can't reach inside a text string and would supress the entire
// manifest if we marked it as sensitive. This allows us to redact the value while
// still being able to surface that something has changed so it appears in the diff.
func hashSensitiveValue(v string) string {
	hash := make([]byte, 8)
	sha3.ShakeSum256(hash, []byte(v))
	return fmt.Sprintf("(sensitive value %x)", hash)
}

// redactSensitiveValues removes values that appear in `set_sensitive` blocks from
// the manifest JSON
func redactSensitiveValues(text string, d resourceGetter) string {
	masked := text

	for _, v := range d.Get("set_sensitive").(*schema.Set).List() {
		vv := v.(map[string]interface{})

		if sensitiveValue, ok := vv["value"].(string); ok {
			h := hashSensitiveValue(sensitiveValue)
			masked = strings.ReplaceAll(masked, sensitiveValue, h)
		}
	}

	return masked
}
