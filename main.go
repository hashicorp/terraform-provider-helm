package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/terraform-providers/terraform-provider-helm/helm"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: helm.Provider,
	})
}
