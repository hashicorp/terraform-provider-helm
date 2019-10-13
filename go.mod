module github.com/terraform-providers/terraform-provider-helm

go 1.12

require (
	// github.com/Azure/go-autorest v11.1.0+incompatible // indirect
	// github.com/DATA-DOG/go-sqlmock v1.3.3 // indirect
	// github.com/Masterminds/semver v1.4.2 // indirect
	// github.com/Masterminds/sprig v2.18.0+incompatible // indirect
	// github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	// github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	// github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	// github.com/elazarl/goproxy v0.0.0-20190911111923-ecfe977594f1 // indirect
	// github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	// github.com/gobuffalo/packr v1.25.0 // indirect
	// github.com/gogo/protobuf v1.3.0 // indirect
	// github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	// github.com/googleapis/gnostic v0.3.1 // indirect
	// github.com/gophercloud/gophercloud v0.4.0 // indirect
	// github.com/gorilla/websocket v1.4.1 // indirect
	// github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	// github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	// github.com/grpc-ecosystem/grpc-gateway v1.11.2 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.1.0
	// github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	// github.com/prometheus/client_golang v1.1.0 // indirect
	// github.com/rubenv/sql-migrate v0.0.0-20190327083759-54bad0a9b051 // indirect
	// github.com/russross/blackfriday v2.0.0+incompatible // indirect
	// github.com/soheilhy/cmux v0.1.4 // indirect
	// github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	// github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	// github.com/ziutek/mymysql v1.5.4 // indirect
	// golang.org/x/net v0.0.0-20191002035440-2ec189313ef0 // indirect
	// golang.org/x/sys v0.0.0-20191002091554-b397fe3ad8ed // indirect
	// google.golang.org/genproto v0.0.0-20190927181202-20e1ac93f88c // indirect
	google.golang.org/grpc v1.24.0
	// gopkg.in/gorp.v1 v1.7.2 // indirect
	// gopkg.in/inf.v0 v0.9.1 // indirect
	// gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	helm.sh/helm/v3 v3.0.0-beta.4.0.20191011210504-34b930cb9db8
	k8s.io/client-go v0.0.0
	k8s.io/kubernetes v1.16.1 // indirect
//vbom.ml/util v0.0.0-20180919145318-efcd4e0f9787 // indirect
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
	k8s.io/helm => helm.sh/helm/v3 v3.0.0-beta.4.0.20191011210504-34b930cb9db8
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
