# Virtual machine workloads

`WorkloadEntry` registers a VM, bare-metal process, or other non-Kubernetes workload as a mesh endpoint. `ServiceEntry` gives the selected workloads a stable service hostname.

## Register a workload

Set `spec.address` in `vm-service.yaml` to an address reachable from mesh clients, then apply it:

```console
kubectl apply -f samples/vm/vm-service.yaml
```

The workload port overrides the matching service port by name. `network`, `locality`, `weight`, and `serviceAccount` are propagated with the endpoint. A missing weight defaults to `1`.

## Publish health

VM health agents or external monitors can update the standard `Ready` or `Healthy` condition. A false or unknown condition publishes the endpoint as unhealthy through xDS; no condition preserves the backward-compatible healthy default.

```console
kubectl patch workloadentry payment-vm-01 --subresource=status --type=merge \
  -p '{"status":{"conditions":[{"type":"Ready","status":"False","reason":"HealthCheckFailed"}]}}'
```

Restore the endpoint after recovery:

```console
kubectl patch workloadentry payment-vm-01 --subresource=status --type=merge \
  -p '{"status":{"conditions":[{"type":"Ready","status":"True","reason":"HealthCheckPassed"}]}}'
```

Inspect the state published by the control plane:

```console
kubectl -n dubbo-system port-forward deploy/dubbod 18080:8080
curl http://127.0.0.1:18080/debug/endpointz
```

The control plane manages discovery and routing. The VM process, network reachability, workload certificate provisioning, and the component that reports its health remain deployment responsibilities.
