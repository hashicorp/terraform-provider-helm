module github.com/terraform-providers/terraform-provider-helm

go 1.14

require (
	github.com/hashicorp/terraform-plugin-sdk v1.9.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	helm.sh/helm/v3 v3.2.0
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.3+incompatible
