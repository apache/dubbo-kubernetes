{{- define "admin.pod" -}}
{{- $admin := .Values -}}
serviceAccountName: admin-sa
containers:
- name: admin
  image: "{{ $admin.image }}:{{ $admin.tag }}"
  imagePullPolicy: IfNotPresent
  args:
  - --config-file=/var/lib/admin/dubbo-cp.yaml
  ports:
  - name: http
    containerPort: 8888
  readinessProbe:
    httpGet:
      path: /admin
      port: 8888
    initialDelaySeconds: 1
    periodSeconds: 3
    timeoutSeconds: 5
  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    runAsNonRoot: false
    capabilities:
      drop:
        - ALL
  volumeMounts:
  - name: data
    mountPath: /var/lib/admin
  - name: config
    mountPath: /var/lib/admin/dubbo-cp.yaml
    subPath: dubbo-cp.yaml
{{- with $admin.volumeMounts }}
{{- toYaml . | nindent 10 }}
{{- end }}
resources:
{{ toYaml $admin.resources | trim | indent 2 }}
volumes:
- name: data
  emptyDir: {}
- name: config
  configMap:
    name: admin-config
    defaultMode: 0755
{{- with $admin.volumes }}
{{- toYaml . | nindent 6 }}
{{- end }}
{{- end -}}