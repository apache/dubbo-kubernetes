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

