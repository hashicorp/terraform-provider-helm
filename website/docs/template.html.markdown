---
layout: "helm"
page_title: "helm: helm_template"
sidebar_current: "docs-helm-template"
description: |-

---

# Data Source: helm_template

Render chart templates locally.

`helm_template` renders chart templates locally and exposes the rendered manifests in the data source attributes. `helm_template` mimics the functionality of the `helm template` command. This data source does not require Tiller. However, any values that would normally be looked up or retrieved in-cluster will be faked locally. Additionally, none of the server-side testing of chart validity (e.g. whether an API is supported) is done.

For further details on the `helm template` command, refer to the [Helm documentation](https://helm.sh/docs/helm/#helm-template).

## Example Usage

### Render all chart templates

The following example renders all templates of the `mariadb` chart of the official Helm stable repository. Concatenated manifests are exposed as output variable `mariadb_instance_manifest`.

```hcl
data "helm_repository" "stable" {
  name = "stable"
  url  = "https://kubernetes-charts.storage.googleapis.com"
}

data "helm_template" "mariadb_instance" {
  name       = "mariadb-instance"
  namespace  = "default"
  repository = data.helm_repository.stable.metadata.0.name

  chart   = "mariadb"
  version = "6.8.2"

  set {
    name  = "service.port"
    value = "13306"
  }

  set_sensitive {
    name = "rootUser.password"
    value = "s3cr3t!"
  }
}

output "mariadb_instance_manifest" {
  value = data.helm_template.mariadb_instance.rendered
}

output "mariadb_instance_manifests" {
  value = data.helm_template.mariadb_instance.manifests
}
```

### Render explicit chart templates

The following example renders only the templates `master-statefulset.yaml` and `master-svc.yaml` of the `mariadb` chart of the official Helm stable repository.

```hcl
data "helm_repository" "stable" {
  name = "stable"
  url  = "https://kubernetes-charts.storage.googleapis.com"
}

data "helm_template" "mariadb_instance" {
  name       = "mariadb-instance"
  namespace  = "default"
  repository = data.helm_repository.stable.metadata.0.name

  chart   = "mariadb"
  version = "6.8.2"

  templates = [
    "templates/master-statefulset.yaml",
    "templates/master-svc.yaml",
  ]
}

output "mariadb_instance_manifest" {
  value = data.helm_template.mariadb_instance.rendered
}

output "mariadb_instance_manifests" {
  value = data.helm_template.mariadb_instance.manifests
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Release name.
* `repository` - (Optional) Repository where to locate the requested chart. If the location is an URL the chart is rendered without installing the repository.
* `chart` - (Required) Chart name to be rendered.
* `devel` - (Optional) Use chart development versions, too. Equivalent to version '>0.0.0-0'. If `version` is set, this is ignored.
* `version` - (Optional) Specify the exact chart version to render. If this is not specified, the latest version is rendered.
* `templates` - (Optional) Explicit list of chart templates to render, as Helm does with the `-x` option. Paths to chart templates are relative to the root folder of the chart, e.g. `templates/deployment.yaml`. If not provided, all templates of the chart are rendered.
* `values` - (Optional) List of values in raw yaml to be used when rendering the chart templates. Values will be merged, in order, as Helm does with multiple `-f` options.
* `set` - (Optional) Value block with custom values to be merged with the values yaml.
* `set_sensitive` - (Optional) Value block with custom sensitive values to be merged with the values yaml that won't be exposed in the plan's diff.
* `set_string` - (Optional) Value block with custom STRING values to be merged with the values yaml.
* `namespace` - (Optional) Namespace of the rendered resources.
* `kube_version` - (Optional) Kubernetes version used as Capabilities.KubeVersion.Major/Minor. Defaults to 1.9 as defined in the Helm package.
* `verify` - (Optional) Verify the package before rendering it.
* `keyring` - (Optional) Location of public keys used for verification.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `rendered` - Concatenated rendered chart templates. This corresponds to the output of the `helm template` command.
* `notes` - Rendered notes if the chart contains a `NOTES.txt`.
* `manifests` - Map of rendered chart templates indexed by the template name.
