---
post_title: "ADR-0007: Start with a Single Backend Process"
author1: Luca Cossaro
post_slug: adr-0007-start-with-single-backend-process
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - backend
  - deployment
  - architecture
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to deploy the initial EventAtlas backend as one process.
post_date: 2026-08-16
---

# ADR-0007: Start with a Single Backend Process

## Status

Accepted

## Context

The EventAtlas backend contains several responsibilities: provider discovery,
periodic reconciliation, observation ingestion, topology normalization,
persistence, and the REST API. These responsibilities have distinct domain
boundaries but initially operate on the same topology model and database.

The product is at an early stage, and its traffic, collection volume, scaling
profile, and operational service-level objectives are not yet known. Splitting
the backend into independently deployed services would require network
contracts, failure handling, deployment coordination, observability, and local
development infrastructure before those boundaries have been validated.

This decision defines the initial runtime and deployment boundary. It does not
merge all code into one package, embed PostgreSQL, include the web frontend in
the backend process, or prevent future extraction of components.

## Decision Drivers

- Deliver the first end-to-end topology slice with low operational overhead.
- Keep transactions and consistency local while workflows are evolving.
- Provide one backend executable and container for local and initial use.
- Avoid distributed failure modes before independent scaling is required.
- Preserve clear internal ownership boundaries for future extraction.
- Make development, testing, debugging, and deployment straightforward.

## Decision

EventAtlas will initially deploy its backend as one Go executable running in
one operating system process. The process will contain:

- provider discovery and reconciliation scheduling;
- observation ingestion;
- topology normalization and application workflows;
- persistence adapters;
- the REST API and operational endpoints.

The backend will use explicit internal module boundaries following ports and
adapters. Domain and application logic must not depend on HTTP, PostgreSQL,
NATS SDKs, or OpenTelemetry transport details. Providers, observation sources,
storage, and API handlers remain adapters around application-owned contracts.

Components will communicate through in-process calls and typed interfaces. The
initial backend will not introduce an internal message bus, service-to-service
API, or distributed workflow solely to separate these responsibilities.

Background collection and reconciliation will use bounded concurrency,
timeouts, cancellation, and coordinated process lifecycle management. The
process may run multiple goroutines; the decision concerns deployment and
failure boundaries, not sequential execution.

PostgreSQL, messaging brokers, the web frontend, and an optional OpenTelemetry
Collector remain external deployment components.

Separating a component into another process requires a superseding ADR and at
least one demonstrated need:

- materially different scaling characteristics;
- unacceptable resource contention or failure propagation;
- a required security or network isolation boundary;
- an independent deployment cadence that improves delivery;
- measured availability objectives that cannot be met in one process.

## Consequences

### Positive

- The initial deployment has one backend executable and container.
- Internal workflows avoid network latency and distributed coordination.
- Topology reconciliation can use local transactions and shared application
  contracts.
- Local development and end-to-end testing require fewer running components.
- Module boundaries can evolve before they become remote contracts.
- Operational telemetry and configuration have one backend entry point.

### Negative

- API and collection workloads share CPU, memory, and a failure boundary.
- The entire backend is deployed when one component changes.
- Components cannot scale independently.
- A blocking or memory-intensive collector can affect API responsiveness.
- Future extraction will require explicit network contracts and data ownership
  decisions.

### Risks

- Provider failures could reduce API availability. Isolate background work with
  timeouts, bounded queues, error handling, and health reporting.
- Collection load could starve request handling. Measure resource usage and
  enforce concurrency and memory limits.
- Internal boundaries could erode because calls are local. Enforce dependency
  direction through packages, interfaces, and architecture tests where useful.
- The process could become difficult to start and stop cleanly. Use one
  lifecycle coordinator with cancellation and graceful shutdown.
- Premature extraction could be justified by preference rather than evidence.
  Require measured criteria and a superseding decision record.

## Alternatives Considered

### Start with Independent Microservices

Discovery, ingestion, topology processing, and API responsibilities could each
be deployed separately. This was not selected because it would introduce
distributed contracts, failure modes, and operational overhead before workload
and ownership boundaries are known.

### Separate Collector and API Processes Immediately

A collector process could write topology to PostgreSQL while an API process
serves reads. This would isolate collection failures and allow independent
scaling. It was not selected because the initial workloads do not demonstrate
that need, and two processes would still require coordination of migrations,
health, version compatibility, and local environments.

### Run Discovery as Scheduled Serverless Jobs

Provider discovery could run as external scheduled functions or jobs. This was
not selected because discovery also needs advisory subscriptions, periodic
reconciliation, shared identity rules, and predictable access to application
state. A long-running process is the simpler initial runtime.

### Load Providers as Separate Plugin Processes

Out-of-process plugins could isolate provider dependencies and failures. This
was not selected because EventAtlas has one initial provider and no proven need
for independent provider distribution. Typed in-process adapters preserve the
boundary without introducing a plugin protocol.

## References

- [ADR-0001: Use Go for the Backend](0001-use-go-for-backend.md)
- [ADR-0004: Use PostgreSQL](0004-use-postgresql-for-persistence.md)
- [ADR-0005: Separate Declared and Observed Topology](0005-separate-declared-and-observed-topology.md)
- [ADR-0006: Treat OpenTelemetry as an Observation Source](0006-treat-opentelemetry-as-observation-source.md)
