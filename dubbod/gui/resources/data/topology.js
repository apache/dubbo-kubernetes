/**
 * Config-plane topology — the console's signature view.
 *
 * Every node and edge is a fact from GET api/overview: registries sync into
 * dubbod, dubbod provisions gateways and distributes xDS to namespaces of
 * mesh services. There is no request-level telemetry feed yet, so edges carry
 * relationship + state, never invented traffic numbers.
 */

const html = window.html;
const { useState, useMemo, useRef, useEffect, useCallback } = window;

const NODE_W = 168;
const NODE_H = 46;
const CP_W = 200;
const CP_H = 92;
const COL_X = [24, 320, 640];
const GROUP_PAD = 14;
const GROUP_HEAD = 30;
const ROW_GAP = 10;
const MAX_VISIBLE = 8;

const statusOf = {
  registry: (r) => (r.synced ? "ok" : "warn"),
  gateway: (g) => (g.isReady ? "ok" : (g.readyReplicas || 0) === 0 ? "err" : "warn"),
  controlplane: (instances) => {
    const ready = instances.filter((i) => i.isReady).length;
    if (instances.length === 0 || ready === 0) return "err";
    return ready === instances.length ? "ok" : "warn";
  },
};

export const buildGraph = (data, { expanded, namespaceFilter }) => {
  const nodes = [];
  const edges = [];

  const registries = data.registries || [];
  const gateways = data.gatewayInstances || [];
  const instances = data.instances || [];
  const services = (data.services || []).filter(
    (s) => !namespaceFilter || s.namespace === namespaceFilter
  );

  // Column 1: registries then gateways.
  let y = 56;
  registries.forEach((r) => {
    nodes.push({
      id: `registry:${r.provider}/${r.cluster}`, type: "registry", label: r.provider,
      sub: r.cluster, status: statusOf.registry(r), x: COL_X[0], y, w: NODE_W, h: NODE_H, data: r,
    });
    y += NODE_H + 18;
  });
  y += 40;
  const gatewayBandY = y;
  gateways.forEach((g) => {
    nodes.push({
      id: `gateway:${g.namespace}/${g.name}`, type: "gateway", label: g.gatewayName || g.name,
      sub: `${g.namespace} · ${g.readyReplicas || 0}/${g.desiredReplicas || 0} ready`,
      status: statusOf.gateway(g), x: COL_X[0], y, w: NODE_W, h: NODE_H, data: g,
    });
    y += NODE_H + 18;
  });
  const col1H = y;

  // Column 3: namespace groups of services.
  const byNamespace = new Map();
  services.forEach((s) => {
    if (!byNamespace.has(s.namespace)) byNamespace.set(s.namespace, []);
    byNamespace.get(s.namespace).push(s);
  });
  let gy = 56;
  const groups = [];
  [...byNamespace.entries()].sort((a, b) => a[0].localeCompare(b[0])).forEach(([ns, list]) => {
    const isExpanded = expanded.has(ns);
    const visible = isExpanded ? list : list.slice(0, MAX_VISIBLE);
    const hidden = list.length - visible.length;
    const rows = visible.length + (hidden > 0 || isExpanded ? 1 : 0);
    const h = GROUP_HEAD + GROUP_PAD + rows * (NODE_H + ROW_GAP);
    const group = { ns, x: COL_X[2], y: gy, w: NODE_W + GROUP_PAD * 2, h, count: list.length };
    groups.push(group);
    visible.forEach((s, i) => {
      nodes.push({
        id: `service:${s.hostname}`, type: "service", label: s.name || s.hostname,
        sub: s.registry + (s.meshExternal ? " · external" : ""), status: "unknown",
        x: group.x + GROUP_PAD, y: gy + GROUP_HEAD + GROUP_PAD + i * (NODE_H + ROW_GAP),
        w: NODE_W, h: NODE_H, data: s, group: ns,
      });
    });
    if (hidden > 0 || isExpanded) {
      nodes.push({
        id: `more:${ns}`, type: "more", label: hidden > 0 ? `+${hidden} more` : "collapse",
        x: group.x + GROUP_PAD, y: gy + GROUP_HEAD + GROUP_PAD + visible.length * (NODE_H + ROW_GAP),
        w: NODE_W, h: 30, group: ns,
      });
    }
    gy += h + 26;
  });
  const col3H = gy;

  // Column 2: control plane vertically centered against the taller side.
  const totalH = Math.max(col1H, col3H, 320);
  const cp = {
    id: "controlplane", type: "controlplane", label: "dubbod",
    sub: `${instances.filter((i) => i.isReady).length}/${instances.length} ready · ${data.counts?.xdsConnections ?? 0} xDS conns`,
    status: statusOf.controlplane(instances),
    x: COL_X[1], y: totalH / 2 - CP_H / 2, w: CP_W, h: CP_H,
    data: { instances, version: data.version, namespace: data.namespace },
  };
  nodes.push(cp);

  // Edges — all real relationships.
  nodes.filter((n) => n.type === "registry").forEach((n) => {
    edges.push({ id: `e:${n.id}`, from: n, to: cp, kind: "sync", ok: n.data.synced, label: n.data.synced ? "synced" : "syncing" });
  });
  nodes.filter((n) => n.type === "gateway").forEach((n) => {
    edges.push({ id: `e:${n.id}`, from: cp, to: n, kind: "provision", ok: n.data.isReady, label: "provisions" });
  });
  groups.forEach((g) => {
    edges.push({
      id: `e:ns:${g.ns}`, from: cp, kind: "xds", ok: true, label: `xDS · ${g.count} svc`,
      toPoint: { x: g.x, y: g.y + GROUP_HEAD / 2 + 8 }, group: g.ns,
    });
  });

  return { nodes, edges, groups, width: COL_X[2] + NODE_W + GROUP_PAD * 2 + 40, height: totalH + 40, gatewayBandY };
};

