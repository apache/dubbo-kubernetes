#!/bin/bash

JAEGER="https://raw.githubusercontent.com/istio/istio/release-1.23/samples/addons/jaeger.yaml"
PROMETHEUS="https://prometheus-community.github.io/helm-charts"
ISTIO="https://istio-release.storage.googleapis.com/charts"

helm pull kube-prometheus-stack --repo $PROMETHEUS --version 65.2.0 --untar
cp -r dashboards/ kube-prometheus-stack/templates/grafana/dashboards-1.14/
cd kube-prometheus-stack
helm install kube-prometheus-stack -f values.yaml .

kubectl apply -f $JAEGER

helm repo add istio $ISTIO
helm repo update

kubectl create namespace istio-system || true

helm install istio-base istio/base -n istio-system --set defaultRevision=default
helm install istiod istio/istiod -n istio-system