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

```hcl
resource "helm_release" "example" {
  name       = "my-redis-release"
  repository = "https://kubernetes-charts.storage.googleapis.com" 
  chart      = "redis"
  version    = "6.0.1"

  values = [
    "${file("values.yaml")}"
  ]

  set {
    name  = "cluster.enabled"
    value = "true"
  }

  set {
    name  = "metrics.enabled"
    value = "true"
  }

  set_string {
    name  = "service.annotations.prometheus\\.io/port"
    value = "9127"
  }
}
```

## Example Usage - Local Chart

In case a Chart is not available from a repository, a path may be used:

```hcl
resource "helm_release" "local" {
  name       = "my-local-chart"
  chart      = "./charts/example"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Release name.
* `chart` - (Required) Chart name to be installed. A path may be used.
* `repository` - (Optional) Repository URL where to locate the requested chart.
* `repository_key_file` - (Optional) The repositories cert key file
* `repository_cert_file` - (Optional) The repositories cert file
* `repository_ca_file` - (Optional) The Repositories CA File. 
* `repository_username` - (Optional) Username for HTTP basic authentication against the repository.
* `repository_password` - (Optional) Password for HTTP basic authentication against the repository.
* `devel` - (Optional) Use chart development versions, too. Equivalent to version '>0.0.0-0'. If version is set, this is ignored.
* `version` - (Optional) Specify the exact chart version to install. If this is not specified, the latest version is installed.
* `namespace` - (Optional) The namespace to install the release into. Defaults to `default`
* `verify` - (Optional) Verify the package before installing it. Defaults to `false`
* `keyring` - (Optional) Location of public keys used for verification. Used only if `verify` is true. Defaults to `/.gnupg/pubring.gpg` in the location set by `home`
* `timeout` - (Optional) Time in seconds to wait for any individual kubernetes operation (like Jobs for hooks). Defaults to `300` seconds.
* `disable_webhooks` - (Optional) Prevent hooks from running. Defaults to `false`
* `reuse_values` - (Optional) When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored. Defaults to `false`.
* `reset_values` - (Optional) When upgrading, reset the values to the ones built into the chart. Defaults to `false`.
* `force_update` - (Optional) Force resource update through delete/recreate if needed. Defaults to `false`.
* `recreate_pods` - (Optional) Perform pods restart during upgrade/rollback. Defaults to `false`.
* `cleanup_on_fail` - (Optional) Allow deletion of new resources created in this upgrade when upgrade fails. Defaults to `false`.
* `max_history` - (Optional) Maximum number of release versions stored per release. Defaults to `0` (no limit).
* `atomic` - (Optional) If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used. Defaults to `false`.
* `skip_crds` - (Optional) If set, no CRDs will be installed. By default, CRDs are installed if not already present. Defaults to `false`.
* `render_subchart_notes` - (Optional) If set, render subchart notes along with the parent. Defaults to `true`.
* `disable_openapi_validation` - (Optional) If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema. Defaults to `false`.
* `wait` - (Optional) Will wait until all resources are in a ready state before marking the release as successful. It will wait for as long as `timeout`. Defaults to `true`.
* `values` - (Optional) List of values in raw yaml to pass to helm. Values will be merged, in order, as Helm does with multiple `-f` options.
* `set` - (Optional) Value block with custom values to be merged with the values yaml.
* `set_sensitive` - (Optional) Value block with custom sensitive values to be merged with the values yaml that won't be exposed in the plan's diff.
* `set_string` - (Optional) Value block with custom STRING values to be merged with the values yaml.
* `dependency_update` - (Optional) Runs helm dependency update before installing the chart. Defaults to `false`.
* `replace` - (Optional) Re-use the given name, even if that name is already used. This is unsafe in production. Defaults to `false`.
* `description` - (Optional) Set release description attribute (visible in the history).
* `postrender` - (Optional) Configure a command to run after helm renders the manifest which can alter the manifest contents.
* `lint` - (Optional) Run the helm chart linter during the plan. Defaults to `false`.
* `create_namespace` - (Optional) Create the namespace if it does not yet exist. Defaults to `false`.
* `upgrade` - (Optional) Enable "upgrade mode" -- the structure of this map is documented below.

The `set` and `set_sensitive` blocks support:

* `name` - (Required) full name of the variable to be set.
* `value` - (Required) value of the variable to be set.
* `type` - (Optional) type of the variable to be set. Valid options are `auto` and `string`.

The `set_strings` block supports:

* `name` - (Required) full name of the variable to be set.
* `value` - (Required) value of the variable to be set.

The `postrender` block supports a single attribute:

* `binary_path` - (Required) relative or full path to command binary.

The `upgrade` block supports:

* `enable`  - (Required) if set to `true`, use the "upgrade" strategy to install the chart. See [upgrade mode](#upgrade_mode) below for details.
* `install` - (Optional) if set to `true`, install the release even if there is no existing release to upgrade. 

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
* `values` - The compounded values from `values` and `set*` attributes.

## Upgrade Mode

When using the Helm CLI directly, it is possible (and fairly common) to use `helm upgrade --install` to
_idempotently_ install a release.  For example, `helm upgrade --install mariadb charts/mariadb --verson 7.1.0`
will check to see if there is already a release called `mariadb`: if there is, ensure that it is set to version
7.1.0, and if there is not, install that version from scratch. (See the documentation for the
[helm upgrade](https://helm.sh/docs/helm/helm_upgrade) command for more details.)

Emulating this behavior in the `helm_release` resource might be desirable if, for example, the initial installation
of a chart is handled out-of-band by a CI/CD system and you want to subsequently add the release to terraform without
having to manually import the release into terraform state each time. But the mechanics of this approach are subtly
different from the defaults and you can easily produce unexpected or undesirable results if you are not careful:
using this approach in production is not necessarily recommended!

If upgrade mode is enabled by setting `enable` to `true` in the `upgrade` map, the provider will first check to see
if a release with the given name already exists.  If that release exists, it will attempt to upgrade the release to
the state defined in the resource, using the same strategy as the [helm upgrade](https://helm.sh/docs/helm/helm_upgrade)
command.  In this case, the `generate_name`, `name_template` and `replace` attributes of the resource (if set) are
ignored, as those attributes are not supported by helm's "upgrade" behavior.

If the release does _not_ exist, the behavior is controlled by the setting of the `install` attribute.  If `install`
is `false` or unset, the apply stage will fail: the provider cannot upgrade a non-existent release.  If `install`
is set to `true`, the provider will perform a from-scratch installation of the chart.  In this case, all resource
attributes are honored.

## Import

A Helm Release resource can be imported using its namespace and name e.g.

```
$ terraform import helm_release.example default/example-name
```

~> **NOTE:** Since the `repository` attribute is not being persisted as metadata by helm, it will not be set to any value by default. All other provider specific attributes will be set to their default values and they can be overriden after running `apply` using the resource definition configuration.
