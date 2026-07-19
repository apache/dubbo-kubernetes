/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Dubbo control plane console (Preact no-build).
 *
 * Data contract: see dubbod/gui/DESIGN.md. Everything rendered here maps to
 * api/overview, api/logs or api/metrics; modules whose backend feed does not
 * exist yet render an explicit capability gate instead of demo data.
 */

import { mockOverview, mockLogs, mockMetrics } from "./mock.js";
import { Topology } from "./topology.js";
import {
  Sparkline, TrendChart, BucketBars, BreakdownBars,
  bucketQuantile, fmtNumber, fmtDuration,
} from "./charts.js";

const html = window.html;
const { useState, useEffect, useMemo, useRef, useCallback } = window;

const CONFIG = (() => {
  const el = document.getElementById("dubbod-gui-config");
  return el ? JSON.parse(el.textContent) : { basePath: "/gui", product: "Dubbo" };
})();

const API = {
  overview: new URL("api/overview", document.baseURI).toString(),
  logs: new URL("api/logs", document.baseURI).toString(),
  metrics: new URL("api/metrics", document.baseURI).toString(),
};

// --- mock mode --------------------------------------------------------------

const mockKey = "dubbod-gui-mock";
const initMock = () => {
  const param = new URLSearchParams(location.search).get("mock");
  if (param === "1") { try { localStorage.setItem(mockKey, "1"); } catch (_) {} return true; }
  if (param === "0") { try { localStorage.removeItem(mockKey); } catch (_) {} return false; }
  try { return localStorage.getItem(mockKey) === "1"; } catch (_) { return false; }
};
let MOCK = initMock();
const setMock = (on) => {
  try { on ? localStorage.setItem(mockKey, "1") : localStorage.removeItem(mockKey); } catch (_) {}
  location.search = on ? "?mock=1" : "";
};

const getJSON = async (url) => {
  const res = await fetch(url);
  const body = await res.json().catch(() => null);
  if (!res.ok) throw new Error(body?.error || `HTTP ${res.status}`);
  return body;
};

const fetchOverview = () => (MOCK ? Promise.resolve(mockOverview()) : getJSON(API.overview));
const fetchMetrics = () => (MOCK ? Promise.resolve(mockMetrics()) : getJSON(API.metrics));
const fetchLogs = (target) => {
  if (MOCK) return Promise.resolve(mockLogs(target));
  const url = new URL(API.logs);
  url.searchParams.set("kind", target.kind);
  url.searchParams.set("namespace", target.namespace || "");
  url.searchParams.set("name", target.name || "");
  if (target.tail) url.searchParams.set("tail", target.tail);
  return getJSON(url.toString());
};

// --- theme ------------------------------------------------------------------

const themeKey = "dubbod-gui-theme";
const getTheme = () => document.documentElement.dataset.theme || "auto";
const applyTheme = (mode) => {
  if (mode === "auto") {
    delete document.documentElement.dataset.theme;
    try { localStorage.removeItem(themeKey); } catch (_) {}
  } else {
    document.documentElement.dataset.theme = mode;
    try { localStorage.setItem(themeKey, mode); } catch (_) {}
  }
};

// --- routing ----------------------------------------------------------------

