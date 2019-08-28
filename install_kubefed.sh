#!/usr/bin/env bash

set -e

PROXY_BASE_URL=${PROXY_BASE_URL:-https://stage-proxy.containership.io}

if [[ -z "$FEDERATION_NAME" ]]; then
    echo "FEDERATION_NAME environment variable must be set" 1>&2
    exit 1
fi

if [[ -z "$GITHUB_TOKEN" ]]; then
    echo "GITHUB_TOKEN environment variable must be set" 1>&2
    exit 1
fi

if [[ -z "$CONTAINERSHIP_TOKEN" ]]; then
    echo "Attempting to get Containership token from csctl config file"
    CONTAINERSHIP_TOKEN=$(grep '^token:' $HOME/.containership/csctl.yaml | awk '{print $2}')
fi

if [[ -z "$CONTAINERSHIP_TOKEN" ]]; then
    echo "CONTAINERSHIP_TOKEN environment variable must be set or token must exist in ~/.containership/csctl.yaml" 1>&2
    exit 1
fi

# This script assumes that kubectl is properly configured to point to the host
# cluster on which we're installing kubefed (and other required resources)

echo "Applying Containership Cluster CRD"
kubectl apply -f deploy/crd/containership-federation-cluster-crd.yaml


kubefed_yaml=$(mktemp /tmp/kubefed.yaml.XXXXXX)

echo Downloading kubefed manifests to "$kubefed_yaml"
curl -H "Authorization: token ${GITHUB_TOKEN}" \
  -H 'Accept: application/vnd.github.v3.raw' \
  -L -o $kubefed_yaml \
  --silent --show-error \
  https://api.github.com/repos/containership/kubefed-manifests/contents/kubefed.yaml >/dev/null

echo Applying kubefed manifests at "$kubefed_yaml"
kubectl apply -f $kubefed_yaml

echo "Creating Containership token secret in kube-federation-system namespace"
containership_token_base64=$(echo -n "$CONTAINERSHIP_TOKEN" | base64)
cat <<EOF | kubectl apply -f -
---
apiVersion: v1
data:
  token: $containership_token_base64
kind: Secret
metadata:
  name: containership-token
  namespace: kube-federation-system
type: Opaque
EOF

echo "Setting cloud-coordinator environment variable DISABLE_CLUSTER_MANAGEMENT_PLUGIN_SYNC=true"
kubectl -n containership-core set env deployment/cloud-coordinator DISABLE_CLUSTER_MANAGEMENT_PLUGIN_SYNC=true

echo "Setting cloud-coordinator environment variable CONTAINERSHIP_CLOUD_PROXY_BASE_URL=${PROXY_BASE_URL}"
kubectl -n containership-core set env deployment/cloud-coordinator CONTAINERSHIP_CLOUD_PROXY_BASE_URL="$PROXY_BASE_URL"

echo "Setting cloud-coordinator environment variable FEDERATION_NAME=$FEDERATION_NAME"
kubectl -n containership-core set env deployment/cloud-coordinator FEDERATION_NAME="$FEDERATION_NAME"

echo "Updating cloud-coordinator image to kubefed tag"
kubectl -n containership-core set image deployment/cloud-coordinator cloud-coordinator=containership/cloud-coordinator:kubefed

echo "Deleting cloud-coordinator pod to force update"
kubectl -n containership-core delete pod -l "containership.io/app=cloud-coordinator"
