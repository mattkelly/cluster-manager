#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

${CODEGEN_PKG}/generate-groups.sh all \
  github.com/containership/cluster-manager/pkg/client github.com/containership/cluster-manager/pkg/apis \
  "containership.io:v3 provision.containership.io:v3 auth.containership.io:v3 federation.containership.io:v3"
