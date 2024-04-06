{{- define "traefik.annotations" -}}
prometheus.io/scrape: "true"
prometheus.io/path: "/metrics"
prometheus.io/port: "9100"
{{- end -}}

{{- define "traefik.ingress.annotations" -}}
ingress.kubernetes.io/ssl-redirect: "true"
ingress.kubernetes.io/proxy-body-size: "0"
nginx.ingress.kubernetes.io/ssl-redirect: "true"
nginx.ingress.kubernetes.io/proxy-body-size: "0"
{{- end -}}