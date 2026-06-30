// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-provider-helm/helm"
	"k8s.io/klog"
)

// Example version string that can be overwritten by a release process
var Version string = "dev"

func main() {
	var debug bool
	debugFlag := flag.Bool("debug", false, "Start provider in stand-alone debug mode.")
	flag.Parse()

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	err := klogFlags.Set("logtostderr", "false")
	if err != nil {
		panic(err)
	}

	opts := providerserver.ServeOpts{
		Address:         "registry.terraform.io/hashicorp/helm",
		Debug:           debug,
		ProtocolVersion: 6,
	}

	if *debugFlag {
		opts.Debug = true
	}

	serveErr := providerserver.Serve(context.Background(), helm.New(Version), opts)
	if serveErr != nil {
		log.Fatal(serveErr.Error())
	}
}
