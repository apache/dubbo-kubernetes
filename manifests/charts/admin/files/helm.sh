#!/bin/bash

set -eux



PROMETHEUS="https://prometheus-community.github.io/helm-charts"
ISTIO="https://istio-release.storage.googleapis.com/charts"

helm pull kube-prometheus-stack --repo "$PROMETHEUS" --version 65.2.0 --untar && \
cp -r "/files/dashboards/" kube-prometheus-stack/templates/grafana/dashboards-1.14/ && \
cd kube-prometheus-stack && \
helm install kube-prometheus-stack -f values.yaml . && \
helm repo add istio "$ISTIO" && \
helm repo update && \
helm install istio-base istio/base -n istio-system --set defaultRevision=default && \
helm install istiod istio/istiod -n istio-system
