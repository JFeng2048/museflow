# Architecture Design Docs

This folder holds **architecture-level** documents: what the system is made of, where each part runs, and how the parts talk to each other.

Division of labour with the other doc folders:

| Folder | Question it answers | Examples |
|--------|---------------------|----------|
| `docs/cn/architecture/` (and this folder) | What the system consists of, how it is deployed, how parts communicate | Deployment architecture, service architecture |
| `docs/en/develop/` | How a specific feature is implemented | Dual-token auth, 2FA, Turnstile, RBAC |
| `docs/en/api/` | Public interface contracts | api-gateway, config-service, user-service, crawl4ai-service |

## Index

| Document | Content |
|----------|---------|
| [Deployment Architecture](deployment.md) | Kubernetes component cheat sheet (Namespace / Deployment / Service / Ingress / …), layered topology, per-hop request path, routing rules, deployment and troubleshooting |
| [Service Architecture](service-architecture.md) | Service and module breakdown, layering and dependency rules, persistence, key designs, links to implementation docs |

## Conventions for new documents

- **Naming**: kebab-case English file names, topic only (`deployment.md`, `messaging-design.md`); no numeric prefixes, so documents can be inserted and removed freely.
- **Opening**: every document starts with a one-sentence summary — conclusion first, details after.
- **Real values**: paths, commands and config keys must be real repository-relative values, never placeholders.
- **Cross links**: link freely inside this folder; use `../develop/xxx.md` for implementation details and `../api/xxx.md` for interfaces.
- **Register**: add a row to the index table above when adding a document.
- **Chinese counterpart**: keep `docs/cn/architecture/` in sync (Chinese file names).
