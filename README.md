Terraform Provider for Helm
[![Build Status](https://travis-ci.org/hashicorp/terraform-provider-helm.svg?branch=master)](https://travis-ci.org/hashicorp/terraform-provider-helm)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/hashicorp/terraform-provider-helm?label=release)](https://github.com/hashicorp/terraform-provider-helm/releases)
[![license](https://img.shields.io/github/license/hashicorp/terraform-provider-helm.svg)]()
===========================

This is a [Helm](https://github.com/kubernetes/helm) provider for [Terraform](https://www.terraform.io/).

The provider manages the installed [Charts](https://github.com/kubernetes/charts) in your Kubernetes cluster, in the same way of Helm does, through Terraform. It will also install Tiller automatically if it is not already present.

Contents
--------

* [Developing the Provider](#developing-the-provider)
* [Example](#example)
* [Documentation](https://www.terraform.io/docs/providers/helm/index.html)
  * [Resource: helm_release](https://www.terraform.io/docs/providers/helm/release.html)
  * [Resource: helm_repository](https://www.terraform.io/docs/providers/helm/repository.html)


Developing the Provider
------------

### Installation from sources

If you wish to compile the provider from source code, you'll first need [Go](http://www.golang.org) installed on your machine (version >=1.9 is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

Clone repository to: `$GOPATH/src/github.com/hashicorp/terraform-provider-helm`

```sh
> mkdir -p $GOPATH/src/github.com/terraform-providers
> git clone https://github.com/hashicorp/terraform-provider-helm.git $GOPATH/src/github.com/hashicorp/terraform-provider-helm
```

Enter the provider directory and build the provider

```sh
> cd $GOPATH/src/github.com/hashicorp/terraform-provider-helm
> make build
```

Now copy the compiled binary to the Terraform's plugins folder, if is your first plugin maybe isn't present.

```sh
> mkdir -p ~/.terraform.d/plugins/
> mv terraform-provider-helm ~/.terraform.d/plugins/
```

Example
-------

This is a small example of how to install the mariadb chart on your default
kubernetes cluster, since the provider was initialized, all the configuration
is retrieved from the environment. Please read the [documentation](https://www.terraform.io/docs/providers/helm/index.html) for more
information.

You should have a local configured copy of kubectl.

```hcl
resource "helm_release" "my_database" {
    name      = "my-database"
    chart     = "stable/mariadb"

    set {
        name  = "mariadbUser"
        value = "foo"
    }

    set {
        name = "mariadbPassword"
        value = "qux"
    }

    set_string {
        name = "image.tags"
        value = "registry\\.io/terraform-provider-helm\\,example\\.io/terraform-provider-helm"
    }
}
```

License
-------

Mozilla Public License 2.0, see [LICENSE](LICENSE)