const edgePath = (x1, y1, x2, y2) => {
  const mx = (x1 + x2) / 2;
  return `M${x1},${y1} C${mx},${y1} ${mx},${y2} ${x2},${y2}`;
};

const NodeIcon = ({ type }) => {
  switch (type) {
    case "registry":
      return html`<svg x="9" y="9" width="14" height="14" viewBox="0 0 24 24" class="topo-icon"><ellipse cx="12" cy="6" rx="8" ry="3" fill="none"/><path d="M4 6v12c0 1.7 3.6 3 8 3s8-1.3 8-3V6" fill="none"/><path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3" fill="none"/></svg>`;
    case "controlplane":
      return html`<svg x="9" y="9" width="14" height="14" viewBox="0 0 24 24" class="topo-icon"><polygon points="12 2 2 7 12 12 22 7 12 2" fill="none"/><polyline points="2 17 12 22 22 17" fill="none"/><polyline points="2 12 12 17 22 12" fill="none"/></svg>`;
    case "gateway":
      return html`<svg x="9" y="9" width="14" height="14" viewBox="0 0 24 24" class="topo-icon"><path d="M4 21V8l8-5 8 5v13" fill="none"/><path d="M9 21v-6h6v6" fill="none"/></svg>`;
    default:
      return html`<svg x="9" y="9" width="14" height="14" viewBox="0 0 24 24" class="topo-icon"><circle cx="12" cy="12" r="9" fill="none"/><circle cx="12" cy="12" r="3.5" fill="none"/></svg>`;
  }
};

