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

