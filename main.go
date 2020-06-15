package main

import (
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/hashicorp/terraform-provider-helm/helm"
	"k8s.io/klog"
)

func main() {
	flag.Parse()
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	err := klogFlags.Set("logtostderr", "false")
	if err != nil {
		panic(err)
	}
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: helm.Provider,
	})
}
