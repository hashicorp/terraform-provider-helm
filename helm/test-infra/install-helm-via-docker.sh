#!/bin/bash
# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

set -e

which docker > /dev/null
if [ "$?" != "0" ]; then
	echo "Error: Docker not found"
	exit 1
fi
if [ "$KUBE_DIR" == "" ]; then
	echo "Error: Please specify KUBE_DIR"
	exit 1
fi
if [ "$HELM_VERSION" == "" ]; then
	echo "Error: Please specify HELM_VERSION"
	exit 1
fi
if [ "$HELM_HOME" == "" ]; then
	HELM_HOME=~/.helm
fi
if [ "$HYPERKUBE_VERSION" == "" ]; then
	URL="https://console.cloud.google.com/gcr/images/google-containers/GLOBAL/hyperkube"
	echo "Error: Please specify HYPERKUBE_VERSION"
	echo " - Available versions: $URL"
	exit 1
fi

echo "HELM_HOME=${HELM_HOME}"

docker version

DOCKER_KUBECTL="docker run -i --rm \
	-v ${KUBE_DIR}:/root/.kube \
	gcr.io/google-containers/hyperkube:${HYPERKUBE_VERSION} \
	kubectl"

$DOCKER_KUBECTL version

echo "Creating Tiller service account ..."
$DOCKER_KUBECTL --namespace kube-system create sa tiller

echo "Creating ClusterRoleBinding for Tiller ..."
$DOCKER_KUBECTL create clusterrolebinding tiller \
    --clusterrole cluster-admin \
    --serviceaccount=kube-system:tiller

mkdir -p $HELM_HOME
ls -la $HELM_HOME
DOCKER_HELM="docker run -i --rm \
	-v $(pwd):/apps \
	-v ${HELM_HOME}:/.helm \
	-v ${KUBE_DIR}:/.kube \
	--user $(id -u):$(id -g) \
	alpine/helm:${HELM_VERSION}"

echo "Helm client version:"
$DOCKER_HELM version -c

echo "Initializing Helm ..."
$DOCKER_HELM init --service-account tiller

echo "Verifying Helm ..."
$DOCKER_KUBECTL get deploy,svc tiller-deploy -n kube-system
