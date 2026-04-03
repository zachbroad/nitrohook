---
title: Kubernetes (Helm)
description: Deploy nitrohook to Kubernetes with the included Helm chart.
---

nitrohook includes a Helm chart in `charts/` that deploys the full stack: API server, fan-out workers, Postgres, and Redis.

## Quick install

```bash
helm install nitrohook ./charts \
  --set api.ingress.host=nitrohook.example.com
```

## What gets deployed

| Component | Kind | Replicas |
|-----------|------|----------|
| API server | Deployment | 1 (configurable) |
| Fan-out worker | Deployment | 3 (configurable) |
| PostgreSQL | StatefulSet | 1 |
| Redis | StatefulSet | 1 |
| Ingress | Ingress | 1 (optional) |

The API deployment runs database migrations as an init container before starting.

## Values reference

### Images

```yaml
image:
  api:
    repository: ghcr.io/zachbroad/nitrohook-api
    tag: main
  worker:
    repository: ghcr.io/zachbroad/nitrohook-worker
    tag: main
```

### API

```yaml
api:
  replicaCount: 1
  migration:
    enabled: true
  ingress:
    enabled: true
    className: nginx
    host: nitrohook.test
    tls: []
    #  - secretName: nitrohook-tls
    #    hosts:
    #      - nitrohook.test
```

### Worker

```yaml
worker:
  replicaCount: 3
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi
```

### Application config

These go into a ConfigMap and map to the same environment variables as Docker Compose:

```yaml
appConfig:
  port: "8080"
  workerConcurrency: "4"
  maxRetries: "5"
  retryBaseDelay: "5s"
  deliveryTimeout: "10s"
  pollInterval: "30s"
```

### Postgres

```yaml
postgres:
  user: relay
  password: relay
  database: nitrohook
  storage: 5Gi
  storageClass: ""
```

### Redis

```yaml
redis:
  storage: 1Gi
  storageClass: ""
```

### Secrets

By default the chart creates a Secret with the database and Redis connection strings. To use an existing secret:

```yaml
secret:
  create: false
  existingSecret: my-nitrohook-secret
```

The secret must contain `DATABASE_URL` and `REDIS_URL` keys.

## Accessing the UI

With ingress enabled, DNS resolution depends on your setup:

- **minikube** — enable the [ingress-dns](https://minikube.sigs.k8s.io/docs/handbook/addons/ingress-dns/) addon to automatically resolve `*.test` domains to your minikube IP:
  ```bash
  minikube addons enable ingress
  minikube addons enable ingress-dns
  ```
- **Cloud / bare metal** — point your DNS record at the ingress controller's external IP.

Without ingress:

```bash
kubectl port-forward svc/nitrohook-api 8080:80
open http://localhost:8080
```
