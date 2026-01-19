#!/bin/bash

set -ex

rm -f "../../manifests/charts/base/files/crd-all.gen.yaml"
cp "../../api/kubernetes/customresourcedefinitions.gen.yaml" "../../manifests/charts/base/files/crd-all.gen.yaml"