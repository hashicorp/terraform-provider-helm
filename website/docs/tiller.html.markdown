---
layout: "helm"
page_title: "helm: helm_tiller"
sidebar_current: "docs-helm-tiller"
description: |-

---

# Resource: helm_tiller

Tiller is the in-cluster component of Helm.

`helm_tiller` describes the desired status of a Tiller installation.

## Example Usage

```hcl
resource "helm_tiller" "install" {
  namespace       = "tiller"
  service_account = "default"
  tiller_image    = "gcr.io/kubernetes-helm/tiller:v2.11.0"
}
```

## Argument Reference

The following arguments are supported:

* `namespace` - (Optional) Set an alternative Tiller namespace. Defaults to `kube-system`.
* `tiller_image` - (Optional) Tiller image to install. Defaults to `gcr.io/kubernetes-helm/tiller:v2.11.0`.
* `service_account` - (Optional) Service account to install Tiller with. Defaults to `default`.
* `automount_service_account_token` - (Optional) Auto-mount the given service account to tiller. Defaults to `true`.
* `override` - (Optional) Override values for the Tiller Deployment manifest. Defaults to `true`.
* `max_history` - (Optional) Maximum number of release versions stored per release. Defaults to `0` (no limit).
* `verify_tls` - (Optional) Whether server should be accessed without verifying the TLS certificate. Defaults to `false`.
* `enable_tls` - (Optional) Enables TLS communications with the Tiller. Defaults to `false`.
* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication. By default read from `$HELM_HOME/key.pem`.
* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication. By default read from `$HELM_HOME/cert.pem`.
* `ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication. By default read from `$HELM_HOME/ca.pem`.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `metadata` - Status of the deployed Tiller.

The `metadata` block supports:

* `name` - Name of the Tiller Deployment.
* `namespace` - Namespace of the Tiller Deployment.
* `generation` - The generation number of the Tiller Deployment.
* `resource_version` - The resource version of the Tiller Deployment.
* `self_link` - The self link URL of the Tiller Deployment.
* `uid` - The unique ID of the Tiller Deployment.
