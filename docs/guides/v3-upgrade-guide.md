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
