{{/*
Return the ZooKeeper client-server authentication credentials secret.
*/}}
{{- define "zoo.client.secretName" -}}
{{- $zoo := .Values.zookeeper -}}
{{- if $zoo.auth.client.existingSecret -}}
    {{- printf "%s" (tpl $zoo.auth.client.existingSecret $) -}}
{{- else -}}
    {{- printf "%s-client-auth" (include "zoo.name" .) -}}
{{- end -}}
{{- end -}}

{{/*
Return the ZooKeeper server-server authentication credentials secret.
*/}}
{{- define "zoo.quorum.secretName" -}}
{{- $zoo := .Values.zookeeper -}}
{{- if $zoo.auth.quorum.existingSecret -}}
    {{- printf "%s" (tpl $zoo.auth.quorum.existingSecret $) -}}
{{- else -}}
    {{- printf "%s-quorum-auth" (include "zoo.name" .) -}}
{{- end -}}
{{- end -}}

{{/*
Return the Dubbo system namespace to use.
*/}}
{{- define "system.namespaces" -}}
{{- printf "dubbo-system" -}}
{{- end -}}