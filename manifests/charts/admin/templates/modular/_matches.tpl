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
Return ZooKeeper matchLabels to use.
*/}}
{{- define "zoo.matchLabels" -}}
app.kubernetes.io/name: {{ template "zoo.name" . }}
helm.sh/chart: {{ include "zoo.name" . }}-{{ .Values.zookeeper.image.tag }}
app.kubernetes.io/instance: {{ template "zoo.name" . }}
app.kubernetes.io/component: {{ template "zoo.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Return Nacos matchLabels to use.
*/}}
{{- define "nacos.matchLabels" -}}
app.kubernetes.io/name: {{ template "nacos.name" . }}
helm.sh/chart: {{ include "nacos.name" . }}-{{ .Values.nacos.image.tag }}
app.kubernetes.io/instance: {{ template "nacos.name" . }}
app.kubernetes.io/component: {{ template "nacos.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Return Traefik matchLabels to use.
*/}}
{{- define "traefik.matchLabels" -}}
app.kubernetes.io/name: {{ template "traefik.name" . }}
{{- end -}}
