/**
 * Development mock data.
 *
 * Every payload here is structurally identical to the backend responses in
 * dubbod/discovery/pkg/bootstrap/gui.go (guiOverview, guiLogsResponse and the
 * api/metrics JSON). Field names and types must stay in lockstep with those
 * structs so the UI cannot drift from the real contract.
 */

const now = () => new Date().toISOString();

export const mockOverview = () => ({
  product: "Dubbo",
  version: "1.1.0-mock (go1.23)",
  clusterId: "Kubernetes",
  namespace: "dubbo-system",
  podName: "dubbod-7c9f6d4b58-x2m4q",
  mesh: {
    trustDomain: "cluster.local",
    rootNamespace: "dubbo-system",
    discoveryAddress: "dubbod.dubbo-system.svc:15010",
  },
  server: {
    guiPath: "/gui",
    httpAddress: "127.0.0.1:26080",
    grpcAddress: ":15010",
    secureGrpcAddress: ":15012",
    overviewPath: "http://127.0.0.1:26080/gui/api/overview",
    metricsPath: "http://127.0.0.1:8080/metrics",
    versionPath: "http://127.0.0.1:8080/version",
    readyPath: "https://127.0.0.1:8443/ready",
  },
  status: {
    xdsServerReady: true,
    cachesSynced: true,
    servicesSynced: true,
    configSynced: true,
    proxylessSynced: true,
    injectorReady: true,
    validationReady: false,
  },
  counts: {
    services: 9,
    endpointServices: 12,
    xdsConnections: 17,
    registries: 2,
    peerAuthentications: 1,
    requestAuthentications: 0,
    authorizationPolicies: 2,
    httpRoutes: 4,
    gatewayClasses: 1,
    gateways: 2,
  },
  configKinds: [
    { kind: "PeerAuthentication", count: 1, description: "mTLS posture and workload identity policy." },
    { kind: "RequestAuthentication", count: 0, description: "JWT request authentication policy." },
    { kind: "AuthorizationPolicy", count: 2, description: "Request authorization policy." },
    { kind: "HTTPRoute", count: 4, description: "Gateway API HTTP routing resources." },
    { kind: "GatewayClass", count: 1, description: "Gateway controller classes in scope." },
    { kind: "Gateway", count: 2, description: "Gateway instances served by the control plane." },
  ],
  registries: [
    { provider: "Kubernetes", cluster: "Kubernetes", synced: true },
    { provider: "Nacos", cluster: "nacos-prod", synced: false },
  ],
  services: [
    { name: "shop-order", hostname: "shop-order.shop.svc.cluster.local", namespace: "shop", registry: "Kubernetes", ports: "grpc:20880/GRPC, http:8080/HTTP", exposure: "internal", serviceAccounts: 1, defaultAddress: "10.96.12.4", meshExternal: false },
    { name: "shop-user", hostname: "shop-user.shop.svc.cluster.local", namespace: "shop", registry: "Kubernetes", ports: "grpc:20880/GRPC", exposure: "internal", serviceAccounts: 1, defaultAddress: "10.96.12.9", meshExternal: false },
    { name: "shop-inventory", hostname: "shop-inventory.shop.svc.cluster.local", namespace: "shop", registry: "Kubernetes", ports: "grpc:20880/GRPC", exposure: "internal", serviceAccounts: 1, defaultAddress: "10.96.13.2", meshExternal: false },
    { name: "shop-payment", hostname: "shop-payment.shop.svc.cluster.local", namespace: "shop", registry: "Kubernetes", ports: "grpc:20880/GRPC, metrics:9090/HTTP", exposure: "internal", serviceAccounts: 2, defaultAddress: "10.96.13.7", meshExternal: false },
    { name: "search-api", hostname: "search-api.search.svc.cluster.local", namespace: "search", registry: "Kubernetes", ports: "http:8080/HTTP", exposure: "clusterip", serviceAccounts: 1, defaultAddress: "10.96.20.3", meshExternal: false },
    { name: "search-indexer", hostname: "search-indexer.search.svc.cluster.local", namespace: "search", registry: "Kubernetes", ports: "grpc:20880/GRPC", exposure: "internal", serviceAccounts: 1, defaultAddress: "10.96.20.8", meshExternal: false },
    { name: "legacy-quote", hostname: "legacy-quote.dubbo.example.com", namespace: "shop", registry: "Nacos", ports: "dubbo:20880/TCP", exposure: "dns", serviceAccounts: 0, defaultAddress: "", meshExternal: true },
    { name: "recs-ranker", hostname: "recs-ranker.recs.svc.cluster.local", namespace: "recs", registry: "Kubernetes", ports: "grpc:20880/GRPC", exposure: "internal", serviceAccounts: 1, defaultAddress: "10.96.31.5", meshExternal: false },
    { name: "recs-features", hostname: "recs-features.recs.svc.cluster.local", namespace: "recs", registry: "Kubernetes", ports: "grpc:20880/GRPC", exposure: "passthrough", serviceAccounts: 1, defaultAddress: "", meshExternal: false },
  ],
  instances: [
    { name: "dubbod-7c9f6d4b58-x2m4q", namespace: "dubbo-system", ip: "10.96.0.14", isReady: true },
    { name: "dubbod-7c9f6d4b58-9kzlp", namespace: "dubbo-system", ip: "10.96.0.15", isReady: true },
  ],
  gatewayInstances: [
    { name: "shop-gateway-dubbo", namespace: "shop", ip: "", isReady: true, gatewayClass: "dubbo", gatewayName: "shop-gateway", readyReplicas: 2, desiredReplicas: 2 },
    { name: "edge-gateway-dubbo", namespace: "edge", ip: "", isReady: false, gatewayClass: "dubbo", gatewayName: "edge-gateway", readyReplicas: 0, desiredReplicas: 1 },
  ],
  updatedAt: now(),
});

