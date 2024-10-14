{{/*
Return Admin Service Selector to use.
*/}}
{{- define "admin.selector" -}}
{{ include "admin.name" . }}
{{- end -}}

{{/*
Return ZooKeeper Service Selector to use.
*/}}
{{- define "zoo.selector" -}}
{{ include "zoo.name" . }}
{{- end -}}

{{/*
Return Nacos Service Selector to use.
*/}}
{{- define "nacos.selector" -}}
{{ include "nacos.name" . }}
{{- end -}}
