#!/bin/bash
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

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

NAMESPACE="grpc-loadtest"
QPS=${QPS:-100}
DURATION=${DURATION:-60s}
CONNECTIONS=${CONNECTIONS:-4}

echo "=== gRPC Load Test Framework ==="
echo "QPS: $QPS"
echo "Duration: $DURATION"
echo "Connections: $CONNECTIONS"
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed"
    exit 1
fi

# Create namespace if it doesn't exist
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Function to wait for deployment to be ready
wait_for_deployment() {
    local deployment=$1
    echo "Waiting for $deployment to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/$deployment -n "$NAMESPACE" || {
        echo "Error: $deployment failed to become ready"
        kubectl describe deployment/$deployment -n "$NAMESPACE"
        exit 1
    }
}

# Function to run load test
run_load_test() {
    local mode=$1
    local job_name="loadtest-client-$mode"
    local target=$2
    
    echo ""
    echo "=== Running $mode test ==="
    
    # Create job manifest
    cat > /tmp/$job_name.yaml <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: $job_name
  namespace: $NAMESPACE
spec:
  template:
    metadata:
      labels:
        app: loadtest-client
        mode: $mode
    spec:
      containers:
      - name: client
        image: loadtest-client:latest
        imagePullPolicy: IfNotPresent
        command: ["./client"]
        args:
        - "-target=$target"
        - "-mode=$mode"
        - "-qps=$QPS"
        - "-duration=$DURATION"
        - "-connections=$CONNECTIONS"
        resources:
          requests:
            cpu: "1500m"
            memory: "1000Mi"
          limits:
            cpu: "1500m"
            memory: "1000Mi"
      restartPolicy: Never
  backoffLimit: 1
EOF

    # Apply job
    kubectl apply -f /tmp/$job_name.yaml
    
    # Wait for job to complete
    echo "Waiting for job to complete..."
    kubectl wait --for=condition=complete --timeout=600s job/$job_name -n "$NAMESPACE" || {
        echo "Error: Job failed or timed out"
        kubectl logs job/$job_name -n "$NAMESPACE" || true
        exit 1
    }
    
    # Show logs
    echo ""
    echo "=== Results for $mode ==="
    kubectl logs job/$job_name -n "$NAMESPACE"
    
    # Cleanup
    kubectl delete job/$job_name -n "$NAMESPACE" --ignore-not-found=true
    rm -f /tmp/$job_name.yaml
}

# Deploy servers
echo "=== Deploying servers ==="
kubectl apply -f k8s/baseline-server.yaml
kubectl apply -f k8s/envoy-server.yaml
kubectl apply -f k8s/xds-server.yaml

# Wait for servers to be ready
wait_for_deployment "loadtest-server-baseline"
wait_for_deployment "loadtest-server-envoy"
wait_for_deployment "loadtest-server-xds"

# Wait a bit for services to stabilize
echo "Waiting for services to stabilize..."
sleep 10

# Run tests
run_load_test "baseline" "loadtest-server-baseline.$NAMESPACE.svc.cluster.local:8080"
run_load_test "envoy" "loadtest-server-envoy.$NAMESPACE.svc.cluster.local:8080"
run_load_test "xds" "xds:///loadtest-server-xds.$NAMESPACE.svc.cluster.local:8080"

echo ""
echo "=== All tests completed ==="
echo "To view server logs:"
echo "  kubectl logs -f deployment/loadtest-server-baseline -n $NAMESPACE"
echo "  kubectl logs -f deployment/loadtest-server-envoy -n $NAMESPACE"
echo "  kubectl logs -f deployment/loadtest-server-xds -n $NAMESPACE"

