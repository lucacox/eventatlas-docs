---
post_title: "ADR-0002: Use React and TypeScript for the Web Frontend"
author1: Luca Cossaro
post_slug: adr-0002-use-react-for-frontend
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - frontend
  - react
  - typescript
  - vite
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to use React, TypeScript, and Vite for the EventAtlas web app.
post_date: 2026-08-16
---

# ADR-0002: Use React and TypeScript for the Web Frontend

## Status

Accepted

## Context

EventAtlas needs an interactive web application for exploring event-driven
topologies. Its primary interface will be a graph with pan, zoom, selection,
filters, custom node rendering, detail panels, and eventually live updates.

The frontend must consume the EventAtlas REST API and represent evolving
topology contracts safely. It should support rapid iteration on graph-specific
interactions without requiring server-side rendering or a full-stack web
framework.

This decision selects the frontend framework, language, and build tool. It does
not select the graph rendering library, state management approach, component
library, or deployment topology.

## Decision Drivers

- Strong support for interactive, component-based user interfaces.
- A mature ecosystem for graph visualization and custom node rendering.
- Static typing for API contracts, graph entities, and UI state.
- Fast local development and production builds with minimal configuration.
- Support for a client-side single-page application.
- Broad availability of testing and accessibility tooling.

## Decision

EventAtlas will implement its web frontend as a React single-page application
written in TypeScript and built with Vite.

TypeScript will use strict type checking. API response types must be modeled at
the frontend boundary and must not be replaced by untyped objects. The web
application will remain a separate repository and deployment artifact from the
Go backend.

The graph renderer will be selected through a focused comparison of React Flow
and Cytoscape.js using representative EventAtlas topology data. This ADR does
not prefer either library. The spike must evaluate graph size, layout quality,
custom nodes, interaction behavior, accessibility, and update performance.

## Consequences

### Positive

- React's component model supports custom nodes, panels, filters, and reusable
  interaction controls.
- TypeScript provides compile-time checks across API data, graph models, and
  component state.
- Vite provides a small setup and fast feedback during UI development.
- Both React Flow and Cytoscape.js can be evaluated without changing the
  selected application platform.
- The SPA can be built as static assets and deployed independently from the
  backend.

### Negative

- The frontend and backend use different languages and build toolchains.
- Client-side rendering shifts initial loading and graph processing to the
  browser.
- The React ecosystem introduces dependency churn and requires deliberate
  package maintenance.
- TypeScript types do not validate untrusted API responses at runtime.

### Risks

- Large topologies may exceed the rendering capacity of the chosen graph
  library. Benchmark representative graph sizes before committing to a
  renderer and add aggregation or virtualization when required.
- UI types may drift from backend contracts. Introduce contract generation or
  schema-based validation when the API stabilizes.
- Graph interactions can be inaccessible by default. Include keyboard
  navigation, focus management, and non-graph topology views in design and
  testing.
- Framework abstractions could leak into topology logic. Keep graph
  transformation and filtering logic independent from React components.

## Alternatives Considered

### Angular

Angular provides an integrated framework, strong TypeScript support, and
established application structure. It was not selected because EventAtlas does
not need its full framework surface, and React offers a more direct fit for the
custom graph component ecosystem under consideration.

### Vue or Svelte

Vue and Svelte provide productive component models and smaller application
footprints. They remain viable technologies, but React has broader direct
support among the graph UI libraries being evaluated for EventAtlas.

### Next.js

Next.js provides routing, server rendering, and full-stack deployment
capabilities. It was not selected because the initial EventAtlas UI is an
authenticated graph application with no identified server-rendering or search
indexing requirement. Vite supplies the needed SPA build surface with less
operational coupling.

### Vanilla TypeScript

A frontend without a component framework would minimize framework dependency.
It was not selected because EventAtlas requires coordinated interactive state,
custom graph elements, filters, and detail panels that benefit from a mature
component lifecycle and ecosystem.

## References

- [React documentation](https://react.dev/)
- [TypeScript documentation](https://www.typescriptlang.org/docs/)
- [Vite documentation](https://vite.dev/guide/)
- [React Flow documentation](https://reactflow.dev/)
- [Cytoscape.js documentation](https://js.cytoscape.org/)
