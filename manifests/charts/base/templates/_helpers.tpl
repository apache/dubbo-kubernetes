{{/*
Return Dubbo Namespace to use.
*/}}
{{- define "dubbo.namespace" -}}
{{- "dubbo-system" | default }}
{{- end }}
