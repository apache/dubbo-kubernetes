{{/*
Return Admin Labels to use.
*/}}
{{- define "admin.labels" -}}
helm.sh/chart: {{ .Chart.Version }}
app.kubernetes.io/name: {{ template "admin.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: {{ .Release.Name }}
{{- end -}}

{{/*
Return ZooKeeper Labels to use.
*/}}
{{- define "zoo.labels" -}}
app.kubernetes.io/name: {{ template "zoo.name" . }}
app.kubernetes.io/instance: {{ template "zoo.name" . }}
app.kubernetes.io/component: {{ template "zoo.name" . }}
{{- end -}}

{{/*
Return Nacos Labels to use.
*/}}
{{- define "nacos.labels" -}}
app.kubernetes.io/name: {{ template "nacos.name" . }}
{{- end -}}

{{/*
Return Traefik Labels.
*/}}
{{- define "traefik.labels" -}}
app.kubernetes.io/name: {{ template "traefik.name" . }}
{{- end -}}

