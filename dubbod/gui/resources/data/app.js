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
 * Dubbo control plane console.
 *
 * Every value on screen comes from api/overview, api/metrics or api/logs —
 * there is no demo/mock path and no widget for a metric this build does not
 * export. See architecture/gui/DESIGN.md for the field-by-field mapping.
 */

import { html, render, useState, useEffect, useMemo, useRef, useCallback } from "./runtime.js";
import { t, LANGUAGES, getLanguage, setLanguage } from "./i18n.js";
import {
  BreakdownBars, StageFlow,
  histStats, fmtDuration,
} from "./charts.js";

const CONFIG = (() => {
  const el = document.getElementById("dubbod-gui-config");
  return el ? JSON.parse(el.textContent) : { basePath: "/gui", product: "Dubbo" };
})();

const API = {
  overview: new URL("api/overview", document.baseURI).toString(),
  logs: new URL("api/logs", document.baseURI).toString(),
  metrics: new URL("api/metrics", document.baseURI).toString(),
};

const getJSON = async (url) => {
  const res = await fetch(url);
  const body = await res.json().catch(() => null);
  if (!res.ok) throw new Error(body?.error || `HTTP ${res.status}`);
  return body;
};

const fetchLogs = (target) => {
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
    try { localStorage.removeItem(themeKey); } catch (_) { /* storage unavailable */ }
  } else {
    document.documentElement.dataset.theme = mode;
    try { localStorage.setItem(themeKey, mode); } catch (_) { /* storage unavailable */ }
  }
};

// --- routing ----------------------------------------------------------------

