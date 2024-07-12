---
layout: "helm"
page_title: "Helm: Upgrade Guide for Helm Provider v2.0.0"
description: |-
  This guide covers the changes introduced in v2.0.0 of the Helm provider and what you may need to do to upgrade your configuration.
---

# Upgrading to v2.0.0 of the Helm provider

This guide covers the changes introduced in v2.0.0 of the Helm provider and what you may need to do to upgrade your configuration.

## Changes in v2.0.0

### Changes to Kubernetes credentials supplied in the provider block

We have made several changes to the way access to Kubernetes is configured in the provider block.

1. The `load_config_file` attribute has been removed.
2. Support for the `KUBECONFIG` environment variable has been dropped and replaced with `KUBE_CONFIG_PATH`.
3. The `config_path` attribute will no longer default to `~/.kube/config` and must be set explicitly.

The above changes have been made to encourage the best practise of configuring access to Kubernetes in the provider block explicitly, instead of relying upon default paths or `KUBECONFIG` being set. We have done this because allowing the provider to configure its access to Kubernetes implicitly caused confusion with a subset of our users. It also created risk for users who use Terraform to manage multiple clusters. Requiring explicit configuring for kubernetes in the provider block eliminates the possibility that the configuration will be applied to the wrong cluster.

You will therefore need to explicity configure access to your Kubernetes cluster in the provider block going forward. For many users this will simply mean specifying the `config_path` attribute in the provider block. Users already explicitly configuring the provider should not be affected by this change, but will need to remove the `load_config_file` attribute if they are currently using it.

When running Terraform inside a Kubernetes cluster no provider configuration is neccessary, as the provider will detect that is has access to a service account token.

### Removal of the `helm_repository` data source

This feature of the provider caused a fair bit of confusion and was a source of instability as data sources are not supposed to be stateful. This data source performed a stateful operation that modified the filesystem, mirroring similar functionality to the `helm repo add` command. It has been the recommendation for some time to configure repository information explicity at the `helm_resource` level and so the data source has been removed. See the example below.

```hcl
resource "helm_release" "redis" {
  name  = "redis"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "redis"
}
```

The provider will continue to work with repositories that are configured with `helm repo add` before Terraform is run.

### Removal of `set_string` in the `helm_release` resource

The addition of a `type` attribute to the `set` block has rendered `set_string` superfluous so it has been removed. See the example below on how to set a string using the `set` block. This is used when the type of a value is an ambigious (e.g strings containing only numbers, true, false) and we want it to be explicitly parsed as a string.

```hcl
resource "helm_release" "redis" {
  name  = "redis"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "redis"

  set {
    name  = "test.value"
    value = "123456"
    type  = "string"
  }
}
```

### Dropped support for Terraform 0.11

All builds of the Helm provider going forward will no longer work with Terraform 0.11. See [Upgrade Guides](https://www.terraform.io/upgrade-guides/index.html) for how to migrate your configurations to a newer version of Terraform.

### Upgrade to v2 of the Terraform Plugin SDK

Contributors to the provider will be interested to know this upgrade has brought the latest version of the [Terraform Plugin SDK](https://github.com/hashicorp/terraform-plugin-sdk) which introduced a number of enhancements to the developer experience. Details of the changes introduced can be found under [Extending Terraform](https://www.terraform.io/docs/extend/guides/v2-upgrade-guide.html).

## Helm 2

We removed support in the provider for Helm 2 earlier this year. In accordance with the [Helm v2 deprecation timeline](https://helm.sh/blog/helm-v2-deprecation-timeline/) we will no longer be accepting PRs or handling issues that relate to Helm 2 going forward.
