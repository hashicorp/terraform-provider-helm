package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/mcuadros/terraform-provider-helm/helm"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: helm.Provider,
	})
}
