# Deploying Custom Providers

## Build

To build the latest version of the provider with your changes run the following:
```
$ export VERSION=2.8.0
$ VERSION=v${VERSION} make packages
a terraform-provider-helm_darwin_amd64
a terraform-provider-helm_darwin_amd64/terraform-provider-helm_v2.8.0
a terraform-provider-helm_linux_amd64
a terraform-provider-helm_linux_amd64/terraform-provider-helm_v2.8.0
```

This will create the release binaries in the `./build` folder.

## Package

Package each binary into a release zip with the following:

```
$ zip terraform-provider-helm_${VERSION}_darwin_amd64.zip ./build/terraform-provider-helm_darwin_amd64/terraform-provider-helm_${VERSION}
  adding: build/terraform-provider-helm_darwin_amd64/terraform-provider-helm_v2.8.0 (deflated 55%)
$ zip terraform-provider-helm_${VERSION}_linux_amd64.zip ./build/terraform-provider-helm_linux_amd64/terraform-provider-helm_${VERSION}
  adding: build/terraform-provider-helm_linux_amd64/terraform-provider-helm_v2.8.0 (deflated 55%)
```

## Sign

Generate SHASUMs of the release archives with:

```
shasum -a 256 *.zip > terraform-provider-helm_${VERSION}_SHA256SUMS
```

