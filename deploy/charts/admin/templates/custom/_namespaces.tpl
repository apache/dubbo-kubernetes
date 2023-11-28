{{/*
Return Admin Namespace to use.
*/}}
{{- define "admin.namespace" -}}
{{- if .Values.namespaceOverride }}
{{- .Values.namespaceOverride }}
{{- else -}}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Return ZooKeeper Namespace to use.
*/}}
{{- define "zoo.namespace" -}}
{{- if .Values.zookeeper.namespaceOverride -}}
{{- .Values.zookeeper.namespaceOverride }}
{{- else -}}
{{- .Release.Namespace }}
{{- end -}}
{{- end -}}

{{/*
Return Nacos Namespace to use.
*/}}
{{- define "nacos.namespace" -}}
{{- if .Values.nacos.namespaceOverride -}}
{{- .Values.nacos.namespaceOverride }}
{{- else -}}
{{- .Release.Namespace }}
{{- end -}}
{{- end -}}

{{/*
Return Nacos Namespace to use.
*/}}
{{- define "ingress.namespace" -}}
{{- if .Values.ingress.namespaceOverride -}}
{{- .Values.ingress.namespaceOverride }}
{{- else -}}
{{- .Release.Namespace }}
{{- end -}}
{{- end -}}