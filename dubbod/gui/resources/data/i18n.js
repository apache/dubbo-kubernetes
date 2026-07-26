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
 * Console copy in English and Simplified Chinese.
 *
 * Only prose is translated. Identifiers stay verbatim in both languages —
 * Kubernetes labels, xDS resource types, mTLS mode names, metric labels and
 * addresses are things the reader will grep for or type into kubectl, and
 * translating them would break that.
 */

export const LANGUAGES = [
  { id: "en", label: "English" },
  { id: "zh", label: "简体中文" },
];

const STRINGS = {
  en: {
    // shell
    "nav.mesh": "Mesh",
    "nav.observe": "Observe",
    "nav.system": "System",
    "nav.overview": "Overview",
    "nav.services": "Services",
    "nav.pipeline": "Pipeline",
    "nav.logs": "Logs",
    "nav.topology": "Topology",
    "nav.configuration": "Configuration",
    "action.refresh": "Refresh",
    "action.refreshing": "Refreshing…",
    "action.refresh.title": "Fetch the current state now",
    "action.retry": "Retry",
    "action.close": "Close",
    "error.request": "Request failed",
    "status.ok": "healthy",
    "status.warn": "degraded",
    "status.err": "unhealthy",
    "status.unknown": "unknown",
    "error.noOverview.before": "The console renders nothing without",
    "error.noOverview.after":
      ". Check that the control plane is ready and that this listener is reachable.",

    // overview
    "overview.title": "Overview",
    "overview.controlPlane": "Control Plane",
    "overview.dataPlane": "Data Plane",
    "overview.externalDataPlane": "External Data Plane",
    "col.pod": "Pod",
    "col.podIP": "Pod IP",
    "col.namespace": "Namespace",
    "col.state": "State",
    "col.injected": "Injected",
    "col.inboundL4": "Inbound L4",
    "col.mtls": "mTLS",
    "col.certificate": "Certificate",
    "col.trustRoot": "Trust root",
    "col.restarts": "Restarts",
    "col.deployment": "Deployment",
    "col.gateway": "Gateway",
    "col.class": "Class",
    "col.replicas": "Replicas",
    "col.role": "Role",
    "col.pods": "Pods",
    "col.service": "Service",
    "col.registry": "Registry",
    "col.ports": "Ports",
    "col.clusterAddress": "Cluster address",
    "col.exposure": "Exposure",
    "state.ready": "ready",
    "state.notReady": "not ready",
    "state.accepting": "accepting",
    "state.notAccepting": (n, total) => `${n} of ${total} not accepting`,
    "state.current": "current",
    "state.superseded": (n) => `${n} superseded`,
    "state.meshDefault": "mesh default",
    "state.unknown": "unknown",
    "state.none": "none",
    "state.expired": "expired",
    "state.left": (d) => `${d} left`,
    "empty.noDubbodPods": "No dubbod pods visible",
    "empty.noDubbodPods.body.before":
      "This console is served by the control plane, so it is running — but no pod labelled",
    "empty.noDubbodPods.body.after": "is readable in",
    "empty.noInjected": "No injected workloads",
    "empty.noInjected.body.before": "Label a namespace with",
    "empty.noInjected.body.after":
      "and restart its pods; each one then runs the inbound proxy and is counted here.",
    "empty.noGateways": "No gateway deployments",
    "empty.noGateways.body.before": "Create a Gateway with",
    "empty.noGateways.body.after": "and the control plane provisions its",
    "empty.noGateways.body.tail": "deployment here.",

    // services
    "services.title": "Services",
    "services.filter": "Filter by name, host or namespace…",
    "services.allNamespaces": "All namespaces",
    "services.allRegistries": "All registries",
    "services.shown": (n) => `${n} shown`,
    "empty.noServices": "No mesh services discovered",
    "empty.noServices.body.before": "Label a namespace with",
    "empty.noServices.body.after": "to bring its services into the mesh.",
    "empty.noServicesMatch": "No services match this filter",
    "empty.noServicesMatch.body": (n) => `Clear the filters to see all ${n} services.`,

    // pipeline
    "pipeline.title": "Config push pipeline",
    "pipeline.pushPath": "Push path",
    "pipeline.meanEndToEnd": (d) => `${d} mean end to end`,
    "pipeline.inboundUpdates": "Inbound updates",
    "pipeline.inboundUpdates.unit": "cumulative by source",
    "pipeline.pushTriggers": "Push triggers",
    "pipeline.pushTriggers.unit": "cumulative by reason",
    "stage.debounce": "Debounce",
    "stage.pushcontext": "PushContext",
    "stage.queue": "Proxy queue",
    "stage.send": "Send",
    "stage.noSamples": "no samples",
    "stage.idle.noConfigChange": "no config change yet",
    "stage.idle.noPushContext": "no push context built yet",
    "stage.idle.needsProxy": "needs a connected proxy",
    "stage.samples": (n, ceiling) => `n=${n} · all ≤ ${ceiling}`,
    "stage.share": (pct) => `${pct}% of path`,

    // logs
    "logs.title": "Logs",
    "logs.role.controlPlane": "control plane",
    "logs.role.externalDataPlane": "external data plane",
    "logs.search": "Search in logs…",
    "logs.level.all": "all",
    "logs.level.info": "info",
    "logs.level.warn": "warn",
    "logs.level.error": "error",
    "logs.tail": (n) => `tail ${n}`,
    "logs.lines": (n) => `${n} lines`,
    "logs.noMatch": "No lines match.",
    "logs.noPods": "No pods matched this deployment",
    "logs.refused": "Kubernetes refused the log stream:",
    "logs.download": "Download",
    "logs.reload": "Refresh",

    // topology
    "topology.title": "Topology",
    "topology.scope.northSouth": "North–south · ingress",
    "topology.scope.eastWest": "East–west · service to service",
    "topology.scope.controlPlane": "Control plane · xDS distribution",
    "topology.legend.required": "mTLS required",
    "topology.legend.plaintext": "plaintext accepted",
    "topology.legend.off": "mTLS off",
    "topology.legend.stream": "live xDS stream",
    "topology.legend.configOnly": "config only, no stream",
    "topology.routes": (n) =>
      `${n} HTTPRoute${n === 1 ? "" : "s"} · configured paths, not measured traffic`,
    "topology.streams": (n) => `${n} live xDS stream${n === 1 ? "" : "s"}`,
    "topology.omitted": (n) => `${n} node${n === 1 ? "" : "s"} not on a path in this view:`,
    "topology.empty.northSouth": "No gateway ingress routes",
    "topology.empty.northSouth.body":
      "No HTTPRoute names a Gateway as its parent, so nothing enters the mesh through one.",
    "topology.empty.eastWest": "No service-to-service routes",
    "topology.empty.eastWest.body":
      "No HTTPRoute names a Service as its parent, so no meshed workload routes to another.",
    "topology.empty.noServices": "No mesh services",
    "topology.empty.noServices.body.before": "Label a namespace with",
    "topology.empty.noServices.body.after": "so its services join the mesh and appear here.",
    "topology.zoomIn": "Zoom in",
    "topology.zoomOut": "Zoom out",
    "topology.reset": "Reset",
    "topology.reset.title": "Reset layout and zoom",
    "topology.aria": "configured call paths between mesh services",

    // drawer
    "drawer.kind": "Kind",
    "drawer.kind.gateway": "managed gateway",
    "drawer.kind.service": "mesh service",
    "drawer.kind.controlPlane": "control plane",
    "drawer.controlPlane.role":
      "Programs every node in this graph over xDS. It carries no application traffic, so it has no inbound mTLS mode and no request path of its own.",
    "drawer.reachedBy": "Reached by",
    "drawer.reachedBy.value": (n) => `${n} route rule${n === 1 ? "" : "s"}`,
    "drawer.forwardsTo": "Forwards to",
    "drawer.forwardsTo.none": "nothing — no HTTPRoute is attached to this service",
    "drawer.hostname": "Hostname",
    "drawer.serviceAccounts": "Service accounts",
    "drawer.meshExternal": "Mesh external",
    "drawer.copyHostname": "Copy hostname",
    "drawer.gatewayClass": "Gateway class",
    "drawer.viewLogs": "View logs",
    "drawer.provider": "Provider",
    "drawer.cluster": "Cluster",
    "drawer.informerSync": "Informer sync",
    "drawer.synced": "synced",
    "drawer.syncing": "syncing",
    "drawer.injectedPods": "Injected pods",
    "drawer.injectedPods.value": (n, total) => `${n} of ${total} running pods`,
    "drawer.injectedPods.gap": " — the rest have no sidecar and are not in the mesh",
    "drawer.inboundProxy": "Inbound proxy",
    "drawer.mtlsPerPort": "mTLS mode per port",
    "drawer.mtlsPerPort.none": "no per-port mode set; the mesh default applies",
    "drawer.soonestExpiry": "Soonest certificate expiry",
    "drawer.soonestExpiry.none": "no certificate found",
    "drawer.trustRoot": "Trust root",
    "drawer.trustRoot.ok": "every pod holds the root this control plane is issuing with",
    "drawer.trustRoot.stale": (n) =>
      `${n} pod(s) still hold a superseded root and need a reissue`,
    "drawer.problem": "Problem",
    "drawer.pods": "Pods",
    "drawer.inboundMTLS": "Inbound mTLS",
    "drawer.mtls.fromPolicy": "set by a PeerAuthentication",
    "drawer.mtls.fallback":
      "no PeerAuthentication selects this workload, so the sidecar falls back to this mode",
    "drawer.role": "Role",

    // configuration
    "config.title": "Configuration",
    "config.grpc": "gRPC (xDS)",
    "config.secureGrpc": "Secure gRPC (mTLS xDS)",
    "config.metrics": "Prometheus metrics",
    "config.ready": "Readiness probe",
    "config.copy": "Click to copy",
    "config.preferences": "Preferences",
    "config.theme": "Theme",
    "config.theme.auto": "auto",
    "config.theme.light": "light",
    "config.theme.dark": "dark",
    "config.language": "Language",

    // mTLS
    "mtls.strict.label": "STRICT — callers must use mTLS",
    "mtls.permissive.label": "PERMISSIVE — plaintext is still accepted",
    "mtls.disable.label": "DISABLE — inbound mTLS is off",
    "mtls.source.policy": "set by PeerAuthentication",
    "mtls.source.fallback": "sidecar fallback, no PeerAuthentication selects this workload",
  },

  zh: {
    "nav.mesh": "网格",
    "nav.observe": "观测",
    "nav.system": "系统",
    "nav.overview": "总览",
    "nav.services": "服务",
    "nav.pipeline": "下发流水线",
    "nav.logs": "日志",
    "nav.topology": "拓扑",
    "nav.configuration": "配置",
    "action.refresh": "刷新",
    "action.refreshing": "刷新中…",
    "action.refresh.title": "立即拉取当前状态",
    "action.retry": "重试",
    "action.close": "关闭",
    "error.request": "请求失败",
    "status.ok": "健康",
    "status.warn": "降级",
    "status.err": "异常",
    "status.unknown": "未知",
    "error.noOverview.before": "没有",
    "error.noOverview.after": "控制台无法渲染。请确认控制面已就绪，且该监听地址可达。",

    "overview.title": "总览",
    "overview.controlPlane": "控制面",
    "overview.dataPlane": "数据面",
    "overview.externalDataPlane": "外部数据面",
    "col.pod": "Pod",
    "col.podIP": "Pod IP",
    "col.namespace": "命名空间",
    "col.state": "状态",
    "col.injected": "已注入",
    "col.inboundL4": "入站 L4",
    "col.mtls": "mTLS",
    "col.certificate": "证书",
    "col.trustRoot": "信任根",
    "col.restarts": "重启次数",
    "col.deployment": "Deployment",
    "col.gateway": "Gateway",
    "col.class": "Class",
    "col.replicas": "副本",
    "col.role": "角色",
    "col.pods": "Pod 数",
    "col.service": "服务",
    "col.registry": "注册中心",
    "col.ports": "端口",
    "col.clusterAddress": "集群地址",
    "col.exposure": "暴露方式",
    "state.ready": "就绪",
    "state.notReady": "未就绪",
    "state.accepting": "正常接收",
    "state.notAccepting": (n, total) => `${total} 个中有 ${n} 个未接收`,
    "state.current": "最新",
    "state.superseded": (n) => `${n} 个已过期`,
    "state.meshDefault": "网格默认",
    "state.unknown": "未知",
    "state.none": "无",
    "state.expired": "已过期",
    "state.left": (d) => `剩余 ${d}`,
    "empty.noDubbodPods": "看不到 dubbod Pod",
    "empty.noDubbodPods.body.before":
      "这个控制台由控制面提供服务，说明它在运行 —— 但在命名空间中读不到带标签",
    "empty.noDubbodPods.body.after": "的 Pod：",
    "empty.noInjected": "没有已注入的工作负载",
    "empty.noInjected.body.before": "给命名空间打上",
    "empty.noInjected.body.after": "标签并重启其 Pod；之后每个 Pod 都会运行入站代理并计入这里。",
    "empty.noGateways": "没有网关 Deployment",
    "empty.noGateways.body.before": "创建一个",
    "empty.noGateways.body.after": "的 Gateway，控制面会在这里为它创建",
    "empty.noGateways.body.tail": "Deployment。",

    "services.title": "服务",
    "services.filter": "按名称、主机名或命名空间过滤…",
    "services.allNamespaces": "全部命名空间",
    "services.allRegistries": "全部注册中心",
    "services.shown": (n) => `显示 ${n} 条`,
    "empty.noServices": "未发现网格服务",
    "empty.noServices.body.before": "给命名空间打上",
    "empty.noServices.body.after": "标签，把它的服务纳入网格。",
    "empty.noServicesMatch": "没有服务匹配当前过滤条件",
    "empty.noServicesMatch.body": (n) => `清除过滤条件可查看全部 ${n} 个服务。`,

    "pipeline.title": "配置下发流水线",
    "pipeline.pushPath": "下发路径",
    "pipeline.meanEndToEnd": (d) => `端到端均值 ${d}`,
    "pipeline.inboundUpdates": "入站更新",
    "pipeline.inboundUpdates.unit": "按来源累计",
    "pipeline.pushTriggers": "下发触发原因",
    "pipeline.pushTriggers.unit": "按原因累计",
    "stage.debounce": "去抖",
    "stage.pushcontext": "PushContext",
    "stage.queue": "代理队列",
    "stage.send": "发送",
    "stage.noSamples": "无样本",
    "stage.idle.noConfigChange": "尚无配置变更",
    "stage.idle.noPushContext": "尚未构建 PushContext",
    "stage.idle.needsProxy": "需要有代理连接",
    "stage.samples": (n, ceiling) => `n=${n} · 全部 ≤ ${ceiling}`,
    "stage.share": (pct) => `占路径 ${pct}%`,

    "logs.title": "日志",
    "logs.role.controlPlane": "控制面",
    "logs.role.externalDataPlane": "外部数据面",
    "logs.search": "在日志中搜索…",
    "logs.level.all": "全部",
    "logs.level.info": "info",
    "logs.level.warn": "warn",
    "logs.level.error": "error",
    "logs.tail": (n) => `末尾 ${n} 行`,
    "logs.lines": (n) => `${n} 行`,
    "logs.noMatch": "没有匹配的行。",
    "logs.noPods": "该 Deployment 下没有匹配的 Pod",
    "logs.refused": "Kubernetes 拒绝了日志流：",
    "logs.download": "下载",
    "logs.reload": "刷新",

    "topology.title": "拓扑",
    "topology.scope.northSouth": "南北向 · 入口流量",
    "topology.scope.eastWest": "东西向 · 服务间调用",
    "topology.scope.controlPlane": "控制面 · xDS 下发",
    "topology.legend.required": "强制 mTLS",
    "topology.legend.plaintext": "接受明文",
    "topology.legend.off": "mTLS 关闭",
    "topology.legend.stream": "有 xDS 流",
    "topology.legend.configOnly": "仅下发配置，无流",
    "topology.routes": (n) => `${n} 条 HTTPRoute · 配置路径，非实测流量`,
    "topology.streams": (n) => `${n} 条活跃 xDS 流`,
    "topology.omitted": (n) => `${n} 个节点不在此视图的任何路径上：`,
    "topology.empty.northSouth": "没有网关入口路由",
    "topology.empty.northSouth.body":
      "没有任何 HTTPRoute 以 Gateway 为父级，因此没有流量经网关进入网格。",
    "topology.empty.eastWest": "没有服务间路由",
    "topology.empty.eastWest.body":
      "没有任何 HTTPRoute 以 Service 为父级，因此网格内的工作负载之间没有路由。",
    "topology.empty.noServices": "没有网格服务",
    "topology.empty.noServices.body.before": "给命名空间打上",
    "topology.empty.noServices.body.after": "标签，它的服务加入网格后会显示在这里。",
    "topology.zoomIn": "放大",
    "topology.zoomOut": "缩小",
    "topology.reset": "重置",
    "topology.reset.title": "重置布局与缩放",
    "topology.aria": "网格服务之间已配置的调用路径",

    "drawer.kind": "类型",
    "drawer.kind.gateway": "托管网关",
    "drawer.kind.service": "网格服务",
    "drawer.kind.controlPlane": "控制面",
    "drawer.controlPlane.role":
      "通过 xDS 为图中每个节点下发配置。它不承载任何应用流量，因此没有入站 mTLS 模式，也没有自己的请求路径。",
    "drawer.reachedBy": "被指向",
    "drawer.reachedBy.value": (n) => `${n} 条路由规则`,
    "drawer.forwardsTo": "转发到",
    "drawer.forwardsTo.none": "无 —— 没有 HTTPRoute 挂在这个服务上",
    "drawer.hostname": "主机名",
    "drawer.serviceAccounts": "ServiceAccount 数",
    "drawer.meshExternal": "网格外部",
    "drawer.copyHostname": "复制主机名",
    "drawer.gatewayClass": "Gateway class",
    "drawer.viewLogs": "查看日志",
    "drawer.provider": "提供方",
    "drawer.cluster": "集群",
    "drawer.informerSync": "Informer 同步",
    "drawer.synced": "已同步",
    "drawer.syncing": "同步中",
    "drawer.injectedPods": "已注入 Pod",
    "drawer.injectedPods.value": (n, total) => `${total} 个运行中的 Pod 里有 ${n} 个`,
    "drawer.injectedPods.gap": " —— 其余没有 sidecar，不在网格内",
    "drawer.inboundProxy": "入站代理",
    "drawer.mtlsPerPort": "各端口 mTLS 模式",
    "drawer.mtlsPerPort.none": "未设置端口级模式，套用网格默认",
    "drawer.soonestExpiry": "最近的证书到期时间",
    "drawer.soonestExpiry.none": "未找到证书",
    "drawer.trustRoot": "信任根",
    "drawer.trustRoot.ok": "所有 Pod 持有的根证书都与控制面当前签发所用的一致",
    "drawer.trustRoot.stale": (n) => `${n} 个 Pod 仍持有已被替换的根证书，需要重新签发`,
    "drawer.problem": "问题",
    "drawer.pods": "Pod 数",
    "drawer.inboundMTLS": "入站 mTLS",
    "drawer.mtls.fromPolicy": "由 PeerAuthentication 设定",
    "drawer.mtls.fallback": "没有 PeerAuthentication 命中该工作负载，sidecar 兜底为此模式",
    "drawer.role": "角色",

    "config.title": "配置",
    "config.grpc": "gRPC (xDS)",
    "config.secureGrpc": "Secure gRPC (mTLS xDS)",
    "config.metrics": "Prometheus 指标",
    "config.ready": "就绪探针",
    "config.copy": "点击复制",
    "config.preferences": "偏好设置",
    "config.theme": "主题",
    "config.theme.auto": "跟随系统",
    "config.theme.light": "浅色",
    "config.theme.dark": "深色",
    "config.language": "语言",

    "mtls.strict.label": "STRICT —— 调用方必须使用 mTLS",
    "mtls.permissive.label": "PERMISSIVE —— 仍然接受明文",
    "mtls.disable.label": "DISABLE —— 入站 mTLS 已关闭",
    "mtls.source.policy": "由 PeerAuthentication 设定",
    "mtls.source.fallback": "sidecar 兜底，没有 PeerAuthentication 命中该工作负载",
  },
};

const langKey = "dubbod-gui-lang";

const detect = () => {
  try {
    const stored = localStorage.getItem(langKey);
    if (stored && STRINGS[stored]) return stored;
  } catch (_) { /* storage unavailable */ }
  return (navigator.language || "").toLowerCase().startsWith("zh") ? "zh" : "en";
};

let current = detect();

export const getLanguage = () => current;

export const setLanguage = (lang) => {
  if (!STRINGS[lang]) return;
  current = lang;
  try { localStorage.setItem(langKey, lang); } catch (_) { /* storage unavailable */ }
  document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
};

/** Falls back to English, then to the key itself, so a gap shows up as the key. */
export const t = (key, ...args) => {
  const value = STRINGS[current]?.[key] ?? STRINGS.en[key];
  if (value == null) return key;
  return typeof value === "function" ? value(...args) : value;
};

document.documentElement.lang = current === "zh" ? "zh-CN" : "en";