const mockLogLines = (name) => {
  const t = Date.now();
  const stamp = (offsetSeconds) => new Date(t - offsetSeconds * 1000).toISOString();
  return [
    `${stamp(340)} info FLAG: --guiAddr=":26080"`,
    `${stamp(339)} info initializing mesh configuration`,
    `${stamp(338)} info kube controller for ${name} started`,
    `${stamp(320)} info ads ADS: new connection for node sidecar~10.96.12.4~shop-order`,
    `${stamp(290)} info ads CDS: PUSH for node:shop-order resources:14 size:9.2kB`,
    `${stamp(289)} info ads EDS: PUSH for node:shop-order resources:12 size:4.1kB`,
    `${stamp(240)} warn ads ADS: node shop-inventory watch stale, forcing full push`,
    `${stamp(180)} info ads LDS: PUSH for node:shop-user resources:6 size:2.8kB`,
    `${stamp(120)} error ads gRPC send timeout for node recs-features, closing stream`,
    `${stamp(60)} info ads RDS: PUSH for node:shop-user resources:4 size:1.2kB`,
    `${stamp(5)} info ads EDS: incremental push for endpoints shard shop/shop-order`,
  ].join("\n");
};

export const mockLogs = ({ kind, namespace, name }) => ({
  kind: kind || "dubbod",
  name: name || "dubbod",
  namespace: namespace || "dubbo-system",
  pods: [
    {
      name: `${name || "dubbod"}-7c9f6d4b58-x2m4q`,
      container: kind === "gateway" ? "dxgate" : "execute",
      phase: "Running",
      ready: true,
      logs: mockLogLines(name || "dubbod"),
    },
    {
      name: `${name || "dubbod"}-7c9f6d4b58-9kzlp`,
      container: kind === "gateway" ? "dxgate" : "execute",
      phase: "Running",
      ready: true,
      logs: mockLogLines(name || "dubbod"),
    },
  ],
  updatedAt: now(),
});

// Mutable counters so successive polls show live movement, like a real
// control plane under light load.
const state = {
  start: Date.now(),
  pushes: { cds: 482, eds: 1290, lds: 466, rds: 431 },
  connections: 17,
  services: 9,
};

export const mockMetrics = () => {
  state.pushes.eds += Math.floor(Math.random() * 6);
  state.pushes.cds += Math.random() < 0.3 ? 1 : 0;
  state.pushes.lds += Math.random() < 0.2 ? 1 : 0;
  state.pushes.rds += Math.random() < 0.2 ? 1 : 0;
  if (Math.random() < 0.1) state.connections += Math.random() < 0.5 ? 1 : -1;

  const uptime = (Date.now() - state.start) / 1000 + 86400 * 3.2;
  const totalPushes = Object.values(state.pushes).reduce((a, b) => a + b, 0);

  return {
    updatedAt: now(),
    families: [
      { name: "dubbod_uptime_seconds", help: "Current dubbod server uptime in seconds", type: "GAUGE", metrics: [{ value: uptime }] },
      { name: "dubbod_info", help: "Dubbod version and build information.", type: "GAUGE", metrics: [{ labels: { version: "1.1.0-mock" }, value: 1 }] },
      { name: "dubbod_xds", help: "Number of endpoints connected to this pilot using XDS.", type: "GAUGE", metrics: [{ labels: { version: "1.1.0-mock" }, value: state.connections }] },
      { name: "dubbod_services", help: "Total services known to pilot.", type: "GAUGE", metrics: [{ value: state.services }] },
      {
        name: "dubbod_xds_pushes", help: "Pilot build and send errors for lds, rds, cds and eds.", type: "COUNTER",
        metrics: Object.entries(state.pushes).map(([type, v]) => ({ labels: { type }, value: v })),
      },
      {
        name: "dubbod_xds_push_time", help: "Total time in seconds Pilot takes to push lds, rds, cds and eds.", type: "HISTOGRAM",
        metrics: [{
          count: totalPushes,
          sum: totalPushes * 0.0125,
          buckets: [
            { le: 0.01, count: Math.floor(totalPushes * 0.55) },
            { le: 0.1, count: Math.floor(totalPushes * 0.86) },
            { le: 1, count: Math.floor(totalPushes * 0.97) },
            { le: 3, count: Math.floor(totalPushes * 0.99) },
            { le: 5, count: totalPushes },
            { le: 10, count: totalPushes },
            { le: 20, count: totalPushes },
            { le: 30, count: totalPushes },
            { le: Infinity, count: totalPushes },
          ],
        }],
      },
      { name: "dubbod_total_xds_internal_errors", help: "Total number of internal XDS errors in pilot.", type: "COUNTER", metrics: [{ value: 2 }] },
      { name: "dubbod_total_xds_rejects", help: "Total number of XDS responses from pilot rejected by proxy.", type: "COUNTER", metrics: [{ value: 0 }] },
      {
        name: "dubbod_push_triggers", help: "Total number of times a push was triggered.", type: "COUNTER",
        metrics: [
          { labels: { type: "endpoint" }, value: 812 },
          { labels: { type: "config" }, value: 133 },
          { labels: { type: "service" }, value: 96 },
          { labels: { type: "proxy" }, value: 44 },
        ],
      },
    ],
  };
};
