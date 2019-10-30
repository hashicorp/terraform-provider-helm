module github.com/terraform-providers/terraform-provider-helm

go 1.12

require (
	github.com/DATA-DOG/go-sqlmock v1.3.3 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.18.0+incompatible // indirect
	github.com/apparentlymart/go-dump v0.0.0-20190214190832-042adf3cf4a0 // indirect
	github.com/aws/aws-sdk-go v1.25.3 // indirect
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190911111923-ecfe977594f1 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/gobuffalo/packr v1.25.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gophercloud/gophercloud v0.4.0 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.11.2 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.1.0
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20190327083759-54bad0a9b051 // indirect
	github.com/russross/blackfriday v2.0.0+incompatible // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/technosophos/moniker v0.0.0-20180509230615-a5dbd03a2245 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/vmihailenco/msgpack v4.0.1+incompatible // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/ziutek/mymysql v1.5.4 // indirect
	golang.org/x/net v0.0.0-20191009170851-d66e71096ffb // indirect
	golang.org/x/sys v0.0.0-20191002091554-b397fe3ad8ed // indirect
	google.golang.org/genproto v0.0.0-20190927181202-20e1ac93f88c // indirect
	google.golang.org/grpc v1.24.0
	gopkg.in/gorp.v1 v1.7.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v0.0.0
	k8s.io/helm v2.15.1+incompatible
	k8s.io/klog v1.0.0 // indirect
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d // indirect
	k8s.io/kubernetes v1.15.5 // indirect
	vbom.ml/util v0.0.0-20180919145318-efcd4e0f9787 // indirect
)

replace github.com/russross/blackfriday => github.com/russross/blackfriday v1.5.2

replace k8s.io/api => k8s.io/api v0.0.0-20191016110246-af539daaa43a

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113439-b64f2075a530

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115701-31ade1b30762

replace k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016111841-d20af8c7efc5

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016113937-7693ce2cae74

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20191016110837-54936ba21026

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115248-b061d4666016

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115051-4323e76404b0

replace k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b

replace k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111234-b8c37ee0c266

replace k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190817025403-3ae76f584e79

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115443-72c16c0ea390

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112329-27bff66d0b7c

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114902-c7514f1b89da

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114328-7650d5e6588e

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114710-682e84547325

replace k8s.io/kubectl => k8s.io/kubectl v0.0.0-20190602132728-7075c07e78bf

replace k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114520-100045381629

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115707-22244e5b01eb

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113728-f445c7b35c1c

replace k8s.io/node-api => k8s.io/node-api v0.0.0-20191016115915-6ca2fdc4797b

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112728-ceb381866e80

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20191016114133-1e47bf8f13f2

replace k8s.io/sample-controller => k8s.io/sample-controller v0.0.0-20191016113108-ba18715371a3
