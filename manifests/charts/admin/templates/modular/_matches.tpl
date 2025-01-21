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
