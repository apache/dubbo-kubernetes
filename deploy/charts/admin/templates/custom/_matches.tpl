{{/*
Return Admin matchLabels to use.
*/}}
{{- define "admin.matchLabels" -}}
app.kubernetes.io/name: {{ template "admin.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: {{ .Release.Name }}
{{- end -}}

{{/*
Return ZooKeeper matchLabels to use.
*/}}
{{- define "zoo.matchLabels" -}}
app.kubernetes.io/name: {{ template "zoo.name" . }}
app.kubernetes.io/instance: {{ template "zoo.name" . }}
app.kubernetes.io/component: {{ template "zoo.name" . }}
{{- end -}}

{{/*
Return Nacos matchLabels to use.
*/}}
{{- define "nacos.matchLabels" -}}
app.kubernetes.io/name: {{ template "nacos.name" . }}
{{- end -}}

{{/*
Return Traefik matchLabels to use.
*/}}
{{- define "traefik.matchLabels" -}}
prometheus.io/scrape: "true"
prometheus.io/path: "/metrics"
prometheus.io/port: "9100"
{{- end -}}
