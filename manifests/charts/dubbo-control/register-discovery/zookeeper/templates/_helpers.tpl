{{/*
Return the ZooKeeper client-server authentication credentials secret.
*/}}
{{- define "zookeeper.client.secretName" -}}
{{- $zoo := .Values.registerCentre.zookeeper -}}
{{- if $zoo.auth.client.existingSecret -}}
    {{- printf "%s" (tpl $zoo.auth.client.existingSecret $) -}}
{{- else -}}
    {{- printf "zookeeper-client-auth" -}}
{{- end -}}
{{- end -}}

{{/*
Return the ZooKeeper server-server authentication credentials secret.
*/}}
{{- define "zookeeper.quorum.secretName" -}}
{{- $zoo := .Values.registerCentre.zookeeper -}}
{{- if $zoo.auth.quorum.existingSecret -}}
    {{- printf "%s" (tpl $zoo.auth.quorum.existingSecret $) -}}
{{- else -}}
    {{- printf "zookeeper-quorum-auth" -}}
{{- end -}}
{{- end -}}