You will then need to sign the SHASUMs using the GPG key stored in [Keybase](keybase://team/fastly_ccs/TFE%20Provider%20GPG%20/tfe_provider_key.gpg)
Import this key into your keyring and use it to sign the SHASUM like so:

```
gpg --import tfe_provider_key.gpg
gpg -u 291B9AEEA25D983658B694CC6584570A987E54B0 --detach-sign terraform-provider-helm_${VERSION}_SHA256SUMS
```

You now have the required files to create a release version in TFE.

## Release
Create a new version of the provider

```
$ cat ./deploy/terraform_provider_version.json | envsubst > terraform_provider_version.json
$ curl https://prod.tf.secretcdn.net/api/v2/organizations/fastly/registry-providers/private/fastly/helm/versions --header "Authorization: Bearer $TFE_TOKEN"  --header "Content-Type: application/vnd.api+json"  --request POST --data @terraform_provider_version.json | jq .
...
"links": {
      "shasums-upload": "https://prod.tf.secretcdn.net/_archivist/v1/object/dmF1bHQ6djE6WE9Tb3ZWL2xNN2lGdXg3aDg1dVZLZVcvOS9RZmIrck1XUUVUOGlCRmVNWDJybzU5TUFsck44RWZQeTR3bVhUc05zTnAvOTltSFZzb0Y2NGNwWWNxYStUTmlnOGd5QTA1eVNOYnJMejdWb3BIVGk4NmpKZHdtOTVXaHJZanF2S1Q3Tm9FNDZmS0loelJXT0RUZEdxU2JiZzFWMHlZaVBra2ZsU1EvSVoybkZYRlFlLzkxNGJxTTQxYTJyYWpvSTI3QzUydXhybHJ6UFo3UUNjMFQ3aVh5cW5JQTBvQkY2L3pDYjFCZ0RaUGM5TE0yR3ZpMTlzUVcyN29aTm5RMThMalo4azJ2eFAvWjAyUGtrMHAvekc1T2NEZmNiaVNiUHV6elZ4cGVreG9XNGc9",
      "shasums-sig-upload": "https://prod.tf.secretcdn.net/_archivist/v1/object/dmF1bHQ6djE6SmxMMVRBVGgzeXdybEZMS0gxRUFsdFdFOHhCL2t2S1NLWFRlVTRuRUh0ZlpRb29yVmRQb3hQKzh2NVVQQU9VenE4MnhSUEVleDVuSXphdkQyUis3dGI5RXdqUjFTS0F4M1FDNXh3QXVFSTI1VVNRRXAya2ZhdFQyUVBCbkllOTZaUUNmblRFZUxyNzNWZWNBeE41R3k1QXZseTZhZkpzbnM5QXdhY1hscGM1UjlWNkZ3WDdJRWhRVmF4eDAzZCsvUnZEZ251QzBCWGdlMi9JWDJHUDUrVWpKUTJyZktjVUpONmM4KzFrMGptdW5URDZidHVSWFVVYk9iakF4Y2krMUdZYU1yUFQxYUFsakoyYWRjZmlCNWJGMWQzNW5hTzF4aEZxMGhLc1NSR2tDMnkyZQ"
    }
```

You want to make note of the 2 attributes from the `links` field provided above.
You need to upload your SHASUM files to these endpoints.

```
curl -T terraform-provider-helm_${VERSION}_SHA256SUMS ${links.shasums-upload}
curl -T terraform-provider-helm_${VERSION}_SHA256SUMS.sig ${links.shasums-sig-upload}
```

Next, create a platform release of your provider using the [template](./terraform_provider_platform.json) for both platforms, eg:

```
$ cat terraform_provider_darwin
{
  "data": {
    "type": "registry-provider-version-platforms",
    "attributes": {
      "os": "darwin",
      "arch": "amd64",
      "shasum": "8b2cdd59567cae65788ab7c398daba91bd2ee11c89e8f30c347173b80f130fe7",
      "filename": "terraform-provider-helm_2.8.0_darwin_amd64.zip"
    }
  }
}

$ cat terraform_provider_linux
{
  "data": {
    "type": "registry-provider-version-platforms",
    "attributes": {
      "os": "linux",
      "arch": "amd64",
      "shasum": "8b2cdd59567cae65788ab7c398daba91bd2ee11c89e8f30c347173b80f130fe7",
      "filename": "terraform-provider-helm_2.8.0_linux_amd64.zip"
    }
  }
}

$ curl \
  --header "Authorization: Bearer $TFE_TOKEN" \
  --header "Content-Type: application/vnd.api+json" \
  --request POST \
  --data @terraform_provider_linux.json \
  https://prod.tf.secretcdn.net/api/v2/organizations/fastly/registry-providers/private/fastly/helm/versions/${VERSION}/platforms | jq
...

"links": {
      "provider-binary-upload": "https://prod.tf.secretcdn.net/_archivist/v1/object/dmF1bHQ6djE6VyszVmE1Mjd6MDNvREZ6RmdXMVFqT053bTR5NUtSMkdaenJ1RG1WR1JscVQ0RHNSRmt3YkdUUDNwTklEQ2QyZ2ttV01RNm4reUFJaVc3K0ljMnAwUnVRcjRJSlB4d0c1ZzlabnJFRDhBQk9IRVpkOUxFTnhaQldMYTVZN291ckNYZUZDMW96N3ZxME5uYVBKUFNwNjZhTkg2UkxiUW5IU3ZyVHJQQ2lIMDRRTWs4NVFDd2NkMTlxUmZrdTM4ZWhiQjUydVJtKzVqTTRtTkhhd0lqZUxRNVdWNGR3VEdDeTkvb0x3MCtMWnhQekJRcnpBNXQ3dU1GTW15RzByK1E9PQ"
    }

$ curl \
  --header "Authorization: Bearer $TFE_TOKEN" \
  --header "Content-Type: application/vnd.api+json" \
  --request POST \
  --data @terraform_provider_darwin.json \
  https://prod.tf.secretcdn.net/api/v2/organizations/fastly/registry-providers/private/fastly/helm/versions/${VERSION}/platforms | jq
...

"links": {
      "provider-binary-upload": "https://prod.tf.secretcdn.net/_archivist/v1/object/dmF1bHQ6djE6VyszVmE1Mjd6MDNvREZ6RmdXMVFqT053bTR5NUtSMkdaenJ1RG1WR1JscVQ0RHNSRmt3YkdUUDNwTklEQ2QyZ2ttV01RNm4reUFJaVc3K0ljMnAwUnVRcjRJSlB4d0c1ZzlabnJFRDhBQk9IRVpkOUxFTnhaQldMYTVZN291ckNYZUZDMW96N3ZxME5uYVBKUFNwNjZhTkg2UkxiUW5IU3ZyVHJQQ2lIMDRRTWs4NVFDd2NkMTlxUmZrdTM4ZWhiQjUydVJtKzVqTTRtTkhhd0lqZUxRNVdWNGR3VEdDeTkvb0x3MCtMWnhQekJRcnpBNXQ3dU1GTW15RzByK1E9PQ"
    }


```

Lastly you need to upload the archives to the link generated from each platform
release

```
curl -T terraform-provider-helm_${VERSION}_darwin_amd64.zip ${links.provider-binary-upload}
curl -T terraform-provider-helm_${VERSION}_linux_amd64.zip ${links.provider-binary-upload}
```


