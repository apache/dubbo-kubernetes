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
 * Hand-rolled display primitives (no external chart library).
 *
 * Colours come from CSS custom properties (--series-*) so light/dark themes swap
 * in one place. Everything here renders a value the control plane reports right
 * now; there are no client-side time series, because the backend keeps no
 * history and a buffer that starts empty on every page load cannot show an
 * operator the event they came to investigate.
 */

import { html, Fragment } from "./runtime.js";
import { t } from "./i18n.js";

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

// --- horizontal breakdown bars (label: value) --------------------------------

export const BreakdownBars = ({ items, format = fmtNumber }) => {
  if (!items || items.length === 0) return html`<div class="chart-empty">${t("state.none")}</div>`;
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

// --- histogram summary ------------------------------------------------------

/**
 * Read a Prometheus histogram sample the honest way.
 *
 * The dubbod pipeline histograms use very coarse bounds (0.01, 0.1, 1, 3, …),
 * so bucket-interpolated percentiles are mostly an artefact of the bucket
 * layout. `sum/count` is exact, so that is what we lead with; the buckets are
 * only used to state the band every sample actually landed in.
 */
export const histStats = (sample) => {
  if (!sample || !sample.count) return null;
  const buckets = sample.buckets || [];
  const perBucket = [];
  let prev = 0;
  for (const b of buckets) {
    perBucket.push({ le: b.le, count: Math.max(0, b.count - prev) });
    prev = b.count;
  }
  const occupied = perBucket.filter((b) => b.count > 0);
  return {
    count: sample.count,
    sum: sample.sum || 0,
    mean: sample.count > 0 ? (sample.sum || 0) / sample.count : null,
    perBucket,
    // Tightest bound that already covers every observation.
    ceiling: occupied.length > 0 ? occupied[occupied.length - 1].le : null,
    spans: occupied.length,
  };
};

/** Compact segmented bar showing which buckets a histogram's samples fell in. */
export const HistogramStrip = ({ stats }) => {
  if (!stats || stats.count === 0) return html`<div class="strip strip-empty" aria-hidden="true" />`;
  const bars = stats.perBucket;
  const total = stats.count || 1;
  return html`
    <div class="strip" role="img" aria-label=${`${stats.count} samples by latency bucket`}>
      ${bars.map((b, i) => b.count > 0 && html`
        <span key=${b.le} class="strip-seg"
          style=${{ flexGrow: b.count, opacity: 0.4 + (0.6 * i) / Math.max(1, bars.length - 1) }}
          title=${`${fmtNumber(b.count)} samples ≤ ${fmtDuration(b.le)} (${((b.count / total) * 100).toFixed(0)}%)`} />
      `)}
    </div>
  `;
};

// --- config push pipeline ----------------------------------------------------

/**
 * Stage-by-stage view of the config push path. Each stage is one real dubbod
 * histogram; stages the control plane has not exercised yet say so instead of
 * rendering an empty chart.
 */
export const StageFlow = ({ stages }) => {
  const timed = stages.filter((s) => s.stats && s.stats.mean != null);
  const totalMean = timed.reduce((a, s) => a + s.stats.mean, 0);
  return html`
    <div class="flow">
      ${stages.map((s, i) => html`
        <${Fragment} key=${s.key}>
          ${i > 0 && html`<div class="flow-arrow" aria-hidden="true">→</div>`}
          <div class=${`flow-stage ${s.stats ? "" : "is-idle"}`}>
            <div class="flow-stage-name">${t(s.label)}</div>
            <div class="flow-stage-value">${s.stats ? fmtDuration(s.stats.mean) : t("stage.noSamples")}</div>
            <div class="flow-stage-meta">
              ${s.stats
                ? t("stage.samples", fmtNumber(s.stats.count), fmtDuration(s.stats.ceiling))
                : t(s.idleHint)}
            </div>
            <${HistogramStrip} stats=${s.stats} />
            ${s.stats && totalMean > 0 && html`
              <div class="flow-stage-share">${t("stage.share", ((s.stats.mean / totalMean) * 100).toFixed(1))}</div>
            `}
          </div>
        </${Fragment}>
      `)}
    </div>
  `;
};
