---
post_title: "ADR-0009: Model Consumer Selectors as Binding Metadata"
author1: Luca Cossaro
post_slug: adr-0009-model-consumer-selectors-as-binding-metadata
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - domain-model
  - topology
  - consumers
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to model provider-native consumer selectors as metadata on consumer bindings rather than topology destinations.
post_date: 2026-08-19
---

# ADR-0009: Model Consumer Selectors as Binding Metadata

## Status

Accepted

## Context

A broker consumer is attached to a resource or destination and may include a
provider-native selector. A JetStream consumer, for example, belongs to one
stream and may configure filter subjects such as `orders.*`. Kafka group
subscriptions may use topic patterns, while RabbitMQ routing constraints are
expressed on bindings.

The initial domain model represented a consumer selector as a `filters` edge
from `Consumer` to `Destination`. Normalizing `orders.*` this way created a
destination node even when no application publishes to that address and the
stream itself captures concrete subjects such as `orders.created`. The graph
then suggested that the consumer read directly from the pattern instead of
receiving messages through its stream.

This obscured the canonical message path:

```text
Service -> Destination -> Messaging Resource -> Consumer -> Service
```

It also contradicted the provider boundary in ADR-0003, which assigns native
wildcard and filter interpretation to adapters rather than the core.

## Decision Drivers

- Keep the navigable graph focused on actual messaging entities and paths.
- Preserve the fact that a consumer selector is scoped to its binding.
- Avoid treating provider-native wildcard expressions as publishable addresses.
- Retain selector configuration for diagnostics and UI presentation.
- Keep wildcard interpretation outside the provider-neutral core.
- Support future providers without introducing a NATS-specific abstraction.

## Decision

EventAtlas will not represent consumer selectors as topology nodes or edges.
The `filters` edge kind is removed from the core domain model.

The declared consumer path is represented by one `has_consumer` relationship:

```text
Messaging Resource or Destination -> has_consumer -> Consumer
```

Provider-native selector configuration qualifies that relationship and is
stored as namespaced metadata on its declared evidence. The relationship
remains meaningful without interpreting the metadata. For JetStream, the
initial metadata contract is:

```text
nats.jetstream.filter_mode
nats.jetstream.filter_subjects
```

`filter_mode` is `all` when the consumer has no explicit selector and
`subjects` when explicit filter subjects are configured. `filter_subjects` is
a deterministic JSON array. The NATS adapter validates, sorts, and de-duplicates
the native values but the core does not interpret their wildcard semantics.

A provider must not create a `Destination` solely because a selector mentions
an address or pattern. It may create pattern destinations for independently
navigable capture or routing configuration, but consumer selector metadata
alone is insufficient.

The canonical end-to-end path is:

```text
publishing Service
  -> publishes -> Destination
  -> captured_by/routes_to -> Messaging Resource or Destination
  -> has_consumer -> Consumer
  -> executed_by -> processing Service
```

`publishes`, `consumes`, and `executed_by` are observed application facts.
`consumes` may provide an additional direct correlation between a service and
a destination, especially when no durable consumer identity is available, but
it does not replace the broker-declared path.

## Consequences

### Positive

- Consumer filters appear in the context of the stream or destination they qualify.
- The graph no longer contains synthetic destinations created only from selectors.
- The primary message path has one consistent left-to-right direction.
- Provider-specific wildcard semantics remain isolated in adapters.
- Existing evidence metadata preserves diagnostics without expanding the core.

### Negative

- Generic graph algorithms cannot interpret provider selector expressions.
- API clients that want to render selectors must understand namespaced metadata.
- Existing snapshots and frontend edge enumerations containing `filters` are incompatible.

### Risks

- Evidence metadata could become a substitute for stable domain concepts. If
  cross-provider selector queries become a product requirement, promote a
  provider-neutral binding constraint value object through a separate ADR.
- A selector may refer to destinations not currently discovered. Preserve the
  native expression without inventing unresolved destination nodes.
- UI clients could present provider metadata as a message path. Display
  selectors as binding properties, never as standalone relationships.

## Alternatives Considered

### Keep `Consumer -> filters -> Destination`

This preserves selector navigation but was rejected because it conflates a
scoped subscription expression with an independently addressable destination.

### Store Selectors on the Consumer Node

This is simple but loses binding scope when a consumer is associated with more
than one resource or destination.

### Add a First-class Consumer Binding Entity

A dedicated binding entity could model selectors with strong types. It was not
selected for the MVP because the existing `has_consumer` relationship plus
evidence metadata preserves the required information without adding a node
that users need to navigate. It can be reconsidered if binding lifecycle or
cross-provider querying becomes a core product capability.

## References

- [EventAtlas domain model](../architecture/domain-model.md)
- [ADR-0003: Use a Provider-neutral Domain Model](0003-use-provider-neutral-domain-model.md)
- [ADR-0005: Separate Declared and Observed Topology](0005-separate-declared-and-observed-topology.md)
- [ADR-0006: Treat OpenTelemetry as an Observation Source](0006-treat-opentelemetry-as-observation-source.md)
