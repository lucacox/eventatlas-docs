---
post_title: "ADR-0006: Treat OpenTelemetry as an Observation Source"
author1: Luca Cossaro
post_slug: adr-0006-treat-opentelemetry-as-observation-source
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - opentelemetry
  - observability
  - topology
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to use OpenTelemetry as a runtime observation source.
post_date: 2026-08-16
---

# ADR-0006: Treat OpenTelemetry as an Observation Source

## Status

Accepted

## Context

Broker discovery can describe configured messaging resources and routing, but
it often cannot identify which logical service publishes to a destination or
executes a consumer. NATS, for example, does not provide a durable historical
mapping from service identity to every subject it has published.

Applications already expose runtime behavior through observability signals.
OpenTelemetry provides standard resource identity, messaging attributes,
traces, metrics, and context propagation across supported languages and
instrumentation libraries.

OpenTelemetry cannot authoritatively enumerate broker resources or determine
that configured topology has been removed. Treating it as a discovery provider
would conflate sampled application behavior with declared infrastructure and
violate the evidence lifecycle established by ADR-0005.

This decision defines OpenTelemetry's architectural role. It does not select
an OpenTelemetry Collector deployment, ingestion protocol, sampling policy, or
final trace and metric pipeline.

## Decision Drivers

- Correlate logical services with messaging destinations and consumers.
- Reuse an open, vendor-neutral observability standard.
- Avoid application code that is coupled directly to EventAtlas.
- Preserve the distinction between measured behavior and broker configuration.
- Support trace context propagation across asynchronous message boundaries.
- Control telemetry cardinality while retaining useful topology identity.

## Decision

EventAtlas will treat OpenTelemetry as an observation source, not as a broker
discovery provider. Facts derived from OpenTelemetry will use evidence mode
`observed` and source system `opentelemetry`.

OpenTelemetry observations may support relationships such as:

- service `publishes` destination;
- service `consumes` destination;
- consumer `executed_by` service.

Broker providers remain responsible for declared resources and relationships,
such as destination capture, stream ownership, consumer configuration, and
routing. OpenTelemetry observations will correlate with provider topology
through shared service, broker, destination, and consumer identity rules.

Messaging instrumentation should be implemented in shared application
libraries at common publish and consume boundaries. Direct publishing, async
publishing, and outbox delivery must use the same instrumented publishing
pipeline so they produce equivalent topology observations.

Instrumentation will use OpenTelemetry resource and messaging semantic
conventions where applicable. `service.name` identifies the logical service,
not a pod or process. Physical and logical destination names must remain
distinct when dynamic addresses would cause unbounded cardinality.

Publish instrumentation will inject trace context into message headers.
Consumer instrumentation will extract that context and create the processing
span with the appropriate parent or link semantics. Application business code
must not implement propagation manually.

Message IDs, trace IDs, correlation IDs, user IDs, and payload values must not
be metric attributes. They may appear in traces when appropriate and compliant
with data-handling policy, but they do not participate in topology identity.

## Consequences

### Positive

- EventAtlas can observe service-level messaging relationships that broker
  configuration does not declare.
- Applications use a standard telemetry API rather than an EventAtlas-specific
  reporting protocol.
- Trace propagation connects messaging operations with upstream and downstream
  application work.
- The same observation model can support services implemented in different
  languages.
- Shared instrumentation applies consistent identity and cardinality rules.

### Negative

- Topology completeness depends on application instrumentation coverage.
- Sampling and export failures can hide valid runtime relationships.
- Semantic conventions and instrumentation libraries can evolve independently
  from EventAtlas.
- Shared messaging libraries require coordinated releases across applications.
- Telemetry ingestion, aggregation, and retention add operational work.

### Risks

- Dynamic destinations could create excessive metric cardinality. Require
  explicit logical destination patterns where physical names are unbounded.
- Incorrect `service.name` values could fragment one logical service into many
  nodes. Validate service resource configuration and keep instance identity in
  separate attributes.
- Trace sampling could be mistaken for absence of traffic. Preserve collection
  status and use suitable aggregated metrics for topology presence and volume.
- Duplicate instrumentation could count one operation more than once. Keep one
  common publish and consume boundary and test direct and outbox paths.
- OpenTelemetry attribute changes could leak into the domain. Normalize signals
  in the observation adapter before creating core evidence.

## Alternatives Considered

### Model OpenTelemetry as a Discovery Provider

This approach would give every integration one common provider interface. It
was not selected because OpenTelemetry reports sampled runtime behavior rather
than authoritative broker configuration and therefore requires a different
ingestion contract and lifecycle.

### Use Broker Monitoring Only

Broker monitoring avoids application instrumentation and can expose
connections, traffic, and resource health. It was not selected as the sole
source because connection identity does not reliably provide durable logical
service-to-destination relationships and cannot propagate application traces.

### Build an EventAtlas-Specific Telemetry SDK

A custom SDK could emit exactly the facts EventAtlas requires. It was not
selected because it would couple applications to EventAtlas, duplicate
observability standards, and require language-specific libraries and pipelines.

### Infer Topology from Network Traffic

Network observation could avoid application changes. It was not selected
because encryption, protocol details, multiplexed connections, and missing
logical service identity make reliable semantic reconstruction difficult.

## References

- [EventAtlas domain model](../architecture/domain-model.md)
- [ADR-0005: Separate Declared and Observed Topology](0005-separate-declared-and-observed-topology.md)
- [OpenTelemetry documentation](https://opentelemetry.io/docs/)
- [OpenTelemetry messaging semantic conventions](https://opentelemetry.io/docs/specs/semconv/messaging/)
