---
post_title: "ADR-0005: Separate Declared and Observed Topology"
author1: Luca Cossaro
post_slug: adr-0005-separate-declared-and-observed-topology
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - topology
  - discovery
  - observability
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to preserve declared and observed topology as distinct evidence.
post_date: 2026-08-16
---

# ADR-0005: Separate Declared and Observed Topology

## Status

Accepted

## Context

EventAtlas reconstructs topology from sources with different knowledge and
authority. Messaging providers expose configured resources and routing, while
application telemetry reveals runtime behavior such as which service publishes
to or processes messages from a destination.

Neither source is complete by itself. Broker configuration does not reliably
identify the logical service behind every client connection or retain a
service-level history of published destinations. Runtime telemetry cannot
authoritatively describe resources that exist but have received no recent
traffic.

The same topology relationship may be supported by several sources, and those
sources may disagree. If EventAtlas collapses all evidence into one
undifferentiated graph, users cannot judge freshness, authority, or the reason
a relationship exists.

This decision defines evidence modes and their lifecycle. It does not select a
telemetry technology, retention duration, confidence algorithm, or persistence
schema.

## Decision Drivers

- Preserve the origin and authority of every topology relationship.
- Represent configured topology even when no traffic is observed.
- Represent application behavior that brokers do not declare.
- Reconcile provider snapshots without deleting runtime history incorrectly.
- Expose corroboration and discrepancies as useful operational information.
- Allow each evidence class to use an appropriate freshness policy.

## Decision

EventAtlas will distinguish two evidence modes: `declared` and `observed`.
Evidence mode is independent from the source system and from the edge kind.

Declared evidence describes topology asserted by a provider API or
configuration source. It is authoritative only within the provider instance
and discovery scope that produced it. A newer complete snapshot may retire
declared evidence absent from that same scope. Partial or failed discovery
must not trigger absence-based deletion.

Observed evidence describes application behavior measured at runtime. It is
time-bound and records when the behavior was first and last seen, together
with count or rate semantics when available. Missing observations in one
collection interval do not prove that a relationship has been removed.
Observed evidence becomes stale or expires according to an explicit retention
policy.

An edge can have evidence from multiple source instances and modes. EventAtlas
will retain those evidence records separately while presenting one normalized
relationship when their source, target, and edge kind match. It will not
silently promote observed evidence to declared evidence or overwrite one source
with another.

Disagreement is a first-class result. EventAtlas should expose relationships
that are declared but not recently observed, observed but not declared, or
reported differently by independent sources.

## Consequences

### Positive

- Users can distinguish configured intent from measured runtime behavior.
- Provider reconciliation cannot erase observations with a different
  lifecycle.
- Multiple sources can corroborate one relationship without losing
  provenance.
- Stale infrastructure and unexpected traffic paths can be detected as
  discrepancies.
- Future providers and observation technologies can use the same evidence
  model.

### Negative

- Storage and API representations must carry evidence and provenance instead
  of exposing only bare edges.
- The UI must explain source, freshness, and discrepancies without creating
  visual noise.
- Declared and observed data require different reconciliation and retention
  logic.
- One normalized edge may have several evidence records with conflicting
  metadata.

### Risks

- Users may interpret missing observed evidence as proof of inactivity. Expose
  sampling, freshness, and collection status with observed relationships.
- Stale observed evidence may clutter the graph. Define explicit retention and
  default visibility policies before production ingestion.
- Provider scopes may be modeled too broadly and cause incorrect deletion.
  Reconcile only complete snapshots with an identical source and scope.
- Source-specific metadata could obscure conflicts. Keep mode, source identity,
  first seen, and last seen as stable core fields.

## Alternatives Considered

### Store One Undifferentiated Topology Graph

This approach would simplify storage and API responses. It was not selected
because a relationship learned from configuration has different authority and
lifecycle from one inferred through sampled telemetry. Collapsing them would
make reconciliation unsafe and troubleshooting ambiguous.

### Use Declared Topology Only

Provider discovery would provide reliable broker configuration with no
telemetry dependency. It was not selected because broker APIs often cannot map
publishers and consumers to logical services or explain which configured paths
are active.

### Use Observed Topology Only

Runtime signals would show paths that applications actually use. It was not
selected because inactive resources and low-traffic paths would disappear,
sampling could hide valid relationships, and telemetry cannot authoritatively
describe broker configuration.

### Promote Observations After a Confidence Threshold

Frequently observed relationships could be converted into declared facts. It
was not selected because repetition does not change the authority of the
source. Confidence and evidence mode answer different questions and must remain
separate.

## References

- [EventAtlas domain model](../architecture/domain-model.md)
- [ADR-0003: Use a Provider-Neutral Domain Model](0003-use-provider-neutral-domain-model.md)
- [ADR-0004: Use PostgreSQL](0004-use-postgresql-for-persistence.md)
