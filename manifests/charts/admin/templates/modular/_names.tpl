{{/*
Return Job Name to use.
*/}}
{{- define "job.name" -}}
{{- printf "jobs" -}}
{{- end -}}

{{/*
Return Admin Control Plane Name to use.
*/}}
{{- define "cp.name" -}}
{{- printf "admin-cp" -}}
{{- end -}}

{{/*
Return Admin Name to use.
*/}}
{{- define "admin.name" -}}
{{- printf "admin" -}}
{{- end -}}

{{/*
Return Dubbo Namespace to use.
*/}}
{{- define "admin.namespace" -}}
{{- "dubbo-system" | default }}
{{- end }}


{{/*
Return ZooKeeper Name to use.
*/}}
{{- define "zoo.name" -}}
{{- printf "zookeeper" -}}
{{- end -}}

{{/*
Return Nacos Name to use.
*/}}
{{- define "nacos.name" -}}
{{- printf "nacos" -}}
{{- end -}}

{{/*
Return Traefik Name to use.
*/}}
{{- define "traefik.name" -}}
{{- printf "traefik" -}}
{{- end -}}

{{/*
Return Prometheus Name to use.
*/}}
{{- define "prom.name" -}}
{{- printf "prometheus-cp-server" -}}
{{- end -}}

{{/*
Return Grafana Name to use.
*/}}
{{- define "grafana.name" -}}
{{- printf "grafana" -}}
{{- end -}}
