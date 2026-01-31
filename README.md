<a href="https://terraform.io">
    <img src=".github/tf.png" alt="Terraform logo" title="Terraform" align="left" height="50" />
</a>

# Helm Provider for Terraform (Community Fork)

[![Actions Status](https://github.com/schnell3526/terraform-provider-helm/workflows/tests/badge.svg)](https://github.com/schnell3526/terraform-provider-helm/actions)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/schnell3526/terraform-provider-helm?label=release)](https://github.com/schnell3526/terraform-provider-helm/releases)
[![license](https://img.shields.io/github/license/schnell3526/terraform-provider-helm.svg)]()

> **This is a community-maintained fork of [hashicorp/terraform-provider-helm](https://github.com/hashicorp/terraform-provider-helm)**

## Why this fork?

We operate Kubernetes platforms at a Japanese tech company and encountered critical bugs in the upstream provider that have not been addressed in a timely manner. Most notably, the [state deletion bug (#1669)](https://github.com/hashicorp/terraform-provider-helm/issues/1669) forced us to maintain permanent `import.tf` files as a workaround. While this prevented immediate failures, it still caused state drift and required constant communication overhead to explain the situation to team members.

This fork aims to:
- Provide timely fixes for critical bugs
- Maintain compatibility with latest Helm versions (including Helm v4 support)
- Keep the provider usable for production workloads

If HashiCorp resumes active maintenance and addresses these issues, this fork may be archived.

## Installation

```hcl
terraform {
  required_providers {
    helm = {
      source  = "schnell3526/helm"
      version = "~> 3.1"
    }
  }
}
```

## Migrating from hashicorp/helm

```diff
terraform {
  required_providers {
    helm = {
-      source  = "hashicorp/helm"
+      source  = "schnell3526/helm"
      version = "~> 3.1"
    }
  }
}
```

```bash
terraform init -upgrade
```

State is compatible - no import required.

## Changes from upstream

- Fix: Preserve Terraform state on failed Helm operations ([#1669](https://github.com/hashicorp/terraform-provider-helm/issues/1669))
- Fix: Nil pointer crash when updating OCI chart dependencies ([#1726](https://github.com/hashicorp/terraform-provider-helm/pull/1726))

## Roadmap

- [ ] Helm v4 support

---

## About

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
