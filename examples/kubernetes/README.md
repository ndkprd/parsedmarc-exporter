# Kubernetes example

Requires the [Prometheus Operator](https://prometheus-operator.dev/) (for example via
kube-prometheus-stack) for the `ServiceMonitor` and `PrometheusRule` resources.

1. The manifests use the published image `registry.gitlab.com/endekasoft/parsedmarc-exporter:latest`. To use your own build or a pinned
   version, build and push it to a registry your cluster can pull from:

   ```bash
   docker build -t registry.gitlab.com/endekasoft/parsedmarc-exporter:0.1.1 --build-arg VERSION=0.1.1 ../..
   docker push registry.gitlab.com/endekasoft/parsedmarc-exporter:0.1.1
   ```

2. Create the database Secret (see `secret.example.yaml`), then set `PARSEDMARC_EXPORTER_DB_HOST` (and
   `image:` if you changed it) in `exporter.yaml`.
3. Apply the resources into the namespace that also contains parsedmarc's PostgreSQL:

   ```bash
   kubectl create namespace parsedmarc
   kubectl -n parsedmarc apply -f secret.yaml -f exporter.yaml -f servicemonitor.yaml
   ```

4. Check that Prometheus discovered the target (`up{job="parsedmarc-exporter"}`), or port-forward
   and read the metrics directly:

   ```bash
   kubectl -n parsedmarc port-forward svc/parsedmarc-exporter 9190:9190
   curl http://127.0.0.1:9190/metrics
   ```

The ServiceMonitor's `release` label and `namespaceSelector` must match your Prometheus
`serviceMonitorSelector` / `serviceMonitorNamespaceSelector`.
