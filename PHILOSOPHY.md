# Philosophy

These principles guide all technical decisions in Jumpgate.

## Design Principles

- **Server-side rendering preferred** — The server renders HTML, the browser displays it. Client-side JS is the exception, not the rule. If in doubt, render on the server.

- **Minimal overhead** — Minimize network requests, JS execution, CPU usage, and page weight. Every dependency, every request, every byte must justify itself.

- **Iterate, don't over-plan** — Build the simple version, ship it, improve based on real usage. Don't design for hypothetical future requirements.

- **Security by default** — `private=true` out of the box, auth required on admin.

- **CDN for static assets** — JS libraries and icon fonts load from CDN. Leverage browser caching. Don't self-host what CDNs do better.

- **Prefer consistency** — When two approaches work equally well, choose the one that matches existing patterns. Consistency makes systems predictable and easier to reason about.

## Implementation Principles

- **Simplicity first** — Minimal dependencies, stdlib where sufficient.

- **Explicit over abstract** — Separate functions for separate operations, no clever indirection.

- **Typed boundaries** — Go structs enforce schema shape. `Datasource` interface defines the backend contract. Templ enforces template correctness at compile time.

- **No translation layers** — Templates consume Go structs directly, no mapping or adapter layers.

- **Interface-driven storage** — The `Datasource` interface decouples handlers from the database implementation.

- **Single binary** — Compiles to one binary with embedded static assets. No runtime filesystem dependencies beyond the database and icon cache.

## Goals

- Be a useful daily-driver startpage/dashboard
- Stay simple enough that one person can understand the entire codebase
- Single binary deployment with minimal configuration
- Work well behind a reverse proxy with delegated auth
- Support multiple color themes for personalization
- Private-by-default for self-hosted security

## Non-Goals

- Built-in user accounts or authentication (delegated to reverse proxy)
- Multi-user with separate dashboards (one dashboard, auth controls visibility)
- Plugin system or extensibility framework
- Self-hosted fonts or JS libraries (CDN preferred)
- TypeScript, React, or any JS framework
- ORM or database abstraction beyond the Datasource interface
- Real-time collaboration or multi-user editing
