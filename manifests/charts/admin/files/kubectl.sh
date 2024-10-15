#!/bin/bash

set -eux

JAEGER="https://raw.githubusercontent.com/istio/istio/release-1.23/samples/addons/jaeger.yaml"

kubectl apply -f $JAEGER

kubectl get namespace istio-system >/dev/null 2>&1 || kubectl create namespace istio-system
