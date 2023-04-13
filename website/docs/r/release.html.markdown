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

## Example Usage - Chart Repository

```hcl
resource "helm_release" "example" {
  name       = "my-redis-release"
  repository = "https://charts.bitnami.com/bitnami"
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

  set {
    name  = "service.annotations.prometheus\\.io/port"
    value = "9127"
    type  = "string"
  }
}
```

## Example Usage - Local Chart

In case a Chart is not available from a repository, a path may be used:

```hcl
resource "helm_release" "example" {
  name       = "my-local-chart"
  chart      = "./charts/example"
}
```

## Example Usage - Chart URL

An absolute URL to the .tgz of the Chart may also be used:

```hcl
resource "helm_release" "example" {
  name  = "redis"
  chart = "https://charts.bitnami.com/bitnami/redis-10.7.16.tgz"
}
```

## Example Usage - Chart Repository configured from OCI Registry

Provider supports grabbing charts from an OCI repository:

```hcl

provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }

  # localhost registry with password protection
  registry {
    url = "oci://localhost:5000"
    username = "username"
    password = "password"
  }
}

resource "helm_release" "example" {
  name        = "testchart"
  namespace   = "helm_registry"
  repository  = "oci://localhost:5000/helm-charts"
  version     = "1.2.3"
  chart       = "test-chart"
}
```

## Example Usage - Chart Repository configured using GCS/S3

The provider also supports helm plugins such as GCS and S3 that add S3/GCS helm repositories by using `helm plugin install`

```hcl

# Install GCS plugin
`helm plugin install https://github.com/hayorov/helm-gcs.git`

# Run follow commands to setup GCS repository

# Init a new repository:
#   helm gcs init gs://bucket/path

# Add your repository to Helm:
#   helm repo add repo-name gs://bucket/path

# Push a chart to your repository:
#   helm gcs push chart.tar.gz repo-name

# Update Helm cache:
#   helm repo update

# Get your chart:

resource "helm_release" "GCS" {
  name        = "GCS"
  repository  = "gcs://tf-test-helm-repo/charts"
  chart       = "chart"
}
```

```hcl

# Install AWS S3 plugin
`helm plugin install https://github.com/hypnoglow/helm-s3.git`

# Run follow commands to setup S3 repository

# Init a new repository:
#   helm s3 init s3://my-helm-charts/stable/myapp

# Add your repository to Helm:
#   helm repo add stable-myapp s3://my-helm-charts/stable/myapp/

# Push a chart to your repository:
#   helm s3 push chart.tar.gz repo-name

# Update Helm cache:
#   helm repo update

# Get your chart:

resource "helm_release" "S3" {
  name        = "S3"
  repository  = "s3://tf-test-helm-repo/charts"
  chart       = "chart"
}
```

## Example Usage - Chart Repository configured outside of Terraform

The provider also supports repositories that are added to the local machine outside of Terraform by running `helm repo add`

```hcl

# run this first: `helm repo add bitnami https://charts.bitnami.com/bitnami`

