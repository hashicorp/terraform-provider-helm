---
page_title: "helm: helm_release"
sidebar_current: "docs-helm-release"
description: |-

---
# List Resource: helm_release

Lists Helm Releases.

## Example Usage – List All Resources

```terraform
list "helm_release" "all" {
  provider = helm

  config {
    all_namespaces = true
  }
}
```

## Example Usage – List All Resources in a specific namespace

```terraform
list "helm_release" "kube_system" {
  provider = helm

  config {
    namespace = "kube-system"
  }
}
```

## Example Usage – Filter by name

```terraform
list "helm_release" "kube_system" {
  provider = helm

  config {
    filter = "test"
  }
}
```

## Argument Reference

This list resource supports the following arguments:

* `namespace` - (Optional) The namespace to list releases in.
* `all_namespaces` (Optional) List reeleases across all namespaces.
* `filter` (Optional) A regular expression to filter the name of releases. 
