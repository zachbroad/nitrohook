---
title: Monitoring
description: Prometheus metrics and Grafana dashboards for nitrohook.
---

Both the API server and fan-out worker expose a `/metrics` endpoint with Prometheus-formatted metrics.

| Endpoint | Port |
|----------|------|
| API server | `8080` (same as HTTP) |
| Worker | `8081` (health/metrics server) |

## Metrics reference

### Counters

| Metric | Labels | Description |
|--------|--------|-------------|
| `nitrohook_webhooks_received_total` | `source` | Total webhooks ingested |
| `nitrohook_deliveries_dispatched_total` | `action_type`, `status` | Total dispatch attempts (`success` or `failed`) |

### Histograms

| Metric | Description |
|--------|-------------|
| `nitrohook_webhook_ingest_duration_seconds` | Time to process an incoming webhook |
| `nitrohook_dispatch_duration_seconds` | Time to dispatch a delivery to an action |

### Gauges

| Metric | Description |
|--------|-------------|
| `nitrohook_pending_deliveries` | Pending deliveries found by the catch-up poller |
| `nitrohook_retryable_attempts` | Retryable attempts found by the retry poller |

## Kubernetes (ServiceMonitor)

If you're running [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack), enable the included ServiceMonitors:

```yaml
# values.yaml
metrics:
  serviceMonitor:
    enabled: true
    interval: 15s
    additionalLabels: {}
```

This creates ServiceMonitor resources for both the API and worker services. If your Prometheus operator uses a label selector (e.g. `release: kube-prometheus-stack`), pass it via `additionalLabels`:

```yaml
metrics:
  serviceMonitor:
    enabled: true
    additionalLabels:
      release: kube-prometheus-stack
```

## Docker Compose

The default `docker-compose.yml` includes a Prometheus instance preconfigured to scrape the API and worker. Access it at [http://localhost:9090](http://localhost:9090).

## Useful PromQL queries

**Webhook ingest rate** (per source, 5-minute window):

```promql
rate(nitrohook_webhooks_received_total[5m])
```

**Ingest latency (p95)**:

```promql
histogram_quantile(0.95, rate(nitrohook_webhook_ingest_duration_seconds_bucket[5m]))
```

**Dispatch success vs failure rate**:

```promql
rate(nitrohook_deliveries_dispatched_total{status="success"}[5m])
rate(nitrohook_deliveries_dispatched_total{status="failed"}[5m])
```

**Dispatch latency (p99)**:

```promql
histogram_quantile(0.99, rate(nitrohook_dispatch_duration_seconds_bucket[5m]))
```

**Failure ratio by action type** (spot broken endpoints):

```promql
rate(nitrohook_deliveries_dispatched_total{status="failed"}[5m])
  /
rate(nitrohook_deliveries_dispatched_total[5m])
```

## Alerting examples

These can be configured as Prometheus alerting rules or in AlertManager:

```yaml
# Delivery backlog growing
- alert: NitrohookHighPendingDeliveries
  expr: nitrohook_pending_deliveries > 100
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Delivery backlog is growing"

# High failure rate
- alert: NitrohookHighFailureRate
  expr: |
    rate(nitrohook_deliveries_dispatched_total{status="failed"}[5m])
      /
    rate(nitrohook_deliveries_dispatched_total[5m])
      > 0.2
  for: 10m
  labels:
    severity: critical
  annotations:
    summary: "Dispatch failure rate above 20%"

# Retry queue building up
- alert: NitrohookRetryBacklog
  expr: nitrohook_retryable_attempts > 50
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Retryable attempts are piling up"
```
