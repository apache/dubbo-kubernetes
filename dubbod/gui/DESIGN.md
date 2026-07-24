# dubbod Console — Information Architecture & Data Mapping

This document is the source of truth for the embedded GUI redesign. Every field
rendered by the GUI maps to a real backend structure listed here; capabilities
the backend does not provide yet are explicitly marked and rendered as
capability gates (never as fabricated data).

## 1. Real data sources

| Endpoint | Handler | Payload |
|---|---|---|
| `GET {gui}/api/overview` | `bootstrap.(*Server).guiOverviewHandler` | `guiOverview` |
| `GET {gui}/api/logs?kind=dubbod\|gateway&namespace=&name=&tail=` | `bootstrap.(*Server).guiLogsHandler` | `guiLogsResponse` |
| `GET {gui}/api/metrics` | `bootstrap.(*Server).guiMetricsHandler` (added with this redesign) | JSON snapshot of the in-process Prometheus registry (`pkg/monitoring.GetRegistry()`) |

Polling: overview every 15 s; metrics at a user-selected interval (5/10/30/60 s).
Log fetches are on demand with `tail` capped at 2000 by the backend.

## 2. Navigation & module map

```
OBSERVE   Overview        real  api/overview
          Metrics         real  api/metrics (control-plane Prometheus registry)
          Logs            real  api/logs
MESH      Services        real  overview.services
          Gateways        real  overview.gatewayInstances + counts
          Registries      real  overview.registries
          Config          real  overview.configKinds (counts only)
PLANNED   Traffic         gated — no request-level telemetry feed exists
          Events          gated — no Kubernetes events feed exists
SYSTEM    Runtime         real  overview.server / mesh / version / instance
```

"PLANNED" items stay in the nav so the IA covers the target scope, but each
renders a capability gate stating exactly which backend feed is missing (see §6).

## 3. Field mapping (frontend ⇄ backend)

### Overview page
| UI element | Source (`guiOverview`) |
|---|---|
| Readiness rail (7 segments) | `status.{xdsServerReady,cachesSynced,servicesSynced,configSynced,proxylessSynced,injectorReady,validationReady}` |
| Health rollup card | derived: readiness flags ∧ registry `synced` ∧ gateway `isReady` |
| Stat tiles | `counts.{services,xdsConnections,registries,gateways,httpRoutes,authorizationPolicies,…}` |
| Control-plane table | `instances[].{name,namespace,ip,isReady}` |
| Gateway table | `gatewayInstances[].{name,namespace,gatewayClass,gatewayName,readyReplicas,desiredReplicas,isReady}` |
| Registry table | `registries[].{provider,cluster,synced}` |
| Header cluster / ns / updated | `clusterId`, `namespace`, `updatedAt` |

### Metrics page (all series from `api/metrics`)
| UI element | Metric family |
|---|---|
| Uptime tile | `dubbod_uptime_seconds` |
| Connections tile + trend | `dubbod_xds` |
| Services tile + trend | `dubbod_services` |
| Push counter + per-type rate lines | `dubbod_xds_pushes{type}` (client-side derivative) |
| Push-time distribution + P50/P95/P99 | `dubbod_xds_push_time` histogram buckets |
| Errors tile | `dubbod_total_xds_internal_errors`, `dubbod_total_xds_rejects`, `dubbod_xds_*_reject` |
| Version chip | `dubbod_info{version}` |

Trend history is a client-side ring buffer sampled while the page is open and is
labeled "since page open" — the backend keeps no time series (marked in §6).

### Services / Gateways / Registries / Config / Logs
Direct rendering of `guiService`, `guiDubbodInstance`, `guiRegistry`,
`guiConfigKind`, `guiLogsResponse` fields. Log level filter/highlight is
client-side parsing of the real log text (`info|warn|error` tokens + timestamps
from `PodLogOptions.Timestamps: true`).

## 4. Drill-down model

Global → module → entity drawer. One right-hand drawer component serves every
entity type (service, gateway, registry, control plane, config kind) so
drill-down behaves identically everywhere. Drawer actions only expose operations
the backend can execute today: fetch logs (dubbod/gateway), open metrics, copy
hostname/address.

## 5. Mock mode

`?mock=1` (or the Runtime page toggle, persisted in `localStorage`) switches the
data layer to `mock.js`. Mock payloads are structurally identical to
`guiOverview`, `guiLogsResponse`, and the `api/metrics` JSON — same field names,
same types — so the UI cannot drift from the real contract. A persistent banner
marks mock mode; it is never on by default.

## 6. Capability ledger (missing backend feeds)

| Capability | Blocked UI | Required feed |
|---|---|---|
| Request-level telemetry (RPS, error rate, P95/P99 per service/edge) | Traffic page; service KPI columns | metrics ingestion from proxyless workloads (e.g. scrape/OTLP) aggregated by service pair |
| Kubernetes events | Events page; event timeline in drawers | `api/events` backed by an events informer |
| Historical time series | Metrics page beyond session buffer; time-range pickers | TSDB-backed query API (or Prometheus proxy) |
| Per-resource config listing/spec | Config page rows → resource list & YAML | `api/configs?kind=` listing endpoint |
| Service instance/endpoint listing | Service drawer endpoints tab | `api/services/{host}/endpoints` |
| Auth / RBAC | permission-denied states | any authn on the GUI listener |
| Cluster switching | header cluster picker (single value today) | multi-cluster overview API |

Each gate page states the required feed in these exact terms.

## 7. Design system (tokens live in `styles.css`)

- Light-first "engineering print" aesthetic with a full dark theme
  (`data-theme` attribute + `prefers-color-scheme` fallback; toggle on the
  top bar, persisted).
- Type: system sans for prose; monospace (`ui-monospace` stack) is the identity
  carrier — eyebrows, data values, hostnames, axis ticks; tabular numerals.
- Status semantics are fixed everywhere (chip, dot, rail, chart):
  ok `--ok`, warning `--warn`, error `--err`, offline `--off`, unknown `--unk`.
  Status is never color-alone (icon/text pairing).
- Chart series palette (validated for CVD + contrast on both surfaces with the
  dataviz validator): light `#2a78d6 #1baf7a #eda100 #008300`,
  dark `#3987e5 #199e70 #c98500 #008300`. Sequential ramps use the blue ramp.
  One y-axis per chart; legend + direct labels for multi-series.
- States implemented everywhere: skeleton (first load), error banner with retry,
  empty, capability gate, mock banner. Reduced motion respected.
- Responsive: nav rail collapses to icons < 1200 px; grids and tables stay
  usable at 1280×800.
