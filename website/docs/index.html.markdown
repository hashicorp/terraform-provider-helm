---
layout: "helm"
page_title: "Provider: Helm"
sidebar_current: "docs-helm-index"
description: |-
  The Helm provider is used to deploy software packages in Kubernetes. The provider needs to be configured with the proper credentials before it can be used.
---

# Helm Provider

The Helm provider is used to deploy software packages in Kubernetes. The provider needs to be configured with the proper credentials before it can be used.

## Data Sources

* [Data Sources: helm_repository](d/repository.html)

## Resources

* [Resource: helm_release](r/release.html)

## Example Usage

```hcl
resource "helm_release" "mydatabase" {
  name  = "mydatabase"
  chart = "stable/mariadb"

  set {
    name  = "mariadbUser"
    value = "foo"
  }

  set {
    name  = "mariadbPassword"
    value = "qux"
  }
}
```

## Requirements

- You must have Kubernetes installed. We recommend version 1.4.1 or later.
- You should also have a local configured copy of kubectl.

## Authentication

There are generally two ways to configure the Helm provider.

### File config

The provider always first tries to load **a config file** (usually `$HOME/.kube/config`), for access kubernetes and reads all the Helm files from home (usually `$HOME/.helm`). You can also define that file with the following setting:

```hcl
provider "helm" {
  kubernetes {
    config_path = "/path/to/kube_cluster.yaml"
  }
}
```

### Statically defined credentials

The other way is **statically** define all the credentials:

```hcl
provider "helm" {
  kubernetes {
    host     = "https://104.196.242.174"
    username = "ClusterMaster"
    password = "MindTheGap"

    client_certificate     = file("~/.kube/client-cert.pem")
    client_key             = file("~/.kube/client-key.pem")
    cluster_ca_certificate = file("~/.kube/cluster-ca-cert.pem")
  }
}
```

If you have **both** valid configuration in a config file and static configuration, the static one is used as override.
i.e. any static field will override its counterpart loaded from the config.

## Argument Reference

"": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "",
			},

The following arguments are supported:
			
* `home` - (Required) Set an alternative location for Helm files. By default, these are stored in `$HOME/.helm`. Can be sourced from `HELM_HOME` environment variable.
* `name` - (Required) Release name.
* `chart` - (Required) Chart name to be installed.
* `kubernetes` - (Required) Kubernetes configuration block.

* `debug` - (Optional) - Debug indicates whether or not Helm is running in Debug mode. Defaults to `false`.
* `repository_key_file` - (Optional) The repositories cert key file
* `repository_cert_file` - (Optional) The repositories cert file
* `repository_ca_file` - (Optional) The Repositories CA File. 
* `repository_username` - (Optional) Username for HTTP basic authentication against the repository.
* `repository_password` - (Optional) Password for HTTP basic authentication against the reposotory.
 * `repository` - (Optional) Repository where to locate the requested chart. If is an URL the chart is installed without install the repository.
* `devel` - (Optional) Use chart development versions, too. Equivalent to version '>0.0.0-0'. If version is set, this is ignored.
* `version` - (Optional) Specify the exact chart version to install. If this is not specified, the latest version is installed.
* `namespace` - (Optional) The namespace to install the release into. Defaults to `default`
* `verify` - (Optional) Verify the package before installing it. Defaults to `false`
* `keyring` - (Optional) Location of public keys used for verification. Used only if `verify` is true. Defaults to `/.gnupg/pubring.gpg` in the location set by `home`
* `timeout` - (Optional) Time in seconds to wait for any individual kubernetes operation. Defaults to `300` seconds.
* `disable_webhooks` - (Optional) Prevent hooks from running. Defauts to `false`
* `reuse_values` - (Optional) When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored. Defaults to `false`.
* `reset_values` - (Optional) When upgrading, reset the values to the ones built into the chart. Defaults to `false`.
* `force_update` - (Optional) Force resource update through delete/recreate if needed. Defaults to `false`.
* `recreate_pods` - (Optional) Perform pods restart during upgrade/rollback. Defaults to `false`.
* `cleanup_on_fail` - (Optional) Allow deletion of new resources created in this upgrade when upgrade fails. Defaults to `false`.
* `max_history` - (Optional) Maximum number of release versions stored per release. Defaults to `0` (no limit).
* `atomic` - (Optional) If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used. Defaults to `false`.
* `skip_crds` - (Optional) If set, no CRDs will be installed. By default, CRDs are installed if not already present. Defaults to `false`.
* `render_subchart_notes` - (Optional) If set, render subchart notes along with the parent. Defaults to `true`.
* `wait` - (Optional) Will wait until all resources are in a ready state before marking the release as successful. Defaults to `true`.

The `kubernetes` block supports:

* `config_path` - (Optional) Path to the kube config file, defaults to `~/.kube/config`. Can be sourced from `KUBE_CONFIG` or `KUBECONFIG`..
* `host` - (Optional) The hostname (in form of URI) of Kubernetes master. Can be sourced from `KUBE_HOST`.
* `username` - (Optional) The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_USER`.
* `password` - (Optional) The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_PASSWORD`.
* `token` - (Optional) The bearer token to use for authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_BEARER_TOKEN`.
* `insecure` - (Optional) Whether server should be accessed without verifying the TLS certificate. Can be sourced from `KUBE_INSECURE`.
* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication. Can be sourced from `KUBE_CLIENT_CERT_DATA`.
* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication. Can be sourced from `KUBE_CLIENT_KEY_DATA`.
* `cluster_ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication. Can be sourced from `KUBE_CLUSTER_CA_CERT_DATA`.
* `config_context` - (Optional) Context to choose from the config file. Can be sourced from `KUBE_CTX`.
* `load_config_file` - (Optional) By default the local config (~/.kube/config) is loaded when you use this provider. This option at false disable this behaviour. Can be sourced from `KUBE_LOAD_CONFIG_FILE`.
