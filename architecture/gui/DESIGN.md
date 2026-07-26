# dubbod Console — Information Architecture & Data Mapping

Source of truth for the embedded GUI. **Every field listed here was verified
against a live cluster**, not against the Go type definitions — a struct field
that exists is not the same as a value the control plane actually produces. The
console ships no demo data, no mock mode, and no widget for a metric this build
does not export.

## 1. Backend APIs

| Endpoint | Handler | Payload |
|---|---|---|
| `GET {gui}/api/overview` | `bootstrap.(*Server).guiOverviewHandler` | `guiOverview` |
| `GET {gui}/api/logs?kind=dubbod\|gateway&namespace=&name=&tail=` | `bootstrap.(*Server).guiLogsHandler` | `guiLogsResponse` |
| `GET {gui}/api/metrics` | `bootstrap.(*Server).guiMetricsHandler` | JSON snapshot of `pkg/monitoring.GetRegistry()` |

Overview polls every 15 s. Metrics polls at a user-selected interval
(5/10/30/60 s) into a browser-side ring buffer. Logs are fetched on demand,
`tail` capped at 2000 by the backend.

`api/logs` needs `pods/log` in the dubbod ClusterRole. Without it the API
returns `unknown (get pods <name>)` for every pod and the Logs module is dead —
see `manifests/charts/dubbod/templates/clusterrole.yaml`.

## 2. Modules

Five modules, each backed by a live feed. There are no placeholder pages: a
module that cannot be populated does not exist in the nav.

```
MESH     Overview   api/overview  — control plane pods, managed gateways, registries
         Services   api/overview  — services[] with filter/sort/drawer
OBSERVE  Pipeline   api/metrics   — config push path, counters, log pressure
         Logs       api/logs      — pod log tail for owned deployments
SYSTEM   Runtime    api/overview  — server addresses, mesh identity, preferences
```

Earlier builds shipped up to eleven nav entries, several of which rendered a
"capability not connected" placeholder (Traffic, Events, Alerts, Topology) and
several of which were a single table apiece (Gateways, Registries, Config).
The placeholders are gone; the single-table pages are folded into Overview.
Their old hash routes redirect (`LEGACY_ROUTES` in `app.js`).

## 3. Field mapping

### Overview
Three tables, nothing else. The page answers "is the control plane up, are its
gateways up, are its registries synced" and defers every count to the page that
owns the underlying list.

| UI element | Source (`guiOverview`) |
|---|---|
| Health chip | derived: `status.*` flags ∧ registry `synced` ∧ gateway `isReady` ∧ some instance ready |
| Control plane table | `instances[].{name,namespace,ip,isReady}` |
| Managed gateways table | `gatewayInstances[].{name,namespace,gatewayClass,gatewayName,readyReplicas,desiredReplicas,isReady}` |
| Service registries table | `registries[].{provider,cluster,synced}` |
| Top bar | `updatedAt` + the health dot |

`status.*` still drives the health chip through `healthRollup`; it is summarised
rather than rendered as a seven-segment rail. `counts.*` and `configKinds[]` are
carried by the API but no longer surfaced — the numbers repeated what the tables
and the Services page already show.

`instances[]` comes from pods labelled `app=dubbod`, matching the chart's
deployment template. An earlier revision queried `app=dubbo-control-plane`,
matched nothing, and fell through to a synthesised row with a hard-coded
`localhost` address — the table looked real and was not.

### Services
Direct rendering of `services[]`. `serviceAccounts` is reported but is 0 for
every service on a stock install, so it appears only in the drawer.

### Pipeline (all series from `api/metrics`)

The metric registry this build exports — verified by enumerating a live
snapshot — is exactly these thirteen families:

```
dubbod_uptime_seconds            dubbod_info
dubbod_xds                       dubbod_services
dubbod_inbound_updates           dubbod_push_triggers
dubbod_xds_pushes                dubbod_log_messages_total
dubbod_debounce_time             dubbod_pushcontext_init_seconds
dubbod_proxy_queue_time          dubbod_xds_send_time
dubbod_proxy_convergence_time
```

Anything else declared in `pkg/xds/monitoring.go` (`dubbod_xds_push_time`,
`dubbod_total_xds_rejects`, `dubbod_total_xds_internal_errors`,
`dubbod_xds_*_reject`, `dubbod_xds_config_size_bytes`, …) is **not** in the
snapshot. A previous version of this page drew a P50/P95/P99 row and a latency
histogram from `dubbod_xds_push_time` and an error tile from the reject
counters; all of them read as `–` or a reassuring `0` that no metric backed.
Those widgets are gone.

