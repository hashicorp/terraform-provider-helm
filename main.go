// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/hashicorp/terraform-provider-helm/helm"
	"k8s.io/klog"
)

// Generate docs for website
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in stand-alone debug mode.")
	flag.Parse()
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	err := klogFlags.Set("logtostderr", "false")
	if err != nil {
		panic(err)
	}
	serveOpts := &plugin.ServeOpts{
		ProviderFunc: helm.Provider,
	}
	if debugFlag != nil && *debugFlag {
		plugin.Debug(context.Background(), "registry.terraform.io/hashicorp/helm", serveOpts)
	} else {
		plugin.Serve(serveOpts)
	}
}
