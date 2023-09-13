apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
  labels:
    app: {{.Name}}
    app-type: dubbo{{range .Labels}}
    {{.Key}}: {{.Value}}{{end}}
spec:
  replicas: {{.Replicas}}
  revisionHistoryLimit: {{.Revisions}}
  selector:
    matchLabels:
      app: {{.Name}}
      app-type: dubbo{{range .Labels}}
      {{.Key}}: {{.Value}}{{end}}
  template:
    metadata:
      labels:
        app: {{.Name}}
        app-type: dubbo{{range .Labels}}
        {{.Key}}: {{.Value}}{{end}}{{if .UseProm}}
      #helm-charts 配置  https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus
      annotations:
        prometheus.io/scrape: "{{.UsePromScrape}}"
        prometheus.io/path: {{.PromPath}}
        prometheus.io/port: "{{.PromPort}}" {{end}}
    spec:{{if .ServiceAccount}}
      serviceAccountName: {{.ServiceAccount}}{{end}}{{if .UseSkywalking}}
      volumes:
        - name: skywalking-agent
          emptyDir: { }
      initContainers:
        - name: agent-container
          image: apache/skywalking-java-agent:8.13.0-java17
          volumeMounts:
            - name: skywalking-agent
              mountPath: /agent
          command: [ "/bin/sh" ]
          args: [ "-c", "cp -R /skywalking/agent /agent/" ]{{end}}
      containers:
      - name: {{.Name}}
        image: {{.Image}}
        {{if .ImagePullPolicy}}imagePullPolicy: {{.ImagePullPolicy}}
        {{end}}ports:
        - containerPort: {{.Port}}
          name: dubbo
          protocol: TCP{{if .UseProm}}
        - containerPort: {{.PromPort}}
          name: metrics
          protocol: TCP{{end}}{{if .UseSkywalking}}
        volumeMounts:
          - name: skywalking-agent
            mountPath: /skywalking{{end}}
        env:
          - name: DUBBO_CTL_VERSION
            value: 0.0.1{{if .UseSkywalking}}
          - name: SW_AGENT_NAME
            value: {{.Name}}
          - name: JAVA_TOOL_OPTIONS
            value: "-javaagent:/skywalking/agent/skywalking-agent.jar"
          - name: SW_AGENT_COLLECTOR_BACKEND_SERVICES
            value: "skywalking-oap-server.{{.Namespace}}.svc:11800"{{end}}{{range .Envs}}
          - name: {{.Name}}
            value: {{.Value}} {{end}}
        readinessProbe:
          tcpSocket:
            port: {{.Port}}
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          tcpSocket:
            port: {{.Port}}
          initialDelaySeconds: 15
          periodSeconds: 20
        resources:
          requests:
            cpu: {{.RequestCpu}}m
            memory: {{.RequestMem}}Mi
          limits:
            cpu: {{.LimitCpu}}m
            memory: {{.LimitMem}}Mi
      {{if .Secret}}imagePullSecrets:
      - name: {{.Secret}}{{end}}
---

apiVersion: v1
kind: Service
metadata:
  name: {{.Name}}-svc
  namespace: {{.Namespace}}
spec:
  ports:
  {{if .UseNodePort}}- nodePort: {{.NodePort}}
    port: {{.Port}}
    protocol: TCP
    targetPort: {{.TargetPort}}
  type: NodePort{{else}}- port: {{.Port}}
    targetPort: {{.TargetPort}}{{end}}{{if .UseProm}}
  - port: {{.PromPort}}
    targetPort: {{.PromPort}}{{end}}
  selector:
    app: {{.Name}}

---

apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{.Name}}-hpa-c
  namespace: {{.Namespace}}
  labels:
    app: {{.Name}}-hpa-c
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{.Name}}
  minReplicas: {{.MinReplicas}}
  maxReplicas: {{.MaxReplicas}}
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 80

---

apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{.Name}}-hpa-m
  namespace: {{.Namespace}}
  labels:
    app: {{.Name}}-hpa-m
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{.Name}}
  minReplicas: {{.MinReplicas}}
  maxReplicas: {{.MaxReplicas}}
  metrics:
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
