# Terraform Provider for Helm

[![Build Status][build_badge]][build_link]
[![GitHub release][release_badge]][release_link]
[![license][license_badge]](LICENSE)

This is a [Helm][helm] provider for [Terraform][terraform].

The provider manage the installed [Charts][charts] in your Kubernetes cluster,
in the same way of Helm does, through Terraform.

## Contents

- [Installation](#installation)
- [Example](#example)
- [Documentation](docs/README.md)
  - [Resource: helm\_release](docs/release.md)
  - [Resource: helm\_repository](docs/repository.md)

## Installation

### Requirements

*terraform-provider-helm* is based on Terraform, this means that you need:

- [Terraform][terraform_download] \>=0.10.0
- [Kubernetes][kubernetes] \>=1.4

### Installation from binaries (recommended)

The recommended way to install *terraform-provider-helm* is use the binary
distributions from the [Releases][release_link] page. The packages are
available for Linux and macOS.

Download and uncompress the latest release for your OS. This example uses the
linux binary.

``` sh
wget https://github.com/mcuadros/terraform-provider-helm/releases/download/v0.4.0/terraform-provider-helm_v0.4.0_linux_amd64.tar.gz
tar -xvf terraform-provider-helm*.tar.gz
```

Now copy the binary to the Terraform’s plugins folder, if is your first plugin
maybe isn’t present.

``` sh
mkdir -p ~/.terraform.d/plugins/
mv terraform-provider-helm*/terraform-provider-helm ~/.terraform.d/plugins/
```

### Installation from sources

If you wish to compile the provider from source code, you’ll first need
[Go][go] installed on your machine (version \>=1.9 is *required*). You’ll also
need to correctly setup a [GOPATH][gopath], as well as adding `$GOPATH/bin` to
your `$PATH`.

Clone repository to:
`$GOPATH/src/github.com/mcuadros/terraform-provider-helm`

``` sh
mkdir -p $GOPATH/src/github.com/mcuadros
git clone https://github.com/mcuadros/terraform-provider-helm.git $GOPATH/src/github.com/mcuadros/terraform-providers
```

Enter the provider directory and build the provider

``` sh
cd $GOPATH/src/github.com/mcuadros/terraform-provider-helm
make build
```

Now copy the compiled binary to the Terraform’s plugins folder, if is
your first plugin maybe isn’t present.

``` sh
mkdir -p ~/.terraform.d/plugins/
mv terraform-provider-helm ~/.terraform.d/plugins/
```

## Example

This is a small example of how to install the mariadb chart on your default
kubernetes cluster, since the provider was initialized, all the configuration
is retrieve from the environment. Please read the
[documentation](docs/README.md) for more information.

You should have a local configured copy of kubectl.

``` hcl
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

## License

Mozilla Public License 2.0, see [LICENSE](LICENSE)

[build_badge]: https://travis-ci.org/mcuadros/terraform-provider-helm.svg?branch=v0.5.0
[build_link]: https://travis-ci.org/mcuadros/terraform-provider-helm
[charts]: https://github.com/kubernetes/charts
[go]: http://www.golang.org
[gopath]: http://golang.org/doc/code.html#GOPATH
[helm]: https://github.com/kubernetes/helm
[kubernetes]: https://kubernetes.io/
[license_badge]: https://img.shields.io/github/license/mcuadros/terraform-provider-helm.svg
[release_badge]: https://img.shields.io/github/release/mcuadros/terraform-provider-helm.svg
[release_link]: https://github.com/mcuadros/terraform-provider-helm/releases
[terraform_download]: https://www.terraform.io/downloads.html
[terraform]: https://www.terraform.io/
