module github.com/terraform-providers/terraform-provider-helm

go 1.12

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/hashicorp/terraform-plugin-sdk v1.1.0
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/lib/pq v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/rubenv/sql-migrate v0.0.0-20191025130928-9355dd04f4b3 // indirect
	google.golang.org/grpc v1.24.0
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	helm.sh/helm/v3 v3.0.0
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v0.0.0
	k8s.io/kubernetes v1.16.1 // indirect
)

replace (
	// github.com/Azure/go-autorest/autorest has different versions for the Go
	// modules than it does for releases on the repository. Note the correct
	// version when updating.
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	github.com/russross/blackfriday => github.com/russross/blackfriday v1.5.2

	// k8s.io/kubernetes has a go.mod file that sets the version of the following
	// modules to v0.0.0. This causes go to throw an error. These need to be set
	// to a version for Go to process them. Here they are set to the same
	// revision as the marked version of Kubernetes. When Kubernetes is updated
	// these need to be updated as well.
	k8s.io/api => k8s.io/kubernetes/staging/src/k8s.io/api v0.0.0-20191001043732-d647ddbd755f
	k8s.io/apiextensions-apiserver => k8s.io/kubernetes/staging/src/k8s.io/apiextensions-apiserver v0.0.0-20191001043732-d647ddbd755f
	k8s.io/apimachinery => k8s.io/kubernetes/staging/src/k8s.io/apimachinery v0.0.0-20191001043732-d647ddbd755f
	k8s.io/apiserver => k8s.io/kubernetes/staging/src/k8s.io/apiserver v0.0.0-20191001043732-d647ddbd755f
	k8s.io/cli-runtime => k8s.io/kubernetes/staging/src/k8s.io/cli-runtime v0.0.0-20191001043732-d647ddbd755f
	k8s.io/client-go => k8s.io/kubernetes/staging/src/k8s.io/client-go v0.0.0-20191001043732-d647ddbd755f
	k8s.io/cloud-provider => k8s.io/kubernetes/staging/src/k8s.io/cloud-provider v0.0.0-20191001043732-d647ddbd755f
	k8s.io/cluster-bootstrap => k8s.io/kubernetes/staging/src/k8s.io/cluster-bootstrap v0.0.0-20191001043732-d647ddbd755f
	k8s.io/code-generator => k8s.io/kubernetes/staging/src/k8s.io/code-generator v0.0.0-20191001043732-d647ddbd755f
	k8s.io/component-base => k8s.io/kubernetes/staging/src/k8s.io/component-base v0.0.0-20191001043732-d647ddbd755f
	k8s.io/cri-api => k8s.io/kubernetes/staging/src/k8s.io/cri-api v0.0.0-20191001043732-d647ddbd755f
	k8s.io/csi-translation-lib => k8s.io/kubernetes/staging/src/k8s.io/csi-translation-lib v0.0.0-20191001043732-d647ddbd755f
	k8s.io/kube-aggregator => k8s.io/kubernetes/staging/src/k8s.io/kube-aggregator v0.0.0-20191001043732-d647ddbd755f
	k8s.io/kube-controller-manager => k8s.io/kubernetes/staging/src/k8s.io/kube-controller-manager v0.0.0-20191001043732-d647ddbd755f
	k8s.io/kube-proxy => k8s.io/kubernetes/staging/src/k8s.io/kube-proxy v0.0.0-20191001043732-d647ddbd755f
	k8s.io/kube-scheduler => k8s.io/kubernetes/staging/src/k8s.io/kube-scheduler v0.0.0-20191001043732-d647ddbd755f
	k8s.io/kubectl => k8s.io/kubernetes/staging/src/k8s.io/kubectl v0.0.0-20191001043732-d647ddbd755f
	k8s.io/kubelet => k8s.io/kubernetes/staging/src/k8s.io/kubelet v0.0.0-20191001043732-d647ddbd755f
	k8s.io/legacy-cloud-providers => k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers v0.0.0-20191001043732-d647ddbd755f
	k8s.io/metrics => k8s.io/kubernetes/staging/src/k8s.io/metrics v0.0.0-20191001043732-d647ddbd755f
	k8s.io/sample-apiserver => k8s.io/kubernetes/staging/src/k8s.io/sample-apiserver v0.0.0-20191001043732-d647ddbd755f
)