export const Topology = ({ data, onSelect, selectedId }) => {
  const [transform, setTransform] = useState({ x: 0, y: 0, k: 1 });
  const [expanded, setExpanded] = useState(new Set());
  const [namespaceFilter, setNamespaceFilter] = useState("");
  const [query, setQuery] = useState("");
  const [fullscreen, setFullscreen] = useState(false);
  const [dragOffsets, setDragOffsets] = useState(new Map());
  const containerRef = useRef(null);
  const gesture = useRef(null);

  const namespaces = useMemo(
    () => [...new Set((data.services || []).map((s) => s.namespace))].sort(),
    [data]
  );

  const graph = useMemo(
    () => buildGraph(data, { expanded, namespaceFilter }),
    [data, expanded, namespaceFilter]
  );

  const offset = (node) => dragOffsets.get(node.id) || { dx: 0, dy: 0 };
  const nx = (node) => node.x + offset(node).dx;
  const ny = (node) => node.y + offset(node).dy;

  const matches = (node) => {
    if (!query) return true;
    const q = query.toLowerCase();
    return [node.label, node.sub, node.data?.hostname, node.data?.cluster]
      .some((v) => v && String(v).toLowerCase().includes(q));
  };

  const fit = useCallback(() => {
    const el = containerRef.current;
    if (!el) return;
    const k = Math.min(el.clientWidth / graph.width, el.clientHeight / graph.height, 1.4);
    setTransform({
      x: (el.clientWidth - graph.width * k) / 2,
      y: Math.max(8, (el.clientHeight - graph.height * k) / 2),
      k,
    });
  }, [graph.width, graph.height]);

  useEffect(() => { fit(); }, [fullscreen, namespaceFilter]);
  useEffect(() => { fit(); }, []);

  const onWheel = (event) => {
    event.preventDefault();
    const el = containerRef.current.getBoundingClientRect();
    const px = event.clientX - el.left;
    const py = event.clientY - el.top;
    setTransform((t) => {
      const k = Math.min(2.5, Math.max(0.3, t.k * (event.deltaY < 0 ? 1.12 : 0.89)));
      return { k, x: px - ((px - t.x) / t.k) * k, y: py - ((py - t.y) / t.k) * k };
    });
  };

  const onPointerDown = (event, node) => {
    event.currentTarget.setPointerCapture?.(event.pointerId);
    gesture.current = {
      node, startX: event.clientX, startY: event.clientY,
      base: node ? { ...offset(node) } : { ...transform }, moved: false,
    };
  };

  const onPointerMove = (event) => {
    const g = gesture.current;
    if (!g) return;
    const dx = event.clientX - g.startX;
    const dy = event.clientY - g.startY;
    if (Math.abs(dx) + Math.abs(dy) > 3) g.moved = true;
    if (!g.moved) return;
    if (g.node) {
      setDragOffsets((prev) => {
        const next = new Map(prev);
        next.set(g.node.id, { dx: g.base.dx + dx / transform.k, dy: g.base.dy + dy / transform.k });
        return next;
      });
    } else {
      setTransform((t) => ({ ...t, x: g.base.x + dx, y: g.base.y + dy }));
    }
  };

  const onPointerUp = (event, node) => {
    const g = gesture.current;
    gesture.current = null;
    if (g && !g.moved && node) {
      if (node.type === "more") {
        setExpanded((prev) => {
          const next = new Set(prev);
          next.has(node.group) ? next.delete(node.group) : next.add(node.group);
          return next;
        });
      } else {
        onSelect?.(node);
      }
    }
  };

  const cp = graph.nodes.find((n) => n.type === "controlplane");

  return html`
    <div class=${`topo ${fullscreen ? "topo-fullscreen" : ""}`}>
      <div class="topo-toolbar">
        <input class="input topo-search" placeholder="Locate node…" value=${query} onInput=${(e) => setQuery(e.target.value)} />
        <select class="input topo-ns" value=${namespaceFilter} onChange=${(e) => setNamespaceFilter(e.target.value)}>
          <option value="">All namespaces</option>
          ${namespaces.map((ns) => html`<option key=${ns} value=${ns}>${ns}</option>`)}
        </select>
        <span class="topo-toolbar-gap" />
        <span class="chip chip-gate" title="Request-level telemetry is not connected; edges show configuration relationships only.">edge telemetry n/a</span>
        <button class="btn btn-ghost" onClick=${() => setTransform((t) => ({ ...t, k: Math.min(2.5, t.k * 1.2) }))} title="Zoom in">+</button>
        <button class="btn btn-ghost" onClick=${() => setTransform((t) => ({ ...t, k: Math.max(0.3, t.k / 1.2) }))} title="Zoom out">−</button>
        <button class="btn btn-ghost" onClick=${fit} title="Fit to view">Fit</button>
        <button class="btn btn-ghost" onClick=${() => setFullscreen(!fullscreen)}>${fullscreen ? "Exit" : "Fullscreen"}</button>
      </div>

      <div
        class="topo-canvas"
        ref=${containerRef}
        onWheel=${onWheel}
        onPointerDown=${(e) => { if (e.target === e.currentTarget || e.target.tagName === "svg") onPointerDown(e, null); }}
        onPointerMove=${onPointerMove}
        onPointerUp=${(e) => onPointerUp(e, null)}
      >
        <svg width="100%" height="100%">
          <g transform=${`translate(${transform.x},${transform.y}) scale(${transform.k})`}>
            <text x=${COL_X[0]} y="30" class="topo-band">REGISTRY PLANE</text>
            <text x=${COL_X[0]} y=${graph.gatewayBandY - 14} class="topo-band">GATEWAY PLANE</text>
            <text x=${COL_X[1]} y="30" class="topo-band">CONTROL PLANE</text>
            <text x=${COL_X[2]} y="30" class="topo-band">WORKLOAD PLANE</text>

            ${graph.edges.map((e) => {
              const from = e.from;
              const x1 = nx(from) + from.w;
              const y1 = ny(from) + from.h / 2;
              const x2 = e.toPoint ? e.toPoint.x : nx(e.to);
              const y2 = e.toPoint ? e.toPoint.y : ny(e.to) + e.to.h / 2;
              // Registry edges enter dubbod's left side; others leave its right.
              const flip = e.kind === "sync";
              const [ax1, ay1, ax2, ay2] = flip
                ? [nx(from) + from.w, y1, nx(e.to), ny(e.to) + e.to.h / 2]
                : [x1, y1, x2, y2];
              const dim = query && !(matches(from) || (e.to && matches(e.to)));
              return html`
                <g key=${e.id} class=${`topo-edge ${e.ok ? "" : "is-degraded"} ${dim ? "is-dim" : ""}`}>
                  <path d=${edgePath(ax1, ay1, ax2, ay2)} class="topo-edge-line" />
                  <text x=${(ax1 + ax2) / 2} y=${(ay1 + ay2) / 2 - 6} class="topo-edge-label" text-anchor="middle">${e.label}</text>
                </g>
              `;
            })}

            ${graph.groups.map((g) => html`
              <g key=${g.ns} class=${query && ![...graph.nodes].some((n) => n.group === g.ns && matches(n)) ? "is-dim" : ""}>
                <rect x=${g.x} y=${g.y} width=${g.w} height=${g.h} rx="10" class="topo-group" />
                <text x=${g.x + GROUP_PAD} y=${g.y + 20} class="topo-group-label">ns/${g.ns}</text>
                <text x=${g.x + g.w - GROUP_PAD} y=${g.y + 20} class="topo-group-count" text-anchor="end">${g.count}</text>
              </g>
            `)}

            ${graph.nodes.map((node) => {
              if (node.type === "more") {
                return html`
                  <g key=${node.id} class="topo-more" transform=${`translate(${nx(node)},${ny(node)})`}
                     onPointerDown=${(e) => { e.stopPropagation(); onPointerDown(e, node); }}
                     onPointerMove=${onPointerMove}
                     onPointerUp=${(e) => { e.stopPropagation(); onPointerUp(e, node); }}>
                    <rect width=${node.w} height=${node.h} rx="6" class="topo-more-box" />
                    <text x=${node.w / 2} y="19" text-anchor="middle" class="topo-more-label">${node.label}</text>
                  </g>
                `;
              }
              const dim = query && !matches(node);
              const hit = query && matches(node);
              return html`
                <g key=${node.id}
                   class=${`topo-node status-${node.status} ${selectedId === node.id ? "is-selected" : ""} ${dim ? "is-dim" : ""} ${hit ? "is-hit" : ""}`}
                   transform=${`translate(${nx(node)},${ny(node)})`}
                   onPointerDown=${(e) => { e.stopPropagation(); onPointerDown(e, node); }}
                   onPointerMove=${onPointerMove}
                   onPointerUp=${(e) => { e.stopPropagation(); onPointerUp(e, node); }}>
                  <rect width=${node.w} height=${node.h} rx="8" class="topo-node-box" />
                  <rect width="3" height=${node.h} rx="1.5" class="topo-node-strip" />
                  <${NodeIcon} type=${node.type} />
                  <text x="30" y=${node.type === "controlplane" ? 34 : 19} class="topo-node-label">${node.label}</text>
                  ${node.sub && html`<text x="30" y=${node.type === "controlplane" ? 52 : 35} class="topo-node-sub">${node.sub}</text>`}
                  ${node.type === "controlplane" && html`<text x="30" y="72" class="topo-node-sub">${(node.data.version || "").split("-")[0]}</text>`}
                </g>
              `;
            })}
          </g>
        </svg>

        <div class="topo-legend">
          <span><span class="dot status-ok" />ready / synced</span>
          <span><span class="dot status-warn" />degraded / syncing</span>
          <span><span class="dot status-err" />down</span>
          <span><span class="dot status-unknown" />no health feed (services)</span>
        </div>
      </div>
    </div>
  `;
};