resource "helm_release" "example" {
  name  = "redis"
  chart = "bitnami/redis"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Release name.
* `chart` - (Required) Chart name to be installed. The chart name can be local path, a URL to a chart, or the name of the chart if `repository` is specified. It is also possible to use the `<repository>/<chart>` format here if you are running Terraform on a system that the repository has been added to with `helm repo add` but this is not recommended.
* `repository` - (Optional) Repository URL where to locate the requested chart.
* `repository_key_file` - (Optional) The repositories cert key file
* `repository_cert_file` - (Optional) The repositories cert file
* `repository_ca_file` - (Optional) The Repositories CA File.
* `repository_username` - (Optional) Username for HTTP basic authentication against the repository.
* `repository_password` - (Optional) Password for HTTP basic authentication against the repository.
* `devel` - (Optional) Use chart development versions, too. Equivalent to version '>0.0.0-0'. If version is set, this is ignored.
* `version` - (Optional) Specify the exact chart version to install. If this is not specified, the latest version is installed. `helm_release` will not automatically grab the latest release, version must explicitly upgraded when upgrading an installed chart.
* `namespace` - (Optional) The namespace to install the release into. Defaults to `default`.
* `verify` - (Optional) Verify the package before installing it. Helm uses a provenance file to verify the integrity of the chart; this must be hosted alongside the chart. For more information see the [Helm Documentation](https://helm.sh/docs/topics/provenance/). Defaults to `false`.
* `keyring` - (Optional) Location of public keys used for verification. Used only if `verify` is true. Defaults to `/.gnupg/pubring.gpg` in the location set by `home`
* `timeout` - (Optional) Time in seconds to wait for any individual kubernetes operation (like Jobs for hooks). Defaults to `300` seconds.
* `disable_webhooks` - (Optional) Prevent hooks from running. Defaults to `false`.
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
* `wait_for_jobs` - (Optional) If wait is enabled, will wait until all Jobs have been completed before marking the release as successful. It will wait for as long as `timeout`.  Defaults to false.

* `values` - (Optional) List of values in raw yaml to pass to helm. Values will be merged, in order, as Helm does with multiple `-f` options.
* `set` - (Optional) Value block with custom values to be merged with the values yaml.
* `set_list` - (Optional) Value block with list of custom values to be merged with the values yaml.
* `set_sensitive` - (Optional) Value block with custom sensitive values to be merged with the values yaml that won't be exposed in the plan's diff.
* `dependency_update` - (Optional) Runs helm dependency update before installing the chart. Defaults to `false`.
* `replace` - (Optional)  Re-use the given name, only if that name is a deleted release which remains in the history. This is unsafe in production. Defaults to `false`.
* `description` - (Optional) Set release description attribute (visible in the history).
* `postrender` - (Optional) Configure a command to run after helm renders the manifest which can alter the manifest contents.
* `pass_credentials` - (Optional) Pass credentials to all domains. Defaults to `false`.
* `lint` - (Optional) Run the helm chart linter during the plan. Defaults to `false`.
* `create_namespace` - (Optional) Create the namespace if it does not yet exist. Defaults to `false`.

The `set`, `set_list`, and `set_sensitive` blocks support:

* `name` - (Required) full name of the variable to be set.
* `value` - (Required) value of the variable to be set.
* `type` - (Optional) type of the variable to be set. Valid options are `auto` and `string`.

Since Terraform Utilizes HCL as well as Helm using the Helm Template Language, it's necessary to  escape certain characters twice in order for it to be parsed. `name` should also be the path that leads to the desired value, where `value` is the desired value that will be set.

```hcl
set {
  name  = "grafana.ingress.annotations\\.alb\\.ingress\\.kubernetes\\.io/group\\.name"
  value = "shared-ingress"
}
```

```hcl
set_list {
  name  = "hashicorp"
  value = ["terraform", "nomad", "vault"]
}
```

The `postrender` block supports two attributes:

* `binary_path` - (Required) relative or full path to command binary.
* `args` - (Optional) a list of arguments to supply to the post-renderer.



## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `manifest` - The rendered manifest of the release as JSON. Enable the `manifest` experiment to use this feature.
* `metadata` - Block status of the deployed release.
* `status` - Status of the release.

The `metadata` block supports:

* `chart` - The name of the chart.
* `name` - Name is the name of the release.
* `namespace` - Namespace is the kubernetes namespace of the release.
* `revision` - Version is an int32 which represents the version of the release.
* `version` - A SemVer 2 conformant version string of the chart.
* `app_version` - The version number of the application being deployed.
* `values` - The compounded values from `values` and `set*` attributes.

## Import

A Helm Release resource can be imported using its namespace and name e.g.

```shell
$ terraform import helm_release.example default/example-name
```

~> **NOTE:** Since the `repository` attribute is not being persisted as metadata by helm, it will not be set to any value by default. All other provider specific attributes will be set to their default values and they can be overriden after running `apply` using the resource definition configuration.
