# Deployment Architecture

**In one sentence**: browsers enter the cluster over HTTPS through the edge Nginx, Traefik routes `/` to the static frontend and `/api` to api-gateway, and everything else (user-service, PostgreSQL, Redis) stays reachable only inside the cluster.

## Kubernetes component cheat sheet

| Concept | Role | Instance in this project |
|---------|------|--------------------------|
| Namespace | Isolation boundary | `museflow` (apps), `traefik` (Ingress Controller) |
| Deployment | Stateless workload, keeps replicas, rolling updates | `web`, `api-gateway`, `user-service`, `user-service-worker` |
| StatefulSet | Stateful workload, stable Pod names and per-Pod PVCs | `postgres`, `redis`, `ollama` |
| Pod | Smallest runnable unit, owned by a Deployment/StatefulSet | `web-xxxx`, `api-gateway-xxxx` |
| Service | Stable in-cluster virtual IP + DNS name, load balances a set of Pods | `web:80`, `api-gateway:5001`, `user-service:5002` |
| Service: ClusterIP | In-cluster only (default) | `web`, `api-gateway`, `user-service` |
| Service: NodePort | Opens one port in `30000-32767` on every node | Traefik: `31380` / `31443` |
| Service: LoadBalancer | Cloud provider or `klipper-lb` allocates an external IP | `postgres`, `redis` (local debugging) |
| Ingress | Declares "host + path → Service" rules (**rules only; it forwards nothing by itself**) | `museflow` (`deploy/k8s/edge/ingress`) |
| IngressClass | Which controller executes the rules | `traefik` (shipped and default in K3s) |
| Ingress Controller | The actual reverse proxy implementing the rules | Traefik in the `traefik` namespace |
| ConfigMap | Non-sensitive config, injected as env vars or files | `api-gateway-config`, `user-service-config`, `web-config` |
| Secret | Sensitive config | `*-secret`, values come from the uncommitted `overlays/secrets.yaml` |
| PVC | Persistent volume claim | `data-postgres-0`, `data-redis-0`, `data-ollama-0` |
| Helm Release | Versioned record of one chart deployment; upgrade/rollback | `postgres`, `redis`, `api-gateway`, `web`, `ingress`… |

**The three most confusing pairs**

- **Service ≠ Ingress**: a Service is a stable layer-4 in-cluster entry (IP + port + load balancing); an Ingress is a layer-7 routing rule (host + path) that must be implemented by an Ingress Controller. An Ingress `backend` points at a Service.
- **NodePort ≠ publicly reachable**: NodePort only opens a port on the node — no host routing, no TLS. Only Traefik's NodePort is handed to the edge Nginx; application services are ClusterIP.
- **IngressClass ≠ Ingress Controller instance**: `ingressClassName: traefik` only says "give these rules to the traefik controller"; the actual work is done by the Pod in the `traefik` namespace.

## Layered topology

```
Browser
  │ HTTPS (certificate on the edge Nginx)
  ▼
Edge Nginx (outside the cluster, the only TLS termination point)
  │ HTTP, preserves the Host header
  ▼
Traefik NodePort 192.168.142.121:31380 (single in-cluster routing entry)
  │ Ingress "museflow" routes by Host + Path
  ├── /      ──▶ Service web:80            (SPA static assets)
  └── /api   ──▶ Service api-gateway:5001  (/api/v1/** passed through as-is)
                     │ gRPC
                     ▼
                Service user-service:5002
                     │
              PostgreSQL / Redis (in-cluster only)
```

Responsibility boundaries:

| Component | Responsibility | Explicitly not |
|-----------|----------------|----------------|
| Edge Nginx | Terminate HTTPS, load certificates, forward everything to Traefik's NodePort | Path-level routing, static hosting |
| Traefik | Single in-cluster entry, routes Host + Path to Services | Certificate termination (edge does it) |
| `web` (frontend) | Serve SPA static assets and SPA fallback | Any `proxy_pass` |
| `api-gateway` | All `/api/v1/**` endpoints, CORS, cookies, JWT verification | Static asset serving |

## Request path, hop by hop

