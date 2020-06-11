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

The following arguments are supported:

* `host` - (Required) Set an alternative Tiller host. The format is host:port. Can be sourced from `HELM_HOST` environment variable.
* `home` - (Required) Set an alternative location for Helm files. By default, these are stored in `$HOME/.helm`. Can be sourced from `HELM_HOME` environment variable.
* `namespace` - (Optional) Set an alternative Tiller namespace. Defaults to `kube-system`.
* `init_helm_home` - (Optional) Initialize Helm home directory configured by the `home` attribute if it is not already initialized, defaults to true.
* `install_tiller` - (Optional) Install Tiller if it is not already installed. Defaults to `true`.
* `tiller_image` - (Optional) Tiller image to install. Defaults to `gcr.io/kubernetes-helm/tiller:v2.16.8`.
* `connection_timeout` - (Optional) Number of seconds Helm will wait before timing out a connection to tiller. Defaults to `60`.
* `service_account` - (Optional) Service account to install Tiller with. Defaults to `default`.
* `automount_service_account_token` - (Optional) Auto-mount the given service account to tiller. Defaults to `true`.
* `override` - (Optional) Override values for the Tiller Deployment manifest. Defaults to `true`.
* `max_history` - (Optional) Maximum number of release versions stored per release. Defaults to `0` (no limit).
* `debug` - (Optional) - Debug indicates whether or not Helm is running in Debug mode. Defaults to `false`.
* `plugins_disable` - (Optional) Disable plugins. Can be sourced from `HELM_NO_PLUGINS` environment variable, set `HELM_NO_PLUGINS=0` to enable plugins. Defaults to `true`.
* `insecure` - (Optional) Whether server should be accessed without verifying the TLS certificate. Defaults to `false`.
* `enable_tls` - (Optional) Enables TLS communications with the Tiller. Defaults to `false`.
* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication. By default read from `key.pem` in the location set by `home`.
* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication. By default read from `cert.pem` in the location set by `home`.
* `ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication. By default read from `ca.pem` in the location set by `home`.
* `kubernetes` - Kubernetes configuration block.

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
