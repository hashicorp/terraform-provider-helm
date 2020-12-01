---
layout: "helm"
page_title: "Provider: Helm"
sidebar_current: "docs-helm-index"
description: |-
  The Helm provider is used to deploy software packages in Kubernetes. The provider needs to be configured with the proper credentials before it can be used.
---

# Helm Provider

The Helm provider is used to deploy software packages in Kubernetes. The provider needs to be configured with the proper credentials before it can be used.

## Resources

* [Resource: helm_release](r/release.html)

## Example Usage

```hcl
provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }
}

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

- You must have a Kubernetes cluster available. We recommend version 1.11.0 or higher.
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

* `debug` - (Optional) - Debug indicates whether or not Helm is running in Debug mode. Defaults to `false`.
* `plugins_path` - (Optional) The path to the plugins directory. Defaults to `HELM_PLUGINS` env if it is set, otherwise uses the default path set by helm.
* `registry_config_path` - (Optional) The path to the registry config file. Defaults to `HELM_REGISTRY_CONFIG` env if it is set, otherwise uses the default path set by helm.
* `repository_config_path` - (Optional) The path to the file containing repository names and URLs. Defaults to `HELM_REPOSITORY_CONFIG` env if it is set, otherwise uses the default path set by helm.
* `repository_cache` - (Optional) The path to the file containing cached repository indexes. Defaults to `HELM_REPOSITORY_CACHE` env if it is set, otherwise uses the default path set by helm.
* `helm_driver` - (Optional) "The backend storage driver. Valid values are: `configmap`, `secret`, `memory`, `sql`. Defaults to `secret`.
  Note: Regarding the sql driver, as of helm v3.2.0 SQL support exists only for the postgres dialect. The connection string can be configured by setting the `HELM_DRIVER_SQL_CONNECTION_STRING` environment variable e.g. `HELM_DRIVER_SQL_CONNECTION_STRING=postgres://username:password@host/dbname` more info [here](https://pkg.go.dev/github.com/lib/pq).
* `kubernetes` - Kubernetes configuration block.

The `kubernetes` block supports:

* `config_path` - (Optional) Path to the kube config file. Can be sourced from `KUBE_CONFIG_PATH`.
* `config_paths` - (Optional) A list of paths to the kube config files. Can be sourced from `KUBE_CONFIG_PATHS`.
* `host` - (Optional) The hostname (in form of URI) of Kubernetes master. Can be sourced from `KUBE_HOST`.
* `username` - (Optional) The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_USER`.
* `password` - (Optional) The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_PASSWORD`.
* `token` - (Optional) The bearer token to use for authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_BEARER_TOKEN`.
* `insecure` - (Optional) Whether server should be accessed without verifying the TLS certificate. Can be sourced from `KUBE_INSECURE`.
* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication. Can be sourced from `KUBE_CLIENT_CERT_DATA`.
* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication. Can be sourced from `KUBE_CLIENT_KEY_DATA`.
* `cluster_ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication. Can be sourced from `KUBE_CLUSTER_CA_CERT_DATA`.
* `config_context` - (Optional) Context to choose from the config file. Can be sourced from `KUBE_CTX`.
* `exec` - (Optional) Configuration block to use an [exec-based credential plugin](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins), e.g. call an external command to receive user credentials.
  * `api_version` - (Required) API version to use when decoding the ExecCredentials resource, e.g. `client.authentication.k8s.io/v1beta1`.
  * `command` - (Required) Command to execute.
  * `args` - (Optional) List of arguments to pass when executing the plugin.
  * `env` - (Optional) Map of environment variables to set when executing the plugin.
