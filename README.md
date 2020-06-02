Terraform Provider for Helm
[![Actions Status](https://github.com/terraform-providers/terraform-provider-helm/workflows/tests/badge.svg)](https://github.com/terraform-providers/terraform-provider-helm/actions)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/terraform-providers/terraform-provider-helm?label=release)](https://github.com/terraform-providers/terraform-provider-helm/releases)
[![license](https://img.shields.io/github/license/terraform-providers/terraform-provider-helm.svg)]()
[![Go Report Card](https://goreportcard.com/badge/github.com/terraform-providers/terraform-provider-helm)](https://goreportcard.com/report/github.com/terraform-providers/terraform-provider-helm)
===========================
<a href="https://terraform.io">
    <img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" alt="Terraform logo" title="Terrafpr," align="right" height="50" />
</a>

- [Documentation](https://www.terraform.io/docs/providers/helm/index.html)
- [Mailing list](http://groups.google.com/group/terraform-tool)
- [#terraform-providers in Kubernetes Slack](https://kubernetes.slack.com/messages/CJY6ATQH4) ([Sign up here](http://slack.k8s.io/))


This is the [Helm](https://github.com/kubernetes/helm) provider for [Terraform](https://www.terraform.io/).

The provider manages the installed [Charts](https://github.com/helm/charts) in your Kubernetes cluster, in the same way Helm does, through Terraform.


## Helm v2 support 


Release `1.0.0` for this provider brought support for Helm v3. This was a breaking change that removed support for Helm v2 and tiller. If you are still using Helm v2 and tiller you will have to [pin your provider version](https://www.terraform.io/docs/configuration/providers.html#provider-versions) to the latest `0.10.x` release. 

We will continue to accept bugfixes for the `0.10.x` version of the provider, please open your pull request against the latest `release-0.10.x` branch. 


## Contents

* [Requirements](#requirements)
* [Getting Started](#getting-started)
* [Contributing to the provider](#contributing)

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) 0.12.x
    - Note that version 0.11.x currently works, but is [deprecated](https://www.hashicorp.com/blog/deprecating-terraform-0-11-support-in-terraform-providers/)
-	[Go](https://golang.org/doc/install) 1.14.x (to build the provider plugin)

## Getting Started

This is a small example of how to install the mariadb chart on your default
kubernetes cluster, since the provider was initialized, all the configuration
is retrieved from the environment. Please read the [documentation](https://www.terraform.io/docs/providers/helm/index.html) for more
information.

You should have a local configured copy of kubectl.

```hcl
resource "helm_release" "my_database" {
    name      = "my-database"
    chart     = "stable/mariadb"

    set {
        name  = "mariadbUser"
        value = "foo"
    }

    set {
        name = "mariadbPassword"
        value = "qux"
    }

    set_string {
        name = "image.tags"
        value = "registry\\.io/terraform-provider-helm\\,example\\.io/terraform-provider-helm"
    }
}
```


## Contributing

The Terraform Kubernetes Provider is the work of many contributors. We appreciate your help!

To contribute, please read the [contribution guidelines](_about/CONTRIBUTING.md). You may also [report an issue](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/new/choose). Once you've filed an issue, it will follow the [issue lifecycle](ISSUES.md).

Also available are some answers to [Frequently Asked Questions](_about/FAQ.md).
