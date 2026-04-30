/**
 * Dubbo Control Plane GUI (React/Preact No-Build)
 * Kuma Style Re-implementation.
 */

const CONFIG = (() => {
  const el = document.getElementById("dubbod-gui-config");
  return el ? JSON.parse(el.textContent) : { basePath: "/gui", product: "Dubbo" };
})();

const API_URL = new URL("api/overview", document.baseURI).toString();
const LOGS_URL = new URL("api/logs", document.baseURI).toString();

// --- Icons (SVG as HTML) ---
const Icons = {
  Home: () => html`<svg className="k-nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>`,
  Mesh: () => html`<svg className="k-nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/></svg>`,
  MeshGateway: () => html`<svg className="k-nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 3a3 3 0 0 0-3 3v12a3 3 0 0 0 3 3 3 3 0 0 0 3-3 3 3 0 0 0-3-3H6a3 3 0 0 0-3 3 3 3 0 0 0 3 3 3 3 0 0 0 3-3V6a3 3 0 0 0-3-3 3 3 0 0 0-3 3 3 3 0 0 0 3 3h12a3 3 0 0 0 3-3 3 3 0 0 0-3-3z"></path></svg>`,
  Config: () => html`<svg className="k-nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`
};

// --- Subcomponents ---
const NavItem = ({ active, onClick, icon, label }) => html`
  <div className=${`k-nav-item ${active ? 'is-active' : ''}`} onClick=${onClick} title=${label}>
    <${icon} />
    <span>${label}</span>
  </div>
`;

const MetricCard = ({ label, value }) => html`
  <div className="k-card k-metric-card">
    <div className="k-metric-label">${label}</div>
    <div className="k-metric-value">${new Intl.NumberFormat().format(value || 0)}</div>
  </div>
`;

const Badge = ({ children, type, style }) => html`
  <span className=${`k-badge k-badge-${type || 'neutral'}`} style=${style}>${children}</span>
`;

const ComponentLink = ({ children, onClick }) => html`
  <button type="button" className="k-component-link" onClick=${onClick}>${children}</button>
`;

const LogsDrawer = ({ state, onClose, onRefresh }) => {
  if (!state) return null;
  const pods = state.data?.pods || [];
  return html`
    <div className="k-log-overlay">
      <div className="k-log-drawer">
        <div className="k-log-header">
          <div>
            <div className="k-log-title">${state.title}</div>
            <div className="k-log-subtitle">${state.namespace || '-'} ${state.kind || ''}</div>
          </div>
          <div className="k-log-actions">
            <button type="button" className="k-icon-button" onClick=${onRefresh} title="Refresh">↻</button>
            <button type="button" className="k-icon-button" onClick=${onClose} title="Close">×</button>
          </div>
        </div>
        <div className="k-log-body">
          ${state.loading && html`<div className="k-log-loading">Loading logs...</div>`}
          ${state.error && html`<div className="k-log-error">${state.error}</div>`}
          ${!state.loading && !state.error && pods.length === 0 && html`<div className="k-log-empty">No pods found.</div>`}
          ${!state.loading && !state.error && pods.map(p => html`
            <section className="k-log-pod" key=${`${p.name}/${p.container}`}>
              <div className="k-log-pod-header">
                <div>
                  <span className="k-log-pod-name">${p.name}</span>
                  <span className="k-log-container">${p.container}</span>
                </div>
                <${Badge} type=${p.ready ? 'success' : 'warning'}>${p.phase || 'unknown'}</${Badge}>
              </div>
              <pre className="k-log-pre">${p.error || p.logs || 'No logs returned.'}</pre>
            </section>
          `)}
        </div>
      </div>
    </div>
  `;
};

