---
post_title: "ADR-0001: Use Go for the Backend"
author1: Luca Cossaro
post_slug: adr-0001-use-go-for-backend
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - backend
  - go
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to use Go for the EventAtlas backend and collectors.
post_date: 2026-08-16
---

# ADR-0001: Use Go for the Backend

## Status

Accepted

## Context

EventAtlas needs a backend that can discover messaging infrastructure, ingest
topology observations, reconcile state, persist data, and expose an HTTP API.
Its first provider will integrate with NATS and JetStream, and application
behavior will later be observed through OpenTelemetry.

The backend will perform long-running network and collection workloads. It
must be straightforward to operate locally and in containers, and it should
have a low distribution burden for users running EventAtlas alongside their
messaging infrastructure.

This decision selects the backend implementation language. It does not decide
the backend process boundaries, package structure, persistence design, or web
frontend stack.

## Decision Drivers

- Mature and actively maintained NATS and JetStream client support.
- First-class support for OpenTelemetry instrumentation and export.
- Good fit for concurrent network services and long-running collectors.
- Simple distribution as a self-contained native executable or container.
- Predictable resource usage for infrastructure tooling.
- Strong standard library support for HTTP, concurrency, and testing.

## Decision

EventAtlas will use Go for its backend and first-party collection components.
This includes the backend executable, domain and topology logic, discovery
providers, observation ingestion, persistence adapters, and HTTP API.

The project will use a supported stable Go release, declared explicitly in
the backend module and continuous integration configuration. Go code will
follow standard tooling and conventions, including `go fmt`, `go vet`, and
the built-in test framework.

Provider boundaries must remain explicit Go interfaces owned by the core.
Choosing Go does not allow provider-specific NATS concepts to leak into the
vendor-neutral domain model.

## Consequences

### Positive

- EventAtlas can use the official `nats.go` client without a language bridge.
- A single native executable simplifies local and container distribution.
- Goroutines and context cancellation suit concurrent discovery and
  collection workloads.
- The standard library covers the initial HTTP and testing requirements with
  few mandatory framework dependencies.
- Go has established OpenTelemetry APIs and exporters.

### Negative

- Contributors must be comfortable with Go's explicit error handling and type
  system.
- The frontend and backend use different languages and build toolchains.
- Advanced type-level domain modeling is more limited than in some
  alternatives.
- Native executables must be built for each supported operating system and
  architecture.

### Risks

- Provider SDKs or telemetry libraries may introduce breaking changes. Pin
  dependencies and isolate them behind adapters owned by EventAtlas.
- Unbounded goroutines could make collectors difficult to operate. Use
  structured cancellation, bounded concurrency, and load-focused tests.
- A convenient NATS client could bias the core model toward NATS. Enforce the
  provider-neutral model through package boundaries and cross-provider tests.

## Alternatives Considered

### TypeScript and Node.js

TypeScript would allow one language across the frontend and backend and offers
fast application development. It was not selected because EventAtlas is an
infrastructure collector with long-running concurrent workloads, and Go offers
simpler native distribution and a stronger direct fit with the NATS ecosystem.

### Rust

Rust offers excellent runtime efficiency, memory safety, and native binaries.
It was not selected because its implementation complexity and compile-time
cost would slow the initial product iteration without solving an identified
performance or safety constraint that Go cannot meet.

### Java or Kotlin

The JVM ecosystem provides mature messaging, telemetry, and web frameworks.
It was not selected because the runtime and deployment footprint are heavier
than needed for the initial collector and API, while providing no decisive
advantage for the first NATS integration.

## References

- [EventAtlas domain model](../architecture/domain-model.md)
- [Go documentation](https://go.dev/doc/)
- [NATS Go client](https://github.com/nats-io/nats.go)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)
