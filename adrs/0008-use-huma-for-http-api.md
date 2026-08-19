---
post_title: "ADR-0008: Use Huma for the HTTP API"
author1: Luca Cossaro
post_slug: adr-0008-use-huma-for-http-api
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - backend
  - api
  - openapi
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to use Huma v2 for the EventAtlas HTTP API adapter and OpenAPI documentation.
post_date: 2026-08-19
---

# ADR-0008: Use Huma for the HTTP API

## Status

Accepted

## Context

The first backend vertical slice must expose the current topology over REST and
JSON. The API needs predictable routing, typed request and response models,
input validation, an OpenAPI contract, and interactive documentation for local
development and future frontend integration.

Implementing these concerns independently with `net/http` would keep the
dependency set small but would require EventAtlas to maintain schema
generation, validation, error serialization, and documentation plumbing. A
larger web framework would provide those capabilities while also introducing
application patterns and dependencies that EventAtlas does not otherwise need.

The HTTP framework is an infrastructure choice. It must not shape the topology
domain or application use cases.

## Decision Drivers

- Generate an OpenAPI contract directly from the implemented operations.
- Provide interactive API documentation with minimal custom infrastructure.
- Preserve typed Go inputs and outputs at the HTTP boundary.
- Continue using the standard library HTTP server and lifecycle management.
- Keep domain and application packages independent from HTTP frameworks.
- Avoid maintaining a parallel hand-written API specification during the MVP.

## Decision

EventAtlas will use Huma v2 for its initial REST API adapter. The first
implementation uses Huma's `net/http` adapter and the standard library
`http.ServeMux`; no additional router framework is required.

Huma owns operation registration, HTTP content negotiation, error responses,
OpenAPI 3.1 generation, JSON Schema generation, and interactive documentation.
Stoplight Elements is the selected documentation renderer. The initial routes
are:

```text
GET /api/v1/topology
GET /openapi.json
GET /docs
```

Huma types, operation definitions, and public transport DTOs remain inside the
HTTP adapter under `internal/api`. Domain entities and application services
must not import Huma or depend on its request, response, validation, or error
types. The composition root passes application-owned use cases to the adapter
through typed interfaces.

The generated specification describes the public API but does not replace
domain validation. Transport DTOs may evolve independently from domain
entities when compatibility, security, or client requirements demand it.

## Consequences

### Positive

- OpenAPI and Stoplight Elements stay synchronized with registered operations.
- API handlers use typed inputs and outputs with less serialization plumbing.
- The backend retains the standard `net/http` server and graceful shutdown.
- Future request validation and generated clients can build on one contract.
- Huma remains replaceable because it is isolated at the adapter boundary.

### Negative

- Huma becomes a direct backend dependency that must be reviewed and upgraded.
- Generated schemas are influenced by Go DTO structure and Huma conventions.
- Code-first API changes can alter the contract unless OpenAPI output is tested.
- Interactive documentation may depend on browser-loaded renderer assets.

### Risks

- Framework types could leak into application code. Enforce package dependency
  direction and keep Huma imports inside `internal/api`.
- An innocuous DTO refactor could break frontend clients. Test important
  OpenAPI operations and introduce compatibility checks as the API stabilizes.
- A Huma upgrade could change generated schemas or error responses. Pin the
  major/minor version through Go modules and inspect API diffs during upgrades.

## Alternatives Considered

### Standard Library Only

`net/http` could serve the initial endpoint without an API framework. It was
not selected because EventAtlas would need separate validation, OpenAPI,
schema, and documentation implementation while gaining little at the domain
boundary.

### Chi with a Separate OpenAPI Toolchain

Chi provides lightweight routing and composes well with `net/http`. It was not
selected for the first slice because routing alone does not provide the typed
OpenAPI and validation workflow. Huma can still use Chi later if routing or
middleware requirements exceed `http.ServeMux`.

### Design-First OpenAPI with Generated Server Interfaces

A hand-authored specification and generated server stubs provide strong
contract governance. This was not selected while the first API and topology
read model are still evolving. It can be reconsidered when external consumers
require a separately governed, compatibility-versioned contract.

## References

- [Huma documentation](https://huma.rocks/)
- [Huma OpenAPI generation](https://huma.rocks/features/openapi-generation/)
- [Huma generated API documentation](https://huma.rocks/features/api-docs/)
- [ADR-0001: Use Go for the Backend](0001-use-go-for-backend.md)
- [ADR-0007: Start with a Single Backend Process](0007-start-with-single-backend-process.md)
