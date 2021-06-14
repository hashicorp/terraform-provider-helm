---
layout: "helm"
page_title: "helm: helm_template"
sidebar_current: "docs-helm-template"
description: |-

---

# Data Source: helm_template

Render chart templates locally.

`helm_template` renders chart templates locally and exposes the rendered manifests in the data source attributes. `helm_template` mimics the functionality of the `helm template` command.

The arguments aim to be identical to the `helm_release` resource.

For further details on the `helm template` command, refer to the [Helm documentation](https://helm.sh/docs/helm/helm_template/).

## Example Usage

### Render all chart templates

The following example renders all templates of the `mariadb` chart of the official Helm stable repository. Concatenated manifests are exposed as output variable `mariadb_instance_manifest_bundle`.

```hcl
data "helm_template" "mariadb_instance" {
  name       = "mariadb-instance"
  namespace  = "default"
  repository = "https://charts.helm.sh/stable"

  chart   = "mariadb"
  version = "7.1.0"

  set {
    name  = "service.port"
    value = "13306"
  }

  set_sensitive {
    name = "rootUser.password"
    value = "s3cr3t!"
  }
}

resource "local_file" "mariadb_manifests" {
  for_each = data.helm_template.mariadb_instance.manifests

  filename = "./${each.key}"
  content  = each.value
}

output "mariadb_instance_manifest_bundle" {
  value = data.helm_template.mariadb_instance.manifest_bundle
}

output "mariadb_instance_manifests" {
  value = data.helm_template.mariadb_instance.manifests
}

output "mariadb_instance_notes" {
  value = data.helm_template.mariadb_instance.notes
}
```

### Render selected chart templates

The following example renders only the templates `master-statefulset.yaml` and `master-svc.yaml` of the `mariadb` chart of the official Helm stable repository.

```hcl
data "helm_template" "mariadb_instance" {
  name       = "mariadb-instance"
  namespace  = "default"
  repository = "https://charts.helm.sh/stable"

  chart   = "mariadb"
  version = "7.1.0"

  templates = [
    "templates/master-statefulset.yaml",
    "templates/master-svc.yaml",
  ]
  
  set {
    name  = "service.port"
    value = "13306"
  }

  set_sensitive {
    name = "rootUser.password"
    value = "s3cr3t!"
  }
}

resource "local_file" "mariadb_manifests" {
  for_each = data.helm_template.mariadb_instance.manifests

  filename = "./${each.key}"
  content  = each.value
}

output "mariadb_instance_manifest_bundle" {
  value = data.helm_template.mariadb_instance.manifest_bundle
}

output "mariadb_instance_manifests" {
  value = data.helm_template.mariadb_instance.manifests
}

output "mariadb_instance_notes" {
  value = data.helm_template.mariadb_instance.notes
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Release name.
* `chart` - (Required) Chart name to be rendered. The chart name can be local path, a URL to a chart, or the name of the chart if `repository` is specified. It is also possible to use the `<repository>/<chart>` format here if you are running Terraform on a system that the repository has been added to with `helm repo add` but this is not recommended.
* `repository` - (Optional) Repository URL where to locate the requested chart.
* `repository_key_file` - (Optional) The repositories cert key file
* `repository_cert_file` - (Optional) The repositories cert file
* `repository_ca_file` - (Optional) The Repositories CA File.
* `repository_username` - (Optional) Username for HTTP basic authentication against the repository.
* `repository_password` - (Optional) Password for HTTP basic authentication against the repository.
* `devel` - (Optional) Use chart development versions, too. Equivalent to version '>0.0.0-0'. If version is set, this is ignored.
* `version` - (Optional) Specify the exact chart version to install. If this is not specified, the latest version is installed.
* `namespace` - (Optional) The namespace to install the release into. Defaults to `default`.
* `verify` - (Optional) Verify the package before installing it. Helm uses a provenance file to verify the integrity of the chart; this must be hosted alongside the chart. For more information see the [Helm Documentation](https://helm.sh/docs/topics/provenance/). Defaults to `false`.
* `keyring` - (Optional) Location of public keys used for verification. Used only if `verify` is true. Defaults to `/.gnupg/pubring.gpg` in the location set by `home`
* `timeout` - (Optional) Time in seconds to wait for any individual kubernetes operation (like Jobs for hooks). Defaults to `300` seconds.
* `disable_webhooks` - (Optional) Prevent hooks from running. Defaults to `false`.
* `reuse_values` - (Optional) When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored. Defaults to `false`.
* `reset_values` - (Optional) When upgrading, reset the values to the ones built into the chart. Defaults to `false`.
* `atomic` - (Optional) If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used. Defaults to `false`.
* `skip_crds` - (Optional) If set, no CRDs will be installed. By default, CRDs are installed if not already present. Defaults to `false`.
* `skip_tests` - (Optional) If set, tests will not be rendered. By default, tests are rendered. Defaults to `false`.
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
* `create_namespace` - (Optional) Create the namespace if it does not yet exist. Defaults to `false`.

The following attributes are specific to the `helm_template` data source and not available in the `helm_release` resource:

* `api_versions` - (Optional) List of Kubernetes api versions used for Capabilities.APIVersions.
* `include_crds` - (Optional) Include CRDs in the templated output. Defaults to `false`.
* `is_upgrade` - (Optional) Set .Release.IsUpgrade instead of .Release.IsInstall. Defaults to `false`.
* `show_only` - (Optional) Explicit list of chart templates to render, as Helm does with the `-s` or `--show-only` option. Paths to chart templates are relative to the root folder of the chart, e.g. `templates/deployment.yaml`. If not provided, all templates of the chart are rendered.
* `validate` - (Optional) Validate your manifests against the Kubernetes cluster you are currently pointing at. This is the same validation performed on an install. Defaults to `false`.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are exported:

* `manifests` - Map of rendered chart templates indexed by the template name.
* `manifest` - Concatenated rendered chart templates. This corresponds to the output of the `helm template` command.
* `notes` - Rendered notes if the chart contains a `NOTES.txt`.