const CustomDropdown = ({ options, value, onChange }) => {
  return html`
    <div className="k-dropdown">
      <div className="k-dropdown-selected">
        ${value}
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style=${{width: '14px', height: '14px', marginLeft: '4px'}}><polyline points="6 9 12 15 18 9"></polyline></svg>
      </div>
      <div className="k-dropdown-menu">
        ${options.map(opt => html`
          <div className=${`k-dropdown-item ${opt === value ? 'is-active' : ''}`} onClick=${() => onChange(opt)}>
            ${opt === value && html`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style=${{width: '12px', height: '12px', marginRight: '6px'}}><polyline points="20 6 9 17 4 12"></polyline></svg>`}
            <span style=${{marginLeft: opt === value ? '0' : '18px'}}>${opt}</span>
          </div>
        `)}
      </div>
    </div>
  `;
};

const App = () => {
  const [data, setData] = useState(null);
  const [activeTab, setActiveTab] = useState('home');
  const [filter, setFilter] = useState('');
  const [selectedHostname, setSelectedHostname] = useState(null);
  const [logsState, setLogsState] = useState(null);

  const fetchData = async () => {
    try {
      const res = await fetch(API_URL);
      const json = await res.json();
      setData(json);
      if (!selectedHostname && json.services.length > 0) {
        setSelectedHostname(json.services[0].hostname);
      }
    } catch (e) {
      console.error("fetch failed", e);
    }
  };

  useEffect(() => {
    fetchData();
    const timer = setInterval(fetchData, 15000);
    return () => clearInterval(timer);
  }, []);

  const filteredServices = useMemo(() => {
    if (!data) return [];
    const f = filter.toLowerCase();
    return data.services.filter(s => 
      [s.name, s.hostname, s.namespace].some(v => v?.toLowerCase().includes(f))
    );
  }, [data, filter]);

  const openLogs = async (target) => {
    const next = { ...target, loading: true, data: null, error: null };
    setLogsState(next);
    try {
      const url = new URL(LOGS_URL);
      url.searchParams.set('kind', target.kind);
      url.searchParams.set('namespace', target.namespace || '');
      url.searchParams.set('name', target.name || '');
      const res = await fetch(url.toString());
      const payload = await res.json();
      if (!res.ok) {
        throw new Error(payload.error || `HTTP ${res.status}`);
      }
      setLogsState({ ...next, loading: false, data: payload });
    } catch (e) {
      setLogsState({ ...next, loading: false, error: e.message || String(e) });
    }
  };

  const refreshLogs = () => {
    if (logsState) {
      openLogs(logsState);
    }
  };

  if (!data) {
    return html`<div className="k-app"><div className="k-content"><div className="k-skeleton" style=${{height: '100%'}}></div></div></div>`;
  }

  return html`
    <div className="k-app fade-in">
      <aside className="k-sidebar">
        <div className="k-sidebar-brand">
          <div className="k-brand-mark" />
        </div>
        <nav className="k-nav">
          <${NavItem} active=${activeTab === 'home'} onClick=${() => setActiveTab('home')} icon=${Icons.Home} label="Home" />
          <${NavItem} active=${activeTab === 'mesh'} onClick=${() => setActiveTab('mesh')} icon=${Icons.Mesh} label="Mesh" />
          <${NavItem} active=${activeTab === 'meshgateway'} onClick=${() => setActiveTab('meshgateway')} icon=${Icons.MeshGateway} label="Mesh Gateway" />
          <${NavItem} active=${activeTab === 'configuration'} onClick=${() => setActiveTab('configuration')} icon=${Icons.Config} label="Configuration" />
        </nav>
      </aside>

      <main className="k-main">
        <header className="k-header">
          <div style=${{color: 'var(--text-muted)', fontSize: '13px', fontWeight: 500, display: 'flex', alignItems: 'center', gap: '8px'}}>
            Cluster: 
            <${CustomDropdown} 
              options=${[data.cluster || 'Kubernetes']} 
              value=${data.cluster || 'Kubernetes'} 
              onChange=${(val) => { console.log('Cluster changed to', val); }} 
            />
          </div>
          <div style=${{display: 'flex', alignItems: 'center', gap: '12px'}}>
            <div className="k-status-dot k-status-dot-active" />
            <span style=${{fontSize: '12px', color: 'var(--text-muted)', fontWeight: 600, textTransform: 'uppercase'}}>Live</span>
          </div>
        </header>

        <div className="k-content">
          ${activeTab === 'home' && html`
            <div className="fade-in">
              <h1 style=${{fontSize: '24px', fontWeight: 700, marginBottom: '24px'}}>Home</h1>

              <h2 style=${{fontSize: '16px', fontWeight: 600, marginBottom: '16px', marginTop: '16px'}}>Dubbo Control Plane</h2>
              <div className="k-table-container">
                <table className="k-table">
                  <thead>
                    <tr>
                      <th width="30%">COMPONENT</th>
                      <th width="20%">READY</th>
                      <th width="50%">NAMESPACE</th>
                    </tr>
                  </thead>
                  <tbody>
                    ${(() => {
                      const total = (data.instances || []).length;
                      const ready = (data.instances || []).filter(i => i.isReady).length;
                      const ns = total > 0 ? data.instances[0].namespace : '-';
                      return html`
                        <tr>
                          <td>
                            <${ComponentLink} onClick=${() => openLogs({ kind: 'dubbod', name: 'dubbod', namespace: ns, title: 'dubbod' })}>dubbod</${ComponentLink}>
                          </td>
                          <td>
                            <span style=${{fontSize: '13px', color: 'var(--text)', fontWeight: 500}}>${ready} / ${total}</span>
                          </td>
                          <td>
                            <span style=${{fontSize: '13px', color: 'var(--text-muted)'}}>${ns}</span>
                          </td>
                        </tr>
                      `;
                    })()}
                  </tbody>
                </table>
              </div>

              <h2 style=${{fontSize: '16px', fontWeight: 600, marginBottom: '16px', marginTop: '32px'}}>Dubbo Kubernetes Gateway</h2>
              <div className="k-table-container">
                <table className="k-table">
                  <thead>
                    <tr>
                      <th width="30%">COMPONENT</th>
                      <th width="20%">READY</th>
                      <th width="25%">NAMESPACE</th>
                      <th width="25%">GATEWAYCLASS</th>
                    </tr>
                  </thead>
                  <tbody>
                    ${(data.gatewayInstances || []).length === 0 && html`
                      <tr>
                        <td>
                          <div style=${{fontWeight: 600, color: 'var(--text-muted)'}}>No managed gateway deployments</div>
                        </td>
                        <td><span style=${{fontSize: '13px', color: 'var(--text-muted)', fontWeight: 500}}>0 / 0</span></td>
                        <td><span style=${{fontSize: '13px', color: 'var(--text-muted)'}}>-</span></td>
                        <td><span style=${{fontSize: '13px', color: 'var(--text-muted)'}}>-</span></td>
                      </tr>
                    `}
                    ${(data.gatewayInstances || []).map(g => html`
                      <tr key=${`${g.namespace}/${g.name}`}>
                        <td>
                          <${ComponentLink} onClick=${() => openLogs({ kind: 'gateway', name: g.name, namespace: g.namespace, title: g.name })}>${g.name}</${ComponentLink}>
                        </td>
                        <td>
                          <span style=${{fontSize: '13px', color: g.isReady ? 'var(--text)' : 'var(--text-muted)', fontWeight: 500}}>${g.readyReplicas || 0} / ${g.desiredReplicas || 0}</span>
                        </td>
                        <td>
                          <span style=${{fontSize: '13px', color: 'var(--text-muted)'}}>${g.namespace || '-'}</span>
                        </td>
                        <td>
                          <span style=${{fontSize: '13px', color: 'var(--text-muted)'}}>${g.gatewayClass || '-'}</span>
                        </td>
                      </tr>
                    `)}
                  </tbody>
                </table>
              </div>
            </div>
          `}

          ${activeTab === 'mesh' && html`
            <div className="fade-in">
              <div style=${{display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px'}}>
                <h1 style=${{fontSize: '24px', fontWeight: 700}}>Mesh Services</h1>
                <input 
                  className="k-search-input" 
                  placeholder="Filter by name, host or namespace..." 
                  value=${filter}
                  onInput=${e => setFilter(e.target.value)}
                />
              </div>

              <div className="k-table-container">
                <table className="k-table">
                  <thead>
                    <tr>
                      <th>Service / Host</th>
                      <th>Namespace</th>
                      <th>Registry</th>
                      <th>Ports</th>
                      <th>Exposure (E-W Traffic)</th>
                    </tr>
                  </thead>
                  <tbody>
                    ${filteredServices.map(s => html`
                      <tr 
                        key=${s.hostname} 
                        className=${s.hostname === selectedHostname ? 'is-active' : ''}
                        onClick=${() => setSelectedHostname(s.hostname)}
                      >
                        <td>
                          <div style=${{fontWeight: 600}}>${s.name || s.hostname}</div>
                          <div style=${{color: 'var(--text-muted)', fontSize: '11px', fontFamily: 'var(--font-mono)'}}>${s.hostname}</div>
                        </td>
                        <td><${Badge} type="neutral" style=${{marginLeft: '-8px'}}>${s.namespace || 'default'}</${Badge}></td>
                        <td>${s.registry}</td>
                        <td style=${{fontFamily: 'var(--font-mono)', fontSize: '12px'}}>${s.ports}</td>
                        <td>
                          <${Badge} type=${s.exposure === 'internal' ? 'success' : 'warning'} style=${{marginLeft: '-8px'}}>${s.exposure}</${Badge}>
                        </td>
                      </tr>
                    `)}
                  </tbody>
                </table>
              </div>
            </div>
          `}

          ${activeTab === 'meshgateway' && html`
            <div className="fade-in">
              <h1 style=${{fontSize: '24px', fontWeight: 700, marginBottom: '24px'}}>Mesh Gateway</h1>
              <div className="k-card">
                <p style=${{color: 'var(--text-muted)'}}>Manage Ingress/Egress, North-South traffic, and Cross-Region Clusters.</p>
                <div style=${{marginTop: '16px', borderTop: '1px solid var(--bg-border)', paddingTop: '16px'}}>
                  <!-- Placeholder for Gateway Configuration List -->
                  <div style=${{display: 'flex', alignItems: 'center', justifyContent: 'center', height: '120px', color: 'var(--text-dim)'}}>
                    No gateways configured currently.
                  </div>
                </div>
              </div>
            </div>
          `}

          ${activeTab === 'configuration' && html`
            <div className="fade-in">
              <h1 style=${{fontSize: '24px', fontWeight: 700, marginBottom: '24px'}}>Configurations</h1>
              <div className="k-table-container">
                <table className="k-table">
                  <thead>
                    <tr>
                      <th>Kind (VirtualService / MeshGlobalSetup)</th>
                      <th>Count</th>
                      <th>Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    ${data.configKinds.map(k => html`
                      <tr key=${k.kind}>
                        <td><div style=${{fontWeight: 600}}>${k.kind}</div></td>
                        <td style=${{fontFamily: 'var(--font-mono)', fontSize: '14px', color: 'var(--primary)'}}>${k.count}</td>
                        <td style=${{color: 'var(--text-muted)'}}>${k.description}</td>
                      </tr>
                    `)}
                  </tbody>
                </table>
              </div>
            </div>
          `}
        </div>
      </main>
      <${LogsDrawer} state=${logsState} onClose=${() => setLogsState(null)} onRefresh=${refreshLogs} />
    </div>
  `;
};

// 轮询直到 Preact 加载完成并渲染
const checkAndRender = () => {
  if (window.render && window.html) {
    window.render(window.html`<${App} />`, document.getElementById('root'));
  } else {
    setTimeout(checkAndRender, 50);
  }
};

checkAndRender();
