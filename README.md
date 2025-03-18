<a href="https://terraform.io">
    <img src=".github/tf.png" alt="Terraform logo" title="Terraform" align="left" height="50" />
</a>

# Helm Provider for Terraform [![Actions Status](https://github.com/hashicorp/terraform-provider-helm/workflows/tests/badge.svg)](https://github.com/hashicorp/terraform-provider-helm/actions)[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/hashicorp/terraform-provider-helm?label=release)](https://github.com/hashicorp/terraform-provider-helm/releases)[![license](https://img.shields.io/github/license/hashicorp/terraform-provider-helm.svg)]()[![Go Report Card](https://goreportcard.com/badge/github.com/hashicorp/terraform-provider-helm)](https://goreportcard.com/report/github.com/hashicorp/terraform-provider-helm)


- [Documentation](https://www.terraform.io/docs/providers/helm/index.html)
- [#terraform-providers in Kubernetes Slack](https://kubernetes.slack.com/messages/CJY6ATQH4) ([Sign up here](http://slack.k8s.io/))

This is the [Helm](https://github.com/kubernetes/helm) provider for [Terraform](https://www.terraform.io/).

This provider allows you to install and manage [Helm Charts](https://artifacthub.io/packages/search?kind=0&sort=relevance&page=1) in your Kubernetes cluster using Terraform.


## Contents

* [Requirements](#requirements)
* [Getting Started](#getting-started)
* [Contributing to the provider](#contributing)

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) v1.x.x
-	[Go](https://golang.org/doc/install) v1.22.x (to build the provider plugin)

## Getting Started

This is a small example of how to install the nginx ingress controller chart. Please read the [documentation](https://www.terraform.io/docs/providers/helm/index.html) for more
information.

```hcl
provider "helm" {
  kubernetes = {
    config_path = "~/.kube/config"
  }
}

resource "helm_release" "nginx_ingress" {
  name       = "nginx-ingress-controller"

  repository = "oci://registry-1.docker.io/bitnamicharts"
  chart      = "nginx-ingress-controller"

  set = [
    {
    name  = "service.type"
    value = "ClusterIP"
    }
  ]
}
```

## Contributing

The Helm Provider for Terraform is the work of many contributors. We appreciate your help!

To contribute, please read the [contribution guidelines](_about/CONTRIBUTING.md). You may also [report an issue](https://github.com/hashicorp/terraform-provider-helm/issues/new/choose). Once you've filed an issue, it will follow the [issue lifecycle](_about/ISSUES.md).

Also available are some answers to [Frequently Asked Questions](_about/FAQ.md).
