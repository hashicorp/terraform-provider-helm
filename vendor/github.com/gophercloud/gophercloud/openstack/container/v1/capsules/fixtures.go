package capsules

// ValidJSONTemplate is a valid OpenStack Capsule template in JSON format
const ValidJSONTemplate = `
{
    "capsuleVersion": "beta",
    "kind": "capsule",
    "metadata": {
        "labels": {
            "app": "web",
            "app1": "web1"
        },
        "name": "template"
    },
    "restartPolicy": "Always",
    "spec": {
        "containers": [
            {
                "command": [
                    "/bin/bash"
                ],
                "env": {
                    "ENV1": "/usr/local/bin",
                    "ENV2": "/usr/bin"
                },
                "image": "ubuntu",
                "imagePullPolicy": "ifnotpresent",
                "ports": [
                    {
                        "containerPort": 80,
                        "hostPort": 80,
                        "name": "nginx-port",
                        "protocol": "TCP"
                    }
                ],
                "resources": {
                    "requests": {
                        "cpu": 1,
                        "memory": 1024
                    }
                },
                "workDir": "/root"
            }
        ]
    }
}
`

// ValidYAMLTemplate is a valid OpenStack Capsule template in YAML format
const ValidYAMLTemplate = `
capsuleVersion: beta
kind: capsule
metadata:
  name: template
  labels:
    app: web
    app1: web1
restartPolicy: Always
spec:
  containers:
  - image: ubuntu
    command:
      - "/bin/bash"
    imagePullPolicy: ifnotpresent
    workDir: /root
    ports:
      - name: nginx-port
        containerPort: 80
        hostPort: 80
        protocol: TCP
    resources:
      requests:
        cpu: 1
        memory: 1024
    env:
      ENV1: /usr/local/bin
      ENV2: /usr/bin
`

// ValidJSONTemplateParsed is the expected parsed version of ValidJSONTemplate
var ValidJSONTemplateParsed = map[string]interface{}{
	"capsuleVersion": "beta",
	"kind":           "capsule",
	"restartPolicy":  "Always",
	"metadata": map[string]interface{}{
		"name": "template",
		"labels": map[string]string{
			"app":  "web",
			"app1": "web1",
		},
	},
	"spec": map[string]interface{}{
		"containers": []map[string]interface{}{
			map[string]interface{}{
				"image": "ubuntu",
				"command": []interface{}{
					"/bin/bash",
				},
				"imagePullPolicy": "ifnotpresent",
				"workDir":         "/root",
				"ports": []interface{}{
					map[string]interface{}{
						"name":          "nginx-port",
						"containerPort": float64(80),
						"hostPort":      float64(80),
						"protocol":      "TCP",
					},
				},
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"cpu":    float64(1),
						"memory": float64(1024),
					},
				},
				"env": map[string]interface{}{
					"ENV1": "/usr/local/bin",
					"ENV2": "/usr/bin",
				},
			},
		},
	},
}

// ValidYAMLTemplateParsed is the expected parsed version of ValidYAMLTemplate
var ValidYAMLTemplateParsed = map[string]interface{}{
	"capsuleVersion": "beta",
	"kind":           "capsule",
	"restartPolicy":  "Always",
	"metadata": map[string]interface{}{
		"name": "template",
		"labels": map[string]string{
			"app":  "web",
			"app1": "web1",
		},
	},
	"spec": map[interface{}]interface{}{
		"containers": []map[interface{}]interface{}{
			map[interface{}]interface{}{
				"image": "ubuntu",
				"command": []interface{}{
					"/bin/bash",
				},
				"imagePullPolicy": "ifnotpresent",
				"workDir":         "/root",
				"ports": []interface{}{
					map[interface{}]interface{}{
						"name":          "nginx-port",
						"containerPort": 80,
						"hostPort":      80,
						"protocol":      "TCP",
					},
				},
				"resources": map[interface{}]interface{}{
					"requests": map[interface{}]interface{}{
						"cpu":    1,
						"memory": 1024,
					},
				},
				"env": map[interface{}]interface{}{
					"ENV1": "/usr/local/bin",
					"ENV2": "/usr/bin",
				},
			},
		},
	},
}
