package main

import (
	"github.com/divoxx/terraform-provider-helm/helm"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: helm.Provider,
	})
}
