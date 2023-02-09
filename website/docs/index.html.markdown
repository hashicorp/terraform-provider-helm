---
layout: "helm"
page_title: "Provider: Helm"
sidebar_current: "docs-helm-index"
description: |-
  The Helm provider is used to deploy software packages in Kubernetes. The provider needs to be configured with the proper credentials before it can be used.
---

# Helm Provider

The Helm provider is used to deploy software packages in Kubernetes. The provider needs to be configured with the proper credentials before it can be used.

Try the [hands-on tutorial](https://learn.hashicorp.com/tutorials/terraform/helm-provider?in=terraform/kubernetes) on the Helm provider on the HashiCorp Learn site.

## Resources

* [Resource: helm_release](r/release.html)

## Data Sources

* [Data Source: helm_template](d/template.html)

## Example Usage

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

  # private registry
  registry {
    url = "oci://private.registry"
    username = "username"
    password = "password"
  }
}

resource "helm_release" "nginx_ingress" {
  name       = "nginx-ingress-controller"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx-ingress-controller"

  set {
    name  = "service.type"
    value = "ClusterIP"
  }
}
```

## Requirements

~> NOTE: The provider does not use the `KUBECONFIG` environment variable by default. See the attribute reference below for the environment variables that map to provider block attributes.

- You must have a Kubernetes cluster available. We support version 1.14.0 or higher.

## Authentication

The Helm provider can get its configuration in two ways:

1. _Explicitly_ by supplying attributes to the provider block. This includes:
   * [Using a kubeconfig file](#file-config)
   * [Supplying credentials](#credentials-config)
   * [Exec plugins](#exec-plugins)
2. _Implicitly_ through environment variables. This includes:
   * [Using the in-cluster config](#in-cluster-config)

For a full list of supported provider authentication arguments and their corresponding environment variables, see the [argument reference](#argument-reference) below.


### File config

The easiest way is to supply a path to your kubeconfig file using the `config_path` attribute or using the `KUBE_CONFIG_PATH` environment variable. A kubeconfig file may have multiple contexts. If `config_context` is not specified, the provider will use the `default` context.

```hcl
provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }
}
```

The provider also supports multiple paths in the same way that kubectl does using the `config_paths` attribute or `KUBE_CONFIG_PATHS` environment variable.

```hcl
provider "helm" {
  kubernetes {
    config_paths = [
      "/path/to/config_a.yaml",
      "/path/to/config_b.yaml"
    ]
  }
}
```

### Credentials config

You can also configure the host, basic auth credentials, and client certificate authentication explicitly or through environment variables.

```hcl
provider "helm" {
  kubernetes {
    host     = "https://cluster_endpoint:port"

    client_certificate     = file("~/.kube/client-cert.pem")
    client_key             = file("~/.kube/client-key.pem")
    cluster_ca_certificate = file("~/.kube/cluster-ca-cert.pem")
  }
}
```

### In-cluster Config

The provider uses the `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` environment variables to detect when it is running inside a cluster, so in this case you do not need to specify any attributes in the provider block if you want to connect to the local kubernetes cluster.

If you want to connect to a different cluster than the one terraform is running inside, configure the provider as [above](#credentials-config).

## Exec plugins

Some cloud providers have short-lived authentication tokens that can expire relatively quickly. To ensure the Kubernetes provider is receiving valid credentials, an exec-based plugin can be used to fetch a new token before initializing the provider. For example, on EKS, the command `eks get-token` can be used:

```hcl
provider "helm" {
  kubernetes {
    host                   = var.cluster_endpoint
    cluster_ca_certificate = base64decode(var.cluster_ca_cert)
    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
      command     = "aws"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `debug` - (Optional) - Debug indicates whether or not Helm is running in Debug mode. Defaults to `false`.
* `plugins_path` - (Optional) The path to the plugins directory. Defaults to `HELM_PLUGINS` env if it is set, otherwise uses the default path set by helm.
* `registry_config_path` - (Optional) The path to the registry config file. Defaults to `HELM_REGISTRY_CONFIG` env if it is set, otherwise uses the default path set by helm.
* `repository_config_path` - (Optional) The path to the file containing repository names and URLs. Defaults to `HELM_REPOSITORY_CONFIG` env if it is set, otherwise uses the default path set by helm.
* `repository_cache` - (Optional) The path to the file containing cached repository indexes. Defaults to `HELM_REPOSITORY_CACHE` env if it is set, otherwise uses the default path set by helm.
* `helm_driver` - (Optional) "The backend storage driver. Valid values are: `configmap`, `secret`, `memory`, `sql`. Defaults to `secret`.
  Note: Regarding the sql driver, as of helm v3.2.0 SQL support exists only for the postgres dialect. The connection string can be configured by setting the `HELM_DRIVER_SQL_CONNECTION_STRING` environment variable e.g. `HELM_DRIVER_SQL_CONNECTION_STRING=postgres://username:password@host/dbname` more info [here](https://pkg.go.dev/github.com/lib/pq).
* `burst_limit` - (Optional) The helm burst limit to use. Set this value higher if your cluster has many CRDs. Default: `100`
* `kubernetes` - Kubernetes configuration block.
* `registry` - Private OCI registry configuration block. Can be specified multiple times.

The `kubernetes` block supports:

* `config_path` - (Optional) Path to the kube config file. Can be sourced from `KUBE_CONFIG_PATH`.
* `config_paths` - (Optional) A list of paths to the kube config files. Can be sourced from `KUBE_CONFIG_PATHS`.
* `host` - (Optional) The hostname (in form of URI) of the Kubernetes API. Can be sourced from `KUBE_HOST`.
* `username` - (Optional) The username to use for HTTP basic authentication when accessing the Kubernetes API. Can be sourced from `KUBE_USER`.
* `password` - (Optional) The password to use for HTTP basic authentication when accessing the Kubernetes API. Can be sourced from `KUBE_PASSWORD`.
* `token` - (Optional) The bearer token to use for authentication when accessing the Kubernetes API. Can be sourced from `KUBE_TOKEN`.
* `insecure` - (Optional) Whether server should be accessed without verifying the TLS certificate. Can be sourced from `KUBE_INSECURE`.
* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication. Can be sourced from `KUBE_CLIENT_CERT_DATA`.
* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication. Can be sourced from `KUBE_CLIENT_KEY_DATA`.
* `cluster_ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication. Can be sourced from `KUBE_CLUSTER_CA_CERT_DATA`.
* `config_context` - (Optional) Context to choose from the config file. Can be sourced from `KUBE_CTX`.
* `proxy_url` - (Optional) URL to the proxy to be used for all API requests. URLs with "http", "https", and "socks5" schemes are supported. Can be sourced from `KUBE_PROXY_URL`.
* `exec` - (Optional) Configuration block to use an [exec-based credential plugin](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins), e.g. call an external command to receive user credentials.
  * `api_version` - (Required) API version to use when decoding the ExecCredentials resource, e.g. `client.authentication.k8s.io/v1beta1`.
  * `command` - (Required) Command to execute.
  * `args` - (Optional) List of arguments to pass when executing the plugin.
  * `env` - (Optional) Map of environment variables to set when executing the plugin.

The `registry` block has options:

* `url` - (Required) url to the registry in format `oci://host:port`
* `username` - (Required) username to registry
* `password` - (Required) password to registry

## Experiments

The provider takes an `experiments` block that allows you enable experimental features by setting them to `true`.

* `manifest` - Enable storing of the rendered manifest for `helm_release` so the full diff of what is changing can been seen in the plan.
