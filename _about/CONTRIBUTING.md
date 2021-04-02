Developing the Provider
------------

Thank you for your interest in contributing to the Helm provider. We welcome your contributions. Here you'll find information to help you get started with provider development.

## Documentation

Our [provider development documentation](https://www.terraform.io/docs/extend/) provides a good start into developing an understanding of provider development. It's the best entry point if you are new to contributing to this provider.

To learn more about how to create issues and pull requests in this repository, and what happens after they are created, you may refer to the resources below:
- [Issue creation and lifecycle](ISSUES.md)
- [Pull Request creation and lifecycle](PULL_REQUESTS.md)

### Installation from sources

If you wish to compile the provider from source code, you'll first need [Go](http://www.golang.org) installed on your machine (version >=1.14 is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

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

Now copy the compiled binary to the Terraform plugins folder.  If this is your first plugin it may not be present.

```sh
> mkdir -p ~/.terraform.d/plugins/
> mv terraform-provider-helm ~/.terraform.d/plugins/
```
