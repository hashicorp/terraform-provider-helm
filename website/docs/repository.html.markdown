---
layout: "helm"
page_title: "helm: helm_repository"
sidebar_current: "docs-helm-repository"
description: |-

---

# Data Source: helm_repository

A chart repository is a location where packaged charts can be stored and shared.

`helm_repository` describes a helm repository.

## Example Usage

```hcl
data "helm_repository" "incubator" {
  name = "incubator"
  url  = "https://kubernetes-charts-incubator.storage.googleapis.com"
}

resource "helm_release" "my_cache" {
  name       = "my-cache"
  repository = data.helm_repository.incubator.metadata[0].name
  chart      = "redis-cache"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Chart repository name.
* `url` - (Required) Chart repository URL.
* `key_file` - (Optional) Identify HTTPS client using this SSL key file
* `cert_file` - (Optional) Identify HTTPS client using this SSL certificate file.
* `ca_file` - (Optional) Verify certificates of HTTPS-enabled servers using this CA bundle
* `username` - (Optional) Username for HTTP basic authentication.
* `password` - (Optional) Password for HTTP basic authentication.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `metadata` - Status of the deployed release.

The `metadata` block supports:

* `name` - Name of the repository read from the home.
* `url` - URL of the repository read from the home.

## Old resource helm_repository

Before 0.9.0 `helm_repository` was a resource and not a data source. The old resource is now a shim to the data source to preserve backwards compatibility. As the use of the resource is deprecated it is strongly suggested to move to the new data source as the compatibility will be removed in a future release.
