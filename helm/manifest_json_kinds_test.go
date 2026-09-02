// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// at walks a dotted path through decoded JSON. Numeric segments index arrays,
// everything else indexes objects, so the same path expresses "the second
// container" before keying and "the container called litellm" after it.
func at(t *testing.T, root any, path string) any {
	t.Helper()
	node := root
	for _, segment := range strings.Split(path, ".") {
		switch typed := node.(type) {
		case map[string]any:
			value, ok := typed[segment]
			require.Truef(t, ok, "no %q while walking %q", segment, path)
			node = value
		case []any:
			i, err := strconv.Atoi(segment)
			require.NoErrorf(t, err, "%q is not an index while walking %q", segment, path)
			require.Lessf(t, i, len(typed), "index %d out of range walking %q", i, path)
			node = typed[i]
		default:
			t.Fatalf("cannot descend into %T at %q of %q", node, segment, path)
		}
	}
	return node
}

// onlyResource decodes a single-resource manifest and returns that resource.
func onlyResource(t *testing.T, manifest string, keyed bool) any {
	t.Helper()
	out, err := convertYAMLManifestToJSON(manifest, keyed)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	require.Len(t, decoded, 1, "expected exactly one resource")
	for _, resource := range decoded {
		return resource
	}
	return nil
}

// leafCount counts scalar leaves, so a transform that drops or duplicates an
// element changes the number even though the shape differs.
func leafCount(node any) int {
	switch typed := node.(type) {
	case map[string]any:
		n := 0
		for _, v := range typed {
			n += leafCount(v)
		}
		return n
	case []any:
		n := 0
		for _, v := range typed {
			n += leafCount(v)
		}
		return n
	case nil:
		return 1
	default:
		return 1
	}
}

const podTemplate = `
      containers:
      - name: app
        image: app:1
        env:
        - name: A
          value: "1"
        - name: B
          value: "2"
        ports:
        - name: http
          containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /etc/app
      initContainers:
      - name: wait-db
        image: busybox
      - name: migrate
        image: app:1
      imagePullSecrets:
      - name: regcred
      hostAliases:
      - ip: "10.0.0.1"
        hostnames: [db.internal]
      volumes:
      - name: config
        configMap:
          name: app-config
      tolerations:
      - key: node-type
        operator: Equal
        value: dev`

