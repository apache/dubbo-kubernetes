# Dubbo Helm Charts

The charts in this directory are the **single source of truth** for every
supported install path:

- `base`: cluster-wide resources (CRDs). Install first.
- `dubbod`: the dubbod control plane (and optional CNI DaemonSet).

Both install paths render exactly these charts:

1. **Helm**: `helm install` the charts directly (see each chart's README).
2. **dubboctl install**: the CLI embeds this directory and renders it locally,
   layered with a profile from `manifests/profiles`.

Any manifest change must be made here; do not fork rendered output elsewhere.
CI lints and renders both charts on every PR (`make lint-helm`), and the kind
smoke test installs them end-to-end (`make test-e2e`).
