Terraform Provider for Helm [![Build Status](https://travis-ci.org/mcuadros/terraform-provider-helm.svg?branch=v0.4.0)](https://travis-ci.org/mcuadros/terraform-provider-helm) [![GitHub release](https://img.shields.io/github/release/mcuadros/terraform-provider-helm.svg)](https://github.com/mcuadros/terraform-provider-helm/releases) [![license](https://img.shields.io/github/license/mcuadros/terraform-provider-helm.svg)]()
===========================

This is a [Helm](https://github.com/kubernetes/helm) provider for [Terraform](https://www.terraform.io/).

The provider manage the installed [Charts](https://github.com/kubernetes/charts) in your Kubernetes cluster, in the same way of Helm does, through Terraform.

Contents
--------

* [Installation](#installation)
* [Example](#example)
* [Documentation](docs/README.md)
  * [Resource: helm_release](docs/release.md)
  * [Resource: helm_repository](docs/repository.md)


Installation
------------

### Requirements

*terraform-provider-helm* is based on [Terraform](golang.org), this means that you need


- [Terraform](https://www.terraform.io/downloads.html) >=0.10.0
- [Kubernetes](https://kubernetes.io/) >=1.4

### Installation from binaries (recommended)

The recommended way to install *terraform-provider-helm* is use the binary
distributions from the [Releases](https://github.com/mcuadros/terraform-provider-helm/releases) page. The packages are available for Linux and macOS.

Download and uncompress the latest release for your OS. This example uses the linux binary.

```sh
> wget https://github.com/mcuadros/terraform-provider-helm/releases/download/v0.5.0/terraform-provider-helm_v0.5.0_linux_amd64.tar.gz
> tar -xvf terraform-provider-helm*.tar.gz
```

Now copy the binary to the Terraform's plugins folder, if is your first plugin maybe isn't present.

```sh
> mkdir -p ~/.terraform.d/plugins/
> mv terraform-provider-helm*/terraform-provider-helm ~/.terraform.d/plugins/
```

### Installation from sources

If you wish to compile the provider from source code, you'll first need [Go](http://www.golang.org) installed on your machine (version >=1.9 is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

Clone repository to: `$GOPATH/src/github.com/mcuadros/terraform-provider-helm`

```sh
> mkdir -p $GOPATH/src/github.com/mcuadros
> git clone https://github.com/mcuadros/terraform-provider-helm.git $GOPATH/src/github.com/mcuadros/terraform-providers
```

Enter the provider directory and build the provider

```sh
> cd $GOPATH/src/github.com/mcuadros/terraform-provider-helm
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
is retrieve from the environment. Please read the [documentation](docs/README.md) for more
information.

You should have a local configured copy of kubectl.

```hcl
resource "helm_release" "my_database" {
    name      = "my_datasase"
    chart     = "stable/mariadb"

    set {
        name  = "mariadbUser"
        value = "foo"
    }

    set {
        name = "mariadbPassword"
        value = "qux"
    }
}
```

License
-------

Mozilla Public License 2.0, see [LICENSE](LICENSE)

