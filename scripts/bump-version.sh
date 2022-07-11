#!/bin/bash
set -ex
OPERATOR_VERSION=$1
MILVUS_HELM_VERSION=$2

SED="sed -i"
if [[ "$(go env GOOS)" == "darwin" ]]; then
    SED="sed -i .bak "
fi
${SED} "s/^VERSION ?= .*/VERSION ?= ${OPERATOR_VERSION}/g" ./Makefile
${SED} "s/^MILVUS_HELM_VERSION ?= milvus-.*/MILVUS_HELM_VERSION ?= milvus-${MILVUS_HELM_VERSION}/g" ./Makefile
${SED} "s/^version: .*/version: ${OPERATOR_VERSION}/g" ./charts/milvus-operator/Chart.yaml
${SED} "s/^appVersion: .*/appVersion: \"${OPERATOR_VERSION}\"/g" ./charts/milvus-operator/Chart.yaml
${SED} "s/^  tag: \".*\"/  tag: \"v${OPERATOR_VERSION}\"/g" ./charts/milvus-operator/values.yaml
${SED} "s|milvus-operator/releases/download/.*/milvus-operator-.*.tgz|milvus-operator/releases/download/v${OPERATOR_VERSION}/milvus-operator-${OPERATOR_VERSION}.tgz|g" ./README.md ./docs/installation/installation.md
${SED} "s|milvus-operator/.*/deploy/manifests/deployment.yaml|milvus-operator/v${OPERATOR_VERSION}/deploy/manifests/deployment.yaml|g" ./README.md

make deploy-manifests
