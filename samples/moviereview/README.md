## moviereview example

```bash
kubectl create ns moviereview
kubectl apply -f samples/moviereview/deployment.yaml
kubectl -n moviereview get svc frontend
```

访问 `http://<NodeIP>:30080`。
