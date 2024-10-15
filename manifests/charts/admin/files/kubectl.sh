#!/bin/bash

set -eux

JAEGER="https://raw.githubusercontent.com/istio/istio/release-1.23/samples/addons/jaeger.yaml"

kubectl apply -f $JAEGER

kubectl create namespace istio-system
if [ $? -eq 0 ]; then
  # shellcheck disable=SC2105
  continue
fi