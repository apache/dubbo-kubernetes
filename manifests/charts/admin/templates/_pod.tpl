{{- define "admin.pod" -}}
{{- $admin := .Values -}}
{{- $zookeeper := .Values.zookeeper }}
{{- $nacos := .Values.nacos }}
serviceAccountName: admin-sa
containers:
- name: admin
  image: "{{ $admin.image }}:{{ $admin.tag }}"
  imagePullPolicy: IfNotPresent
  env:
  - name: DUBBO_DEPLOY_MODE
    value: {{ .Values.deployMode | quote }}
  - name: DUBBO_MODE
    value: {{ .Values.mode | quote }}
  {{- if $nacos.enabled }}
  - name: DUBBO_STORE_TRADITIONAL_REGISTRY
    value: {{ .Values.nacosAddress | quote }}
  - name: DUBBO_STORE_TRADITIONAL_CONFIG_CENTER
    value: {{ .Values.nacosAddress | quote }}
  - name: DUBBO_STORE_TRADITIONAL_METADATA_REPORT
    value: {{ .Values.nacosAddress | quote }}
  {{- end }}
  {{- if $zookeeper.enabled }}
  - name: DUBBO_STORE_TRADITIONAL_REGISTRY
    value: {{ .Values.zookeeperAddress | quote }}
  - name: DUBBO_STORE_TRADITIONAL_CONFIG_CENTER
    value: {{ .Values.zookeeperAddress | quote }}
  - name: DUBBO_STORE_TRADITIONAL_METADATA_REPORT
    value: {{ .Values.zookeeperAddress | quote }}
  {{- else if not ($nacos.enabled) }}
  - name: DUBBO_STORE_TRADITIONAL_REGISTRY
    value: {{ .Values.nacosAddress | quote }}
  - name: DUBBO_STORE_TRADITIONAL_CONFIG_CENTER
    value: {{ .Values.nacosAddress | quote }}
  - name: DUBBO_STORE_TRADITIONAL_METADATA_REPORT
    value: {{ .Values.nacosAddress | quote }}
  {{- end }}
  - name: ADMIN_METRICDASHBOARDS_APPLICATION_BASEURL
    value: {{ .Values.grafanaAddress }}/d/a0b114ca-edf7-4dfe-ac2c-34a4fc545fed/application
  - name: ADMIN_METRICDASHBOARDS_INSTANCE_BASEURL
    value: {{ .Values.grafanaAddress }}/d/dcf5defe-d198-4704-9edf-6520838880e9/instance
  - name: ADMIN_METRICDASHBOARDS_SERVICE_BASEURL
    value: {{ .Values.grafanaAddress }}/d/ec689613-b4a1-45b1-b8bd-9d557059f970/service/
  - name: ADMIN_TRACEDASHBOARDS_APPLICATION_BASEURL
    value: {{ .Values.grafanaAddress }}/d/e968a89b-f03d-42e3-8ad3-930ae815cb0f/application
  - name: ADMIN_TRACEDASHBOARDS_INSTANCE_BASEURL
    value: {{ .Values.grafanaAddress }}/d/f5f48f75-13ec-489b-88ae-635ae38d8618/instance
  - name: ADMIN_TRACEDASHBOARDS_SERVICE_BASEURL
    value: {{ .Values.grafanaAddress }}/d/b2e178fb-ada3-4d5e-9f54-de99e7f07662/service
  - name: ADMIN_PROMETHEUS
    value: {{ .Values.prometheusAddress | quote }}
  - name: ADMIN_GRAFANA
    value: {{ .Values.grafanaAddress | quote }}
  - name: ADMIN_AUTH_USER
    value: {{ .Values.user | quote }}
  - name: ADMIN_AUTH_PASSWORD
    value: {{ .Values.password | quote }}
  - name: ADMIN_AUTH_EXPIRATIONTIME
    value: {{ .Values.expirationTime | quote }}
  ports:
  - containerPort: 8888
  readinessProbe:
    httpGet:
      path: /admin
      port: 8888
      scheme: HTTP
    initialDelaySeconds: 5
    periodSeconds: 30
  livenessProbe:
    httpGet:
      path: /admin
      port: 8888
      scheme: HTTP
    initialDelaySeconds: 5
    periodSeconds: 30
  startupProbe:
    httpGet:
      path: /admin
      port: 8888
      scheme: HTTP
    failureThreshold: 6
    initialDelaySeconds: 30
    periodSeconds: 10
  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: false
    runAsNonRoot: false
    capabilities:
      drop:
        - ALL
  volumeMounts:
  - name: data
    mountPath: /var/lib/admin
  - name: data
    mountPath: /log
{{- with $admin.volumeMounts }}
{{- toYaml . | nindent 10 }}
{{- end }}
resources:
{{ toYaml $admin.resources | trim | indent 2 }}
volumes:
- name: data
  emptyDir: {}
{{- with $admin.volumes }}
{{- toYaml . | nindent 6 }}
{{- end }}
{{- end -}}