const LEGACY_ROUTES = { home: "overview", mesh: "services", meshgateway: "gateways", configuration: "config" };
const ROUTES = [
  "overview", "topology", "metrics", "logs",
  "services", "gateways", "registries", "config",
  "traffic", "events", "alerts", "runtime",
];
const parseRoute = () => {
  let hash = (location.hash || "").replace(/^#\/?/, "");
  hash = LEGACY_ROUTES[hash] || hash;
  return ROUTES.includes(hash) ? hash : "overview";
};

// --- metric helpers ---------------------------------------------------------

const family = (snapshot, name) => snapshot?.families?.find((f) => f.name === name);
const firstValue = (snapshot, name) => family(snapshot, name)?.metrics?.[0]?.value ?? null;
const labeledValues = (snapshot, name, label) => {
  const out = new Map();
  for (const m of family(snapshot, name)?.metrics || []) {
    out.set(m.labels?.[label] ?? "", (out.get(m.labels?.[label] ?? "") || 0) + (m.value || 0));
  }
  return out;
};
const totalValue = (snapshot, name) =>
  (family(snapshot, name)?.metrics || []).reduce((a, m) => a + (m.value || 0), 0);

const HISTORY_MAX = 720;

// --- shared components --------------------------------------------------------

const STATUS_LABEL = { ok: "healthy", warn: "degraded", err: "unhealthy", off: "offline", unknown: "unknown" };
const Dot = ({ status }) => html`<span class=${`dot status-${status}`} aria-hidden="true" />`;
const StatusChip = ({ status, children }) => html`
  <span class=${`chip chip-${status}`}><${Dot} status=${status} />${children ?? STATUS_LABEL[status]}</span>
`;

const Eyebrow = ({ children }) => html`<div class="eyebrow">${children}</div>`;
const SectionTitle = ({ children, aside }) => html`
  <div class="section-head">
    <h2 class="section-title">${children}</h2>
    ${aside && html`<div class="section-aside">${aside}</div>`}
  </div>
`;

const StatTile = ({ label, value, hint, status, spark, sparkColor, onClick }) => html`
  <div class=${`tile ${onClick ? "tile-click" : ""}`} onClick=${onClick} role=${onClick ? "button" : undefined} tabIndex=${onClick ? 0 : undefined}>
    <div class="tile-top">
      <span class="tile-label">${label}</span>
      ${status && html`<${Dot} status=${status} />`}
    </div>
    <div class="tile-value">${value}</div>
    <div class="tile-foot">
      ${hint && html`<span class="tile-hint">${hint}</span>`}
      ${spark && html`<${Sparkline} points=${spark} colorVar=${sparkColor || "--series-1"} />`}
    </div>
  </div>
`;

const EmptyState = ({ title, children }) => html`
  <div class="empty">
    <div class="empty-title">${title}</div>
    ${children && html`<div class="empty-body">${children}</div>`}
  </div>
`;

const ErrorBanner = ({ error, onRetry }) => html`
  <div class="banner banner-err" role="alert">
    <span class="banner-text">Request failed: ${error}</span>
    ${onRetry && html`<button class="btn btn-small" onClick=${onRetry}>Retry</button>`}
  </div>
`;

const Skeleton = ({ h = 120 }) => html`<div class="skeleton" style=${{ height: `${h}px` }} />`;

const CapabilityGate = ({ title, purpose, missing, wouldShow }) => html`
  <div class="gate">
    <div class="gate-mark">CAPABILITY NOT CONNECTED</div>
    <h2 class="gate-title">${title}</h2>
    <p class="gate-purpose">${purpose}</p>
    <div class="gate-block">
      <div class="gate-block-label">Would display</div>
      <ul>${wouldShow.map((w) => html`<li key=${w}>${w}</li>`)}</ul>
    </div>
    <div class="gate-block gate-block-missing">
      <div class="gate-block-label">Required backend feed</div>
      <p>${missing}</p>
    </div>
    <p class="gate-foot">This module intentionally renders no demo data. The full capability ledger lives in <span class="mono">dubbod/gui/DESIGN.md</span>.</p>
  </div>
`;

const Field = ({ label, children, mono }) => html`
  <div class="field">
    <div class="field-label">${label}</div>
    <div class=${`field-value ${mono ? "mono" : ""}`}>${children ?? "–"}</div>
  </div>
`;

const copyText = (text) => { try { navigator.clipboard?.writeText(text); } catch (_) {} };

// --- drawer -------------------------------------------------------------------

const Drawer = ({ item, onClose, onOpenLogs, navigate }) => {
  useEffect(() => {
    const onKey = (e) => { if (e.key === "Escape") onClose(); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  if (!item) return null;
  const { type, data } = item;

  return html`
    <div class="drawer-overlay" onClick=${(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <aside class="drawer">
        <div class="drawer-head">
          <div>
            <${Eyebrow}>${type}</${Eyebrow}>
            <div class="drawer-title">${item.title}</div>
          </div>
          <button class="btn btn-ghost" onClick=${onClose} aria-label="Close">✕</button>
        </div>
        <div class="drawer-body">
          ${item.status && html`<div class="drawer-status"><${StatusChip} status=${item.status} /></div>`}

          ${type === "service" && html`
            <${Field} label="Hostname" mono>${data.hostname}</${Field}>
            <${Field} label="Namespace">${data.namespace}</${Field}>
            <${Field} label="Registry">${data.registry}</${Field}>
            <${Field} label="Ports" mono>${data.ports}</${Field}>
            <${Field} label="Exposure">${data.exposure}</${Field}>
            <${Field} label="Default address" mono>${data.defaultAddress || "–"}</${Field}>
            <${Field} label="Service accounts">${data.serviceAccounts}</${Field}>
            <${Field} label="Mesh external">${data.meshExternal ? "yes" : "no"}</${Field}>
            <div class="drawer-gaps">
              <span class="chip chip-gate">endpoint list n/a</span>
              <span class="chip chip-gate">request metrics n/a</span>
            </div>
            <div class="drawer-actions">
              <button class="btn" onClick=${() => { onClose(); navigate("topology"); }}>Locate in topology</button>
              <button class="btn btn-ghost" onClick=${() => copyText(data.hostname)}>Copy hostname</button>
            </div>
          `}

          ${type === "gateway" && html`
            <${Field} label="Deployment" mono>${data.name}</${Field}>
            <${Field} label="Gateway">${data.gatewayName || "–"}</${Field}>
            <${Field} label="Namespace">${data.namespace}</${Field}>
            <${Field} label="Gateway class">${data.gatewayClass || "–"}</${Field}>
            <${Field} label="Replicas" mono>${data.readyReplicas || 0} / ${data.desiredReplicas || 0} ready</${Field}>
            <div class="drawer-actions">
              <button class="btn" onClick=${() => onOpenLogs({ kind: "gateway", name: data.name, namespace: data.namespace })}>View logs</button>
              <button class="btn btn-ghost" onClick=${() => { onClose(); navigate("topology"); }}>Locate in topology</button>
            </div>
          `}

          ${type === "registry" && html`
            <${Field} label="Provider">${data.provider}</${Field}>
            <${Field} label="Cluster" mono>${data.cluster}</${Field}>
            <${Field} label="Sync">${data.synced ? "synced" : "syncing"}</${Field}>
          `}

          ${type === "controlplane" && html`
            <${Field} label="Version" mono>${data.version || "–"}</${Field}>
            <${Field} label="Namespace">${data.namespace}</${Field}>
            ${(data.instances || []).map((i) => html`
              <div class="drawer-row" key=${i.name}>
                <${Dot} status=${i.isReady ? "ok" : "err"} />
                <span class="mono drawer-row-main">${i.name}</span>
                <span class="mono drawer-row-side">${i.ip}</span>
              </div>
            `)}
            <div class="drawer-actions">
              <button class="btn" onClick=${() => onOpenLogs({ kind: "dubbod", name: "dubbod", namespace: data.namespace })}>View logs</button>
              <button class="btn btn-ghost" onClick=${() => { onClose(); navigate("metrics"); }}>Open metrics</button>
            </div>
          `}

          ${type === "config" && html`
            <${Field} label="Kind" mono>${data.kind}</${Field}>
            <${Field} label="Count">${data.count}</${Field}>
            <${Field} label="Purpose">${data.description}</${Field}>
            <div class="drawer-gaps"><span class="chip chip-gate">per-resource listing n/a</span></div>
          `}
        </div>
      </aside>
    </div>
  `;
};

// --- logs drawer (modal over any page) ----------------------------------------

// Gateway (dxgate) logs arrive with ANSI color codes; strip them for display.
const stripAnsi = (text) => text.replace(/\u001b\[[0-9;]*m/g, "");

const levelOf = (line) => {
  const m = line.match(/\b(error|erro|warn|warning|info|debug)\b/i);
  if (!m) return "info";
  const level = m[1].toLowerCase();
  if (level.startsWith("err")) return "error";
  if (level.startsWith("warn")) return "warn";
  if (level === "debug") return "debug";
  return "info";
};

const LogLine = ({ line, query }) => {
  const level = levelOf(line);
  let content = line;
  if (query) {
    const idx = line.toLowerCase().indexOf(query.toLowerCase());
    if (idx >= 0) {
      content = html`${line.slice(0, idx)}<mark>${line.slice(idx, idx + query.length)}</mark>${line.slice(idx + query.length)}`;
    }
  }
  return html`<div class=${`log-line log-${level}`}>${content}</div>`;
};

const LogsPanel = ({ state, onClose, onReload }) => {
  const [query, setQuery] = useState("");
  const [level, setLevel] = useState("all");
  const [tail, setTail] = useState(state?.tail || 200);
  if (!state) return null;

  const pods = state.data?.pods || [];
  const filterLines = (raw) => {
    const lines = stripAnsi(raw || "").split("\n").filter(Boolean);
    return lines.filter((l) => {
      if (level !== "all" && levelOf(l) !== level) return false;
      if (query && !l.toLowerCase().includes(query.toLowerCase())) return false;
      return true;
    });
  };
  const download = () => {
    const text = pods.map((p) => `==== ${p.name} / ${p.container} ====\n${p.logs || p.error || ""}`).join("\n\n");
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([text], { type: "text/plain" }));
    a.download = `${state.name}-logs.txt`;
    a.click();
    URL.revokeObjectURL(a.href);
  };

  return html`
    <div class="drawer-overlay" onClick=${(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <aside class="drawer drawer-wide">
        <div class="drawer-head">
          <div>
            <${Eyebrow}>logs · ${state.kind}</${Eyebrow}>
            <div class="drawer-title">${state.title || state.name}</div>
          </div>
          <div class="drawer-head-actions">
            <select class="input" value=${tail} onChange=${(e) => { setTail(e.target.value); onReload(Number(e.target.value)); }}>
              ${[200, 500, 1000, 2000].map((n) => html`<option key=${n} value=${n}>tail ${n}</option>`)}
            </select>
            <button class="btn btn-ghost" onClick=${() => onReload(Number(tail))} title="Refresh">↻</button>
            <button class="btn btn-ghost" onClick=${download} title="Download">↓</button>
            <button class="btn btn-ghost" onClick=${onClose} aria-label="Close">✕</button>
          </div>
        </div>
        <div class="log-controls">
          <input class="input log-search" placeholder="Search in logs…" value=${query} onInput=${(e) => setQuery(e.target.value)} />
          ${["all", "info", "warn", "error"].map((lv) => html`
            <button key=${lv} class=${`chip chip-toggle ${level === lv ? "is-on" : ""}`} onClick=${() => setLevel(lv)}>${lv}</button>
          `)}
        </div>
        <div class="drawer-body log-body">
          ${state.loading && html`<${Skeleton} h=${160} />`}
          ${state.error && html`<${ErrorBanner} error=${state.error} onRetry=${() => onReload(Number(tail))} />`}
          ${!state.loading && !state.error && pods.length === 0 && html`<${EmptyState} title="No pods found" />`}
          ${!state.loading && !state.error && pods.map((p) => {
            const lines = filterLines(p.logs);
            return html`
              <section class="log-pod" key=${`${p.name}/${p.container}`}>
                <div class="log-pod-head">
                  <span class="mono log-pod-name">${p.name}</span>
                  <span class="mono log-pod-container">${p.container}</span>
                  <${StatusChip} status=${p.ready ? "ok" : "warn"}>${p.phase || "unknown"}</${StatusChip}>
                  <span class="log-pod-count">${lines.length} lines</span>
                </div>
                ${p.error && html`<div class="log-line log-error">${p.error}</div>`}
                <div class="log-pre">
                  ${lines.length === 0 && !p.error && html`<div class="log-line log-muted">No lines match.</div>`}
                  ${lines.map((l, i) => html`<${LogLine} key=${i} line=${l} query=${query} />`)}
                </div>
              </section>
            `;
          })}
        </div>
      </aside>
    </div>
  `;
};

// --- pages --------------------------------------------------------------------

const healthRollup = (data) => {
  const flags = Object.values(data.status || {});
  const readyFlags = flags.filter(Boolean).length;
  const registriesOK = (data.registries || []).every((r) => r.synced);
  const gatewaysOK = (data.gatewayInstances || []).every((g) => g.isReady);
  const instancesOK = (data.instances || []).some((i) => i.isReady);
  if (!instancesOK || readyFlags === 0) return "err";
  if (readyFlags < flags.length || !registriesOK || !gatewaysOK) return "warn";
  return "ok";
};

const SYNC_ITEMS = [
  ["xdsServerReady", "xDS server"],
  ["cachesSynced", "Caches"],
  ["servicesSynced", "Services"],
  ["configSynced", "Config"],
  ["proxylessSynced", "Proxyless"],
  ["injectorReady", "Injector"],
  ["validationReady", "Validation"],
];

const SyncRail = ({ status }) => html`
  <div class="rail" role="list">
    ${SYNC_ITEMS.map(([key, label]) => {
      const ok = !!(status && status[key]);
      return html`
        <div class=${`rail-seg ${ok ? "is-ok" : "is-pending"}`} role="listitem" key=${key}>
          <div class="rail-bar" />
          <div class="rail-label">${label}</div>
          <div class="rail-state">${ok ? "Ready" : "Pending"}</div>
        </div>
      `;
    })}
  </div>
`;

const OverviewPage = ({ data, openDrawer, openLogs, navigate, history }) => {
  const counts = data.counts || {};
  const health = healthRollup(data);
  const gateways = data.gatewayInstances || [];
  const registries = data.registries || [];
  const instances = data.instances || [];
  const ready = instances.filter((i) => i.isReady).length;
  const gwReady = gateways.filter((g) => g.isReady).length;
  const regSynced = registries.filter((r) => r.synced).length;
  const policies = (counts.peerAuthentications || 0) + (counts.requestAuthentications || 0) + (counts.authorizationPolicies || 0);
  const connSpark = history.map((h) => ({ t: h.t, v: h.connections })).filter((p) => p.v != null);

  return html`
    <div class="page">
      <div class="page-head">
        <div>
          <${Eyebrow}>observe / overview</${Eyebrow}>
          <h1 class="page-title">Mesh overview</h1>
        </div>
        <${StatusChip} status=${health}>control plane ${STATUS_LABEL[health]}</${StatusChip}>
      </div>

      <section class="section">
        <${SectionTitle}>Control plane readiness</${SectionTitle}>
        <${SyncRail} status=${data.status} />
      </section>

      <section class="tile-grid">
        <${StatTile} label="Mesh services" value=${fmtNumber(counts.services)} hint="in injected namespaces" onClick=${() => navigate("services")} />
        <${StatTile} label="xDS connections" value=${fmtNumber(counts.xdsConnections)} spark=${connSpark} onClick=${() => navigate("metrics")} />
        <${StatTile} label="Control plane" value=${`${ready}/${instances.length}`} hint="instances ready" status=${ready === instances.length && ready > 0 ? "ok" : ready > 0 ? "warn" : "err"}
          onClick=${() => openDrawer({ type: "controlplane", title: "dubbod", status: statusOfInstances(instances), data: { instances, version: data.version, namespace: data.namespace } })} />
        <${StatTile} label="Gateways" value=${`${gwReady}/${gateways.length}`} hint="deployments ready" status=${gateways.length === 0 ? "unknown" : gwReady === gateways.length ? "ok" : "warn"} onClick=${() => navigate("gateways")} />
        <${StatTile} label="Registries" value=${`${regSynced}/${registries.length}`} hint="synced" status=${registries.length === 0 ? "unknown" : regSynced === registries.length ? "ok" : "warn"} onClick=${() => navigate("registries")} />
        <${StatTile} label="Routes & policies" value=${fmtNumber((counts.httpRoutes || 0) + policies)} hint=${`${counts.httpRoutes || 0} routes · ${policies} policies`} onClick=${() => navigate("config")} />
      </section>

      <div class="grid-2">
        <section class="section">
          <${SectionTitle} aside=${html`<button class="btn btn-ghost btn-small" onClick=${() => openLogs({ kind: "dubbod", name: "dubbod", namespace: instances[0]?.namespace || data.namespace, title: "dubbod" })}>logs</button>`}>Control plane instances</${SectionTitle}>
          <div class="table-wrap">
            <table class="table">
              <thead><tr><th>Pod</th><th>IP</th><th>State</th></tr></thead>
              <tbody>
                ${instances.map((i) => html`
                  <tr key=${i.name}>
                    <td class="mono cell-strong">${i.name}</td>
                    <td class="mono cell-muted">${i.ip || "–"}</td>
                    <td><${StatusChip} status=${i.isReady ? "ok" : "err"}>${i.isReady ? "ready" : "not ready"}</${StatusChip}></td>
                  </tr>
                `)}
              </tbody>
            </table>
          </div>
        </section>

        <section class="section">
          <${SectionTitle}>Registries</${SectionTitle}>
          ${registries.length === 0 && html`<${EmptyState} title="No registries connected" />`}
          ${registries.length > 0 && html`
            <div class="table-wrap">
              <table class="table">
                <thead><tr><th>Provider</th><th>Cluster</th><th>Sync</th></tr></thead>
                <tbody>
                  ${registries.map((r) => html`
                    <tr key=${`${r.provider}/${r.cluster}`} class="row-click" onClick=${() => openDrawer({ type: "registry", title: r.provider, status: r.synced ? "ok" : "warn", data: r })}>
                      <td class="cell-strong">${r.provider}</td>
                      <td class="mono cell-muted">${r.cluster}</td>
                      <td><${StatusChip} status=${r.synced ? "ok" : "warn"}>${r.synced ? "synced" : "syncing"}</${StatusChip}></td>
                    </tr>
                  `)}
                </tbody>
              </table>
            </div>
          `}
        </section>
      </div>

      <section class="section">
        <${SectionTitle}>Managed gateways</${SectionTitle}>
        ${gateways.length === 0 && html`
          <${EmptyState} title="No managed gateway deployments">
            Create a Gateway resource with <span class="mono">gatewayClassName: dubbo</span> and the control plane will provision its deployment here.
          </${EmptyState}>
        `}
        ${gateways.length > 0 && html`
          <div class="table-wrap">
            <table class="table">
              <thead><tr><th>Deployment</th><th>Gateway</th><th>Namespace</th><th>Class</th><th>Replicas</th><th>State</th></tr></thead>
              <tbody>
                ${gateways.map((g) => html`
                  <tr key=${`${g.namespace}/${g.name}`} class="row-click" onClick=${() => openDrawer({ type: "gateway", title: g.name, status: g.isReady ? "ok" : (g.readyReplicas || 0) === 0 ? "err" : "warn", data: g })}>
                    <td class="mono cell-strong">${g.name}</td>
                    <td class="cell-muted">${g.gatewayName || "–"}</td>
                    <td class="cell-muted">${g.namespace}</td>
                    <td class="cell-muted">${g.gatewayClass || "–"}</td>
                    <td class="mono">${g.readyReplicas || 0}/${g.desiredReplicas || 0}</td>
                    <td><${StatusChip} status=${g.isReady ? "ok" : (g.readyReplicas || 0) === 0 ? "err" : "warn"} /></td>
                  </tr>
                `)}
              </tbody>
            </table>
          </div>
        `}
      </section>
    </div>
  `;
};

const statusOfInstances = (instances) => {
  const ready = instances.filter((i) => i.isReady).length;
  if (instances.length === 0 || ready === 0) return "err";
  return ready === instances.length ? "ok" : "warn";
};

const ServicesPage = ({ data, openDrawer }) => {
  const [query, setQuery] = useState("");
  const [namespace, setNamespace] = useState("");
  const [registry, setRegistry] = useState("");
  const [sort, setSort] = useState({ key: "name", dir: 1 });

  const services = data.services || [];
  const namespaces = [...new Set(services.map((s) => s.namespace))].sort();
  const registries = [...new Set(services.map((s) => s.registry))].sort();

  const filtered = useMemo(() => {
    const q = query.toLowerCase();
    return services
      .filter((s) => (!namespace || s.namespace === namespace) && (!registry || s.registry === registry))
      .filter((s) => !q || [s.name, s.hostname, s.namespace].some((v) => v?.toLowerCase().includes(q)))
      .sort((a, b) => sort.dir * String(a[sort.key] || "").localeCompare(String(b[sort.key] || "")));
  }, [services, query, namespace, registry, sort]);

  const Th = ({ k, children }) => html`
    <th class="th-sort" onClick=${() => setSort((s) => ({ key: k, dir: s.key === k ? -s.dir : 1 }))}>
      ${children}${sort.key === k ? (sort.dir > 0 ? " ↑" : " ↓") : ""}
    </th>
  `;

  return html`
    <div class="page">
      <div class="page-head">
        <div>
          <${Eyebrow}>mesh / services</${Eyebrow}>
          <h1 class="page-title">Services</h1>
          <div class="page-sub">${filtered.length} of ${services.length} mesh services · <span class="chip chip-gate">request metrics n/a</span></div>
        </div>
      </div>

      <div class="filters">
        <input class="input" placeholder="Filter by name, host or namespace…" value=${query} onInput=${(e) => setQuery(e.target.value)} />
        <select class="input" value=${namespace} onChange=${(e) => setNamespace(e.target.value)}>
          <option value="">All namespaces</option>
          ${namespaces.map((ns) => html`<option key=${ns} value=${ns}>${ns}</option>`)}
        </select>
        <select class="input" value=${registry} onChange=${(e) => setRegistry(e.target.value)}>
          <option value="">All registries</option>
          ${registries.map((r) => html`<option key=${r} value=${r}>${r}</option>`)}
        </select>
      </div>

      ${services.length === 0 && html`
        <${EmptyState} title="No mesh services discovered">
          Label a namespace with <span class="mono">dubbo-injection=enabled</span> to bring its services into the mesh.
        </${EmptyState}>
      `}
      ${services.length > 0 && filtered.length === 0 && html`
        <${EmptyState} title="No services match this filter">Clear the filters to see all ${services.length} services.</${EmptyState}>
      `}
      ${filtered.length > 0 && html`
        <div class="table-wrap">
          <table class="table table-wide">
            <thead>
              <tr>
                <${Th} k="name">Service</${Th}>
                <${Th} k="namespace">Namespace</${Th}>
                <${Th} k="registry">Registry</${Th}>
                <th>Ports</th>
                <${Th} k="exposure">Exposure</${Th}>
              </tr>
            </thead>
            <tbody>
              ${filtered.map((s) => html`
                <tr key=${s.hostname} class="row-click" onClick=${() => openDrawer({ type: "service", title: s.name || s.hostname, status: "unknown", data: s })}>
                  <td>
                    <div class="cell-strong">${s.name || s.hostname}</div>
                    <div class="mono cell-host">${s.hostname}</div>
                  </td>
                  <td><span class="chip">${s.namespace || "default"}</span></td>
                  <td class="cell-muted">${s.registry}</td>
                  <td class="mono cell-muted">${s.ports}</td>
                  <td><span class=${`chip ${s.meshExternal ? "chip-warn" : "chip-ok"}`}>${s.exposure}</span></td>
                </tr>
              `)}
            </tbody>
          </table>
        </div>
      `}
    </div>
  `;
};

const GatewaysPage = ({ data, openDrawer, openLogs }) => {
  const counts = data.counts || {};
  const gateways = data.gatewayInstances || [];
  return html`
    <div class="page">
      <div class="page-head">
        <div>
          <${Eyebrow}>mesh / gateways</${Eyebrow}>
          <h1 class="page-title">Gateways</h1>
          <div class="page-sub">North–south entry points provisioned by the control plane.</div>
        </div>
      </div>
      <section class="tile-grid tile-grid-3">
        <${StatTile} label="Gateways" value=${fmtNumber(counts.gateways)} hint="Gateway resources" />
        <${StatTile} label="Gateway classes" value=${fmtNumber(counts.gatewayClasses)} />
        <${StatTile} label="HTTP routes" value=${fmtNumber(counts.httpRoutes)} hint="attached routing rules" />
      </section>
      ${gateways.length === 0 && html`
        <${EmptyState} title="No gateways configured">
          Create a Gateway resource with <span class="mono">gatewayClassName: dubbo</span> to provision one.
        </${EmptyState}>
      `}
      ${gateways.length > 0 && html`
        <div class="table-wrap">
          <table class="table">
            <thead><tr><th>Deployment</th><th>Gateway</th><th>Namespace</th><th>Replicas</th><th>State</th><th></th></tr></thead>
            <tbody>
              ${gateways.map((g) => html`
                <tr key=${`${g.namespace}/${g.name}`} class="row-click" onClick=${() => openDrawer({ type: "gateway", title: g.name, status: g.isReady ? "ok" : (g.readyReplicas || 0) === 0 ? "err" : "warn", data: g })}>
                  <td class="mono cell-strong">${g.name}</td>
                  <td class="cell-muted">${g.gatewayName || "–"}</td>
                  <td class="cell-muted">${g.namespace}</td>
                  <td class="mono">${g.readyReplicas || 0}/${g.desiredReplicas || 0}</td>
                  <td><${StatusChip} status=${g.isReady ? "ok" : (g.readyReplicas || 0) === 0 ? "err" : "warn"} /></td>
                  <td><button class="btn btn-ghost btn-small" onClick=${(e) => { e.stopPropagation(); openLogs({ kind: "gateway", name: g.name, namespace: g.namespace, title: g.name }); }}>logs</button></td>
                </tr>
              `)}
            </tbody>
          </table>
        </div>
      `}
    </div>
  `;
};

const RegistriesPage = ({ data, openDrawer }) => {
  const registries = data.registries || [];
  return html`
    <div class="page">
      <div class="page-head">
        <div>
          <${Eyebrow}>mesh / registries</${Eyebrow}>
          <h1 class="page-title">Registries</h1>
          <div class="page-sub">Service discovery backends feeding this control plane.</div>
        </div>
      </div>
      ${registries.length === 0 && html`<${EmptyState} title="No registries connected" />`}
      ${registries.length > 0 && html`
        <div class="table-wrap">
          <table class="table">
            <thead><tr><th>Provider</th><th>Cluster</th><th>Sync</th></tr></thead>
            <tbody>
              ${registries.map((r) => html`
                <tr key=${`${r.provider}/${r.cluster}`} class="row-click" onClick=${() => openDrawer({ type: "registry", title: r.provider, status: r.synced ? "ok" : "warn", data: r })}>
                  <td class="cell-strong">${r.provider}</td>
                  <td class="mono cell-muted">${r.cluster}</td>
                  <td><${StatusChip} status=${r.synced ? "ok" : "warn"}>${r.synced ? "synced" : "syncing"}</${StatusChip}></td>
                </tr>
              `)}
            </tbody>
          </table>
        </div>
      `}
    </div>
  `;
};

const ConfigPage = ({ data, openDrawer }) => html`
  <div class="page">
    <div class="page-head">
      <div>
        <${Eyebrow}>mesh / config</${Eyebrow}>
        <h1 class="page-title">Configuration</h1>
        <div class="page-sub">Gateway API and policy resources in scope · <span class="chip chip-gate">per-resource listing n/a</span></div>
      </div>
    </div>
    <div class="table-wrap">
      <table class="table">
        <thead><tr><th>Kind</th><th>Count</th><th>Purpose</th></tr></thead>
        <tbody>
          ${(data.configKinds || []).map((k) => html`
            <tr key=${k.kind} class="row-click" onClick=${() => openDrawer({ type: "config", title: k.kind, data: k })}>
              <td class="mono cell-strong">${k.kind}</td>
              <td><span class=${`count ${k.count > 0 ? "is-set" : ""}`}>${k.count}</span></td>
              <td class="cell-muted">${k.description}</td>
            </tr>
          `)}
        </tbody>
      </table>
    </div>
  </div>
`;

const MetricsPage = ({ history, interval, setInterval, paused, setPaused, error, retry }) => {
  const last = history[history.length - 1];
  const snapshot = last?.snapshot;

  const seriesOf = (extract, name) => ({
    name,
    points: history.map((h) => ({ t: h.t, v: extract(h) })).filter((p) => p.v != null),
  });

  const rateSeries = (label) => {
    const points = [];
    for (let i = 1; i < history.length; i++) {
      const prev = history[i - 1].pushes.get(label);
      const cur = history[i].pushes.get(label);
      const dt = (history[i].t - history[i - 1].t) / 1000;
      if (prev != null && cur != null && dt > 0 && cur >= prev) {
        points.push({ t: history[i].t, v: (cur - prev) / dt });
      }
    }
    return { name: label, points };
  };

  const pushTypes = last ? [...last.pushes.keys()] : [];
  const preferred = ["cds", "eds", "lds", "rds"].filter((t) => pushTypes.includes(t));
  const mainTypes = preferred.length > 0
    ? preferred
    : pushTypes.sort((a, b) => (last.pushes.get(b) || 0) - (last.pushes.get(a) || 0)).slice(0, 4);
  const buckets = snapshot ? family(snapshot, "dubbod_xds_push_time")?.metrics?.[0]?.buckets : null;
  const p50 = bucketQuantile(buckets, 0.5);
  const p95 = bucketQuantile(buckets, 0.95);
  const p99 = bucketQuantile(buckets, 0.99);
  const triggers = snapshot ? [...labeledValues(snapshot, "dubbod_push_triggers", "type").entries()].map(([label, value]) => ({ label: label || "(none)", value })).sort((a, b) => b.value - a.value) : [];
  const errors = snapshot
    ? (totalValue(snapshot, "dubbod_total_xds_internal_errors") + totalValue(snapshot, "dubbod_total_xds_rejects"))
    : null;
  const version = snapshot ? family(snapshot, "dubbod_info")?.metrics?.[0]?.labels?.version : null;

  return html`
    <div class="page">
      <div class="page-head">
        <div>
          <${Eyebrow}>observe / metrics</${Eyebrow}>
          <h1 class="page-title">Control plane metrics</h1>
          <div class="page-sub">
            Live from <span class="mono">api/metrics</span> · trends buffer since page open ·
            <span class="chip chip-gate" title="No TSDB behind this GUI; longer ranges need a Prometheus-backed query API.">history beyond session n/a</span>
          </div>
        </div>
        <div class="page-actions">
          <select class="input" value=${interval} onChange=${(e) => setInterval(Number(e.target.value))}>
            ${[5, 10, 30, 60].map((s) => html`<option key=${s} value=${s}>every ${s}s</option>`)}
          </select>
          <button class=${`btn ${paused ? "" : "btn-ghost"}`} onClick=${() => setPaused(!paused)}>${paused ? "Resume" : "Pause"}</button>
        </div>
      </div>

      ${error && html`<${ErrorBanner} error=${error} onRetry=${retry} />`}
      ${!snapshot && !error && html`<${Skeleton} h=${260} />`}

      ${snapshot && html`
        <section class="tile-grid">
          <${StatTile} label="Uptime" value=${fmtDuration(firstValue(snapshot, "dubbod_uptime_seconds"))}
            hint=${version && /\d/.test(String(version)) ? `v${String(version).split("-")[0]}` : ""} />
          <${StatTile} label="xDS connections" value=${fmtNumber(last.connections)} spark=${history.map((h) => ({ t: h.t, v: h.connections })).filter((p) => p.v != null)} />
          <${StatTile} label="Known services" value=${fmtNumber(firstValue(snapshot, "dubbod_services"))} spark=${history.map((h) => ({ t: h.t, v: h.services })).filter((p) => p.v != null)} sparkColor="--series-2" />
          <${StatTile} label="xDS pushes (total)" value=${fmtNumber(last.pushTotal)} />
          <${StatTile} label="xDS errors + rejects" value=${fmtNumber(errors)} status=${errors > 0 ? "warn" : "ok"} />
          <${StatTile} label="Push P95" value=${p95 != null ? fmtDuration(p95) : "–"} hint="from live histogram" />
        </section>

        <section class="section">
          <${SectionTitle}>xDS push rate <span class="unit">pushes/s by type</span></${SectionTitle}>
          <${TrendChart} series=${mainTypes.map(rateSeries)} format=${(v) => v.toFixed(2)}
            emptyHint="Collecting samples — rates appear after two polls." />
        </section>

        <div class="grid-2">
          <section class="section">
            <${SectionTitle}>Push duration distribution
              <span class="unit">
                P50 ${p50 != null ? fmtDuration(p50) : "–"} · P95 ${p95 != null ? fmtDuration(p95) : "–"} · P99 ${p99 != null ? fmtDuration(p99) : "–"}
              </span>
            </${SectionTitle}>
            <${BucketBars} buckets=${buckets} />
          </section>
          <section class="section">
            <${SectionTitle}>Push triggers <span class="unit">cumulative</span></${SectionTitle}>
            <${BreakdownBars} items=${triggers} />
          </section>
        </div>

        <section class="section">
          <${SectionTitle}>xDS connections <span class="unit">gauge</span></${SectionTitle}>
          <${TrendChart} series=${[seriesOf((h) => h.connections, "connections")]} format=${fmtNumber} />
        </section>
      `}
    </div>
  `;
};

const LogsPage = ({ data, openLogs }) => {
  const targets = [
    { kind: "dubbod", name: "dubbod", namespace: (data.instances || [])[0]?.namespace || data.namespace, title: "dubbod (control plane)" },
    ...(data.gatewayInstances || []).map((g) => ({ kind: "gateway", name: g.name, namespace: g.namespace, title: `${g.name} (gateway)` })),
  ];
  return html`
    <div class="page">
      <div class="page-head">
        <div>
          <${Eyebrow}>observe / logs</${Eyebrow}>
          <h1 class="page-title">Logs</h1>
          <div class="page-sub">
            Tail pod logs from control plane and gateway deployments ·
            <span class="chip chip-gate" title="Workload/application log search needs a log aggregation backend.">workload log search n/a</span>
          </div>
        </div>
      </div>
      <div class="log-targets">
        ${targets.map((t) => html`
          <button key=${`${t.kind}/${t.namespace}/${t.name}`} class="log-target" onClick=${() => openLogs(t)}>
            <span class="log-target-kind">${t.kind}</span>
            <span class="log-target-name mono">${t.name}</span>
            <span class="log-target-ns">${t.namespace}</span>
            <span class="log-target-open">open →</span>
          </button>
        `)}
      </div>
    </div>
  `;
};

const RuntimePage = ({ data }) => {
  const server = data.server || {};
  const [theme, setTheme] = useState(getTheme());
  const rows = [
    ["GUI base path", server.guiPath],
    ["GUI HTTP address", server.httpAddress],
    ["gRPC (xDS)", server.grpcAddress],
    ["Secure gRPC", server.secureGrpcAddress],
    ["Overview API", server.overviewPath],
    ["Prometheus metrics", server.metricsPath],
    ["Version endpoint", server.versionPath],
    ["Readiness", server.readyPath],
  ];
  return html`
    <div class="page">
      <div class="page-head">
        <div>
          <${Eyebrow}>system / runtime</${Eyebrow}>
          <h1 class="page-title">Runtime</h1>
          <div class="page-sub mono">${data.version}</div>
        </div>
      </div>
      <div class="grid-2">
        <section class="section">
          <${SectionTitle}>Instance</${SectionTitle}>
          <${Field} label="Cluster" mono>${data.clusterId}</${Field}>
          <${Field} label="Namespace" mono>${data.namespace}</${Field}>
          <${Field} label="Pod" mono>${data.podName || "–"}</${Field}>
          <${Field} label="Trust domain" mono>${data.mesh?.trustDomain || "–"}</${Field}>
          <${Field} label="Root namespace" mono>${data.mesh?.rootNamespace || "–"}</${Field}>
          <${Field} label="Discovery address" mono>${data.mesh?.discoveryAddress || "–"}</${Field}>
        </section>
        <section class="section">
          <${SectionTitle}>Endpoints</${SectionTitle}>
          ${rows.map(([label, value]) => html`
            <div class="field field-row" key=${label}>
              <div class="field-label">${label}</div>
              <div class="field-value mono field-copy" title="Click to copy" onClick=${() => value && copyText(value)}>${value || "–"}</div>
            </div>
          `)}
        </section>
      </div>
      <section class="section">
        <${SectionTitle}>Console preferences</${SectionTitle}>
        <div class="pref-row">
          <div>
            <div class="cell-strong">Theme</div>
            <div class="cell-muted">Follows the OS unless overridden.</div>
          </div>
          <div class="seg">
            ${["auto", "light", "dark"].map((m) => html`
              <button key=${m} class=${`seg-item ${theme === m ? "is-on" : ""}`} onClick=${() => { applyTheme(m); setTheme(m); }}>${m}</button>
            `)}
          </div>
        </div>
        <div class="pref-row">
          <div>
            <div class="cell-strong">Mock data</div>
            <div class="cell-muted">Development only. Payloads mirror the real API contract exactly (see <span class="mono">mock.js</span>).</div>
          </div>
          <button class=${`btn ${MOCK ? "" : "btn-ghost"}`} onClick=${() => setMock(!MOCK)}>${MOCK ? "Disable mock" : "Enable mock"}</button>
        </div>
      </section>
    </div>
  `;
};

const GATED = {
  traffic: {
    title: "Traffic",
    purpose: "Per-service and per-edge request analytics: RPS, success rate, error breakdown, latency percentiles, protocol split and drill-down to slow calls.",
    missing: "Request-level telemetry from proxyless gRPC workloads — a metrics ingestion path (scrape or OTLP) aggregated by service pair. The xds-api wire protocol carries no per-request stats today.",
    wouldShow: ["RPS / error-rate / P95 per service and per edge", "traffic-weighted topology edges", "status-code and protocol breakdowns", "time-range comparison and anomaly flags"],
  },
  events: {
    title: "Events",
    purpose: "Timeline of mesh-relevant Kubernetes events: push failures, gateway provisioning, injector activity, config rejections — each linked to its resource.",
    missing: "An api/events feed backed by a Kubernetes events informer (or an internal event store) scoped to mesh resources.",
    wouldShow: ["chronological event stream with severity", "affected-resource links into services/gateways", "filters by kind, namespace, and reason"],
  },
  alerts: {
    title: "Alerts",
    purpose: "Active and historical alerts with acknowledge / silence / close lifecycle, wired to the metrics that triggered them.",
    missing: "An alerting rule engine and alert store. Nothing evaluates thresholds server-side today; the GUI will not fabricate alert states.",
    wouldShow: ["severity-grouped active alerts", "ack / silence / close actions with audit trail", "links to the triggering metric and topology node"],
  },
};

// --- shell ---------------------------------------------------------------------

const NAV = [
  { group: "Observe", items: [
    { id: "overview", label: "Overview" },
    { id: "topology", label: "Topology" },
    { id: "metrics", label: "Metrics" },
    { id: "logs", label: "Logs" },
  ]},
  { group: "Mesh", items: [
    { id: "services", label: "Services" },
    { id: "gateways", label: "Gateways" },
    { id: "registries", label: "Registries" },
    { id: "config", label: "Config" },
  ]},
  { group: "Planned", items: [
    { id: "traffic", label: "Traffic", gated: true },
    { id: "events", label: "Events", gated: true },
    { id: "alerts", label: "Alerts", gated: true },
  ]},
  { group: "System", items: [
    { id: "runtime", label: "Runtime" },
  ]},
];

const NavIcon = ({ id }) => {
  const paths = {
    overview: html`<path d="M3 12h5V3H3zM10 21h5v-9h-5zM17 8h4V3h-4zM3 21h5v-6H3zM10 9h5V3h-5zM17 21h4V11h-4z"/>`,
    topology: html`<circle cx="5" cy="12" r="2.5"/><circle cx="19" cy="6" r="2.5"/><circle cx="19" cy="18" r="2.5"/><path d="M7.4 11l9.2-4.2M7.4 13l9.2 4.2" fill="none"/>`,
    metrics: html`<path d="M3 20h18M6 16l4-6 4 3 5-8" fill="none"/>`,
    logs: html`<path d="M4 5h16M4 10h16M4 15h10M4 20h7" fill="none"/>`,
    services: html`<circle cx="12" cy="12" r="8.5" fill="none"/><circle cx="12" cy="12" r="3"/>`,
    gateways: html`<path d="M4 21V8l8-5 8 5v13M9 21v-6h6v6" fill="none"/>`,
    registries: html`<ellipse cx="12" cy="5.5" rx="8" ry="2.8" fill="none"/><path d="M4 5.5v13c0 1.6 3.6 2.9 8 2.9s8-1.3 8-2.9v-13M4 12c0 1.6 3.6 2.9 8 2.9s8-1.3 8-2.9" fill="none"/>`,
    config: html`<circle cx="12" cy="12" r="3" fill="none"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1" fill="none"/>`,
    traffic: html`<path d="M3 17c4 0 4-10 9-10s5 10 9 10" fill="none"/>`,
    events: html`<circle cx="12" cy="12" r="8.5" fill="none"/><path d="M12 7v5l3.5 2" fill="none"/>`,
    alerts: html`<path d="M12 3l9.5 17h-19zM12 10v4M12 17.3v.4" fill="none"/>`,
    runtime: html`<rect x="4" y="4" width="16" height="16" rx="2" fill="none"/><path d="M9 9h6v6H9z" fill="none"/>`,
  };
  return html`<svg viewBox="0 0 24 24" class="nav-icon" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">${paths[id] || paths.overview}</svg>`;
};

const App = () => {
  const [route, setRoute] = useState(parseRoute);
  const [data, setData] = useState(null);
  const [overviewError, setOverviewError] = useState(null);
  const [drawer, setDrawer] = useState(null);
  const [logsState, setLogsState] = useState(null);

  // Metrics polling with a session ring buffer.
  const [metricsInterval, setMetricsInterval] = useState(10);
  const [metricsPaused, setMetricsPaused] = useState(false);
  const [metricsError, setMetricsError] = useState(null);
  const [history, setHistory] = useState([]);

  const navigate = useCallback((id) => {
    setRoute(id);
    history_replace(id);
  }, []);
  const history_replace = (id) => { window.history.replaceState(null, "", `#/${id}`); };

  useEffect(() => {
    const onHash = () => setRoute(parseRoute());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  const loadOverview = useCallback(async () => {
    try {
      const json = await fetchOverview();
      setData(json);
      setOverviewError(null);
    } catch (e) {
      setOverviewError(e.message || String(e));
    }
  }, []);

  useEffect(() => {
    loadOverview();
    const timer = window.setInterval(loadOverview, 15000);
    return () => window.clearInterval(timer);
  }, [loadOverview]);

  const pollMetrics = useCallback(async () => {
    try {
      const snapshot = await fetchMetrics();
      setMetricsError(null);
      setHistory((prev) => {
        const entry = {
          t: Date.now(),
          snapshot,
          connections: firstValue(snapshot, "dubbod_xds"),
          services: firstValue(snapshot, "dubbod_services"),
          pushes: labeledValues(snapshot, "dubbod_xds_pushes", "type"),
          pushTotal: totalValue(snapshot, "dubbod_xds_pushes"),
        };
        const next = [...prev, entry];
        return next.length > HISTORY_MAX ? next.slice(next.length - HISTORY_MAX) : next;
      });
    } catch (e) {
      setMetricsError(e.message || String(e));
    }
  }, []);

  useEffect(() => {
    if (metricsPaused) return;
    pollMetrics();
    const timer = window.setInterval(pollMetrics, metricsInterval * 1000);
    return () => window.clearInterval(timer);
  }, [metricsInterval, metricsPaused, pollMetrics]);

  const openLogs = useCallback(async (target, tail = 200) => {
    const next = { ...target, tail, title: target.title || target.name, loading: true, data: null, error: null };
    setLogsState(next);
    try {
      const payload = await fetchLogs({ ...target, tail });
      setLogsState({ ...next, loading: false, data: payload });
    } catch (e) {
      setLogsState({ ...next, loading: false, error: e.message || String(e) });
    }
  }, []);

  if (!data && !overviewError) {
    return html`
      <div class="app">
        <aside class="sidebar" />
        <main class="main">
          <header class="topbar" />
          <div class="content">
            <${Skeleton} h=${64} /><div style=${{ height: "16px" }} />
            <${Skeleton} h=${140} /><div style=${{ height: "16px" }} />
            <${Skeleton} h=${300} />
          </div>
        </main>
      </div>
    `;
  }

  if (!data && overviewError) {
    return html`
      <div class="app">
        <main class="main main-solo">
          <div class="content content-center">
            <${ErrorBanner} error=${overviewError} onRetry=${loadOverview} />
            <p class="cell-muted">The console needs <span class="mono">api/overview</span> to render. You can also inspect the UI with <a href="?mock=1">mock data</a>.</p>
          </div>
        </main>
      </div>
    `;
  }

  const updatedAt = data.updatedAt ? new Date(data.updatedAt).toLocaleTimeString() : null;
  const health = healthRollup(data);

  const pageProps = { data, openDrawer: setDrawer, openLogs, navigate, history };

  return html`
    <div class="app">
      <aside class="sidebar">
        <div class="brand">
          <img class="brand-logo" src="./dubbo-logo.png" alt="" />
          <span class="brand-name">${CONFIG.product || "Dubbo"}<span class="brand-sub">console</span></span>
        </div>
        <nav class="nav">
          ${NAV.map((group) => html`
            <div class="nav-group" key=${group.group}>
              <div class="nav-group-label">${group.group}</div>
              ${group.items.map((item) => html`
                <button key=${item.id} type="button"
                  class=${`nav-item ${route === item.id ? "is-active" : ""} ${item.gated ? "is-gated" : ""}`}
                  onClick=${() => navigate(item.id)} aria-current=${route === item.id ? "page" : undefined}>
                  <${NavIcon} id=${item.id} />
                  <span>${item.label}</span>
                  ${item.gated && html`<span class="nav-gate" title="Backend capability not connected yet">n/a</span>`}
                </button>
              `)}
            </div>
          `)}
        </nav>
        ${data.version && html`<div class="sidebar-foot mono" title=${data.version}>${data.version}</div>`}
      </aside>

      <main class="main">
        <header class="topbar">
          <div class="topbar-context">
            <span class="topbar-kv">cluster <span class="mono">${data.clusterId || "–"}</span></span>
            <span class="topbar-kv">ns <span class="mono">${data.namespace || "–"}</span></span>
            ${data.mesh?.trustDomain && html`<span class="topbar-kv topbar-kv-wide">trust <span class="mono">${data.mesh.trustDomain}</span></span>`}
          </div>
          <div class="topbar-status">
            ${MOCK && html`<span class="chip chip-mock">MOCK DATA</span>`}
            ${overviewError && html`<span class="chip chip-err" title=${overviewError}>stale — retrying</span>`}
            ${updatedAt && !overviewError && html`<span class="topbar-updated">updated ${updatedAt}</span>`}
            <${Dot} status=${overviewError ? "warn" : health} />
            <span class="topbar-live">${overviewError ? "reconnecting" : "live"}</span>
          </div>
        </header>

        <div class="content" key=${route}>
          ${route === "overview" && html`<${OverviewPage} ...${pageProps} />`}
          ${route === "topology" && html`
            <div class="page page-flush">
              <div class="page-head">
                <div>
                  <${Eyebrow}>observe / topology</${Eyebrow}>
                  <h1 class="page-title">Topology</h1>
                  <div class="page-sub">Configuration-plane relationships as of ${updatedAt || "–"} — registries sync into dubbod; dubbod provisions gateways and pushes xDS to workloads.</div>
                </div>
              </div>
              <${Topology} data=${data} selectedId=${drawer?.nodeId}
                onSelect=${(node) => {
                  const map = {
                    registry: () => setDrawer({ type: "registry", nodeId: node.id, title: node.label, status: node.status, data: node.data }),
                    gateway: () => setDrawer({ type: "gateway", nodeId: node.id, title: node.label, status: node.status, data: node.data }),
                    service: () => setDrawer({ type: "service", nodeId: node.id, title: node.label, status: "unknown", data: node.data }),
                    controlplane: () => setDrawer({ type: "controlplane", nodeId: node.id, title: "dubbod", status: node.status, data: node.data }),
                  };
                  map[node.type]?.();
                }} />
            </div>
          `}
          ${route === "metrics" && html`
            <${MetricsPage} history=${history} interval=${metricsInterval} setInterval=${setMetricsInterval}
              paused=${metricsPaused} setPaused=${setMetricsPaused} error=${metricsError} retry=${pollMetrics} />
          `}
          ${route === "logs" && html`<${LogsPage} ...${pageProps} />`}
          ${route === "services" && html`<${ServicesPage} ...${pageProps} />`}
          ${route === "gateways" && html`<${GatewaysPage} ...${pageProps} />`}
          ${route === "registries" && html`<${RegistriesPage} ...${pageProps} />`}
          ${route === "config" && html`<${ConfigPage} ...${pageProps} />`}
          ${route === "runtime" && html`<${RuntimePage} ...${pageProps} />`}
          ${GATED[route] && html`<div class="page"><${CapabilityGate} ...${GATED[route]} /></div>`}
        </div>
      </main>

      <${Drawer} item=${drawer} onClose=${() => setDrawer(null)} onOpenLogs=${(t) => { setDrawer(null); openLogs(t); }} navigate=${navigate} />
      <${LogsPanel} state=${logsState} onClose=${() => setLogsState(null)} onReload=${(tail) => openLogs(logsState, tail)} />
    </div>
  `;
};

const checkAndRender = () => {
  if (window.render && window.html) {
    window.render(window.html`<${App} />`, document.getElementById("root"));
  } else {
    setTimeout(checkAndRender, 50);
  }
};

checkAndRender();
