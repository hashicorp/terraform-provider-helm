#!/bin/bash

set -e
# Prerequisites
if ! command -v gox > /dev/null; then
  go get -u github.com/mitchellh/gox
fi

# setup environment
PROVIDER_NAME="helm"
TARGET_DIR="$(pwd)/results"
XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
XC_OS=${XC_OS:=linux darwin windows freebsd openbsd solaris}
XC_EXCLUDE_OSARCH="!darwin/arm !darwin/386 !solaris/amd64"
LD_FLAGS="-s -w"
export CGO_ENABLED=0

rm -rf "${TARGET_DIR}"
mkdir -p "${TARGET_DIR}"

# Compile
gox \
  -os="${XC_OS}" \
  -arch="${XC_ARCH}" \
  -osarch="${XC_EXCLUDE_OSARCH}" \
  -ldflags "${LD_FLAGS}" \
  -output "$TARGET_DIR/{{.OS}}_{{.Arch}}/terraform-provider-${PROVIDER_NAME}_v0.0.0_x4" \
  -verbose \
  -rebuild \
