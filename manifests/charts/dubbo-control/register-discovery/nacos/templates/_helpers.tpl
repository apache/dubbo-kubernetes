{{/*
Return the appropriate apiVersion for deployment or statefulset.
*/}}
{{- define "apiVersion" -}}
{{- if and ($.Capabilities.APIVersions.Has "apps/v1") (semverCompare ">= 1.14-0" .Capabilities.KubeVersion.Version) }}
{{- print "apps/v1" }}
{{- else }}
{{- print "extensions/v1beta1" }}
{{- end }}
{{- end }}

{{/*
Return Nacos Name to use.
*/}}
{{- define "nacos.name" -}}
{{- printf "nacos" -}}
{{- end -}}

{{/*
Return Nacos Labels to use.
*/}}
{{- define "nacos.labels" -}}
app: {{ template "nacos.name" . }}
app.kubernetes.io/name: {{ template "nacos.name" . }}
app.kubernetes.io/instance: {{ template "nacos.name" . }}
app.kubernetes.io/component: {{ template "nacos.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Return Nacos matchLabels to use.
*/}}
{{- define "nacos.matchLabels" -}}
app.kubernetes.io/name: {{ template "nacos.name" . }}
app.kubernetes.io/instance: {{ template "nacos.name" . }}
app.kubernetes.io/component: {{ template "nacos.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Return Nacos Service Selector to use.
*/}}
{{- define "nacos.selector" -}}
{{ include "nacos.name" . }}
{{- end -}}

{{/*
Return Nacos Port to use.
*/}}
{{- define "nacos.port" -}}
{{- printf "8848" -}}
{{- end -}}