| UI element | Metric family |
|---|---|
| Uptime tile | `dubbod_uptime_seconds`, `dubbod_info{version}` |
| xDS connections tile + trend | `dubbod_xds` |
| Services known tile + trend | `dubbod_services` |
| xDS send errors tile | `dubbod_xds_pushes{type=*_senderr}` |
| Push path stages | `dubbod_debounce_time`, `dubbod_pushcontext_init_seconds`, `dubbod_proxy_queue_time`, `dubbod_xds_send_time`, `dubbod_proxy_convergence_time` |
| Where the time goes | mean of each stage above |
| Inbound updates + rate | `dubbod_inbound_updates{type}` |
| Push triggers | `dubbod_push_triggers{type}` |
| Log pressure | `dubbod_log_messages_total{level,scope}` |

Two naming traps worth stating plainly:

- `dubbod_xds_pushes` is **not** a push counter. Its help text is "Dubbod build
  and send errors for lds, rds, cds and eds" and its label values are
  `cds_senderr` / `eds_senderr` / `lds_senderr` / `rds_senderr`. The console
  labels it "xDS send errors".
- `dubbod_push_triggers` has a label value spelled `depdendentresource` in
  `dubbod/discovery/pkg/xds/monitoring.go`. The console renders label values
  verbatim rather than prettifying them, so the typo is visible; fixing it is a
  breaking change for anyone's existing dashboards and is left alone.

**Histogram reading.** The pipeline histograms use very coarse bounds
(0.01, 0.1, 1, 3, 5, 10, 20, 30 s). On a quiet cluster every debounce sample
lands in the `≤1 s` bucket, so an interpolated P50 reports ~500 ms while the
true mean is ~127 ms — the percentile would be an artefact of the bucket
layout, not a measurement. The page therefore leads with the exact
`sum / count` mean and uses the buckets only to state the band every sample
actually fell in ("all ≤ 1.0s"). Stages with `count == 0` say "no samples" and
name what would produce one, instead of drawing an empty chart.

Trend history is a browser-side ring buffer sampled while the page is open and
is labelled as such — the backend keeps no time series.

### Logs
`guiLogsResponse` rendered directly. Level filter and highlight are client-side
parsing of the real log text. Two display fixes on real output: dxgate emits
ANSI colour codes (stripped), and because the API requests
`PodLogOptions.Timestamps`, every line carries a Kubernetes timestamp followed
by the process's own — the duplicate outer one is dropped.

### Runtime
`server.*`, `mesh.*`, `clusterId`, `namespace`, `podName`, `version`. When
dubbod runs off-cluster the pod field reads "not running in a pod" rather than
inventing an identity.

## 4. Drill-down

Global → module → entity drawer. One drawer component serves every entity type
(service, gateway, registry). Drawer actions expose only what the backend can do
today: fetch logs, copy hostname.

## 5. Assets

`index.html`, `styles.css`, `runtime.js`, `charts.js`, `app.js`, `vendor.js`
and the logo are embedded in the dubbod binary via `resources/fs.go`. `vendor.js`
is a pinned preact + htm bundle regenerated by `tools/gui/vendor.sh`; it exists
so the console loads nothing from a CDN and works in an air-gapped cluster.
The console issues no cross-origin requests — this is asserted by the browser
verification pass, not assumed.

## 6. What the control plane still cannot show

Not placeholder pages — just the honest boundary of the API:

| Capability | Required feed |
|---|---|
| Per-request telemetry (RPS, error rate, latency per service/edge) | metrics ingestion from proxyless workloads aggregated by service pair; the xds-api wire protocol carries no per-request stats |
| Kubernetes events | an `api/events` feed backed by an events informer |
| History beyond the session | a TSDB-backed query API or Prometheus proxy |
| Per-resource config specs | an `api/configs?kind=` listing endpoint |
| Service endpoint listing | an `api/services/{host}/endpoints` endpoint |
| Multi-cluster | a multi-cluster overview API (`clusterId` is a single value) |

Where a boundary affects a visible module, the module says so in one line
(Services notes that request telemetry is out of scope). Nothing gets a page of
its own for being absent.

## 7. Design system (tokens in `styles.css`)

- Light theme is a pale-blue wash falling to white (`--canvas-grad`, a fixed
  `linear-gradient` on `body`); the dark theme mirrors it with a flat surface.
  `data-theme` attribute + `prefers-color-scheme` fallback, persisted.
- Type: one sans stack (Inter-first) carries all chrome and prose. Monospace is
  reserved for values that are literally identifiers — pod names, IPs,
  hostnames, addresses, ports, log lines, axis ticks. Tabular numerals on any
  figure that changes in place.
- Status semantics are fixed everywhere (chip, dot, rail, chart):
  ok `--ok`, warning `--warn`, error `--err`, unknown `--unk`. Status is never
  colour-alone.
- Chart series palette, validated for CVD and contrast on both surfaces:
  light `#2a78d6 #1baf7a #eda100 #008300`, dark `#3987e5 #199e70 #c98500 #008300`.
  One y-axis per chart; legend plus direct end-labels for multi-series.
- States implemented everywhere: skeleton, error banner with retry, empty,
  no-samples. Reduced motion respected.
- Responsive: nav rail collapses to icons below 1200 px; grids and tables stay
  usable at 1280×800.
