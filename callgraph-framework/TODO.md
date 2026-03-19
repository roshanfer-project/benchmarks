# TODO

## Out of Scope (per plan / user)

- **Rajomon/Dagor** — not needed initially
- **Redis** — no stateful backends
- **Prometheus** — not included initially

---

## In Social, Not Yet in Callgraph Framework

### Deployment modes
- Rajomon mode (app-grpc.yaml, rajomon-client, rajomon.env)
- Dagor mode (dagor.env, dagor interceptors)

### Observability
- Prometheus + Pushgateway
- OpenTelemetry (otelgrpc, otel_tool)
- Counter/metrics interceptor (utils/counter.go)
- Context propagation interceptor

### Infrastructure
- Redis (state storage for graph, posts, home, user)

### Application logic
- Init/population (e.g. populateUsersAndFollows, populate posts)
- Domain-specific protos and handlers (social graph, posts, timelines)
