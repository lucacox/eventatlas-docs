---
post_title: EventAtlas Documentation
author1: Luca Cossaro
post_slug: eventatlas-documentation
featured_image: ""
categories:
  - architecture
tags:
  - eventatlas
  - documentation
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Architecture and development documentation for EventAtlas.
post_date: 2026-08-16
---

# EventAtlas Documentation

**EventAtlas — Discover and visualize your event-driven topology.**

This repository contains cross-project architecture, decision records,
integration guides, and development documentation for EventAtlas.

EventAtlas discovers and visualizes event-driven systems by combining topology
declared by messaging infrastructure with runtime behavior observed through
application telemetry. NATS and JetStream are the first integration, while the
core model remains independent of any messaging provider.

## Project Principles

- Model event-driven topology using vendor-neutral concepts.
- Keep declared infrastructure separate from observed application behavior.
- Preserve provider-specific details without promoting them into the core.
- Start with one Go process, one PostgreSQL database, and one React SPA.
- Introduce operational complexity only when product requirements justify it.

## Documentation

### Architecture

- [Domain model](architecture/domain-model.md) defines the core entities,
  relationships, evidence, snapshots, and provider boundaries.
- [Topology persistence](architecture/persistence.md) defines source-scoped
  PostgreSQL storage, reconciliation, and startup behavior.
- [Observation architecture](architecture/observations.md) defines OTLP trace
  ingestion, normalization, retention, correlation, and merged projections.

### Architecture Decision Records

- [ADR template](adrs/adr_template.md) defines the format used to record
  architectural decisions.
- [ADR-0001: Use Go for the Backend](adrs/0001-use-go-for-backend.md)
  describes the decision to use Go for the backend and first-party collectors.
- [ADR-0002: Use React and TypeScript](adrs/0002-use-react-for-frontend.md)
  defines the frontend platform and defers the graph renderer to a spike.
- [ADR-0003: Use a Provider-Neutral Domain Model](adrs/0003-use-provider-neutral-domain-model.md)
  keeps provider-native concepts behind normalization boundaries.
- [ADR-0004: Use PostgreSQL](adrs/0004-use-postgresql-for-persistence.md)
  selects the initial durable persistence layer.
- [ADR-0005](adrs/0005-separate-declared-and-observed-topology.md)
  separates declared topology from observed topology evidence.
- [ADR-0006](adrs/0006-treat-opentelemetry-as-observation-source.md)
  treats OpenTelemetry as an observation source, not a broker provider.
- [ADR-0007](adrs/0007-start-with-single-backend-process.md)
  starts the backend as one process with explicit internal boundaries.
- [ADR-0008](adrs/0008-use-huma-for-http-api.md)
  selects Huma for the HTTP adapter, generated OpenAPI, and Stoplight Elements.
- [ADR-0009](adrs/0009-model-consumer-selectors-as-binding-metadata.md)
  keeps provider-native consumer selectors on consumer-binding evidence.
- [ADR-0010](adrs/0010-ingest-observations-from-otlp-traces.md)
  selects OTLP/HTTP traces as the first observation ingestion boundary.

The initial ADR set will document the backend and frontend stacks, the
provider-neutral model, persistence, topology evidence, OpenTelemetry's role,
and the single-process deployment strategy.

## Planned Documentation

Documentation will be added incrementally as the corresponding design or
implementation work begins:

- NATS and JetStream integration;
- local development, testing, and contribution guides.

Planned documents are not placeholders for decisions. Each document should
describe behavior that has been agreed upon or implemented.

## Repository Map

| Repository | Responsibility |
| --- | --- |
| [eventatlas](https://github.com/lucacox/eventatlas) | Backend, discovery, topology engine, storage, and API |
| [eventatlas-web](https://github.com/lucacox/eventatlas-web) | Web application and graph visualization |
| [eventatlas-deploy](https://github.com/lucacox/eventatlas-deploy) | Local environments, containers, and deployment assets |
| [eventatlas-docs](https://github.com/lucacox/eventatlas-docs) | Cross-project architecture and documentation |

## Project Status

EventAtlas has an end-to-end read-only vertical slice for NATS and JetStream,
including periodic discovery, PostgreSQL persistence, a topology API, an
interactive web graph, a complete local stack, and CI validation. OpenTelemetry
observation ingestion is the current implementation phase. APIs, schemas, and
deployment contracts remain expected to evolve before the first usable release.

## License

See the [LICENSE](LICENSE) file for license information.
