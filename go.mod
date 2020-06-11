module github.com/hashicorp/terraform-provider-helm

go 1.14

require (
	github.com/DATA-DOG/go-sqlmock v1.4.1 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.13.1
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/lib/pq v1.7.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/rubenv/sql-migrate v0.0.0-20200429072036-ae26b214fa43 // indirect
	google.golang.org/grpc v1.27.1
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/apimachinery v0.16.10
	k8s.io/client-go v0.16.10
	k8s.io/helm v2.16.8+incompatible
	k8s.io/kubernetes v1.16.1 // indirect
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.3+incompatible
	k8s.io/api => k8s.io/api v0.0.0-20191003000013-35e20aa79eb8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191003002041-49e3d608220c
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191003001037-3c8b233e046c
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191003002408-6e42c232ac7d
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191003000419-f68efa97b39e
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191003003426-b4b1f434fead
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191003003255-c493acd9e2ff
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190927045949-f81bca4f5e85
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191003000551-f573d376509c
	k8s.io/cri-api => k8s.io/cri-api v0.16.5-beta.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191003003551-0eecdcdcc049
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191003001317-a019a9d85a86
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191003003129-09316795c0dd
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191003002707-f6b7b0f55cc0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191003003001-314f0beee0a9
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191003002833-e367e4712542
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191003003732-7d49cdad1c12
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191003002233-837aead57baf
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191003001538-80f33ca02582
)

// needed because k8s.io/helm/pkg/kube imports k8s.io/kubernetes/pkg/kubectl/cmd/get,
// which only exists in older versions of kubernetes
replace k8s.io/kubernetes => k8s.io/kubernetes v1.16.1

// Needed for helm2, since it imports package "k8s.io/kubectl/pkg/util/printers"
replace k8s.io/kubectl => k8s.io/kubectl v0.16.10
