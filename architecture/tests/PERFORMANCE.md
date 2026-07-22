<!--
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# Performance Engineering

The performance suite keeps two layers separate:

1. Control-plane Go benchmarks measure indexing, client selection, and xDS
   response generation without Kubernetes or network noise.
2. Cluster load tests measure end-to-end request latency and resource usage.
   Those results must identify the cluster, traffic generator, workload, and
   observability setup; they must not be compared directly with Go benchmark
   timings.

## Control-plane scale matrix

`BenchmarkProxylessPushScale` exercises the real targeted endpoint-update path:

```text
clientsForPush -> pushConnection -> EDS generation -> DiscoveryResponse send
```

Every synthetic proxyless connection watches every generated service. A
single service endpoint update therefore selects every connection and emits
one EDS response per connection. The matrix changes one dimension at a time
from the `100 services / 10 endpoints / 100 connections` baseline:

| Dimension | Cases |
|---|---|
| Services | 10, 100, 1,000 |
| Endpoints per service | 1, 10, 100 |
| Connections | 10, 100, 1,000 |

`BenchmarkProxylessFullPushScale` uses the same dimensions but forces a full
push that emits every watched service. Its baseline uses 10 connections, and
its largest cases cap the generated work at 100,000 endpoint copies per
operation:

| Dimension | Cases |
|---|---|
| Services | 10, 100, 1,000 |
| Endpoints per service | 1, 10, 100 |
| Connections | 1, 10, 100 |

Both benchmarks report `ns/op`, `B/op`, `allocs/op`, the three scale values,
and the number of generated endpoint copies per operation.
`ns/op` is the serial cost of completing one synthetic push wave on the
benchmark CPU. Production pushes may execute concurrently, so this number is
for regression and scaling comparisons rather than a direct production SLO.

The existing `BenchmarkClientsForPushProxylessTargetedScale` separately
measures connection selection through 100,000 connected clients, while
`BenchmarkKRTFetch` measures indexed and label-scan lookups over 10,000
workloads.

## Running benchmarks

Run the representative targeted and full-push scale cases used by pull-request
CI:

```bash
make benchmark-smoke
```

Run the complete matrix with repeated samples:

```bash
make benchmark BENCHMARK_COUNT=5 BENCHMARK_TIME=1s BENCHMARK_CPU=1
```

The Make variables can narrow the benchmark expression or adjust measurement
time and CPU count:

```bash
make benchmark \
  BENCHMARK_PATTERN=BenchmarkProxylessPushScale \
  BENCHMARK_COUNT=10 \
  BENCHMARK_TIME=2s \
  BENCHMARK_CPU=1
```

Use the same Go version, CPU model, power mode, `BENCHMARK_CPU`, and background
load for baseline and candidate runs. Keep at least five samples and compare
them with `benchstat`; do not draw a capacity conclusion from one run.

## Continuous evidence

The main CI workflow runs `make benchmark-smoke` to catch fixture failures and
gross scale regressions on every change. The scheduled `Performance
Benchmarks` workflow runs five samples of the complete matrix each week and
retains the raw output plus runner details for 30 days. Shared runner variance
makes a universal absolute latency threshold unreliable; compare like-for-like
runs and investigate changes in both time and allocations.

## Cluster load tests

Use `samples/httpbin/httpbin.yaml` as the minimal data-plane target and the
Prometheus sample under `samples/addons/` for control-plane metrics. Record at
least request rate, p50/p95/p99 latency, error rate, CPU, memory, connected xDS
clients, services, and endpoints. Change one scale dimension at a time and
include warm-up, steady-state, and configuration-churn phases in the result.
