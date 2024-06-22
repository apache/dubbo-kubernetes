{{/*
Return Admin Port to use.
*/}}
{{- define "admin.port" -}}
{{- printf "8080" -}}
{{- end -}}

{{/*
Return xds Container Port to use.
*/}}
{{- define "admin.xds.containerPort" -}}
{{- print "5678" -}}
{{- end -}}

{{/*
Return Admin Container Port to use.
*/}}
{{- define "admin.web.containerPort" -}}
{{- printf "8888" -}}
{{- end -}}

{{/*
Return Admin admissionServer Container Port to use.
*/}}
{{- define "admin.admissionServer.containerPort" -}}
{{- printf "5443" -}}
{{- end -}}

{{/*
Return ZooKeeper Client Port to use.
*/}}
{{- define "zoo.client" -}}
{{- printf "2181" -}}
{{- end -}}

{{/*
Return ZooKeeper Follower Port to use.
*/}}
{{- define "zoo.follower" -}}
{{- printf "2888" -}}
{{- end -}}

{{/*
Return ZooKeeper Election Port to use.
*/}}
{{- define "zoo.election" -}}
{{- printf "3888" -}}
{{- end -}}

{{/*
Return Nacos Port to use.
*/}}
{{- define "nacos.port" -}}
{{- printf "8848" -}}
{{- end -}}

{{- define "traefik.metrics.containerPort" -}}
{{- printf "9100" -}}
{{- end -}}

{{- define "traefik.metrics.hostPort" -}}
{{- printf "9101" -}}
{{- end -}}

{{- define "traefik.traefik.containerPort" -}}
{{- printf "9000" -}}
{{- end -}}

{{- define "traefik.web.containerPort" -}}
{{- printf "8000" -}}
{{- end -}}

{{- define "traefik.web.hostPort" -}}
{{- printf "80" -}}
{{- end -}}

{{- define "traefik.websecure.containerPort" -}}
{{- printf "8443" -}}
{{- end -}}

{{- define "traefik.websecure.hostPort" -}}
{{- printf "443" -}}
{{- end -}}

{{- define "prom.port" -}}
{{- printf "9090" -}}
{{- end -}}

{{- define "grafana.port" -}}
{{- printf "3000" -}}
{{- end -}}