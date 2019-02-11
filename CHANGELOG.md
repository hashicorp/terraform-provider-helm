## 0.8.0 (Unreleased)

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