const ROUTES = ["overview", "services", "pipeline", "logs", "topology", "configuration"];
// Older builds shipped a page per resource kind plus placeholder pages; those
// URLs now land on the page that absorbed them.
const LEGACY_ROUTES = {
  home: "overview", mesh: "services", meshgateway: "overview", runtime: "configuration",
  gateways: "overview", registries: "overview", config: "overview",
  traffic: "overview", events: "overview", alerts: "overview", metrics: "pipeline",
};
const parseRoute = () => {
  let hash = (location.hash || "").replace(/^#\/?/, "");
  hash = LEGACY_ROUTES[hash] || hash;
  return ROUTES.includes(hash) ? hash : "overview";
};

// --- metric helpers ---------------------------------------------------------

const family = (snapshot, name) => snapshot?.families?.find((f) => f.name === name);
const firstSample = (snapshot, name) => family(snapshot, name)?.metrics?.[0] ?? null;
const labeledValues = (snapshot, name, label) => {
  const out = new Map();
  for (const m of family(snapshot, name)?.metrics || []) {
    const key = m.labels?.[label] ?? "";
    out.set(key, (out.get(key) || 0) + (m.value || 0));
  }
  return out;
};
const breakdown = (snapshot, name, label) =>
  [...labeledValues(snapshot, name, label).entries()]
    .map(([key, value]) => ({ label: key || "(unlabeled)", value }))
    .sort((a, b) => b.value - a.value);

// --- shared components ------------------------------------------------------

// Default chip text when a caller supplies no children.
const StatusChip = ({ status, children }) => html`
  <span class=${`chip chip-${status}`}>${children ?? t("status." + status)}</span>
`;

const Eyebrow = ({ children }) => html`<div class="eyebrow">${children}</div>`;
// A section can carry only an aside — dropping the heading should not leave an
// empty h2 holding the row open.
const SectionTitle = ({ children, aside }) => html`
  <div class=${`section-head ${children ? "" : "is-titleless"}`}>
    ${children && html`<h2 class="section-title">${children}</h2>`}
    ${aside && html`<div class="section-aside">${aside}</div>`}
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
    <span class="banner-text">${t("error.request")}: ${error}</span>
    ${onRetry && html`<button class="btn btn-small" onClick=${onRetry}>${t("action.retry")}</button>`}
  </div>
`;

const Skeleton = ({ h = 120 }) => html`<div class="skeleton" style=${{ height: `${h}px` }} />`;

const RefreshButton = ({ onRefresh, refreshing }) => html`
  <button class="btn btn-ghost btn-small" onClick=${onRefresh} disabled=${refreshing}
    title=${t("action.refresh.title")}>
    ${refreshing ? t("action.refreshing") : t("action.refresh")}
  </button>
`;

const Field = ({ label, children, mono }) => html`
  <div class="field">
    <div class="field-label">${label}</div>
    <div class=${`field-value ${mono ? "mono" : ""}`}>${children ?? "–"}</div>
  </div>
`;

const copyText = (text) => {
  try { navigator.clipboard?.writeText(text); } catch (_) { /* clipboard blocked */ }
};

// --- entity drawer ----------------------------------------------------------

const Drawer = ({ item, onClose, onOpenLogs }) => {
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
          <button class="btn btn-ghost" onClick=${onClose} aria-label=${t("action.close")}>✕</button>
        </div>
        <div class="drawer-body">
          ${item.status && html`<div class="drawer-status"><${StatusChip} status=${item.status} /></div>`}

          ${type === "service" && html`
            <${Field} label=${t("drawer.hostname")} mono>${data.hostname}</${Field}>
            <${Field} label=${t("col.namespace")}>${data.namespace}</${Field}>
            <${Field} label=${t("col.registry")}>${data.registry}</${Field}>
            <${Field} label=${t("col.ports")} mono>${data.ports}</${Field}>
            <${Field} label=${t("col.exposure")}>${data.exposure}</${Field}>
            <${Field} label=${t("col.clusterAddress")} mono>${data.defaultAddress || "–"}</${Field}>
            <${Field} label=${t("drawer.serviceAccounts")}>${data.serviceAccounts}</${Field}>
            <${Field} label=${t("drawer.meshExternal")}>${data.meshExternal ? "yes" : "no"}</${Field}>
            <div class="drawer-actions">
              <button class="btn" onClick=${() => copyText(data.hostname)}>${t("drawer.copyHostname")}</button>
            </div>
          `}

          ${type === "gateway" && html`
            <${Field} label=${t("col.deployment")} mono>${data.name}</${Field}>
            <${Field} label=${t("col.gateway")}>${data.gatewayName || "–"}</${Field}>
            <${Field} label=${t("col.namespace")}>${data.namespace}</${Field}>
            <${Field} label=${t("drawer.gatewayClass")}>${data.gatewayClass || "–"}</${Field}>
            <${Field} label=${t("col.replicas")} mono>${data.readyReplicas || 0} / ${data.desiredReplicas || 0} ready</${Field}>
            <div class="drawer-actions">
              <button class="btn" onClick=${() => onOpenLogs({ kind: "gateway", name: data.name, namespace: data.namespace })}>${t("drawer.viewLogs")}</button>
            </div>
          `}

          ${type === "registry" && html`
            <${Field} label=${t("drawer.provider")}>${data.provider}</${Field}>
            <${Field} label=${t("drawer.cluster")} mono>${data.cluster}</${Field}>
            <${Field} label=${t("drawer.informerSync")}>${data.synced ? t("drawer.synced") : t("drawer.syncing")}</${Field}>
          `}

          ${type === "node" && html`
            <${Field} label=${t("drawer.kind")}>
              ${data.kind === "gateway" ? t("drawer.kind.gateway") : data.kind === "controlplane" ? t("drawer.kind.controlPlane") : t("drawer.kind.service")}
            </${Field}>
            ${data.kind === "controlplane" && html`
              <${Field} label=${t("drawer.pods")}>${data.pods}</${Field}>
              <${Field} label=${t("drawer.role")}>${t("drawer.controlPlane.role")}</${Field}>
            `}
            <${Field} label=${t("col.namespace")}>${data.namespace || "–"}</${Field}>
            ${data.ports && html`<${Field} label=${t("col.ports")} mono>${data.ports}</${Field}>`}
            ${data.mtlsMode && html`
              <${Field} label=${t("drawer.inboundMTLS")}>
                ${data.mtlsMode} — ${data.mtlsFromPolicy ? t("drawer.mtls.fromPolicy") : t("drawer.mtls.fallback")}
              </${Field}>
            `}
            <${Field} label=${t("drawer.reachedBy")}>${t("drawer.reachedBy.value", data.edgesIn)}</${Field}>
            ${data.edgesOut.length === 0
              ? html`<${Field} label=${t("drawer.forwardsTo")}>${t("drawer.forwardsTo.none")}</${Field}>`
              : html`
                <div class="drawer-subhead">${t("drawer.forwardsTo")}</div>
                ${data.edgesOut.map((e, i) => html`
                  <div class="drawer-row" key=${i}>
                    <span class="mono drawer-row-main">${e.match} → ${e.to}${e.port ? ":" + e.port : ""}</span>
                    <span class="drawer-row-side">${e.share != null ? e.share + "%" : e.route}</span>
                  </div>
                `)}
              `}
          `}

          ${type === "namespace" && html`
            <${Field} label=${t("drawer.injectedPods")}>
              ${t("drawer.injectedPods.value", data.injected, data.candidates)}${data.injected === data.candidates ? "" : t("drawer.injectedPods.gap")}
            </${Field}>
            <${Field} label=${t("drawer.inboundProxy")} mono>${data.inbound || "–"}</${Field}>
            <${Field} label=${t("drawer.mtlsPerPort")}>
              ${(data.mtlsModes || []).join(" · ") || t("drawer.mtlsPerPort.none")}
            </${Field}>
            <${Field} label=${t("drawer.soonestExpiry")}>
              ${data.soonestExpiry ? new Date(data.soonestExpiry).toLocaleString() : t("drawer.soonestExpiry.none")}
            </${Field}>
            <${Field} label=${t("drawer.trustRoot")}>
              ${data.rootStale === 0 ? t("drawer.trustRoot.ok") : t("drawer.trustRoot.stale", data.rootStale)}
            </${Field}>
            ${data.configError && html`<${Field} label=${t("drawer.problem")}>${data.configError}</${Field}>`}

            <div class="drawer-subhead">${t("drawer.pods")}</div>
            ${(data.pods || []).map((w) => html`
              <div class="drawer-row" key=${w.name}>
                <span class="mono drawer-row-main">${w.name}</span>
                <span class="drawer-row-side">
                  ${w.sidecarReady ? t("state.accepting") : t("state.notReady")}${w.restarts > 0 ? ` · ${w.restarts} restarts` : ""}
                </span>
              </div>
            `)}
          `}
        </div>
      </aside>
    </div>
  `;
};

// --- logs panel -------------------------------------------------------------

// Gateway (dxgate) logs arrive with ANSI color codes; strip them for display.
const stripAnsi = (text) => text.replace(/\u001b\[[0-9;]*m/g, "");

// The API asks Kubernetes for timestamps, but dubbod and dxgate already stamp
// every line themselves. Drop the outer one when a second timestamp follows it;
// that is worth ~30 columns in the log panel.
const dedupeTimestamp = (line) =>
  line.replace(/^\d{4}-\d{2}-\d{2}T[\d:.]+Z\s+(?=\d{4}-\d{2}-\d{2}T)/, "");

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
    const lines = stripAnsi(raw || "").split("\n").filter(Boolean).map(dedupeTimestamp);
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
              ${[200, 500, 1000, 2000].map((n) => html`<option key=${n} value=${n}>${t("logs.tail", n)}</option>`)}
            </select>
            <button class="btn btn-ghost" onClick=${() => onReload(Number(tail))} title=${t("logs.reload")}>↻</button>
            <button class="btn btn-ghost" onClick=${download} title=${t("logs.download")}>↓</button>
            <button class="btn btn-ghost" onClick=${onClose} aria-label=${t("action.close")}>✕</button>
          </div>
        </div>
        <div class="log-controls">
          <input class="input log-search" placeholder=${t("logs.search")} value=${query} onInput=${(e) => setQuery(e.target.value)} />
          ${["all", "info", "warn", "error"].map((lv) => html`
            <button key=${lv} class=${`chip chip-toggle ${level === lv ? "is-on" : ""}`} onClick=${() => setLevel(lv)}>${t("logs.level." + lv)}</button>
          `)}
        </div>
        <div class="drawer-body log-body">
          ${state.loading && html`<${Skeleton} h=${160} />`}
          ${state.error && html`<${ErrorBanner} error=${state.error} onRetry=${() => onReload(Number(tail))} />`}
          ${!state.loading && !state.error && pods.length === 0 && html`<${EmptyState} title=${t("logs.noPods")} />`}
          ${!state.loading && !state.error && pods.map((p) => {
            const lines = filterLines(p.logs);
            return html`
              <section class="log-pod" key=${`${p.name}/${p.container}`}>
                <div class="log-pod-head">
                  <span class="mono log-pod-name">${p.name}</span>
                  <span class="mono log-pod-container">${p.container}</span>
                  <${StatusChip} status=${p.ready ? "ok" : "warn"}>${p.phase || "unknown"}</${StatusChip}>
                  <span class="log-pod-count">${t("logs.lines", lines.length)}</span>
                </div>
                ${p.error && html`
                  <div class="banner banner-err">
                    <span class="banner-text">${t("logs.refused")} ${p.error}</span>
                  </div>
                `}
                <div class="log-pre">
                  ${lines.length === 0 && !p.error && html`<div class="log-line log-muted">${t("logs.noMatch")}</div>`}
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

// --- overview ---------------------------------------------------------------

// Roll the injected pods up per namespace: the pod list itself lives in the
// drawer, the table answers "is this namespace's data plane healthy".
const byNamespace = (workloads, podTotals = {}) => {
  const rows = new Map();
  for (const w of workloads) {
    const row = rows.get(w.namespace) || {
      namespace: w.namespace, injected: 0, accepting: 0, restarts: 0,
      rootStale: 0, modes: new Set(), soonestExpiry: null, configError: "", pods: [],
    };
    row.injected += 1;
    if (w.sidecarReady) row.accepting += 1;
    row.restarts += w.restarts || 0;
    if (w.certExpiresAt && !w.certRootActive) row.rootStale += 1;
    for (const mode of w.mtlsModes || []) row.modes.add(mode);
    if (w.certExpiresAt) {
      const at = new Date(w.certExpiresAt).getTime();
      if (row.soonestExpiry == null || at < row.soonestExpiry) row.soonestExpiry = at;
    }
    if (w.configError && !row.configError) row.configError = w.configError;
    row.inbound = w.inbound && w.upstream ? `${w.inbound} → ${w.upstream}` : row.inbound || "";
    row.pods.push(w);
    rows.set(w.namespace, row);
  }
  return [...rows.values()]
    .map((r) => ({
      ...r,
      mtlsModes: [...r.modes].sort(),
      // Falls back to the injected count so the ratio never reads below 100%
      // just because the pod total was unavailable.
      candidates: Math.max(podTotals[r.namespace] ?? r.injected, r.injected),
    }))
    .sort((a, b) => a.namespace.localeCompare(b.namespace));
};

// The sidecar enforces one mTLS mode per service port. dubbod generated that
// config, so the modes shown here are the ones actually in force.
const mtlsCell = (row) => {
  if (row.configError) return html`<${StatusChip} status="err">${t("state.unknown")}</${StatusChip}>`;
  const modes = row.mtlsModes || [];
  if (modes.length === 0) return html`<span class="cell-muted">${t("state.meshDefault")}</span>`;
  const worst = modes.includes("DISABLE") ? "err" : modes.includes("PERMISSIVE") ? "warn" : "ok";
  return html`<${StatusChip} status=${worst}>${modes.join(" · ")}</${StatusChip}>`;
};

// Certificates are reissued well before expiry, so a short remaining life is the
// signal that rotation has stopped working.
const certCell = (row) => {
  if (row.soonestExpiry == null) return html`<${StatusChip} status="err">${t("state.none")}</${StatusChip}>`;
  const msLeft = row.soonestExpiry - Date.now();
  const status = msLeft <= 0 ? "err" : msLeft / 3600000 < 2 ? "warn" : "ok";
  return html`<${StatusChip} status=${status}>${msLeft <= 0 ? t("state.expired") : t("state.left", fmtDuration(msLeft / 1000))}</${StatusChip}>`;
};

const OverviewPage = ({ data, openDrawer, onRefresh, refreshing }) => {
  const instances = data.instances || [];
  const namespaces = byNamespace(data.dataPlane || [], data.dataPlanePods || {});
  const gateways = data.gatewayInstances || [];


  return html`
    <div class="page">
      <div class="page-head">
        <h1 class="page-title">${t("overview.title")}</h1>
        <${RefreshButton} onRefresh=${onRefresh} refreshing=${refreshing} />
      </div>

      <section class="section">
        <${SectionTitle}>${t("overview.controlPlane")}</${SectionTitle}>

        ${instances.length === 0 && html`
          <${EmptyState} title=${t("empty.noDubbodPods")}>
            ${t("empty.noDubbodPods.body.before")} <span class="mono">app=dubbod</span>
            ${t("empty.noDubbodPods.body.after")} <span class="mono">${data.namespace}</span>
          </${EmptyState}>
        `}
        ${instances.length > 0 && html`
          <div class="table-wrap">
            <table class="table">
              <thead><tr><th>${t("col.pod")}</th><th>${t("col.podIP")}</th><th>${t("col.namespace")}</th><th>${t("col.state")}</th></tr></thead>
              <tbody>
                ${instances.map((i) => html`
                  <tr key=${i.name}>
                    <td class="mono cell-strong">${i.name || "–"}</td>
                    <td class="mono cell-muted">${i.ip || "–"}</td>
                    <td class="cell-muted">${i.namespace}</td>
                    <td><${StatusChip} status=${i.isReady ? "ok" : "err"}>${i.isReady ? t("state.ready") : t("state.notReady")}</${StatusChip}></td>
                  </tr>
                `)}
              </tbody>
            </table>
          </div>
        `}
      </section>

      <section class="section">
        <${SectionTitle}>${t("overview.dataPlane")}</${SectionTitle}>

        ${namespaces.length === 0 && html`
          <${EmptyState} title=${t("empty.noInjected")}>
            ${t("empty.noInjected.body.before")} <span class="mono">dubbo-injection=enabled</span>
            ${t("empty.noInjected.body.after")}
          </${EmptyState}>
        `}
        ${namespaces.length > 0 && html`
          <div class="table-wrap">
            <table class="table">
              <thead>
                <tr><th>${t("col.namespace")}</th><th>${t("col.injected")}</th><th>${t("col.inboundL4")}</th><th>${t("col.mtls")}</th><th>${t("col.certificate")}</th><th>${t("col.trustRoot")}</th><th>${t("col.restarts")}</th></tr>
              </thead>
              <tbody>
                ${namespaces.map((ns) => html`
                  <tr key=${ns.namespace} class="row-click"
                    onClick=${() => openDrawer({ type: "namespace", title: ns.namespace, data: ns })}>
                    <td class="cell-strong">${ns.namespace}</td>
                    <td class=${`mono ${ns.injected === ns.candidates ? "" : "cell-warn"}`}>${ns.injected}/${ns.candidates}</td>
                    <td>
                      <${StatusChip} status=${ns.accepting === ns.injected ? "ok" : "err"}>
                        ${ns.accepting === ns.injected
                          ? t("state.accepting")
                          : t("state.notAccepting", ns.injected - ns.accepting, ns.injected)}
                      </${StatusChip}>
                    </td>
                    <td>${mtlsCell(ns)}</td>
                    <td>${certCell(ns)}</td>
                    <td>
                      <${StatusChip} status=${ns.rootStale === 0 ? "ok" : "warn"}>
                        ${ns.rootStale === 0 ? t("state.current") : t("state.superseded", ns.rootStale)}
                      </${StatusChip}>
                    </td>
                    <td class=${`mono ${ns.restarts > 0 ? "cell-warn" : "cell-muted"}`}>${ns.restarts}</td>
                  </tr>
                `)}
              </tbody>
            </table>
          </div>
        `}
      </section>

      <section class="section">
        <${SectionTitle}>${t("overview.externalDataPlane")}</${SectionTitle}>

        ${gateways.length === 0 && html`
          <${EmptyState} title=${t("empty.noGateways")}>
            ${t("empty.noGateways.body.before")} <span class="mono">gatewayClassName: dubbo</span>
            ${t("empty.noGateways.body.after")} <span class="mono">dxgate</span> ${t("empty.noGateways.body.tail")}
          </${EmptyState}>
        `}
        ${gateways.length > 0 && html`
          <div class="table-wrap">
            <table class="table">
              <thead>
                <tr><th>${t("col.deployment")}</th><th>${t("col.gateway")}</th><th>${t("col.namespace")}</th><th>${t("col.class")}</th><th>${t("col.replicas")}</th><th>${t("col.state")}</th></tr>
              </thead>
              <tbody>
                ${gateways.map((g) => {
                  const status = g.isReady ? "ok" : (g.readyReplicas || 0) === 0 ? "err" : "warn";
                  return html`
                    <tr key=${`${g.namespace}/${g.name}`} class="row-click"
                      onClick=${() => openDrawer({ type: "gateway", title: g.name, status, data: g })}>
                      <td class="mono cell-strong">${g.name}</td>
                      <td class="cell-muted">${g.gatewayName || "–"}</td>
                      <td class="cell-muted">${g.namespace}</td>
                      <td class="cell-muted">${g.gatewayClass || "–"}</td>
                      <td class="mono">${g.readyReplicas || 0}/${g.desiredReplicas || 0}</td>
                      <td><${StatusChip} status=${status} /></td>
                    </tr>
                  `;
                })}
              </tbody>
            </table>
          </div>
        `}
      </section>
    </div>
  `;
};

// --- services ---------------------------------------------------------------

const ServicesPage = ({ data, openDrawer, onRefresh, refreshing }) => {
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
        <h1 class="page-title">${t("services.title")}</h1>
        <${RefreshButton} onRefresh=${onRefresh} refreshing=${refreshing} />
      </div>

      <div class="filters">
        <input class="input" placeholder=${t("services.filter")} value=${query} onInput=${(e) => setQuery(e.target.value)} />
        <select class="input" value=${namespace} onChange=${(e) => setNamespace(e.target.value)}>
          <option value="">${t("services.allNamespaces")}</option>
          ${namespaces.map((ns) => html`<option key=${ns} value=${ns}>${ns}</option>`)}
        </select>
        <select class="input" value=${registry} onChange=${(e) => setRegistry(e.target.value)}>
          <option value="">${t("services.allRegistries")}</option>
          ${registries.map((r) => html`<option key=${r} value=${r}>${r}</option>`)}
        </select>
        <span class="filters-count">${t("services.shown", filtered.length)}</span>
      </div>

      ${services.length === 0 && html`
        <${EmptyState} title=${t("empty.noServices")}>
          ${t("empty.noServices.body.before")} <span class="mono">dubbo-injection=enabled</span>
          ${t("empty.noServices.body.after")}
        </${EmptyState}>
      `}
      ${services.length > 0 && filtered.length === 0 && html`
        <${EmptyState} title=${t("empty.noServicesMatch")}>${t("empty.noServicesMatch.body", services.length)}</${EmptyState}>
      `}
      ${filtered.length > 0 && html`
        <div class="table-wrap">
          <table class="table table-wide">
            <thead>
              <tr>
                <${Th} k="name">${t("col.service")}</${Th}>
                <${Th} k="namespace">${t("col.namespace")}</${Th}>
                <${Th} k="registry">${t("col.registry")}</${Th}>
                <th>${t("col.ports")}</th>
                <th>${t("col.clusterAddress")}</th>
                <${Th} k="exposure">${t("col.exposure")}</${Th}>
              </tr>
            </thead>
            <tbody>
              ${filtered.map((s) => html`
                <tr key=${s.hostname} class="row-click"
                  onClick=${() => openDrawer({ type: "service", title: s.name || s.hostname, data: s })}>
                  <td>
                    <div class="cell-strong">${s.name || s.hostname}</div>
                    <div class="mono cell-host">${s.hostname}</div>
                  </td>
                  <td><span class="chip">${s.namespace || "default"}</span></td>
                  <td class="cell-muted">${s.registry}</td>
                  <td class="mono cell-muted">${s.ports}</td>
                  <td class="mono cell-muted">${s.defaultAddress || "–"}</td>
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

// --- config push pipeline ---------------------------------------------------

/**
 * Every stage below is a histogram dubbod actually exports. The order is the
 * real code path a config change travels: a watch fires, the update is
 * debounced into a batch, a PushContext is built, each proxy is queued, the
 * generated config is written to the wire, and the proxy acknowledges.
 */
// The first two stages run on every config change; the last two only produce
// samples once a proxy is connected, so an idle control plane legitimately shows
// them empty. dubbod_proxy_convergence_time is deliberately absent: it is
// registered but has no Record() call site anywhere in the tree, so a stage for
// it could only ever say "no samples".
const PIPELINE_STAGES = [
  { key: "debounce", label: "stage.debounce", metric: "dubbod_debounce_time",
    idleHint: "stage.idle.noConfigChange" },
  { key: "pushcontext", label: "stage.pushcontext", metric: "dubbod_pushcontext_init_seconds",
    idleHint: "stage.idle.noPushContext" },
  { key: "queue", label: "stage.queue", metric: "dubbod_proxy_queue_time",
    idleHint: "stage.idle.needsProxy" },
  { key: "send", label: "stage.send", metric: "dubbod_xds_send_time",
    idleHint: "stage.idle.needsProxy" },
];

const PipelinePage = ({ snapshot, error, retry, onRefresh, refreshing }) => {
  const stages = PIPELINE_STAGES.map((s) => ({ ...s, stats: histStats(firstSample(snapshot, s.metric)) }));
  const timedStages = stages.filter((s) => s.stats && s.stats.mean != null);
  const endToEnd = timedStages.reduce((a, s) => a + s.stats.mean, 0);

  const inbound = snapshot ? breakdown(snapshot, "dubbod_inbound_updates", "type") : [];
  const triggers = snapshot ? breakdown(snapshot, "dubbod_push_triggers", "type") : [];


  return html`
    <div class="page">
      <div class="page-head">
        <div>
          <h1 class="page-title">${t("pipeline.title")}</h1>
        </div>
        <${RefreshButton} onRefresh=${onRefresh} refreshing=${refreshing} />
      </div>

      ${error && html`<${ErrorBanner} error=${error} onRetry=${retry} />`}
      ${!snapshot && !error && html`<${Skeleton} h=${280} />`}

      ${snapshot && html`
        <section class="section">
          <${SectionTitle} aside=${timedStages.length > 0 && html`
            <span class="section-count">${t("pipeline.meanEndToEnd", fmtDuration(endToEnd))}</span>
          `}>${t("pipeline.pushPath")}</${SectionTitle}>
          <${StageFlow} stages=${stages} />
        </section>

        <div class="grid-2">
          <section class="section">
            <${SectionTitle}>${t("pipeline.inboundUpdates")} <span class="unit">${t("pipeline.inboundUpdates.unit")}</span></${SectionTitle}>
            <${BreakdownBars} items=${inbound} />
          </section>
          <section class="section">
            <${SectionTitle}>${t("pipeline.pushTriggers")} <span class="unit">${t("pipeline.pushTriggers.unit")}</span></${SectionTitle}>
            <${BreakdownBars} items=${triggers} />
          </section>
        </div>

      `}
    </div>
  `;
};

// --- logs -------------------------------------------------------------------

const LogsPage = ({ data, openLogs, onRefresh, refreshing }) => {
  const targets = [
    {
      kind: "dubbod", role: t("logs.role.controlPlane"), name: "dubbod",
      namespace: (data.instances || [])[0]?.namespace || data.namespace,
      replicas: (data.instances || []).length,
    },
    ...(data.gatewayInstances || []).map((g) => ({
      kind: "gateway", role: t("logs.role.externalDataPlane"), name: g.name,
      namespace: g.namespace, replicas: g.readyReplicas || 0,
    })),
  ];

  return html`
    <div class="page">
      <div class="page-head">
        <h1 class="page-title">${t("logs.title")}</h1>
        <${RefreshButton} onRefresh=${onRefresh} refreshing=${refreshing} />
      </div>

      <div class="table-wrap">
        <table class="table">
          <thead><tr><th>${t("col.deployment")}</th><th>${t("col.role")}</th><th>${t("col.namespace")}</th><th>${t("col.pods")}</th></tr></thead>
          <tbody>
            ${targets.map((t) => html`
              <tr key=${`${t.kind}/${t.namespace}/${t.name}`} class="row-click"
                onClick=${() => openLogs({ ...t, title: t.name })}>
                <td class="mono cell-strong">${t.name}</td>
                <td class="cell-muted">${t.role}</td>
                <td class="cell-muted">${t.namespace}</td>
                <td class="mono cell-muted">${t.replicas}</td>
              </tr>
            `)}
          </tbody>
        </table>
      </div>
    </div>
  `;
};

// --- topology ---------------------------------------------------------------

// Drawn as a badge on the node ring. A closed shackle means every caller must
// present a certificate; an open one means the port still takes plaintext; no
// shackle at all means inbound mTLS is off.
// Drawn on a 24x24 grid so the shackle stays legible at node scale: closed =
// every caller must present a certificate, open = the port still takes
// plaintext, absent = inbound mTLS is off.
const MTLS_LOCK = {
  STRICT: { shackle: "M8 10.5V7.5a4 4 0 0 1 8 0v3", status: "ok", label: "mtls.strict.label" },
  PERMISSIVE: { shackle: "M8 10.5V7.5a4 4 0 0 1 7.4-2.1", status: "warn", label: "mtls.permissive.label" },
  DISABLE: { shackle: "", status: "err", label: "mtls.disable.label" },
};

const LOCK_BODY = { x: 5, y: 10, w: 14, h: 10, r: 2 };

const LockGlyph = ({ mode, x = 0, y = 0, fromPolicy }) => {
  const spec = MTLS_LOCK[mode];
  if (!spec) return null;
  return html`
    <g class=${`lock lock-${spec.status}`} transform=${`translate(${x}, ${y})`}>
      <title>${t(spec.label)} · ${fromPolicy ? t("mtls.source.policy") : t("mtls.source.fallback")}</title>
      <circle class="lock-halo" cx="12" cy="13" r="12" />
      <rect x=${LOCK_BODY.x} y=${LOCK_BODY.y} width=${LOCK_BODY.w} height=${LOCK_BODY.h} rx=${LOCK_BODY.r} />
      ${spec.shackle && html`<path d=${spec.shackle} />`}
    </g>
  `;
};

/**
 * Service graph in the Kiali idiom: round nodes, labels underneath, curved
 * edges, drag / zoom / pan, click for detail.
 *
 * Nodes are everything the control plane serves config for — managed gateways
 * plus every mesh service. Edges come from HTTPRoutes: a route names a parent
 * (a Gateway for north-south, a Service for east-west) and the backends each
 * rule forwards to. Services no route mentions still get a node, because a
 * workload that is in the mesh but unreachable is exactly what an operator
 * needs to see.
 *
 * Edges carry the route match and traffic weight — both configuration. Kiali
 * additionally animates edges at the observed request rate and colours them by
 * error rate; dubbod exposes no request-level telemetry, so those channels are
 * deliberately left unused rather than driven by invented numbers.
 */
const GRAPH_SCOPES = [
  { id: "north-south", label: "topology.scope.northSouth" },
  { id: "east-west", label: "topology.scope.eastWest" },
  { id: "control-plane", label: "topology.scope.controlPlane" },
];

const buildServiceGraph = (data, scope) => {
  const services = data.services || [];
  const gateways = (data.gatewayInstances || []).filter((g) => g.gatewayName);
  const routes = data.routes || [];

  const nodes = new Map();
  const put = (id, extra) => {
    const node = nodes.get(id) || { id, name: id, edgesOut: [], edgesIn: 0 };
    nodes.set(id, Object.assign(node, extra));
    return nodes.get(id);
  };

  const meta = new Map();
  for (const g of gateways) {
    meta.set(g.gatewayName, { kind: "gateway", namespace: g.namespace, ready: g.isReady });
  }
  for (const svc of services) {
    meta.set(svc.name, {
      kind: "service", namespace: svc.namespace, ports: svc.ports,
      mtlsMode: svc.mtlsMode, mtlsFromPolicy: svc.mtlsFromPolicy,
    });
  }
  // Only nodes this view is about get drawn. Rendering every service in every
  // view stacks the unrelated ones into a column that dwarfs the actual paths.
  const enter = (id) => put(id, meta.get(id) || { kind: "service" });
  if (scope === "control-plane") for (const id of meta.keys()) enter(id);

  const gatewayNames = new Set(gateways.map((g) => g.gatewayName));
  // A route attached to a Gateway is ingress; one attached to a Service is
  // traffic between meshed workloads.
  const inScope = (parent) =>
    scope === "control-plane" ||
    (scope === "north-south" ? gatewayNames.has(parent) : !gatewayNames.has(parent));

  for (const r of routes) {
    for (const parent of r.parents || []) {
      if (!inScope(parent)) continue;
      const from = enter(parent);
      for (const rule of r.rules || []) {
        const total = (rule.backends || []).reduce((a, b) => a + (b.weight || 0), 0);
        const split = (rule.backends || []).length > 1;
        for (const b of rule.backends || []) {
          enter(b.name).edgesIn += 1;
          from.edgesOut.push({
            to: b.name, match: rule.match, port: b.port, route: r.name,
            share: split && total > 0 ? Math.round(((b.weight || 0) / total) * 100) : null,
          });
        }
      }
    }
  }

  // The control plane carries no application traffic, so it is a separate layer
  // rather than another hop on a call path: dashed edges mean "programs this
  // node", solid edges mean "forwards requests to".
  if (scope === "control-plane") {
    const gatewayDeployments = data.gatewayInstances || [];
    const streaming = new Set();
    for (const client of data.xdsClients || []) {
      // Node IDs are pod names; a pod belongs to the deployment it is prefixed
      // with, which for a managed gateway maps back to its Gateway resource.
      const podName = (client.nodeId || "").split(".")[0];
      for (const g of gatewayDeployments) {
        if (g.gatewayName && podName.startsWith(g.name)) streaming.add(g.gatewayName);
      }
      for (const svc of services) {
        if (podName.startsWith(svc.name + "-")) streaming.add(svc.name);
      }
    }

    const cp = put("dubbod", {
      kind: "controlplane",
      namespace: data.namespace,
      pods: (data.instances || []).length,
    });
    for (const node of [...nodes.values()]) {
      if (node.kind === "controlplane") continue;
      cp.edgesOut.push({ to: node.id, config: true, streaming: streaming.has(node.id) });
    }
  }

  // Layer by longest path from an entry point so callers sit left of callees.
  const layer = new Map();
  const assign = (id, depth, seen) => {
    if (seen.has(id) || (layer.get(id) ?? -1) >= depth) return;
    layer.set(id, depth);
    const next = new Set(seen).add(id);
    for (const e of nodes.get(id)?.edgesOut || []) assign(e.to, depth + 1, next);
  };
  for (const node of nodes.values()) if (node.edgesIn === 0) assign(node.id, 0, new Set());
  for (const node of nodes.values()) if (!layer.has(node.id)) assign(node.id, 0, new Set());

  const columns = [];
  for (const node of nodes.values()) {
    const depth = layer.get(node.id) || 0;
    (columns[depth] = columns[depth] || []).push(node);
  }
  for (const col of columns) {
    col.sort((a, b) => (b.edgesOut.length - a.edgesOut.length) || a.name.localeCompare(b.name));
  }
  const omitted = [...meta.keys()].filter((id) => !nodes.has(id));
  return { nodes, columns, omitted };
};

const R = 26;            // node radius
const COL_GAP = 210;     // horizontal spacing between layers
const ROW_GAP = 104;     // vertical spacing within a layer

const ServiceGraph = ({ data, onSelect, scope }) => {
  const { nodes, columns, omitted } = useMemo(() => buildServiceGraph(data, scope), [data, scope]);
  const [moved, setMoved] = useState({});
  const [hover, setHover] = useState(null);
  const [view, setView] = useState({ zoom: 1, x: 0, y: 0 });
  const drag = useRef(null);
  const pan = useRef(null);
  const svgRef = useRef(null);

  const layout = useMemo(() => {
    const rows = Math.max(...columns.map((c) => c.length), 1);
    const width = Math.max((columns.length - 1) * COL_GAP + R * 2, 200);
    const height = Math.max((rows - 1) * ROW_GAP + R * 2, 160);
    const base = new Map();
    columns.forEach((col, ci) => {
      const colHeight = (col.length - 1) * ROW_GAP;
      col.forEach((node, ri) => {
        base.set(node.id, {
          x: R + ci * COL_GAP,
          y: R + (height - R * 2 - colHeight) / 2 + ri * ROW_GAP,
        });
      });
    });
    return { base, width, height };
  }, [columns]);

  const at = (id) => moved[id] || layout.base.get(id) || { x: 0, y: 0 };

  const toGraph = (event) => {
    const svg = svgRef.current;
    const rect = svg.getBoundingClientRect();
    const vb = svg.viewBox.baseVal;
    return {
      x: vb.x + ((event.clientX - rect.left) / rect.width) * vb.width,
      y: vb.y + ((event.clientY - rect.top) / rect.height) * vb.height,
    };
  };

  const onNodeDown = (event, id) => {
    event.stopPropagation();
    const p = toGraph(event);
    const start = at(id);
    drag.current = { id, dx: p.x - start.x, dy: p.y - start.y, moved: false };
    event.currentTarget.setPointerCapture(event.pointerId);
  };
  const onCanvasDown = (event) => {
    pan.current = { x: event.clientX, y: event.clientY, ox: view.x, oy: view.y };
  };
  const onMove = (event) => {
    if (drag.current) {
      const p = toGraph(event);
      drag.current.moved = true;
      setMoved((prev) => ({ ...prev, [drag.current.id]: { x: p.x - drag.current.dx, y: p.y - drag.current.dy } }));
      return;
    }
    if (pan.current) {
      const scale = layout.width / (svgRef.current?.getBoundingClientRect().width || 1) / view.zoom;
      setView((v) => ({
        ...v,
        x: pan.current.ox - (event.clientX - pan.current.x) * scale,
        y: pan.current.oy - (event.clientY - pan.current.y) * scale,
      }));
    }
  };
  const onNodeUp = (node) => {
    const wasDrag = drag.current?.moved;
    drag.current = null;
    if (!wasDrag) onSelect(node);
  };
  const onWheel = (event) => {
    event.preventDefault();
    setView((v) => ({ ...v, zoom: Math.min(2.6, Math.max(0.45, v.zoom * (event.deltaY < 0 ? 1.12 : 0.89))) }));
  };

  const edges = [];
  for (const node of nodes.values()) {
    for (const e of node.edgesOut) {
      if (!nodes.has(e.to)) continue;
      const a = at(node.id);
      const b = at(e.to);
      const dist = Math.hypot(b.x - a.x, b.y - a.y) || 1;
      // Stop the line on the circle edge so the arrowhead touches the ring.
      const ux = (b.x - a.x) / dist;
      const uy = (b.y - a.y) / dist;
      const x1 = a.x + ux * R;
      const y1 = a.y + uy * R;
      const x2 = b.x - ux * (R + 9);
      const y2 = b.y - uy * (R + 9);
      const bow = Math.max(26, Math.abs(x2 - x1) / 2.4);
      edges.push({
        ...e, from: node.id,
        d: `M${x1},${y1} C${x1 + bow},${y1} ${x2 - bow},${y2} ${x2},${y2}`,
        lx: (x1 + x2) / 2, ly: (y1 + y2) / 2,
      });
    }
  }

  const related = hover
    ? new Set([hover, ...edges.filter((e) => e.from === hover || e.to === hover).flatMap((e) => [e.from, e.to])])
    : null;

  const pad = 70;
  const vw = (layout.width + pad * 2) / view.zoom;
  const vh = (layout.height + pad * 2) / view.zoom;

  if (nodes.size === 0) {
    return html`
      <${EmptyState} title=${scope === "north-south" ? t("topology.empty.northSouth") : t("topology.empty.eastWest")}>
        ${scope === "north-south" ? t("topology.empty.northSouth.body") : t("topology.empty.eastWest.body")}
      </${EmptyState}>
    `;
  }

  return html`
    <div class="graph-board">
      <div class="graph-tools">
        <button class="btn btn-ghost btn-small" onClick=${() => setView((v) => ({ ...v, zoom: Math.min(2.6, v.zoom * 1.2) }))} title=${t("topology.zoomIn")}>+</button>
        <button class="btn btn-ghost btn-small" onClick=${() => setView((v) => ({ ...v, zoom: Math.max(0.45, v.zoom / 1.2) }))} title=${t("topology.zoomOut")}>−</button>
        <button class="btn btn-ghost btn-small" onClick=${() => { setView({ zoom: 1, x: 0, y: 0 }); setMoved({}); }} title=${t("topology.reset.title")}>${t("topology.reset")}</button>
      </div>

      <svg ref=${svgRef} class="graph"
        viewBox=${`${-pad + view.x} ${-pad + view.y} ${vw} ${vh}`}
        onPointerDown=${onCanvasDown} onPointerMove=${onMove}
        onPointerUp=${() => { pan.current = null; }}
        onPointerLeave=${() => { pan.current = null; drag.current = null; setHover(null); }}
        onWheel=${onWheel}
        role="img" aria-label=${t("topology.aria")}>
        <defs>
          <marker id="gedge" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6.5" markerHeight="6.5" orient="auto">
            <path d="M0,0 L10,5 L0,10 z" class="graph-arrow" />
          </marker>
        </defs>

        ${edges.map((e) => html`
          <g key=${`${e.from}-${e.to}-${e.match || "config"}`}
            class=${`graph-edge ${e.config ? (e.streaming ? "is-xds" : "is-config") : ""} ${related && !(related.has(e.from) && related.has(e.to)) ? "is-dim" : ""}`}>
            <path d=${e.d} marker-end="url(#gedge)" />
            ${!e.config && html`
              <text x=${e.lx} y=${e.ly - 8} text-anchor="middle" class="graph-edge-label">
                ${e.match}${e.share != null ? ` · ${e.share}%` : ""}
              </text>
            `}
          </g>
        `)}

        ${[...nodes.values()].map((node) => {
          const p = at(node.id);
          const isGw = node.kind === "gateway";
          const isolated = node.kind !== "controlplane" && node.edgesIn === 0 && node.edgesOut.length === 0;
          return html`
            <g key=${node.id}
              class=${`graph-node ${isGw ? "is-gateway" : ""} ${node.kind === "controlplane" ? "is-cp" : ""} ${isolated ? "is-isolated" : ""} ${related && !related.has(node.id) ? "is-dim" : ""}`}
              transform=${`translate(${p.x}, ${p.y})`}
              onPointerDown=${(e) => onNodeDown(e, node.id)}
              onPointerUp=${() => onNodeUp(node)}
              onMouseEnter=${() => setHover(node.id)}
              onMouseLeave=${() => setHover(null)}
              tabIndex="0">
              ${node.kind === "controlplane"
                ? html`<rect x=${-R - 6} y=${-R + 6} width=${(R + 6) * 2} height=${(R - 6) * 2} rx="6" />`
                : isGw
                  ? html`<rect x=${-R} y=${-R} width=${R * 2} height=${R * 2} rx="7" transform="rotate(45)" />`
                  : html`<circle r=${R} />`}
              ${node.mtlsMode && html`
                <${LockGlyph} mode=${node.mtlsMode} fromPolicy=${node.mtlsFromPolicy} x=${R - 14} y=${-R - 10} />
              `}
              <text y=${R + 17} text-anchor="middle" class="graph-node-name">${node.name}</text>
              <text y=${R + 30} text-anchor="middle" class="graph-node-sub">${node.namespace || ""}</text>
              ${isGw && html`<text y="4" text-anchor="middle" class="graph-node-tag">GW</text>`}
              ${node.kind === "controlplane" && html`<text y="4" text-anchor="middle" class="graph-node-tag">CP</text>`}
            </g>
          `;
        })}
      </svg>

      ${omitted.length > 0 && html`
        <div class="graph-omitted">
          ${t("topology.omitted", omitted.length)}
          <span class="mono">${omitted.join(", ")}</span>
        </div>
      `}
    </div>
  `;
};

const TopologyPage = ({ data, openDrawer, onRefresh, refreshing }) => {
  const routes = data.routes || [];
  const services = data.services || [];
  const [scope, setScope] = useState("north-south");
  const streaming = (data.xdsClients || []).length;

  return html`
    <div class="page page-flush">
      <div class="page-head">
        <h1 class="page-title">${t("topology.title")}</h1>
        <div class="page-actions">
          <select class="input" value=${scope} onChange=${(e) => setScope(e.target.value)}>
            ${GRAPH_SCOPES.map((v) => html`<option key=${v.id} value=${v.id}>${t(v.label)}</option>`)}
          </select>
          <${RefreshButton} onRefresh=${onRefresh} refreshing=${refreshing} />
        </div>
      </div>

      ${services.length === 0 && html`
        <${EmptyState} title=${t("topology.empty.noServices")}>
          ${t("topology.empty.noServices.body.before")} <span class="mono">dubbo-injection=enabled</span>
          ${t("topology.empty.noServices.body.after")}
        </${EmptyState}>
      `}
      ${services.length > 0 && html`
        <section class="section">
          <${SectionTitle} aside=${html`
            <span class="legend">
              <span class="legend-item"><svg class="lock lock-ok" viewBox="2 3 20 19"><rect x="5" y="10" width="14" height="10" rx="2"/><path d="M8 10.5V7.5a4 4 0 0 1 8 0v3"/></svg>${t("topology.legend.required")}</span>
              <span class="legend-item"><svg class="lock lock-warn" viewBox="2 3 20 19"><rect x="5" y="10" width="14" height="10" rx="2"/><path d="M8 10.5V7.5a4 4 0 0 1 7.4-2.1"/></svg>${t("topology.legend.plaintext")}</span>
              <span class="legend-item"><svg class="lock lock-err" viewBox="2 3 20 19"><rect x="5" y="10" width="14" height="10" rx="2"/></svg>${t("topology.legend.off")}</span>
              ${scope === "control-plane" && html`
                <span class="legend-item"><span class="legend-line is-xds"></span>${t("topology.legend.stream")}</span>
                <span class="legend-item"><span class="legend-line is-config"></span>${t("topology.legend.configOnly")}</span>
              `}
              <span class="section-count">
                ${scope === "control-plane"
                  ? t("topology.streams", streaming)
                  : t("topology.routes", routes.length)}
              </span>
            </span>
          `} />
          <${ServiceGraph} data=${data} scope=${scope}
            onSelect=${(node) => openDrawer({ type: "node", title: node.name, data: node })} />
        </section>
      `}
    </div>
  `;
};

// --- configuration ----------------------------------------------------------

const ConfigurationPage = ({ data, onRefresh, refreshing, lang, onLanguage }) => {
  const server = data.server || {};
  const [theme, setTheme] = useState(getTheme());
  // Only addresses something outside this page has to dial. The GUI base path
  // and listener are already in the address bar, the overview API is what this
  // page itself is calling, and the version endpoint just repeats the version
  // printed above — none of them tell an operator anything they cannot see.
  const rows = [
    [t("config.grpc"), server.grpcAddress],
    [t("config.secureGrpc"), server.secureGrpcAddress],
    [t("config.metrics"), server.metricsPath],
    [t("config.ready"), server.readyPath],
  ];
  return html`
    <div class="page">
      <div class="page-head">
        <div>
          <h1 class="page-title">${t("config.title")}</h1>
        </div>
        <${RefreshButton} onRefresh=${onRefresh} refreshing=${refreshing} />
      </div>
      <section class="section">
        ${rows.map(([label, value]) => html`
          <div class="field field-row" key=${label}>
            <div class="field-label">${label}</div>
            <div class="field-value mono field-copy" title=${t("config.copy")} onClick=${() => value && copyText(value)}>${value || "–"}</div>
          </div>
        `)}
      </section>
      <section class="section">
        <${SectionTitle}>${t("config.preferences")}</${SectionTitle}>
        <div class="pref-row">
          <div class="cell-strong">${t("config.theme")}</div>
          <div class="seg">
            ${["auto", "light", "dark"].map((m) => html`
              <button key=${m} class=${`seg-item ${theme === m ? "is-on" : ""}`} onClick=${() => { applyTheme(m); setTheme(m); }}>${t("config.theme." + m)}</button>
            `)}
          </div>
        </div>
        <div class="pref-row">
          <div class="cell-strong">${t("config.language")}</div>
          <div class="seg">
            ${LANGUAGES.map((l) => html`
              <button key=${l.id} class=${`seg-item ${lang === l.id ? "is-on" : ""}`}
                onClick=${() => onLanguage(l.id)}>${l.label}</button>
            `)}
          </div>
        </div>
      </section>
    </div>
  `;
};

// --- shell ------------------------------------------------------------------

const NAV = [
  { group: t("nav.mesh"), items: [
    { id: "overview", label: t("nav.overview") },
    { id: "services", label: t("nav.services") },
  ]},
  { group: t("nav.observe"), items: [
    { id: "pipeline", label: t("nav.pipeline") },
    { id: "logs", label: t("nav.logs") },
    { id: "topology", label: t("nav.topology") },
  ]},
  { group: t("nav.system"), items: [
    { id: "configuration", label: t("nav.configuration") },
  ]},
];

const NavIcon = ({ id }) => {
  const paths = {
    overview: html`<path d="M3 12h5V3H3zM10 21h5v-9h-5zM17 8h4V3h-4zM3 21h5v-6H3zM10 9h5V3h-5zM17 21h4V11h-4z"/>`,
    services: html`<circle cx="12" cy="12" r="8.5" fill="none"/><circle cx="12" cy="12" r="3"/>`,
    pipeline: html`<path d="M3 20h18M6 16l4-6 4 3 5-8" fill="none"/>`,
    logs: html`<path d="M4 5h16M4 10h16M4 15h10M4 20h7" fill="none"/>`,
    topology: html`<circle cx="5" cy="12" r="2.5" fill="none"/><circle cx="19" cy="6" r="2.5" fill="none"/><circle cx="19" cy="18" r="2.5" fill="none"/><path d="M7.5 11 16.6 6.9M7.5 13l9.1 4.1" fill="none"/>`,
    configuration: html`<rect x="4" y="4" width="16" height="16" rx="2" fill="none"/><path d="M9 9h6v6H9z" fill="none"/>`,
  };
  return html`<svg viewBox="0 0 24 24" class="nav-icon" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">${paths[id] || paths.overview}</svg>`;
};

const App = () => {
  const [route, setRoute] = useState(parseRoute);
  const [data, setData] = useState(null);
  const [overviewError, setOverviewError] = useState(null);
  const [drawer, setDrawer] = useState(null);
  const [logsState, setLogsState] = useState(null);

  const [metricsError, setMetricsError] = useState(null);
  const [refreshing, setRefreshing] = useState(false);
  const [lang, setLang] = useState(getLanguage());

  // Copy is resolved at render time, so re-rendering the tree is what applies a
  // language change; nothing else needs to know about it.
  const onLanguage = useCallback((next) => {
    setLanguage(next);
    setLang(next);
  }, []);
  const [metrics, setMetrics] = useState(null);

  const navigate = useCallback((id) => {
    setRoute(id);
    window.history.replaceState(null, "", `#/${id}`);
  }, []);

  useEffect(() => {
    const onHash = () => setRoute(parseRoute());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  const loadOverview = useCallback(async () => {
    try {
      setData(await getJSON(API.overview));
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
      setMetrics(await getJSON(API.metrics));
      setMetricsError(null);
    } catch (e) {
      setMetricsError(e.message || String(e));
    }
  }, []);

  useEffect(() => {
    pollMetrics();
    const timer = window.setInterval(pollMetrics, 15000);
    return () => window.clearInterval(timer);
  }, [pollMetrics]);

  // One control per page: pull both feeds so the button means the same thing
  // everywhere, and hold the disabled state long enough to be visible.
  const refresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([loadOverview(), pollMetrics()]);
    } finally {
      setRefreshing(false);
    }
  }, [loadOverview, pollMetrics]);

  const openLogs = useCallback(async (target, tail = 200) => {
    const next = { ...target, tail, title: target.title || target.name, loading: true, data: null, error: null };
    setLogsState(next);
    try {
      setLogsState({ ...next, loading: false, data: await fetchLogs({ ...target, tail }) });
    } catch (e) {
      setLogsState({ ...next, loading: false, error: e.message || String(e) });
    }
  }, []);

  if (!data && !overviewError) {
    return html`
      <div class="app">
        <aside class="sidebar" />
        <main class="main">
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
            <p class="cell-muted">
              ${t("error.noOverview.before")} <span class="mono">api/overview</span>${t("error.noOverview.after")}
            </p>
          </div>
        </main>
      </div>
    `;
  }

  const pageProps = { data, openDrawer: setDrawer, openLogs, navigate, onRefresh: refresh, refreshing, lang, onLanguage };

  return html`
    <div class="app">
      <aside class="sidebar">
        <div class="brand">
          <span class="brand-name">Dubbod GUI</span>
        </div>
        <nav class="nav">
          ${NAV.map((group) => html`
            <div class="nav-group" key=${group.group}>
              <div class="nav-group-label">${group.group}</div>
              ${group.items.map((item) => html`
                <button key=${item.id} type="button"
                  class=${`nav-item ${route === item.id ? "is-active" : ""}`}
                  onClick=${() => navigate(item.id)} aria-current=${route === item.id ? "page" : undefined}>
                  <${NavIcon} id=${item.id} />
                  <span>${item.label}</span>
                </button>
              `)}
            </div>
          `)}
        </nav>
      </aside>

      <main class="main">
        <div class="content" key=${route}>
          ${route === "overview" && html`<${OverviewPage} ...${pageProps} />`}
          ${route === "services" && html`<${ServicesPage} ...${pageProps} />`}
          ${route === "pipeline" && html`
            <${PipelinePage} snapshot=${metrics} error=${metricsError} retry=${pollMetrics}
              onRefresh=${refresh} refreshing=${refreshing} />
          `}
          ${route === "logs" && html`<${LogsPage} ...${pageProps} />`}
          ${route === "topology" && html`<${TopologyPage} ...${pageProps} />`}
          ${route === "configuration" && html`<${ConfigurationPage} ...${pageProps} />`}
        </div>
      </main>

      <${Drawer} item=${drawer} onClose=${() => setDrawer(null)} onOpenLogs=${(t) => { setDrawer(null); openLogs(t); }} />
      <${LogsPanel} state=${logsState} onClose=${() => setLogsState(null)} onReload=${(tail) => openLogs(logsState, tail)} />
    </div>
  `;
};

render(html`<${App} />`, document.getElementById("root"));
