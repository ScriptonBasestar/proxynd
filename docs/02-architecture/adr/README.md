# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the ProxyND project.

## What are ADRs?

An Architecture Decision Record (ADR) is a document that captures an important architectural decision made along with its context and consequences.

## ADR Format

We use the following format for our ADRs:

```markdown
# ADR-NNN: Title

## Status
[Accepted | Superseded by ADR-NNN | Deprecated | ...]

## Context
What is the issue that we're seeing that is motivating this decision?

## Decision
What is the change that we're proposing and/or doing?

## Consequences
What becomes easier or more difficult to do because of this change?
```

## Index

1. [ADR-001: Use Architecture Decision Records](001-use-adrs.md)
2. [ADR-002: Adopt Fiber as Web Framework](002-adopt-fiber-framework.md)
3. [ADR-003: Implement Repository Pattern for Data Access](003-repository-pattern.md)
4. [ADR-004: Use Dependency Injection Container](004-dependency-injection.md)
5. [ADR-005: Separate Configuration from Code](005-configuration-separation.md)
6. [ADR-006: Implement Structured Logging](006-structured-logging.md)
7. [ADR-007: Cache Strategy and TTL Management](007-cache-strategy.md)
8. [ADR-008: Multi-Proxy Architecture](008-multi-proxy-architecture.md)

## Creating New ADRs

To create a new ADR:

1. Copy the template from `template.md`
2. Name it `NNN-short-title.md` where NNN is the next number
3. Fill in all sections
4. Update this README with a link to the new ADR
5. Submit a PR with the new ADR

## Tools

We recommend using [adr-tools](https://github.com/npryce/adr-tools) for managing ADRs, though it's not required.