---
layout: "helm"
page_title: "helm: helm_release"
sidebar_current: "docs-helm-release"
description: |-

---

# Resource: helm_release

A Release is an instance of a chart running in a Kubernetes cluster.
A Chart is a Helm package. It contains all of the resource definitions necessary to run an application, tool, or service inside of a Kubernetes cluster.

`helm_release` describes the desired status of a chart in a kubernetes cluster.

## Example Usage

```
resource "helm_release" "example" {
  name = "my_redis"
  chart = "redis"
  values = [
    "${file("values.yaml")}"
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Release name.
* `repository` - (Optional) Repository where to locate the requested chart. If is an URL the chart is installed without install the repository.
* `chart` - (Required) Chart name to be installed.
* `devel` - (Optional) Use chart development versions, too. Equivalent to version '>0.0.0-0'. If version is set, this is ignored.
* `version` - (Optional) Specify the exact chart version to install. If this is not specified, the latest version is installed.
* `values` - (Optional) List of values in raw yaml to pass to helm. Values will be merged, in order, as Helm does with multiple `-f` options.
* `set` - (Optional) Value block with custom values to be merge with the values.yaml.
* `namespace` - (Optional) Namespace to install the release into.
* `verify` - (Optional) Verify the package before installing it.
* `keyring` - (Optional) Location of public keys used for verification.
* `timeout` - (Optional) Time in seconds to wait for any individual kubernetes operation.
* `disable_webhooks` - (Optional) Prevent hooks from running.
* `force_update` - (Optional) Force resource update through delete/recreate if needed.
* `recreate_pods` - (Optional) On update performs pods restart for the resource if applicable.
* `reuse_values` - (Optional) Reuse values from previous revision when upgrading a release. Same as `--reuse-values` flag in Helm CLI. Default is false.
* `reuse` - (Optional) Instructs Tiller to re-use an existing name. Default is true.

The `set` block supports:

* `name` - (Required) full name of the variable to be set.
* `value` - (Required) value of the variable to be set.


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
* `values` - The compounded values from `values` and `set`

## Import

helm_release can be imported using the , e.g.

```
$ terraform import helm_release.example ...
```
