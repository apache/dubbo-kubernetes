{{/*
Return Dubbo Control Plane Name to use.
*/}}
{{- define "dubbo.cp.name" -}}
{{- printf "admin-cp" -}}
{{- end -}}

{{/*
Return Admin Name to use.
*/}}
{{- define "admin.name" -}}
{{- printf "admin" -}}
{{- end -}}

{{/*
Return ZooKeeper Name to use.
*/}}
{{- define "zoo.name" -}}
{{- printf "zookeeper" -}}
{{- end -}}

{{/*
Return Nacos Name to use.
*/}}
{{- define "nacos.name" -}}
{{- printf "nacos" -}}
{{- end -}}

{{/*
Return Traefik Name to use.
*/}}
{{- define "traefik.name" -}}
{{- printf "traefik" -}}
{{- end -}}

{{/*
Return Kube-Prometheus-Stack Name to use.
*/}}
{{- define "prom.stack.name" -}}
{{- printf "kube-prometheus-stack" -}}
{{- end -}}

{{/*
Return Kube-Prometheus-Stack Name to use.
*/}}
{{- define "prom.name" -}}
{{- printf "kube-prometheus" -}}
{{- end -}}

{{/*
Return kube-prometheus-grafana to use.
*/}}
{{- define "grafana.stack.name" -}}
{{- printf "kube-prometheus-grafana" -}}
{{- end -}}

{{/*
Return Grafana Name to use.
*/}}
{{- define "grafana.name" -}}
{{- printf "grafana" -}}
{{- end -}}

{{/*
Return Job Name to use.
*/}}
{{- define "job.name" -}}
{{- printf "jobs" -}}
{{- end -}}

{{/*
Return Dubbo Namespace to use.
*/}}
{{- define "admin.namespace" -}}
{{- "dubbo-system" | default }}
{{- end }}

{{/*
Return Jobs Namespace to use.
*/}}
{{- define "job.namespace" -}}
{{- if .Values.jobs.namespaceOverride -}}
{{- .Values.jobs.namespaceOverride }}
{{- else -}}
{{- .Release.Namespace }}
{{- end -}}
{{- end -}}

{{/*
Return ingress Namespace to use.
*/}}
{{- define "ingress.namespace" -}}
{{- if .Values.ingress.namespaceOverride -}}
{{- .Values.ingress.namespaceOverride }}
{{- else -}}
{{- .Release.Namespace }}
{{- end -}}
{{- end -}}

{{/*
Return Admin Labels to use.
*/}}
{{- define "admin.labels" -}}
app: {{ template "admin.name" . }}
app.kubernetes.io/name: {{ template "admin.name" . }}
helm.sh/chart: {{ include "admin.name" . }}-{{ .Values.image.tag }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Return Jobs Labels to use.
*/}}
{{- define "jobs.labels" -}}
app.kubernetes.io/name: {{ template "job.name" . }}
helm.sh/chart: {{ include "job.name" . }}-{{ .Values.jobs.image.tag }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Return Traefik Labels.
*/}}
{{- define "traefik.labels" -}}
app.kubernetes.io/name: {{ template "traefik.name" . }}
{{- end -}}

{{/*
Return Admin matchLabels to use.
*/}}
{{- define "admin.matchLabels" -}}
app.kubernetes.io/name: {{ template "admin.name" . }}
helm.sh/chart: {{ include "admin.name" . }}-{{ .Values.image.tag }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Return Traefik matchLabels to use.
*/}}
{{- define "traefik.matchLabels" -}}
app.kubernetes.io/name: {{ template "traefik.name" . }}
{{- end -}}

{{/*
Return Admin Service Selector to use.
*/}}
{{- define "admin.selector" -}}
{{ include "admin.name" . }}
{{- end -}}

{{/*
Return Admin Port to use.
*/}}
{{- define "admin.port" -}}
{{- printf "8080" -}}
{{- end -}}

{{/*
Return xds Port to use.
*/}}
{{- define "admin.xds.Port" -}}
{{- print "5678" -}}
{{- end -}}

{{/*
Return Admin Port to use.
*/}}
{{- define "admin.web.port" -}}
{{- printf "8888" -}}
{{- end -}}

{{/*
Return Admin admissionServer Container Port to use.
*/}}
{{- define "admin.admissionServer.containerPort" -}}
{{- printf "5443" -}}
{{- end -}}

{{/*
Return ZooKeeper Client Port to use.
*/}}
{{- define "zoo.client" -}}
{{- printf "2181" -}}
{{- end -}}

{{/*
Return ZooKeeper Follower Port to use.
*/}}
{{- define "zoo.follower" -}}
{{- printf "2888" -}}
{{- end -}}

{{/*
Return ZooKeeper Election Port to use.
*/}}
{{- define "zoo.election" -}}
{{- printf "3888" -}}
{{- end -}}

{{/*
Return Nacos Port to use.
*/}}
{{- define "nacos.port" -}}
{{- printf "8848" -}}
{{- end -}}

{{- define "traefik.metrics.containerPort" -}}
{{- printf "9100" -}}
{{- end -}}

{{- define "traefik.metrics.hostPort" -}}
{{- printf "9101" -}}
{{- end -}}

{{- define "traefik.traefik.containerPort" -}}
{{- printf "9000" -}}
{{- end -}}

{{- define "traefik.web.containerPort" -}}
{{- printf "8000" -}}
{{- end -}}

{{- define "traefik.web.hostPort" -}}
{{- printf "80" -}}
{{- end -}}

{{- define "traefik.websecure.containerPort" -}}
{{- printf "8443" -}}
{{- end -}}

{{- define "traefik.websecure.hostPort" -}}
{{- printf "443" -}}
{{- end -}}

{{- define "prom.port" -}}
{{- printf "9090" -}}
{{- end -}}

{{- define "grafana.port" -}}
{{- printf "3000" -}}
{{- end -}}

{{/*
Return jobs-1 annotations to use.
*/}}
{{- define "jobs.1.annotations" -}}
"helm.sh/hook": "pre-install"
"helm.sh/hook-weight": "1"
"helm.sh/hook-delete-policy": "hook-succeeded"
{{- end -}}

{{/*
Return jobs-2 annotations to use.
*/}}
{{- define "jobs.2.annotations" -}}
"helm.sh/hook": "pre-install"
"helm.sh/hook-weight": "2"
"helm.sh/hook-delete-policy": "hook-succeeded"
{{- end -}}

{{/*
Return jobs serviceAccount annotations to use.
*/}}
{{- define "jobs.sa.annotations" -}}
"helm.sh/hook": "pre-install"
"helm.sh/hook-delete-policy": "before-hook-creation"
{{- end -}}

{{- define "traefik.annotations" -}}
prometheus.io/scrape: "true"
prometheus.io/path: "/metrics"
prometheus.io/port: "9100"
{{- end -}}

{{- define "traefik.ingress.annotations" -}}
ingress.kubernetes.io/ssl-redirect: "true"
ingress.kubernetes.io/proxy-body-size: "0"
nginx.ingress.kubernetes.io/ssl-redirect: "true"
nginx.ingress.kubernetes.io/proxy-body-size: "0"
{{- end -}}

