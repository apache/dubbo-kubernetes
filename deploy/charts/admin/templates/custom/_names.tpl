{{/*
Return Dubbo Name to use.
*/}}
{{- define "dubbo.name" -}}
{{- printf "dubbo" -}}
{{- end -}}

{{/*
Return Admin Name to use.
*/}}
{{- define "admin.name" -}}
{{- printf "admin" -}}
{{- end -}}

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
{{- printf "prometheus" -}}
{{- end -}}

{{/*
Return Grafana Name to use.
*/}}
{{- define "grafana.name" -}}
{{- printf "grafana" -}}
{{- end -}}
