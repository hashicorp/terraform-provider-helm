## 3.1.1 (Nov 17, 2025)

BUG FIXES:

* `resource/helm_release`: Fix "inconsistent result after apply" error by moving recomputeMetadata function call [[GH-1713](https://github.com/hashicorp/terraform-provider-helm/issues/1713)]

## 3.1.0 (Oct 27, 2025)

FEATURES:

* Add `qps` field to Helm provider configuration [[GH-1668](https://github.com/hashicorp/terraform-provider-helm/issues/1668)]
* Add `resources` attribute to manifest experimental feature [[GH-1693](https://github.com/hashicorp/terraform-provider-helm/issues/1693)]
* `helm_template`: Add `set_wo` write-only attribute [[GH-1703](https://github.com/hashicorp/terraform-provider-helm/issues/1703)]
* `helm_release`: Add support for the `take_ownership` field [[GH-1680](https://github.com/hashicorp/terraform-provider-helm/pull/1680)]

ENHANCEMENT:

* Introduce the `timeouts` field to the helm_release resource and helm_template data source, enabling configurable operation timeouts for create, read, update, and delete actions. [[GH-1702](https://github.com/hashicorp/terraform-provider-helm/issues/1702)]

BUG FIXES:

* Port missing field `upgrade_install` [[GH-1675](https://github.com/hashicorp/terraform-provider-helm/pull/1675)]
 

## 3.0.2 (Jun 23, 2025)

This is a patch release that fixes a number of bugs discovered in the v3.x.x release. 

BUG FIXES:

* `helm_release`: Fix description field causing inconsistent plan [[GH-1648](https://github.com/hashicorp/terraform-provider-helm/issues/1648)]
* `helm_release`: Fix plan error when `devel = false` is set and `version` is provided [[GH-1656](https://github.com/hashicorp/terraform-provider-helm/issues/1656)]
* `helm_release`: Fix postrender being run when binaryPath is nil [[GH-1649](https://github.com/hashicorp/terraform-provider-helm/issues/1649)]
* `helm_release`: Fix shallow clone bug causing nested sensitive values to be redacted in the k8s API [[GH-1644](https://github.com/hashicorp/terraform-provider-helm/issues/1644)]
* `provider`: Fix namespace override logic in Kubernetes client initialization [[GH-1650](https://github.com/hashicorp/terraform-provider-helm/issues/1650)]
* `provider`: Restore support for the `KUBE_PROXY_URL` environment variable [[GH-1655](https://github.com/hashicorp/terraform-provider-helm/issues/1655)]

## 3.0.1 (Jun 18, 2025)

This is a hotfix release.

HOTFIX:

- `helm_release`: Fix state upgrader code to use correct type for "values" attribute. [[GH-1638](https://github.com/hashicorp/terraform-provider-helm/pull/1638)]
 

## 3.0.0 (Jun 18, 2025)

This release migrates ports the provider project from `terraform-plugin-sdk/v2` to `terraform-plugin-framework` [[GH-1379](https://github.com/hashicorp/terraform-provider-helm/pull/1379)]

Please refer to the [migration guide](./docs/guides/v3-upgrade-guide.md).

BREAKING CHANGES:

- **Blocks to Nested Objects**: Blocks like `kubernetes`, `registry`, and `experiments` are now represented as nested objects.
- **List Syntax for Nested Attributes**: Attributes like `set`, `set_list`, and `set_sensitive` in `helm_release` and `helm_template` are now lists of nested objects instead of blocks
- The new framework code uses [Terraform Plugin Protocol Version 6](https://developer.hashicorp.com/terraform/plugin/terraform-plugin-protocol#protocol-version-6) which is compatible with Terraform versions 1.0 and above. Users of earlier versions of Terraform can continue to use the Helm provider by pinning their configuration to the 2.x version.

FEATURES:

* Add `"literal"` as a supported `type` for the `set` block [[GH-1615](https://github.com/hashicorp/terraform-provider-helm/issues/1615)]

* `helm_release`: Add support for ResourceIdentity. [[GH-1625](https://github.com/hashicorp/terraform-provider-helm/issues/1625)]

* `helm_release`: Add `set_wo` write-only attribute [[GH-1592](https://github.com/hashicorp/terraform-provider-helm/issues/1592)]

ENHANCEMENT:

* `helm_release`: Add `UpgradeState` logic to support migration from SDKv2 to Plugin Framework [[GH-1633](https://github.com/hashicorp/terraform-provider-helm/issues/1633)]
* update helm dependency to v3.17.2 [[GH-1608](https://github.com/hashicorp/terraform-provider-helm/issues/1608)]

BUG FIXES:

* `helm_release`: Fix namespace behaviour for dependency charts in non-default namespaces [[GH-1583](https://github.com/hashicorp/terraform-provider-helm/issues/1583)]

* change `set.value` && `set_list.value` to optional instead of required [[GH-1572](https://github.com/hashicorp/terraform-provider-helm/issues/1572)]

## 3.0.0-pre2 (Feb 27, 2025)

FEATURES:

* `helm_release`: Add `set_wo` write-only attribute [[GH-1592](https://github.com/hashicorp/terraform-provider-helm/issues/1592)]

BUG FIXES:

* change `set.value` && `set_list.value` to optional instead of required [[GH-1572](https://github.com/hashicorp/terraform-provider-helm/issues/1572)]


## 3.0.0-pre1 (Jan 16, 2025)

* This pre-release migrates ports the provider project from `terraform-plugin-sdk/v2` to `terraform-plugin-framework` [[GH-1379](https://github.com/hashicorp/terraform-provider-helm/pull/1379)]

Please refer to the [migration guide](./docs/guides/v3-upgrade-guide.md).

## 2.17.0 (Dec 19, 2024)

ENHANCEMENT:

* `resource/helm_release`: the dry-run option is now set to `server` to execute any chart lookups against the server during the plan stage. [[GH-1335](https://github.com/hashicorp/terraform-provider-helm/issues/1335)]

BUG FIXES:

* `resource/helm_release`: fix an issue where `postrender.args` is not parsed correctly. [[GH-1534](https://github.com/hashicorp/terraform-provider-helm/issues/1534)]

## 2.16.1 (Oct 15, 2024)

BUG FIXES:

* `helm_release`: Fix nil pointer deref panic on destroy when helm release is not found [[GH-1501](https://github.com/hashicorp/terraform-provider-helm/issues/1501)]

## 2.16.0 (Oct 10, 2024)

BUG FIXES:

* `helm_release`: On destroy, do not error when release is not found [[GH-1487](https://github.com/hashicorp/terraform-provider-helm/issues/1487)]
* `resource/helm_release`: Fix: only recompute metadata when the version in the metadata changes [[GH-1458](https://github.com/hashicorp/terraform-provider-helm/issues/1458)]

## 2.15.0 (Aug 14, 2024)

ENHANCEMENT:

* resource/helm_release: add `upgrade_install` boolean attribute to enable idempotent release installation, addressing components of [GH-425](https://github.com/hashicorp/terraform-provider-helm/issues/425) [[GH-1247](https://github.com/hashicorp/terraform-provider-helm/issues/1247)]

## 2.14.1 (Aug 7, 2024)

DEPENDENCIES:

* Bump golang.org/x/crypto from v0.23.0 to v0.25.0 [[GH-1399](https://github.com/hashicorp/terraform-provider-helm/pull/1399)]
* Bump k8s.io/api from v0.30.0 to v0.30.3 [[GH-1436](https://github.com/hashicorp/terraform-provider-helm/pull/1436)]
* Bump k8s.io/apimachinery from v0.30.0 to v0.30.3 [[GH-1436](https://github.com/hashicorp/terraform-provider-helm/pull/1436)]
* Bump k8s.io/client-go from v0.30.0 to v0.30.3 [[GH-1436](https://github.com/hashicorp/terraform-provider-helm/pull/1436)]
* Bump helm.sh/helm/v3 from v3.13.2 to v3.15.3 [[GH-1422](https://github.com/hashicorp/terraform-provider-helm/pull/1422)]

## 2.14.0 (June 13, 2024)

ENHANCEMENT:

* Add support for Terraform's experimental deferred actions [[GH-1377](https://github.com/hashicorp/terraform-provider-helm/issues/1377)]
* `helm_release`: add new attributes metadata.last_deployed, metadata.first_deployed, metadata.notes [[GH-1380](https://github.com/hashicorp/terraform-provider-helm/issues/1380)]

## 2.13.2 (May 8, 2024)

DEPENDENCIES:

* Bump github.com/docker/docker from 24.0.7 to 24.0.9
* Bump golang.org/x/net from 0.21.0 to 0.23.0
* Bundle license file with TF provider release artifacts

## 2.13.1 (Apr 15, 2024)

HOTFIX:

* `helm_release`: Fix regression causing errors at plan time. 

## 2.13.0 (Apr 4, 2024)

BUG FIXES:

* `provider`: Fix manifest diff rendering for OCI charts. [[GH-1326](https://github.com/hashicorp/terraform-provider-helm/issues/1326)]

DOCS:

* `docs`: Use templatefile() instead of "template_file" provider in GKE example. [[GH-1329](https://github.com/hashicorp/terraform-provider-helm/issues/1329)]

## 2.12.1 (Nov 30, 2023)

DEPENDENCIES:

* Bump Golang from `1.20` to `1.21`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]
* Bump github.com/hashicorp/go-cty from `v1.4.1-0.20200414143053-d3edf31b6320` to `v1.4.1-0.20200723130312-85980079f637`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]
* Bump github.com/hashicorp/terraform-plugin-docs from `v0.14.1` to `v0.16.0`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]
* Bump github.com/hashicorp/terraform-plugin-sdk/v2 from `v2.26.1` to `v2.30.0`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]
* Bump golang.org/x/crypto from `v0.14.0` to `v0.16.0`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]
* Bump helm.sh/helm/v3 from `v3.13.1` to `v3.13.2`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]
* Bump k8s.io/api from `v0.28.3` to `v0.28.4`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]
* Bump k8s.io/apimachinery from `v0.28.3` to `v0.28.4`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]
* Bump k8s.io/client-go from `v0.28.3` to `v0.28.4`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]
* Bump sigs.k8s.io/yaml from `v1.3.0` to `v1.4.0`. [[GH-1300](https://github.com/hashicorp/terraform-provider-helm/issues/1300)]

## 2.12.0 (Nov 27, 2023)

BUG FIXES:

* `helm_release`: Fix perpetual diff when version attribute is an empty string [[GH-1246](https://github.com/hashicorp/terraform-provider-helm/issues/1246)]

DEPENDENCIES:

* Bump `helm.sh/helm/v3` from `3.12.0` to `3.13.1` 

## 2.11.0 (Aug 24, 2023)

ENHANCEMENT:

* `kubernetes/provider.go`: Add `tls_server_name` kubernetes provider options. [[GH-839](https://github.com/hashicorp/terraform-provider-helm/issues/839)]
* `resource/helm_release`: add `name` field validation to be limited to 53 characters. [[GH-1228](https://github.com/hashicorp/terraform-provider-helm/issues/1228)]

BUG FIXES:

* `helm/resource_release.go`: Fix: version conflicts when using local chart [[GH-1176](https://github.com/hashicorp/terraform-provider-helm/issues/1176)]
* `resource/helm_release`: Add nil check for `set_list.value` to prevent provider ChartPathOptions [[GH-1231](https://github.com/hashicorp/terraform-provider-helm/issues/1231)]

## 2.10.1 (Jun 5, 2023)

HOTFIX:

* `helm_release`: Fix: Only recompute metadata if version actually changes. [[GH-1150](https://github.com/hashicorp/terraform-provider-helm/issues/1150)]

## 2.10.0 (May 30, 2023)

FEATURES:

* `helm_release`: Add `set_list` attribute [[GH-1071](https://github.com/hashicorp/terraform-provider-helm/issues/1071)]

BUG FIXES:

* `helm_release`: Always recompute metadata when a release is updated [[GH-1097](https://github.com/hashicorp/terraform-provider-helm/issues/1097)]

DEPENDENCIES:

* Bump `helm.sh/helm/v3` from `3.11.2` to `3.12.0` [[GH-1143](https://github.com/hashicorp/terraform-provider-helm/issues/1143)]

## 2.9.0 (February 14, 2023)

FEATURES:

* `provider`: Add a new attribute `burst_limit` for client-side throttling limit configuration. [[GH-1012](https://github.com/hashicorp/terraform-provider-helm/issues/1012)]

ENHANCEMENT:

* `data_source/helm_template`: Add a new attribute `crds` which when `include_crds` is set to `true` will be populated with a list of the manifests from the `crds/` folder of the chart. [[GH-1050](https://github.com/hashicorp/terraform-provider-helm/issues/1050)]

BUG FIXES:

* `resource/helm_release`: Fix an issue when the provider crashes with the error message `Provider produced inconsistent final plan` after upgrading from `v2.5.1` to `v2.6.0` and higher. That happened due to changes in the provider schema and the introduction of a new attribute `pass_credentials` that was not properly handled. [[GH-982](https://github.com/hashicorp/terraform-provider-helm/issues/982)]

DOCS:

* `data_source/helm_template`: Add a new attribute `crds` [[GH-1050](https://github.com/hashicorp/terraform-provider-helm/issues/1050)]
* `data_source/helm_template`: Correct some errors in examples. [[GH-1027](https://github.com/hashicorp/terraform-provider-helm/issues/1027)]
* `provider`: Add a new attribute `burst_limit`. [[GH-1012](https://github.com/hashicorp/terraform-provider-helm/issues/1012)]
* `provider`: Add a note regarding the `KUBECONFIG` environment variable. [[GH-1051](https://github.com/hashicorp/terraform-provider-helm/issues/1051)]
* `resource/helm_release`: Add usage example for `OCI` repositories. [[GH-1030](https://github.com/hashicorp/terraform-provider-helm/issues/1030)]
* `resource/helm_release`: Add usage examples for `GCS` and `S3` plugins. [[GH-1026](https://github.com/hashicorp/terraform-provider-helm/issues/1026)]

DEPENDENCIES:

* Bump `github.com/containerd/containerd` from `1.6.6` to `1.6.12` [[GH-1029](https://github.com/hashicorp/terraform-provider-helm/issues/1029)]
* Bump `golang.org/x/crypto` from `0.5.0` to `0.6.0` [[GH-1055](https://github.com/hashicorp/terraform-provider-helm/issues/1055)]
* Bump `helm.sh/helm/v3` from `3.9.4` to `3.11.1` [[GH-1036](https://github.com/hashicorp/terraform-provider-helm/issues/1036)] [[GH-1054](https://github.com/hashicorp/terraform-provider-helm/issues/1054)]
* Bump `k8s.io/client-go` from `0.24.2` to `0.26.1` [[GH-1037](https://github.com/hashicorp/terraform-provider-helm/issues/1037)]

NOTES:

* `provider`: `kubernetes.exec.api_version` no longer supports `client.authentication.k8s.io/v1alpha1`. Please, switch to `client.authentication.k8s.io/v1beta1` or `client.authentication.k8s.io/v1`. [[GH-1037](https://github.com/hashicorp/terraform-provider-helm/issues/1037)]

## Community Contributors :raised_hands:
- @loafoe made their contribution in https://github.com/hashicorp/terraform-provider-helm/pull/1012

## 2.8.0 (December 13, 2022)

FEATURES:

* Add support for configuring OCI registries inside provider block [[GH-862](https://github.com/hashicorp/terraform-provider-helm/issues/862)]
* Add support for setting kube version on helm_template data source [[GH-994](https://github.com/hashicorp/terraform-provider-helm/issues/994)]

BUG FIXES:

* Fix larger diff than expected when updating helm_release "set" block value [[GH-915](https://github.com/hashicorp/terraform-provider-helm/issues/915)]

## 2.7.1 (October 12, 2022)

BUG FIXES:

* Crash Fix: Fix Unknown Value in Manifest Diff [[GH-966](https://github.com/hashicorp/terraform-provider-helm/issues/966)]

## 2.7.0 (September 28, 2022)

FEATURES:

* Update helm package to 3.9.4 (#945)
* Show Manifest when creating release [[GH-903](https://github.com/hashicorp/terraform-provider-helm/issues/903)]

BUG FIXES:

* Do dependency update in resourceDiff #771 (#855)
* Crash: Fix `show_only` crash when string is empty [[GH-950](https://github.com/hashicorp/terraform-provider-helm/issues/950)]

## 2.6.0 (June 17, 2022)

IMPROVEMENTS:
* Upgrade helm dependency to 3.9.0 (#867)
* Add `args` attribute in `post_render` block in (#869)
* Add `pass_credentials` attribute (#841)
* Add `proxy_url` attribute to provider block (#843)

BUG FIXES:
* Don't persist state when update causes an error (#857)

## 2.5.1 (April 11, 2022)

FIX:
* Only run OCI login on create and update (#846)
* OCI login concurrency issue (#848)

## 2.5.0 (March 28, 2022)

* Upgrade helm dependency to v3.8.1
* Add support for OCI registries 

## 2.4.1 (November 09, 2021)

HOTFIX:
* Fix exec plugin interactive mode regression (#798) 

## 2.4.0 (November 08, 2021)

* Upgrade helm to 3.7.1

## 2.3.0 (August 27, 2021)

* Support templates with multiple resources in helm_template data source (#772)
* Upgrade helm to 3.6.2

## 2.2.0 (June 10, 2021)

* Add support for stand-alone debug mode (launch with -debug argument) (#748)
* Add helm_template data source to render chart templates locally (#483)
* Surface diagnostics when helm release creation fails (#727)

## 2.1.2 (April 27, 2021)

* Fix dependency download on resource update (#580)
* Add support for the --wait-for-jobs option (#720)

## 2.1.1 (April 16, 2021)

* Fix dry-run happening at plan when manifest is not enabled (#724)

## 2.1.0 (April 01, 2021)

IMPROVEMENTS:
* Add chart diff support by storing the rendered manifest (#702)
* Update to Helm 3.5.3 (#709)
* Docs: add link to Learn tutorial (#714)

BUG FIXES:
* Remove kubeconfig file check (#708)

## 2.0.3 (March 11, 2021)

BUG FIXES:
* Fix documentation for KUBE_TOKEN env var name (#684)
* Fix destroy stage error for charts with "helm.sh/resource-policy:keep" annotation (#671)
* Fix read function to set resource id to null when not found (#674)

IMPROVEMENTS:
* Update provider configuration docs (#673)

## 2.0.2 (January 18, 2021)

BUG FIXES:
* Remove check for empty kubernetes block 

## 2.0.1 (December 19, 2020)

BUG FIXES:
* Move kubernetes config check out of providerConfigure (#648)

## 2.0.0 (December 19, 2020)

BREAKING CHANGES:
Please review our [upgrade guide](https://github.com/hashicorp/terraform-provider-helm/blob/master/website/docs/guides/v2-upgrade-guide.markdown).

* Update Terraform SDK to v2 (#594). 
* Remove deprecated helm_repository resource and data source (#600)
* Remove implicit support for KUBECONFIG (#604)
* Remove load_config_file attribute (#604)
* Remove set_string attribute from helm_release (#608)

IMPROVEMENTS:
* Add support for multiple paths to kubeconfig files (#636)
* Remove remote dependencies from test-fixtures (#638)
* Set up matrix build to run acc tests against different tf versions (#637)


## 1.3.2 (October 07, 2020)

BUG FIXES:
* Fix nil pointer crash when using Helm plugins (#598)

## 1.3.1 (September 29, 2020)

IMPROVEMENTS:
* Upgrade Helm to 3.3.4 (#572)

## 1.3.0 (September 02, 2020)

IMPROVEMENTS:
* Added app_version to metadata attribute block (#532)

BUG FIXES:
* Fix nil path for `dependency_update` flag (#482)

## 1.2.4 (July 22, 2020)

BUG FIXES:

* Update go-version for CVE-2020-14039 (#548)

## 1.2.3 (June 16, 2020)

BUG FIXES:

* Fix concurrent read/write crash (#525)
* Fix for provider hang (#505)

## 1.2.2 (June 01, 2020)

BUG FIXES:

* Add a lint attribute to helm_release (#514)

## 1.2.1 (May 08, 2020)

BUG FIXES:

* Fix linter crash (#487)

## 1.2.0 (May 06, 2020)

IMPROVEMENTS:

* Cloak sensitive values in metadata field (#480)
* Upgrade to Helm 3.2.0
* Deprecate helm_repository data source
* Lint chart at plan time

## 1.1.1 (March 26, 2020)

BUG FIXES:

* Fix chart path bug causing unwanted diff (#449)

## 1.1.0 (March 19, 2020)

IMPROVEMENTS:

* Add import feature for helm_release (#394)
* Run acceptance tests in travis-ci using kind
* Upgrade helm to version v3.1.2 (#440)
* Add description attribute
* Add post-rendering support

BUG FIXES:

* Fix errors being swallowed when creating a helm_release (#406)
* Various documentation fixes

## 1.0.0 (February 05, 2020)

BREAKING CHANGES:

* No longer supports helm v2 (#378)
* Provider no longer supports the following parameters
    * host
    * home
    * namespace
    * init_helm_home
    * install_tiller
    * tiller_image
    * connection_timeout
    * service_account
    * automount_service_account_token
    * override
    * max_history (Moved to the release)
    * plugins_disable
    * insecure
    * enable_tls
    * client_key
    * client_certificate
    * ca_certificate
* Release no longer supports the following parameters
    * disable_crd_hooks
* Release Parameters that were renamed
    * reuse was renamed to replace to match the rename in helm v3

IMPROVEMENTS:

* Upgrade Helm to v3.0
* Adds the following parameters to the provider
    * plugins_path - (Optional) The path to the plugins directory. Defaults to `HELM_PLUGINS` env if it is set, otherwise uses the default path set by helm.
    * registry_config_path - (Optional) The path to the registry config file. Defaults to `HELM_REGISTRY_CONFIG` env if it is set, otherwise uses the default path set by helm.
    * repository_config_path - (Optional) The path to the file containing repository names and URLs. Defaults to `HELM_REPOSITORY_CONFIG` env if it is set, otherwise uses the default path set by helm.
    * repository_cache - (Optional) The path to the file containing cached repository indexes. Defaults to `HELM_REPOSITORY_CACHE` env if it is set, otherwise uses the default path set by helm.
    * helm_driver - (Optional) "The backend storage driver. Valid values are: `configmap`, `secret`, `memory`. Defaults to `secret`
* Adds the following parameters to the release
    * repository_key_file - (Optional) The repositories cert key file
    * repository_cert_file - (Optional) The repositories cert file
    * repository_ca_file - (Optional) The Repositories CA File.
    * repository_username - (Optional) Username for HTTP basic authentication against the repository.
    * repository_password - (Optional) Password for HTTP basic authentication against the reposotory.
    * reset_values - (Optional) When upgrading, reset the values to the ones built into the chart. Defaults to `false`.
    * cleanup_on_fail - (Optional) Allow deletion of new resources created in this upgrade when upgrade fails. Defaults to `false`.
    * max_history - (Optional) Maximum number of release versions stored per release. Defaults to 0 (no limit).
    * atomic - (Optional) If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used. Defaults to false.
    * skip_crds - (Optional) If set, no CRDs will be installed. By default, CRDs are installed if not already present. Defaults to false.
    * render_subchart_notes - (Optional) If set, render subchart notes along with the parent. Defaults to true.
    * dependency_update - (Optional) Runs helm dependency update before installing the chart. Defaults to false


## 0.10.5 (Unreleased)
## 0.10.4 (October 28, 2019)

BUG FIXES:

* Tiller installed version should match helm client (#365)

## 0.10.3 (October 27, 2019)

IMPROVEMENTS:

* Upgrade Helm to v2.15.1 and Kubernetes to v1.15.5
* Migrate to terraform-plugin-sdk
* Allow for colon separated KUBECONFIG (#98)
* Modernise docs

BUG FIXES:

* Remove manual installation instructions
* Fix loading kubeconfig when disabled (#307)
* Don't enable TLS if `enable_tls` is false (#245)
* Remove ForceNew on repo and chart changes (#173)

## 0.10.2 (August 07, 2019)

BUG FIXES:

* Revert "Escape commas in set_string" (#310)

## 0.10.1 (July 30, 2019)

IMPROVEMENTS:

* Update helm and tiller to 2.14.1 (#294)
* Wait for tiller if it's not ready (#295)

## 0.10.0 (June 18, 2019)

FEATURES:

* Automatically initialize the configured helm home directory (#185)

IMPROVEMENTS:

* Update helm and tiller to 2.14.0 (#277)
* Update terraform to 0.12.1 (#289 #290)

BUG FIXES:

* Fix concurrency issues reading multiple repos (#272)
* Documentation fixes (#262 #270 #276)
* helm/resource_release: typo fixes (#282)

## 0.9.1 (April 24, 2019)

FEATURES:

IMPROVEMENTS:

* Migrate to Terraform 0.12 SDK
* Move to Go modules for dep-management

BUG FIXES:

* Properly handle commas in attribute values
* Documentation fixes

## 0.9.0 (March 07, 2019)
FEATURES:

* `helm_repository` is now a data source. We retain backwards compatibility through `DataSourceResourceShim` (#221)
* Use configured helm home when reading default TLS settings (#210)
* Added `load_config_file` option to enable or disable the load of kubernetes config file (#231)

IMPROVEMENTS:

* CI and doc improvements

## 0.8.0 (February 11, 2019)

FEATURES:

* Added the possibility to set sensitive values (#153)

IMPROVEMENTS:

* Multiple README, logs and docs improvements
* Go 1.11 and modules (#179, #200 and #201)
* Default tiller version v2.11.0 (#194)
* Suppress diff of "keyring" and "devel" attributes (#193)
* Add entries to .gitignore to roughly match the Google provider (#206)

BUG FIXES:

* Fix when Helm provider ignores FAILED release state (#161)
* Use `127.0.0.1` as default `localhost` (#207)

## 0.7.0 (December 17, 2018)

- Based on Helm 2.11

## 0.6.2 (October 26, 2018)

- Bug fix: A recursion between the read and create methods as described in PR #137

## 0.6.1 (October 25, 2018)

- Re-release after induction into 'terraform-providers'. This is to align to the de-facto repository version sequence.

## 0.1.0 (October 10, 2018)

- Initial Release by Hashicorp