For `https://<domain>/api/v1/auth/login`:

1. **Browser → edge Nginx:443** — DNS points at the edge server; TLS terminates here.
2. **Edge Nginx → `192.168.142.121:31380`** — plain reverse proxy that keeps the `Host` header, which Traefik needs for rule matching.
3. **Traefik → Ingress rule** — matches `Host: <domain>` + `Path: /api` and selects the `api-gateway` route.
4. **Traefik → Service `api-gateway:5001`** — the ClusterIP Service load balances to one api-gateway Pod.
5. **api-gateway → `user-service:5002`** — gRPC over the Service name, resolved by in-cluster DNS.
6. **user-service → `postgres:15432` / `redis:6379`** — again by Service name; addresses come from `DB_*` / `REDIS_ADDR` in `overlays/secrets.yaml`.

Static assets take the short path: `/` matches `web:80` and the container's Nginx serves the built frontend directly.

## Routing rules

| Path | Backend | Notes |
|------|---------|-------|
| `/` | `web:80` | Static assets + SPA fallback |
| `/api` | `api-gateway:5001` | `/api/v1/**` passed through as-is |
| `/health` | `api-gateway:5001` | Gateway health check; removable from `paths` |

Traefik matches the longest prefix first, so `/api` is never shadowed by `/`. Rules live in the Helm chart `deploy/k8s/edge/ingress`; the path list is `ingress.paths` in `edge/ingress/values.yaml`.

## Exposure of each component

| Component | In-cluster address | Outside |
|-----------|--------------------|---------|
| `web` | `http://web:80` | Edge Nginx + Ingress (`/`) |
| `api-gateway` | `http://api-gateway:5001` | Edge Nginx + Ingress (`/api`, `/health`) |
| `user-service` | `user-service:5002` (gRPC) | not exposed |
| `postgres` | `postgres:15432` | not exposed by default; `postgres.service` in `base/overlays/values.yaml` can temporarily become NodePort / LoadBalancer for local debugging |
| `redis` | `redis:6379` | same as above |
| Traefik | — | `192.168.142.121:31380` (HTTP) / `31443` (HTTPS) |
| Ollama / SearXNG | `ollama:*`, `searxng:*` | not exposed |

> Production should keep `user-service`, PostgreSQL and Redis in-cluster only; open database ports temporarily for local debugging at most.

## Domain and certificates

The domain is environment data; its single source of truth is `deploy/k8s/applications/overlays/secrets.yaml` (not committed).
Example domains in this document always use `museflow.com`:

```yaml
domain:
  host: museflow.com
  scheme: https
```

Every other domain-related value is derived by chart templates; never repeat it in `values.yaml`:

| Derived value | Rule | Implementation |
|---------------|------|----------------|
| Ingress `rules[].host` | `domain.host` | `deploy/k8s/edge/ingress/templates/_helpers.tpl` |
| `GATEWAY_ALLOW_ORIGINS` | `<domain.scheme>://<domain.host>` plus `apiGateway.extraAllowOrigins` | `deploy/k8s/applications/services/api-gateway/templates/_helpers.tpl` |
| `USER_TURNSTILE_ALLOWED_HOSTNAMES` | `domain.host` plus `userService.extraTurnstileHostnames` | `deploy/k8s/applications/services/user-service/templates/_helpers.tpl` |

When `domain.host` is missing, `helm template` / `helm lint` fails loudly instead of rendering an empty or stale domain.

TLS terminates outside the cluster; the config is `deploy/k8s/edge/nginx/museflow.com.conf` (example domain `museflow.com`, replace it with `domain.host`). Certificate issuance and server layout are documented in `deploy/k8s/README.md` ("访问入口与域名").

Changing the domain:

1. Update `domain.host` in `deploy/k8s/applications/overlays/secrets.yaml`.
2. Update the edge Nginx `server_name` and certificate directory; keep proxying to `192.168.142.121:31380`.
3. Redeploy the `app` scope (or just the `ingress`, `api-gateway`, `user-service` releases).
4. Verify: `kubectl get ingress -n museflow`, then hit `/` and `/api/v1/**` through the edge entry.

