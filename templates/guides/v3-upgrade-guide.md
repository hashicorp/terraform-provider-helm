---
layout: "helm"
page_title: "Helm: Upgrade Guide for Helm Provider v3.0.0"
description: |-
  This guide covers the changes introduced in v3.0.0 of the Helm provider and what you may need to do to upgrade your configuration.
---

# Upgrading to v3.0.0 of the Helm provider


This guide covers the changes introduced in v3.0.0 of the Helm provider and what you may need to do to upgrade your configuration.

## Changes in v3.0.0

### Adoption of the Terraform Plugin Framework

The Helm provider has been migrated from the legacy [Terraform Plugin SDKv2](https://github.com/hashicorp/terraform-plugin-sdk) to the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). This migration introduces structural changes to the schema, affecting nested blocks, attribute names, and how configurations are represented. Users must update their configurations to align with the new framework. Key changes include:

- **Blocks to Nested Objects**: Blocks like `kubernetes`, `registry`, and `experiments` are now represented as nested objects.
- **List Syntax for Nested Attributes**: Attributes like `set`, `set_list`, and `set_sensitive` in `helm_release` and `helm_template` are now lists of nested objects instead of blocks.

### Terraform Version Compatability

The new framework code uses [Terraform Plugin Protocol Version 6](https://developer.hashicorp.com/terraform/plugin/terraform-plugin-protocol#protocol-version-6) which is compatible with Terraform versions 1.0 and aboove. Users of earlier versions of Terraform can continue to use the Helm provider by pinning their configuration to the 2.x version.

---

### Changes to Provider Attributes

#### Kubernetes Configuration (`kubernetes`)

The `kubernetes` block has been updated to a single nested object.

**Old SDKv2 Configuration:**

```hcl
provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }

  registry {
    url      = "oci://localhost:5000"
    username = "username"
    password = "password"
  }

  registry {
    url      = "oci://private.registry"
    username = "username"
    password = "password"
  }
}
```

**New Plugin Framework Configuration:**

```hcl
provider "helm" {
  kubernetes = {
    config_path = "~/.kube/config"
  }

  registries = [
    {
      url      = "oci://localhost:5000"
      username = "username"
      password = "password"
    },
    {
      url      = "oci://private.registry"
      username = "username"
      password = "password"
    }
  ]
}
```

**What Changed?**

- `kubernetes` is now a single nested object attribute using `{ ... }`.
- `registry` blocks have been replaced by a `registries` list attribute.

#### Experiments Configuration (experiments)

The `experiments` block has been updated to a list of nested objects.

**Old SDKv2 Configuration:**

```hcl
provider "helm" {
  experiments {
    manifest = true
  }
}
```

**New Plugin Framework Configuration:**

```hcl
provider "helm" {
  experiments = {
    manifest = true
  }
}
```

**What Changed?**

- `experiments` is now a single nested object attribute using `{ ... }`.

### Changes to helm_release Resource

#### `set`, `set_list`, and `set_sensitive` Configuration

Attributes  `set`, `set_list`, and `set_sensitive` are now represented as lists of nested objects instead of individual blocks.

**Old SDKv2 Configuration:**

```hcl
resource "helm_release" "nginx_ingress" {
  name       = "nginx-ingress-controller"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx-ingress-controller"

  set {
    name  = "service.type"
    value = "ClusterIP"
  }

  set_list {
    name  = "allowed.hosts"
    value = ["host1", "host2"]
  }

  set_sensitive {
    name  = "api.key"
    value = "super-secret-key"
  }
}
```

**New Plugin Framework Configuration:**

```hcl
resource "helm_release" "nginx_ingress" {
  name       = "nginx-ingress-controller"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx-ingress-controller"

  set = [
    {
      name  = "service.type"
      value = "ClusterIP"
    }
  ]

  set_list = [
    {
      name  = "allowed.hosts"
      value = ["host1", "host2"]
    }
  ]

  set_sensitive = [
    {
      name  = "api.key"
      value = "super-secret-key"
    }
  ]
}
```

**What Changed?**

- `set`, `set_list`, and `set_sensitive` is now a list of nested objects using `[ { ... } ]`.

### Changes to helm_template Data Source

#### `set`, `set_list`, and `set_sensitive` Configuration

Attributes  `set`, `set_list`, and `set_sensitive` are now represented as lists of nested objects instead of individual blocks.

**Old SDKv2 Configuration:**

```hcl
data "helm_template" "example" {
  name       = "my-release"
  chart      = "my-chart"
  namespace  = "my-namespace"
  values     = ["custom-values.yaml"]

  set {
    name  = "image.tag"
    value = "1.2.3"
  }

  set_list {
    name  = "allowed.hosts"
    value = ["host1", "host2"]
  }

  set_sensitive {
    name  = "api.key"
    value = "super-secret-key"
  }
}
```

**New Plugin Framework Configuration:**

```hcl
data "helm_template" "example" {
  name       = "my-release"
  chart      = "my-chart"
  namespace  = "my-namespace"
  values     = ["custom-values.yaml"]

  set = [
    {
      name  = "image.tag"
      value = "1.2.3"
    }
  ]

  set_list = [
    {
      name  = "allowed.hosts"
      value = ["host1", "host2"]
    }
  ]

  set_sensitive = [
    {
      name  = "api.key"
      value = "super-secret-key"
    }
  ]
}
```

**What Changed?**

- `set`, `set_list`, and `set_sensitive` is now a list of nested objects using `[ { ... } ]`.

## Troubleshooting

### Error: Blocks of type "kubernetes" are not expected here

After upgrading to v3.0.0 you may see an error similar to:

```text
Error: Unsupported block type

  on main.tf line 2, in provider "helm":
   2:   kubernetes {

Blocks of type "kubernetes" are not expected here. Did you mean to define
argument "kubernetes"? If so, use the equals sign to assign it a value.
```

This happens because v3.0.0 migrated the provider from the [Terraform Plugin SDKv2](https://github.com/hashicorp/terraform-plugin-sdk) to the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). Configuration that previously used **blocks** (no equals sign) must now be written as **nested attributes** (with an equals sign). The same applies to `registry` (now `registries`), `experiments`, and the `set`, `set_list`, and `set_sensitive` arguments in `helm_release` and `helm_template`.

Update your configuration as follows:

| v2 (block syntax)              | v3 (attribute syntax)          |
| ------------------------------ | ------------------------------ |
| `kubernetes { ... }`           | `kubernetes = { ... }`         |
| `registry { ... }` (repeatable) | `registries = [ { ... } ]`    |
| `experiments { ... }`          | `experiments = { ... }`        |
| `set { ... }`                  | `set = [ { ... } ]`            |
| `set_list { ... }`             | `set_list = [ { ... } ]`       |
| `set_sensitive { ... }`        | `set_sensitive = [ { ... } ]`  |

See [Changes to Provider Attributes](#changes-to-provider-attributes) above for complete before/after examples.

If you are not ready to migrate, you can pin the provider to the latest v2 release until your configuration has been updated:

```hcl
terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.17"
    }
  }
}
```

### Error: could not login to OCI registry (The specified item already exists in the keychain)

On macOS you may encounter an error similar to:

```text
Error: could not login to OCI registry "registry.example.com": error storing
credentials - err: exit status 1, out: `The specified item already exists in
the keychain.`
```

This is caused by an interaction between Helm's OCI registry login and the macOS `docker-credential-osxkeychain` credential helper, which fails when a stale or conflicting entry already exists in the login keychain.

The recommended workaround is to configure the registry credentials directly in the provider block using the `registries` attribute, so Helm authenticates with the supplied username and password:

```hcl
provider "helm" {
  registries = [
    {
      url      = "oci://registry.example.com"
      username = "username"
      password = "password"
    }
  ]
}
```

If the error persists, remove the stale keychain entry (replace the server with your registry host) and re-run Terraform:

```sh
security delete-internet-password -s registry.example.com
```

Alternatively, remove the `credsStore` or `credHelpers` entry for the registry from your Docker configuration (`~/.docker/config.json`) so credentials are not written to the macOS keychain.
