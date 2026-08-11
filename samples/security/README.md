# Security policy sample

This sample exercises the security path shared by grpc-engine and dxgate:
PERMISSIVE-to-STRICT mTLS migration, JWT validation and claim headers, ALLOW,
DENY, AUDIT, and CUSTOM external authorization.

Install the provider and mesh defaults with the control plane:

```bash
helm upgrade --install dubbod ./manifests/charts/dubbod \
  --namespace dubbo-system \
  -f samples/security/helm-values.yaml
kubectl apply -f samples/security/policies.yaml
```

Start with `PERMISSIVE` while plaintext clients are being migrated. Change the
`PeerAuthentication` mode to `STRICT` after every caller has a workload
certificate. The CUSTOM policy requires the `opa` service named in
`helm-values.yaml`; remove that policy when no external authorizer is running.

