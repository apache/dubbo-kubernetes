/**
 * Hand-rolled SVG chart primitives (no external chart library).
 *
 * Colors are read from CSS custom properties (--series-*, chart chrome tokens)
 * so light/dark themes swap in one place. Every multi-point chart ships a
 * hover layer; series identity never relies on color alone (legend + direct
 * labels).
 */

const html = window.html;
const { useState, useRef, useMemo } = window;

// --- formatting -----------------------------------------------------------

export const fmtNumber = (value) => {
  if (value == null || Number.isNaN(value)) return "–";
  const abs = Math.abs(value);
  if (abs >= 1e9) return (value / 1e9).toFixed(1) + "B";
  if (abs >= 1e6) return (value / 1e6).toFixed(1) + "M";
  if (abs >= 1e4) return (value / 1e3).toFixed(1) + "k";
  if (abs >= 100 || Number.isInteger(value)) return new Intl.NumberFormat("en-US").format(Math.round(value));
  return value.toFixed(abs >= 1 ? 1 : 2);
};

export const fmtDuration = (seconds) => {
  if (seconds == null || Number.isNaN(seconds)) return "–";
  if (seconds < 0.001) return (seconds * 1e6).toFixed(0) + "µs";
  if (seconds < 1) return (seconds * 1000).toFixed(seconds < 0.01 ? 1 : 0) + "ms";
  if (seconds < 90) return seconds.toFixed(1) + "s";
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m ${Math.floor(seconds % 60)}s`;
};

const fmtClock = (ms) => {
  const d = new Date(ms);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
};

const niceTicks = (max, count = 3) => {
  if (max <= 0) return [0];
  const raw = max / count;
  const mag = Math.pow(10, Math.floor(Math.log10(raw)));
  const norm = raw / mag;
  const step = (norm <= 1 ? 1 : norm <= 2 ? 2 : norm <= 5 ? 5 : 10) * mag;
  const ticks = [];
  for (let v = 0; v <= max + step * 0.001; v += step) ticks.push(v);
  return ticks;
};

// --- sparkline (stat tiles) ------------------------------------------------

export const Sparkline = ({ points, width = 120, height = 34, colorVar = "--series-1" }) => {
  if (!points || points.length < 2) {
    return html`<svg class="spark" width=${width} height=${height} aria-hidden="true"><line x1="0" y1=${height - 6} x2=${width} y2=${height - 6} class="spark-flat" /></svg>`;
  }
  const vs = points.map((p) => p.v);
  const min = Math.min(...vs);
  const max = Math.max(...vs);
  const span = max - min || 1;
  const t0 = points[0].t;
  const t1 = points[points.length - 1].t;
  const tSpan = t1 - t0 || 1;
  const px = (p) => ((p.t - t0) / tSpan) * (width - 4) + 2;
  const py = (p) => height - 5 - ((p.v - min) / span) * (height - 10);
  const d = points.map((p, i) => `${i ? "L" : "M"}${px(p).toFixed(1)},${py(p).toFixed(1)}`).join(" ");
  const area = `${d} L${px(points[points.length - 1]).toFixed(1)},${height - 2} L${px(points[0]).toFixed(1)},${height - 2} Z`;
  return html`
    <svg class="spark" width=${width} height=${height} aria-hidden="true">
      <path d=${area} fill=${`var(${colorVar})`} opacity="0.12" />
      <path d=${d} fill="none" stroke=${`var(${colorVar})`} stroke-width="1.5" stroke-linejoin="round" />
      <circle cx=${px(points[points.length - 1])} cy=${py(points[points.length - 1])} r="2.5" fill=${`var(${colorVar})`} />
    </svg>
  `;
};

// --- multi-series trend chart ----------------------------------------------

const PAD = { top: 14, right: 84, bottom: 24, left: 44 };

export const TrendChart = ({ series, height = 220, unit = "", format = fmtNumber, emptyHint }) => {
  const [hover, setHover] = useState(null);
  const wrapRef = useRef(null);
  const width = 760; // viewBox width; scales responsively via CSS

  const live = (series || []).filter((s) => s.points && s.points.length > 1);
  const flat = useMemo(() => live.flatMap((s) => s.points), [series]);

  if (live.length === 0) {
    return html`<div class="chart-empty">${emptyHint || "Collecting samples…"}</div>`;
  }

  const t0 = Math.min(...flat.map((p) => p.t));
  const t1 = Math.max(...flat.map((p) => p.t));
  const vMax = Math.max(...flat.map((p) => p.v), 0.0001);
  const ticks = niceTicks(vMax);
  const yMax = ticks[ticks.length - 1] || vMax;
  const plotW = width - PAD.left - PAD.right;
  const plotH = height - PAD.top - PAD.bottom;
  const x = (t) => PAD.left + ((t - t0) / (t1 - t0 || 1)) * plotW;
  const y = (v) => PAD.top + plotH - (v / yMax) * plotH;

  const onMove = (event) => {
    const svg = wrapRef.current?.querySelector("svg");
    if (!svg) return;
    const rect = svg.getBoundingClientRect();
    const fx = ((event.clientX - rect.left) / rect.width) * width;
    const t = t0 + ((fx - PAD.left) / plotW) * (t1 - t0);
    let best = null;
    for (const s of live) {
      let candidate = s.points[0];
      for (const p of s.points) if (Math.abs(p.t - t) < Math.abs(candidate.t - t)) candidate = p;
      if (!best || Math.abs(candidate.t - t) < Math.abs(best.t - t)) best = candidate;
    }
    if (best) setHover({ t: best.t, px: x(best.t) });
  };

  return html`
    <div class="chart-wrap" ref=${wrapRef} onMouseMove=${onMove} onMouseLeave=${() => setHover(null)}>
      <svg viewBox=${`0 0 ${width} ${height}`} class="chart" role="img" aria-label="trend chart">
        ${ticks.map((v) => html`
          <g key=${v}>
            <line x1=${PAD.left} y1=${y(v)} x2=${width - PAD.right} y2=${y(v)} class="chart-grid" />
            <text x=${PAD.left - 6} y=${y(v) + 3} class="chart-tick" text-anchor="end">${format(v)}</text>
          </g>
        `)}
        <line x1=${PAD.left} y1=${PAD.top + plotH} x2=${width - PAD.right} y2=${PAD.top + plotH} class="chart-axis" />
        <text x=${PAD.left} y=${height - 6} class="chart-tick">${fmtClock(t0)}</text>
        <text x=${width - PAD.right} y=${height - 6} class="chart-tick" text-anchor="end">${fmtClock(t1)}</text>

        ${(() => {
          // Direct end-labels, nudged apart when series ends collide.
          const labelYs = [];
          const placeLabel = (v) => {
            let ly = y(v) + 3;
            while (labelYs.some((other) => Math.abs(other - ly) < 11)) ly -= 11;
            labelYs.push(ly);
            return ly;
          };
          return live.map((s, i) => {
            const d = s.points.map((p, j) => `${j ? "L" : "M"}${x(p.t).toFixed(1)},${y(p.v).toFixed(1)}`).join(" ");
            const last = s.points[s.points.length - 1];
            return html`
              <g key=${s.name}>
                ${live.length === 1 && html`
                  <path d=${`${d} L${x(last.t).toFixed(1)},${PAD.top + plotH} L${x(s.points[0].t).toFixed(1)},${PAD.top + plotH} Z`}
                    fill=${`var(--series-${(i % 4) + 1})`} opacity="0.10" />
                `}
                <path d=${d} fill="none" stroke=${`var(--series-${(i % 4) + 1})`} stroke-width="2" stroke-linejoin="round" />
                <text x=${x(last.t) + 6} y=${placeLabel(last.v)} class="chart-series-label" fill=${`var(--series-${(i % 4) + 1})`}>${s.name}</text>
              </g>
            `;
          });
        })()}

        ${hover && html`<line x1=${hover.px} y1=${PAD.top} x2=${hover.px} y2=${PAD.top + plotH} class="chart-crosshair" />`}
        ${hover && live.map((s, i) => {
          const p = s.points.reduce((a, b) => (Math.abs(b.t - hover.t) < Math.abs(a.t - hover.t) ? b : a));
          return html`<circle key=${s.name} cx=${x(p.t)} cy=${y(p.v)} r="3.5" fill=${`var(--series-${(i % 4) + 1})`} stroke="var(--surface)" stroke-width="1.5" />`;
        })}
      </svg>
      ${hover && html`
        <div class="chart-tooltip" style=${{ left: `${(hover.px / width) * 100}%` }}>
          <div class="chart-tooltip-time">${fmtClock(hover.t)}</div>
          ${live.map((s, i) => {
            const p = s.points.reduce((a, b) => (Math.abs(b.t - hover.t) < Math.abs(a.t - hover.t) ? b : a));
            return html`
              <div class="chart-tooltip-row" key=${s.name}>
                <span class="chart-tooltip-swatch" style=${{ background: `var(--series-${(i % 4) + 1})` }} />
                <span class="chart-tooltip-name">${s.name}</span>
                <span class="chart-tooltip-value">${format(p.v)}${unit}</span>
              </div>
            `;
          })}
        </div>
      `}
      ${live.length > 1 && html`
        <div class="chart-legend">
          ${live.map((s, i) => html`
            <span class="chart-legend-item" key=${s.name}>
              <span class="chart-tooltip-swatch" style=${{ background: `var(--series-${(i % 4) + 1})` }} />${s.name}
            </span>
          `)}
        </div>
      `}
    </div>
  `;
};

// --- histogram bucket bars --------------------------------------------------

export const BucketBars = ({ buckets, height = 190 }) => {
  const [hover, setHover] = useState(null);
  // Convert cumulative Prometheus buckets to per-bucket counts.
  const bars = useMemo(() => {
    if (!buckets || buckets.length === 0) return [];
    const out = [];
    let prev = 0;
    for (const b of buckets) {
      out.push({ le: b.le, count: Math.max(0, b.count - prev) });
      prev = b.count;
    }
    while (out.length > 1 && out[out.length - 1].count === 0) out.pop();
    return out;
  }, [buckets]);

  if (bars.length === 0) return html`<div class="chart-empty">No histogram samples yet.</div>`;

  const width = 760;
  const pad = { top: 12, right: 12, bottom: 26, left: 44 };
  const plotW = width - pad.left - pad.right;
  const plotH = height - pad.top - pad.bottom;
  const max = Math.max(...bars.map((b) => b.count), 1);
  const bw = plotW / bars.length;
  const ticks = niceTicks(max);
  const yMax = ticks[ticks.length - 1] || max;
  const leLabel = (le) => (le === Infinity || le === "+Inf" ? "+Inf" : fmtDuration(le));
  // Sequential single-hue ramp: deeper blue = higher bucket bound.
  const rampOpacity = (i) => 0.35 + (0.65 * i) / Math.max(1, bars.length - 1);

  return html`
    <div class="chart-wrap">
      <svg viewBox=${`0 0 ${width} ${height}`} class="chart" role="img" aria-label="latency distribution">
        ${ticks.map((v) => html`
          <g key=${v}>
            <line x1=${pad.left} y1=${pad.top + plotH - (v / yMax) * plotH} x2=${width - pad.right} y2=${pad.top + plotH - (v / yMax) * plotH} class="chart-grid" />
            <text x=${pad.left - 6} y=${pad.top + plotH - (v / yMax) * plotH + 3} class="chart-tick" text-anchor="end">${fmtNumber(v)}</text>
          </g>
        `)}
        ${bars.map((b, i) => {
          const h = (b.count / yMax) * plotH;
          const bx = pad.left + i * bw + 3;
          return html`
            <g key=${i} onMouseEnter=${() => setHover(i)} onMouseLeave=${() => setHover(null)}>
              <rect x=${pad.left + i * bw} y=${pad.top} width=${bw} height=${plotH} fill="transparent" />
              <rect x=${bx} y=${pad.top + plotH - h} width=${Math.max(2, bw - 6)} height=${Math.max(h, b.count > 0 ? 2 : 0)}
                rx="3" fill="var(--series-1)" opacity=${hover === i ? 1 : rampOpacity(i)} />
              <text x=${pad.left + i * bw + bw / 2} y=${height - 8} class="chart-tick" text-anchor="middle">≤${leLabel(b.le)}</text>
              ${hover === i && html`
                <text x=${pad.left + i * bw + bw / 2} y=${pad.top + plotH - h - 6} class="chart-bar-label" text-anchor="middle">${fmtNumber(b.count)}</text>
              `}
            </g>
          `;
        })}
        <line x1=${pad.left} y1=${pad.top + plotH} x2=${width - pad.right} y2=${pad.top + plotH} class="chart-axis" />
      </svg>
    </div>
  `;
};

// Estimate a quantile from cumulative Prometheus buckets (linear interpolation).
export const bucketQuantile = (buckets, q) => {
  if (!buckets || buckets.length === 0) return null;
  const total = buckets[buckets.length - 1].count;
  if (!total) return null;
  const rank = q * total;
  let prevCount = 0;
  let prevLE = 0;
  for (const b of buckets) {
    if (b.count >= rank) {
      const le = b.le === Infinity || b.le === "+Inf" ? prevLE * 2 || 1 : Number(b.le);
      const span = b.count - prevCount;
      if (span <= 0) return le;
      return prevLE + ((rank - prevCount) / span) * (le - prevLE);
    }
    prevCount = b.count;
    prevLE = b.le === Infinity || b.le === "+Inf" ? prevLE : Number(b.le);
  }
  return prevLE;
};

// --- horizontal breakdown bars (label: value) --------------------------------

export const BreakdownBars = ({ items, format = fmtNumber }) => {
  if (!items || items.length === 0) return html`<div class="chart-empty">No data.</div>`;
  const max = Math.max(...items.map((d) => d.value), 1);
  return html`
    <div class="hbars">
      ${items.map((d, i) => html`
        <div class="hbar-row" key=${d.label}>
          <span class="hbar-label">${d.label}</span>
          <span class="hbar-track">
            <span class="hbar-fill" style=${{ width: `${Math.max(1, (d.value / max) * 100)}%`, background: `var(--series-${(i % 4) + 1})` }} />
          </span>
          <span class="hbar-value">${format(d.value)}</span>
        </div>
      `)}
    </div>
  `;
};
