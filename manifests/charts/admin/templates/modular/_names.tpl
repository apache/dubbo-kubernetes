{{/*
Return Dubbo Control Plane Name to use.
*/}}
{{- define "dubbo.cp.name" -}}
{{- printf "admin-cp" -}}
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
Return Kube-Prometheus-Stack Name to use.
*/}}
{{- define "prom.stack.name" -}}
{{- printf "kube-prometheus-stack" -}}
{{- end -}}

{{/*
Return Kube-Prometheus-Stack Name to use.
*/}}
{{- define "prom.name" -}}
{{- printf "kube-prometheus" -}}
{{- end -}}


{{/*
Return kube-prometheus-grafana to use.
*/}}
{{- define "grafana.stack.name" -}}
{{- printf "kube-prometheus-grafana" -}}
{{- end -}}

{{/*
Return Grafana Name to use.
*/}}
{{- define "grafana.name" -}}
{{- printf "grafana" -}}
{{- end -}}



{{/*
Return Job Name to use.
*/}}
{{- define "job.name" -}}
{{- printf "jobs" -}}
{{- end -}}
