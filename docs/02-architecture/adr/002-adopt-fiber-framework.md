# ADR-002: Adopt Fiber as Web Framework

## Status

Accepted

## Context

ProxyND requires a high-performance HTTP server to handle package manager proxy requests efficiently. The choice of web framework significantly impacts:

- Request handling performance
- Development velocity
- Middleware ecosystem
- Learning curve for developers
- Long-term maintainability

Requirements:
- High throughput for serving package files (potentially large)
- Low latency for metadata requests
- Support for streaming large files
- Robust middleware support (auth, logging, metrics)
- Good developer experience

## Decision

We have chosen Fiber v2 as the web framework for ProxyND. Fiber is an Express-inspired web framework built on top of Fasthttp, the fastest HTTP engine for Go.

Key factors in this decision:
1. **Performance**: Built on Fasthttp, significantly faster than net/http
2. **Familiar API**: Express-like API reduces learning curve
3. **Rich middleware ecosystem**: Extensive built-in and community middleware
4. **Zero memory allocation**: Optimized for high-load scenarios
5. **Built-in features**: WebSocket, streaming, static file serving

## Consequences

### Positive

- Superior performance compared to standard library and other frameworks
- Faster development with familiar Express-like API
- Rich middleware ecosystem reduces development time
- Excellent performance for serving large package files
- Built-in support for streaming responses
- Active community and good documentation

### Negative

- Not compatible with net/http interfaces
- Fewer third-party libraries compared to standard library
- Fasthttp has some quirks that differ from net/http
- Context handling is different from standard library
- Team needs to learn Fiber-specific patterns

### Neutral

- Commits to Fiber ecosystem for HTTP handling
- May need custom adapters for some net/http libraries
- Performance gains most noticeable under high load

## Alternatives Considered

1. **Standard library (net/http)**
   - Pros: Standard, wide compatibility, well-understood
   - Cons: Lower performance, more boilerplate code
   - Rejected due to performance requirements

2. **Gin**
   - Pros: Popular, good performance, extensive middleware
   - Cons: Not as fast as Fiber, uses standard net/http
   - Rejected for performance reasons

3. **Echo**
   - Pros: Minimalist, good performance, clean API
   - Cons: Smaller ecosystem, not as fast as Fiber
   - Rejected for performance and ecosystem reasons

4. **Chi**
   - Pros: Lightweight, compatible with net/http
   - Cons: More minimal, less built-in functionality
   - Rejected due to need for more features out-of-box

## References

- [Fiber Documentation](https://docs.gofiber.io/)
- [Fasthttp Benchmarks](https://github.com/valyala/fasthttp#benchmarks)
- [TechEmpower Framework Benchmarks](https://www.techempower.com/benchmarks/)
- Internal performance testing results
