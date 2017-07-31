---
layout: "helm"
page_title: "Helm: helm_chart"
sidebar_current: "docs-helm-chart"
description: |-
  A Chart is a Helm package. It contains all of the resource definitions necessary to run an application, tool, or service inside of a Kubernetes cluster.
---

# helm_chart

A Chart is a Helm package. It contains all of the resource definitions necessary to run an application, tool, or service inside of a Kubernetes cluster.

`helm_chart` describes the desired status of a chart in a kubernetes cluster.

## Example Usage

```
resource "helm_chart" "example" {
  name = "my_redis"
  chart = "redis"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Release name.
* `repository` - (Optional) Repository where to locate the requested chart. If is an URL the chart is installed without install the repository.
* `chart` - (Required) Chart name to be installed.
* `version` - (Optional) Specify the exact chart version to install. If this is not specified, the latest version is installed.
* `value` - (Optional) Value block with custom values to be merge with the values.yaml.
* `namespace` - (Optional) Namespace to install the release into.
* `repository_url` - (Optional) Repository URL where to locate the requested chart without install the repository.
* `verify` - (Optional) Verify the package before installing it.
* `keyring` - (Optional) Location of public keys used for verification.
* `timeout` - (Optional) Time in seconds to wait for any individual kubernetes operation.
* `disable_webhooks` - (Optional) Prevent hooks from running.
* `force_update` - (Optional) Force resource update through delete/recreate if needed.
* `recreate_pods` - (Optional) On update performs pods restart for the resource if applicable.

The `value` block supports:

* `content` - (Required)
* `name` - (Required)


## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `metadata` - Block status of the deployed release.

The `metadata` block supports:

* `chart` - The name of the chart.
* `name` - Name is the name of the release.
* `namespace` - Namespace is the kubernetes namespace of the release.
* `revision` - Version is an int32 which represents the version of the release.
* `status` - Status of the release.
* `version` - A SemVer 2 conformant version string of the chart.


## Import

helm_chart can be imported using the , e.g.

```
$ terraform import helm_chart.example ...
```
