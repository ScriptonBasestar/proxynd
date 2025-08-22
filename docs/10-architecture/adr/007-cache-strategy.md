# ADR-007: Cache Strategy and TTL Management

## Status

Accepted

## Context

ProxyND acts as a caching proxy for various package managers. Effective caching is crucial for:
- Reducing bandwidth usage
- Improving package download speeds
- Reducing load on upstream servers
- Providing offline capability

Challenges:
- Different package types have different update patterns
- Metadata changes more frequently than packages
- Need to balance freshness with performance
- Storage limitations require cache eviction
- Must respect upstream cache headers

Current issues:
- Fixed TTL for all content types
- No consideration of upstream cache headers
- No intelligent cache eviction
- No stale-while-revalidate support

## Decision

Implement a sophisticated cache strategy with the following components:

1. **Hierarchical TTL Configuration**:
   ```yaml
   cache:
     ttl: 3600  # Default 1 hour
     package_ttls:
       apt: 3600
       npm: 1800  # 30 minutes - more frequent updates
       maven: 5400  # 90 minutes
     pattern_ttls:
       "*-SNAPSHOT*": 300  # 5 minutes for snapshots
       "*.json": 600       # 10 minutes for metadata
   ```

2. **Cache Header Respect**:
   - Honor Cache-Control headers from upstream
   - Implement max-age and s-maxage support
   - Configure min/max TTL boundaries
   - Support for ETags and conditional requests

3. **Stale-While-Revalidate**:
   - Serve stale content while fetching fresh
   - Configurable stale window
   - Background refresh for popular content

4. **Eviction Strategy**:
   - LRU (Least Recently Used) as primary strategy
   - Size-based limits with high/low watermarks
   - Protected entries for frequently accessed content

5. **Cache Warming**:
   - Pre-fetch popular packages
   - Refresh before expiration for hot content

## Consequences

### Positive

- Optimal cache hit rates for different content types
- Reduced upstream bandwidth usage
- Better user experience with stale-while-revalidate
- Respect for upstream caching policies
- Efficient storage usage with smart eviction
- Improved performance for frequently accessed packages

### Negative

- Increased complexity in cache management
- More configuration options to understand
- Potential for stale content if misconfigured
- Background refresh adds CPU/network overhead

### Neutral

- Requires monitoring of cache effectiveness
- Need to tune TTLs based on usage patterns
- Storage requirements depend on eviction settings

## Implementation Details

1. **Cache Storage Structure**:
   ```
   storage/
   ├── apt/
   │   ├── packages/
   │   └── metadata/
   ├── maven/
   │   ├── artifacts/
   │   └── metadata/
   └── cache.db  # Cache metadata
   ```

2. **Cache Metadata Tracking**:
   - Last access time
   - Hit count
   - Size
   - TTL and expiration
   - ETag/Last-Modified

3. **Background Tasks**:
   - Periodic eviction runs
   - Stale content refresh
   - Cache statistics collection

## Alternatives Considered

1. **Simple TTL-only caching**
   - Pros: Simple to implement and understand
   - Cons: Not optimal for varied content types
   - Rejected for lack of flexibility

2. **No caching (pure proxy)**
   - Pros: Always fresh content
   - Cons: High bandwidth, slow performance
   - Rejected for performance reasons

3. **External cache (Redis/Memcached)**
   - Pros: Proven solutions, distributed caching
   - Cons: Additional dependency, complexity
   - Rejected for operational simplicity

4. **CDN-style caching**
   - Pros: Sophisticated caching rules
   - Cons: Complex implementation
   - Rejected as overkill for our use case

## Monitoring and Metrics

Track the following metrics:
- Cache hit/miss ratio
- Cache size and eviction rate
- Average response time (cached vs uncached)
- Bandwidth savings
- Popular packages
- Stale serves

## References

- [HTTP Caching RFC 7234](https://tools.ietf.org/html/rfc7234)
- [Stale-While-Revalidate RFC 5861](https://tools.ietf.org/html/rfc5861)
- [Cache-Control for Civilians](https://csswizardry.com/2019/03/cache-control-for-civilians/)
- Current implementation in `internal/services/cache/`
