{{/*
Return jobs env annotations to use.
*/}}
{{- define "jobs.env.annotations" -}}
"helm.sh/hook": "pre-install"
"helm.sh/hook-weight": "0"
"helm.sh/hook-delete-policy": "before-hook-creation,hook-succeeded"
{{- end -}}


{{/*
Return jobs-1 annotations to use.
*/}}
{{- define "jobs.1.annotations" -}}
"helm.sh/hook": "pre-install"
"helm.sh/hook-weight": "1"
"helm.sh/hook-delete-policy": "hook-succeeded"
{{- end -}}

{{/*
Return jobs-2 annotations to use.
*/}}
{{- define "jobs.2.annotations" -}}
"helm.sh/hook": "pre-install"
"helm.sh/hook-weight": "2"
"helm.sh/hook-delete-policy": "hook-succeeded"
{{- end -}}

{{/*
Return jobs serviceAccount annotations to use.
*/}}
{{- define "jobs.sa.annotations" -}}
"helm.sh/hook": "pre-install"
"helm.sh/hook-delete-policy": "before-hook-creation"
{{- end -}}
