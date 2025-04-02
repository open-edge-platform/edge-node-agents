#!/bin/bash

# This script adds 'ALLOW_MISSING_CLIENTS' env to intel-southbound-handler with cluster-agent client
# info so that intel-southbound-handler can skip rbac for cluster-agent

DEPLOYMENT_NAME="intel-infra-provider-southbound"
NAMESPACE="default"
ENV_NAME="ALLOW_MISSING_AUTH_CLIENTS"
ENV_VALUE="cluster-agent"

kubectl get deployment $DEPLOYMENT_NAME -n $NAMESPACE -o json | \
jq --arg name "$ENV_NAME" --arg value "$ENV_VALUE" '
  if (.spec.template.spec.containers[].env // [] | map(select(.name == $name)) | length) == 0 then
    .spec.template.spec.containers[].env += [{"name": $name, "value": $value}]
  else
    .
  end' | kubectl apply -f -