## Deploying

Charts come in two groups: `base/` (PostgreSQL, Redis, Ollama, SearXNG) and `applications/` (business apps + `edge/ingress`).
Real secrets and the domain live in two uncommitted `secrets.yaml` files; the deploy scripts verify they exist first.

```bat
:: Windows: pick a scope, default is all
deploy\k8s\scripts\deploy.bat base --kubeconfig=E:\kubeconfig\xxx.yaml
deploy\k8s\scripts\deploy.bat app  --kubeconfig=E:\kubeconfig\xxx.yaml
deploy\k8s\scripts\deploy.bat all  --kubeconfig=E:\kubeconfig\xxx.yaml
```

```bash
# Linux/macOS
./deploy/k8s/scripts/deploy.sh app --kubeconfig=/path/to/kubeconfig
```

The edge Nginx is not managed by Helm; copy it to the server manually:

| Repository path | Server path |
|-----------------|-------------|
| `deploy/k8s/edge/nginx/museflow.com.conf` | `/etc/nginx/conf.d/museflow.com.conf` |
| — | `/etc/nginx/certs/<domain>/fullchain.pem`, `privkey.pem` |

Manual Helm commands, validation, upgrade/rollback and uninstall notes are in `deploy/k8s/README.md`.

When the node cannot pull images, use `deploy/k8s/scripts/push-image.sh` (or `.bat` / `.ps1`) to
`docker save` locally built images, upload them to `/docker/images` on the node and import them into
K3s containerd with `ctr -n k8s.io images import`; see "离线镜像导入" in `deploy/k8s/README.md`.

## Troubleshooting

| Symptom | Look at |
|---------|---------|
| 502/503 on the domain | `kubectl get pods -n museflow`; then check the Ingress backend has endpoints: `kubectl get endpoints -n museflow` |
| 404 with a JSON body | Ingress matched but the path is wrong: see `ingress.paths`; the gateway only serves `/api/v1/**` |
| 404 rendered by Traefik | Ingress did not match: `kubectl get ingress -n museflow` — is `host` equal to the browser's domain? |
| Cookie not sent by the browser | Is the edge really HTTPS? With `GATEWAY_COOKIE_SECURE=true` browsers drop cookies over HTTP |
| SSE stream not real time | Both `proxy_buffering off` on the edge Nginx and `X-Accel-Buffering: no` from the gateway must be present |
| Registration fails with `duplicate key ... "user_pkey"` yet succeeds on retry | The sequence is out of sync with the seeded `max(id)`: seed rows insert explicit ids and the sequence never advances. Realign with `SELECT setval(pg_get_serial_sequence('user_svc."user"', 'id'), (SELECT max(id) FROM user_svc."user"), true)`; both `database/user_svc.sql` and `platform_svc.sql` now realign by `max(id)` automatically |
| `invalid-input-secret` in the logs | Turnstile is holding the frontend Site Key or a placeholder: fix `USER_TURNSTILE_SECRET` in `applications/overlays/secrets.yaml`; leaving it empty skips verification (local only) |

Handy commands:

```bash
kubectl get pods,svc,deploy,statefulset,pvc,ingress -n museflow --kubeconfig=<kubeconfig>
kubectl get events -n museflow --sort-by=.lastTimestamp --kubeconfig=<kubeconfig>
kubectl logs -n museflow deployment/api-gateway --tail=100 --kubeconfig=<kubeconfig>
helm list -n museflow --kubeconfig=<kubeconfig>
helm history api-gateway -n museflow --kubeconfig=<kubeconfig>
```

## Directory layout

```
deploy/k8s/
├── applications/          # business charts: services/, web/, overlays/ (secrets + domain)
├── base/                  # infrastructure charts: postgres, redis, ollama, searxng
├── edge/                  # public entry layer
│   ├── ingress/           # in-cluster routing: Traefik Ingress (Helm chart)
│   └── nginx/             # external edge Nginx: TLS termination + Traefik proxy
├── scripts/               # deploy.sh / deploy.bat / deploy.ps1
└── README.md              # deployment command reference
```
