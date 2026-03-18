# TODO

## Out of Scope (per plan / user)

- **Weight on edges** — ignored; all edges treated as 1 call
- **USER server** — no server; USER is the external caller
- **Sidecar/Envoy mode** — plain only for now
- **Rajomon/Dagor** — not needed initially
- **Redis** — no stateful backends
- **Prometheus** — not included initially

---

## In Social, Not Yet in Callgraph Framework

### Deployment modes
- Sidecar mode (sidecar container, sidecar-configs.yaml, multi-API, which includes separate port for every API and building wrapper scripts accordingly)
- Rajomon mode (app-grpc.yaml, rajomon-client, rajomon.env)
- Dagor mode (dagor.env, dagor interceptors)
- Ingress (for sidecar mode)

### Observability
- Prometheus + Pushgateway
- OpenTelemetry (otelgrpc, otel_tool)
- Counter/metrics interceptor (utils/counter.go)
- Context propagation interceptor

### Infrastructure
- Redis (state storage for graph, posts, home, user)
- Different ports per service (social: 2001–2007)

### Application logic
- Init/population (e.g. populateUsersAndFollows, populate posts)
- Domain-specific protos and handlers (social graph, posts, timelines)
