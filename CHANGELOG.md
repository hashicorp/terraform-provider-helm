## 1.1.1 (March 26, 2020)
FIXES:
* Fix chart path bug causing unwanted diff (#449)

## 1.1.0 (March 19, 2020)
IMPROVEMENTS:
* Add import feature for helm_release (#394)
* Run acceptance tests in travis-ci using kind
* Upgrade helm to version v3.1.2 (#440)
* Add description attribute
* Add post-rendering support

FIXES:
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

Improvements:
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

FIXES:

* Tiller installed version should match helm client (#365)

## 0.10.3 (October 27, 2019)

IMPROVEMENTS:

* Upgrade Helm to v2.15.1 and Kubernetes to v1.15.5
* Migrate to terraform-plugin-sdk
* Allow for colon separated KUBECONFIG (#98)
* Modernise docs

FIXES:

* Remove manual installation instructions
* Fix loading kubeconfig when disabled (#307)
* Don't enable TLS if `enable_tls` is false (#245)
* Remove ForceNew on repo and chart changes (#173)

## 0.10.2 (August 07, 2019)

FIXES:

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

FIXES:

* Fix concurrency issues reading multiple repos (#272)
* Documentation fixes (#262 #270 #276)
* helm/resource_release: typo fixes (#282)

## 0.9.1 (April 24, 2019)

FEATURES:

IMPROVEMENTS:

* Migrate to Terraform 0.12 SDK
* Move to Go modules for dep-management

FIXES:

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

FIXES:

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
