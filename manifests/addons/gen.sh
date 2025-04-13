#!/usr/bin/env bash

# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


WORKDIR=$(dirname "$0")
WORKDIR=$(cd "$WORKDIR"; pwd)

set -eux

# script set up the plain text rendered

KUBERNETES=${WORKDIR}/../kubernetes
mkdir -p "${KUBERNETES}"
kubectl create ns dubbo-system
if [ $? -ne 0 ];then
  kubectl delete ns dubbo-system ; kubectl create ns dubbo-system
fi

# Set up prometheus
helm template prometheus prometheus \
  --namespace dubbo-system \
  --version 27.5.1 \
  --repo https://prometheus-community.github.io/helm-charts \
  -f "${WD}/values-prometheus.yaml" \
  > "${ADDONS}/prometheus.yaml"