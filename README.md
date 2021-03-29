Helm Provider for Terraform
[![Actions Status](https://github.com/hashicorp/terraform-provider-helm/workflows/tests/badge.svg)](https://github.com/hashicorp/terraform-provider-helm/actions)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/hashicorp/terraform-provider-helm?label=release)](https://github.com/hashicorp/terraform-provider-helm/releases)
[![license](https://img.shields.io/github/license/hashicorp/terraform-provider-helm.svg)]()
[![Go Report Card](https://goreportcard.com/badge/github.com/hashicorp/terraform-provider-helm)](https://goreportcard.com/report/github.com/hashicorp/terraform-provider-helm)
===========================
<a href="https://terraform.io">
    <img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" alt="Terraform logo" title="Terrafpr," align="right" height="50" />
</a>

- [Documentation](https://www.terraform.io/docs/providers/helm/index.html)
- [Mailing list](http://groups.google.com/group/terraform-tool)
- [#terraform-providers in Kubernetes Slack](https://kubernetes.slack.com/messages/CJY6ATQH4) ([Sign up here](http://slack.k8s.io/))
- [Roadmap: Q3 2020](_about/ROADMAP.md)

This is the [Helm](https://github.com/kubernetes/helm) provider for [Terraform](https://www.terraform.io/).

The provider manages the installed [Charts](https://github.com/helm/charts) in your Kubernetes cluster, in the same way Helm does, through Terraform.


## Contents

* [Requirements](#requirements)
* [Getting Started](#getting-started)
* [Contributing to the provider](#contributing)

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) v0.12.x or later
-	[Go](https://golang.org/doc/install) v1.16.x (to build the provider plugin)
-   [Helm](https://github.com/helm/helm/releases) v3.x.x to deploy your charts

### Legacy provider versions

Helm provider v0.10.x
-	See latest `release-0.10.x` branch
-	Terraform v0.11.x or later (at least including v0.14.x)
-	[Helm v2.x.x](https://v2.helm.sh/)

Helm provider v1.x.x
-	See latest `release-1.x.x` branch
-	Terraform v0.11.x or later (at least including v0.14.x)
-	Helm v3.x.x

How to upgrade from Helm v2.x.x to v3.x.x:
-	https://helm.sh/blog/migrate-from-helm-v2-to-helm-v3/
-	https://helm.sh/docs/topics/v2_v3_migration/

## Getting Started

This is a small example of how to install the nginx ingress controller chart. Please read the [documentation](https://www.terraform.io/docs/providers/helm/index.html) for more
information.

```hcl
provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }
}

resource helm_release nginx_ingress {
  name       = "nginx-ingress-controller"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx-ingress-controller"

  set {
    name  = "service.type"
    value = "ClusterIP"
  }
}
```

## Contributing

The Helm Provider for Terraform is the work of many contributors. We appreciate your help!

To contribute, please read the [contribution guidelines](_about/CONTRIBUTING.md). You may also [report an issue](https://github.com/hashicorp/terraform-provider-helm/issues/new/choose). Once you've filed an issue, it will follow the [issue lifecycle](_about/ISSUES.md).

Also available are some answers to [Frequently Asked Questions](_about/FAQ.md).
