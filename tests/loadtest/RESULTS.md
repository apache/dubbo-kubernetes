# Load Test Results Comparison

## Test Results Summary

```
Load Test Results Comparison
├── Baseline Mode
│   ├── Duration: 1m0.008528758s
│   ├── Total Requests: 22,285
│   ├── Successful: 22,285
│   ├── Errors: 0
│   ├── Average QPS: 371.36
│   └── Latency Percentiles
│       ├── P50:  600.411µs
│       ├── P90:  1.102656ms
│       ├── P95:  1.369779ms
│       ├── P99:  2.599231ms
│       └── P99.9: 25.780049ms
│
├── xDS Mode
│   ├── Duration: 1m30.976317937s (includes 30s connection setup)
│   ├── Total Requests: 23,920
│   ├── Successful: 23,920
│   ├── Errors: 0
│   ├── Average QPS: 262.93
│   └── Latency Percentiles
│       ├── P50:  547.081µs
│       ├── P90:  1.05883ms
│       ├── P95:  1.333871ms
│       ├── P99:  2.521575ms
│       └── P99.9: 8.311473ms
│
└── Envoy Mode
    ├── Duration: 1m0.009001221s
    ├── Total Requests: 22,254
    ├── Successful: 22,254
    ├── Errors: 0
    ├── Average QPS: 370.84
    └── Latency Percentiles
        ├── P50:  2.439575ms
        ├── P90:  3.888403ms
        ├── P95:  4.390359ms
        ├── P99:  7.067849ms
        └── P99.9: 41.936625ms
```

## Performance Comparison

### QPS Comparison
```
Baseline:  ████████████████████████████████████████ 371.36 QPS
Envoy:     ████████████████████████████████████████ 370.84 QPS
xDS:       ████████████████████████████████████     262.93 QPS
```

### Latency Comparison (P50)
```
Baseline:  ████ 600.411µs
xDS:       ████ 547.081µs
Envoy:     ████████████ 2.439575ms
```

### Latency Comparison (P99)
```
Baseline:  ████████ 2.599231ms
xDS:       ████████ 2.521575ms
Envoy:     ████████████████████ 7.067849ms
```

### Latency Comparison (P99.9)
```
xDS:       ████ 8.311473ms
Baseline:  ████████████████████████ 25.780049ms
Envoy:     ████████████████████████████████████████████████ 41.936625ms
```

## Key Observations

1. **QPS Performance**
   - Baseline and Envoy modes achieve similar QPS (~371 QPS)
   - xDS mode has lower QPS (262.93 QPS) due to connection setup overhead

2. **Latency (P50/P99)**
   - xDS mode has the lowest latency at P50 (547µs)
   - Baseline and xDS have similar P99 latency (~2.5ms)
   - Envoy mode has higher latency (2.4ms P50, 7ms P99)

3. **Tail Latency (P99.9)**
   - xDS mode has the best tail latency (8.3ms)
   - Baseline mode has moderate tail latency (25.8ms)
   - Envoy mode has the highest tail latency (41.9ms)

4. **Connection Setup**
   - xDS mode requires ~30 seconds for connection establishment
   - Baseline and Envoy modes start immediately

## Test Configuration

- **QPS**: 100 requests/second per connection
- **Connections**: 4 concurrent connections
- **Duration**: 60 seconds (actual test time)
- **Payload**: 0 bytes

## Summary
- Highest Throughput: Baseline / Envoy
- Lowest Latency: xDS
- Most Balanced Overall Performance: xDS (when ignoring startup overhead)

The Baseline mode represents the raw performance upper bound, while the xDS mode delivers the best latency and overall efficiency in a real service mesh environment.