func manifestCases() map[string]struct {
	manifest string
	keyed    []string
	arrays   []string
} {
	return map[string]struct {
		manifest string
		keyed    []string
		arrays   []string
	}{
		"Deployment": {
			manifest: "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: app\nspec:\n  template:\n    spec:" + podTemplate,
			keyed: []string{
				"spec.template.spec.containers",
				"spec.template.spec.containers.app.env",
				"spec.template.spec.containers.app.ports",
				"spec.template.spec.containers.app.volumeMounts",
				"spec.template.spec.initContainers",
				"spec.template.spec.imagePullSecrets",
				"spec.template.spec.volumes",
				"spec.template.spec.hostAliases",
			},
			arrays: []string{"spec.template.spec.tolerations"},
		},
		"StatefulSet with volumeClaimTemplates": {
			manifest: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: db
spec:
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ReadWriteOnce]
  template:
    spec:
      containers:
      - name: db
        image: pg:16
`,
			keyed: []string{"spec.template.spec.containers"},
			// identity lives at metadata.name, not name, so it stays ordered
			arrays: []string{"spec.volumeClaimTemplates", "spec.volumeClaimTemplates.0.spec.accessModes"},
		},
		"CronJob nested pod template": {
			manifest: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: nightly
spec:
  schedule: "0 3 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: worker
            image: worker:1
            env:
            - name: MODE
              value: batch
            command: ["/bin/sh", "-c", "run"]
`,
			keyed: []string{
				"spec.jobTemplate.spec.template.spec.containers",
				"spec.jobTemplate.spec.template.spec.containers.worker.env",
			},
			arrays: []string{"spec.jobTemplate.spec.template.spec.containers.worker.command"},
		},
		"Service with named ports": {
			manifest: `apiVersion: v1
kind: Service
metadata:
  name: app
spec:
  ports:
  - name: http
    port: 80
  - name: metrics
    port: 9090
`,
			keyed: []string{"spec.ports"},
		},
		"Service with unnamed ports": {
			manifest: "apiVersion: v1\nkind: Service\nmetadata:\n  name: app\nspec:\n  ports:\n  - port: 80\n  - port: 443\n",
			arrays:   []string{"spec.ports"},
		},
		"Service with partially named ports": {
			manifest: "apiVersion: v1\nkind: Service\nmetadata:\n  name: app\nspec:\n  ports:\n  - name: http\n    port: 80\n  - port: 443\n",
			arrays:   []string{"spec.ports"},
		},
		"NetworkPolicy ports have no name": {
			manifest: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: db
spec:
  ingress:
  - ports:
    - port: 5432
      protocol: TCP
  egress:
  - {}
`,
			arrays: []string{"spec.ingress", "spec.ingress.0.ports", "spec.egress"},
		},
		"Ingress rules stay ordered": {
			manifest: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app
spec:
  rules:
  - host: a.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: app
            port:
              name: http
`,
			arrays: []string{"spec.rules", "spec.rules.0.http.paths"},
		},
		"ServiceAccount secrets": {
			manifest: "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: app\nsecrets:\n- name: app-token\nimagePullSecrets:\n- name: regcred\n",
			keyed:    []string{"secrets", "imagePullSecrets"},
		},
		"ClusterRole rules stay ordered": {
			manifest: `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: app
rules:
- apiGroups: [""]
  resources: [pods]
  verbs: [get, list]
- apiGroups: ["apps"]
  resources: [deployments]
  verbs: [get]
`,
			arrays: []string{"rules", "rules.0.verbs", "rules.0.resources"},
		},
		"CustomResourceDefinition versions": {
			manifest: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  versions:
  - name: v1alpha1
    served: true
    storage: false
  - name: v1
    served: true
    storage: true
`,
			// versions is a genuine list-map, but the field is not on the
			// allowlist because CRDs use the name too freely; array is the safe
			// answer.
			arrays: []string{"spec.versions"},
		},
		"HorizontalPodAutoscaler behaviour policies": {
			manifest: `apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: app
spec:
  behavior:
    scaleDown:
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 80
`,
			arrays: []string{"spec.behavior.scaleDown.policies", "spec.metrics"},
		},
		"Pod with ephemeral and sysctls": {
			manifest: `apiVersion: v1
kind: Pod
metadata:
  name: debug
spec:
  securityContext:
    sysctls:
    - name: net.core.somaxconn
      value: "1024"
  ephemeralContainers:
  - name: debugger
    image: busybox
  containers:
  - name: app
    image: app:1
`,
			keyed: []string{"spec.containers", "spec.ephemeralContainers", "spec.securityContext.sysctls"},
		},
		"custom resource whose env is an ordered scalar list": {
			manifest: `apiVersion: example.com/v1
kind: Widget
metadata:
  name: w
spec:
  env:
  - FOO=1
  - BAR=2
  containers: 3
`,
			arrays: []string{"spec.env"},
		},
	}
}

func TestKeyedLists_AcrossResourceKinds(t *testing.T) {
	for name, tc := range manifestCases() {
		t.Run(name, func(t *testing.T) {
			resource := onlyResource(t, tc.manifest, true)

			for _, path := range tc.keyed {
				assert.IsTypef(t, map[string]any{}, at(t, resource, path), "%s should be keyed", path)
			}
			for _, path := range tc.arrays {
				assert.IsTypef(t, []any{}, at(t, resource, path), "%s must stay an array", path)
			}
		})
	}
}

// Keying must never lose, duplicate or invent data.
// hostAliases are keyed by ip rather than name, and the hostnames inside one
// remain an ordered list of scalars.
func TestKeyedLists_HostAliasesKeyedByIP(t *testing.T) {
	resource := onlyResource(t, "apiVersion: v1\nkind: Pod\nmetadata:\n  name: p\nspec:\n  hostAliases:\n  - ip: 10.0.0.1\n    hostnames: [db.internal, db]\n  containers:\n  - name: app\n", true)

	aliases := at(t, resource, "spec.hostAliases")
	require.IsType(t, map[string]any{}, aliases)
	assert.Contains(t, aliases, "10.0.0.1")
	assert.IsType(t, []any{}, aliases.(map[string]any)["10.0.0.1"].(map[string]any)["hostnames"])
}

// An empty or comment-only document is not a resource and must not appear.
func TestKeyedLists_EmptyDocumentsAreSkipped(t *testing.T) {
	manifest := "apiVersion: v1\nkind: Service\nmetadata:\n  name: app\nspec:\n  ports:\n  - name: http\n    port: 80\n---\n# only a comment\n---\n\n"

	for _, keyed := range []bool{false, true} {
		out, err := convertYAMLManifestToJSON(manifest, keyed)
		require.NoError(t, err)
		assert.NotContains(t, out, `"//"`, "empty documents must not be stored")

		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(out), &decoded))
		assert.Len(t, decoded, 1)
	}
}

func TestKeyedLists_IsLossless(t *testing.T) {
	for name, tc := range manifestCases() {
		t.Run(name, func(t *testing.T) {
			plain := onlyResource(t, tc.manifest, false)
			keyed := onlyResource(t, tc.manifest, true)
			assert.Equal(t, leafCount(plain), leafCount(keyed),
				"keying changed the number of leaf values")
		})
	}
}

// A list-map field carrying duplicate identities is exactly what the litellm
// chart used to render for DATABASE_HOST. Keying it would silently drop an
// entry, so it has to stay an array.
func TestKeyedLists_DuplicateNamesStayAnArray(t *testing.T) {
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
      - name: app
        env:
        - name: DATABASE_HOST
          value: from-values
        - name: DATABASE_HOST
          valueFrom:
            secretKeyRef:
              name: db
              key: host
`
	resource := onlyResource(t, manifest, true)
	env := at(t, resource, "spec.template.spec.containers.app.env")
	require.IsType(t, []any{}, env, "duplicate names must not be keyed")
	assert.Len(t, env, 2, "neither entry may be dropped")
}

func TestKeyedLists_EmptyAndNullListsSurvive(t *testing.T) {
	manifest := `apiVersion: v1
kind: Pod
metadata:
  name: p
spec:
  containers:
  - name: app
    env: []
    volumeMounts: null
`
	resource := onlyResource(t, manifest, true)
	container := at(t, resource, "spec.containers.app")

	assert.IsType(t, []any{}, container.(map[string]any)["env"], "an empty list has no identities to key on")
	assert.Nil(t, container.(map[string]any)["volumeMounts"])
}

// Multi-document manifests, empty documents and comments are what Helm actually
// emits between templates.
func TestKeyedLists_MultiDocumentManifest(t *testing.T) {
	manifest := `# leading comment
apiVersion: v1
kind: Service
metadata:
  name: app
spec:
  ports:
  - name: http
    port: 80
---
# an empty document follows
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: app
secrets:
- name: token
`
	out, err := convertYAMLManifestToJSON(manifest, true)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	assert.Len(t, decoded, 2, "both real documents should survive")

	assert.IsType(t, map[string]any{}, at(t, decoded, "service/v1/app.spec.ports"))
	assert.IsType(t, map[string]any{}, at(t, decoded, "serviceaccount/v1/app.secrets"))
}

// The same kind in two namespaces must stay two separate entries.
func TestKeyedLists_NamespacedKeysStaySeparate(t *testing.T) {
	manifest := `apiVersion: v1
kind: Service
metadata:
  name: app
  namespace: one
spec:
  ports:
  - name: http
    port: 80
---
apiVersion: v1
kind: Service
metadata:
  name: app
  namespace: two
spec:
  ports:
  - name: http
    port: 81
`
	out, err := convertYAMLManifestToJSON(manifest, true)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	assert.Len(t, decoded, 2)
}
