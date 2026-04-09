# TODO

## Out of Scope (per plan / user)

- **Dagor** — not in callgraph framework yet
- **Redis** — no stateful backends

---

## In Social/Hotel, Not Yet in Callgraph Framework

### Deployment modes

- Dagor mode (dagor.env, dagor interceptors)

### Observability

- OpenTelemetry (otelgrpc, otel_tool)

### Infrastructure

- Redis (state storage for graph, posts, home, user)

### Application logic

- Init/population (e.g. populateUsersAndFollows, populate posts)
- Domain-specific protos and handlers (social graph, posts, timelines)
